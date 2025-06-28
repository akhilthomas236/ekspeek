// Package k8s contains Kubernetes-related functionality
package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetEFSCSIStatus(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "efs-csi-controller-0",
				Namespace: "kube-system",
				Labels: map[string]string{
					"app": "efs-csi-controller",
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
	)

	client, err := NewKubeClient(clientset)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	status, err := client.GetEFSCSIStatus(context.Background())
	if err != nil {
		t.Fatalf("GetEFSCSIStatus failed: %v", err)
	}

	if len(status) != 1 {
		t.Errorf("Expected 1 pod status, got %d", len(status))
	}

	if status[0].Status != string(corev1.PodRunning) {
		t.Errorf("Expected pod status %s, got %s", corev1.PodRunning, status[0].Status)
	}
}

func TestGetClusterResources(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: "test-node",
			Containers: []corev1.Container{
				{
					Name: "test-container",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(node, pod)
	client, err := NewKubeClient(clientset)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	resources, err := client.GetClusterResources(context.Background())
	if err != nil {
		t.Fatalf("GetClusterResources failed: %v", err)
	}

	expectedCPUPercent := 25.0 // 1 CPU requested out of 4 total
	if resources.CPUPercentage != expectedCPUPercent {
		t.Errorf("Expected CPU percentage %f, got %f", expectedCPUPercent, resources.CPUPercentage)
	}
}

func TestValidatePodWebIdentityToken(t *testing.T) {
	testCases := []struct {
		name        string
		pod         *corev1.Pod
		expectError bool
	}{
		{
			name: "Valid IRSA configuration",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "aws-token",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											ServiceAccountToken: &corev1.ServiceAccountTokenProjection{},
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{Name: "AWS_ROLE_ARN", Value: "arn:aws:iam::123456789012:role/test-role"},
								{Name: "AWS_WEB_IDENTITY_TOKEN_FILE", Value: "/var/run/secrets/eks.amazonaws.com/serviceaccount/token"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Missing token volume",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{
								{Name: "AWS_ROLE_ARN", Value: "arn:aws:iam::123456789012:role/test-role"},
								{Name: "AWS_WEB_IDENTITY_TOKEN_FILE", Value: "/var/run/secrets/eks.amazonaws.com/serviceaccount/token"},
							},
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePodWebIdentityToken(tc.pod)
			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
