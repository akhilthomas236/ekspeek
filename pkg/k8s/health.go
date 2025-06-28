package k8s

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterHealthStatus contains comprehensive health check results
type ClusterHealthStatus struct {
	NodeVersions        map[string][]string // Maps Kubernetes versions to node names
	DeprecatedAPIs     []string
	LoggingStatus      LoggingStatus
	NetworkingStatus   NetworkingStatus
	LoadBalancerStatus LoadBalancerStatus
	SchedulingStatus   SchedulingStatus
	AuthStatus         AuthStatus
	NodeStatus         NodeStatus
}

type LoggingStatus struct {
	FluentBitStatus    []PodStatus
	CloudWatchStatus   []PodStatus
	MetricsServerStatus []PodStatus
}

type NetworkingStatus struct {
	CNIStatus        []PodStatus
	CoreDNSStatus    []PodStatus
	ExternalAccess   bool
	DNSResolution    bool
}

type LoadBalancerStatus struct {
	PendingServices []string
	IngressStatus   []IngressStatus
}

type IngressStatus struct {
	Name      string
	Namespace string
	Status    string
	Problems  []string
}

type SchedulingStatus struct {
	PendingPods    []PodSchedulingIssue
	ResourceIssues []ResourceIssue
}

type PodSchedulingIssue struct {
	Pod       string
	Namespace string
	Reason    string
}

type ResourceIssue struct {
	NodeName string
	CPU      ResourceStats
	Memory   ResourceStats
}

type ResourceStats struct {
	Capacity    int64
	Allocated   int64
	Utilization float64
}

type AuthStatus struct {
	IRSAIssues    []string
	RBACIssues    []string
	IAMAuthIssues []string
}

type NodeStatus struct {
	NotReady        []string
	ASGIssues       []string
	BootstrapIssues []string
}

type PodStatus struct {
	Name      string
	Namespace string
	Status    string
	Message   string
}

// CheckClusterHealth performs comprehensive health checks
func (k *KubeClient) CheckClusterHealth(ctx context.Context) (*ClusterHealthStatus, error) {
	status := &ClusterHealthStatus{
		NodeVersions: make(map[string][]string),
	}

	// Check node versions and control plane compatibility
	if err := k.checkVersionMismatch(ctx, status); err != nil {
		return nil, err
	}

	// Check for deprecated API usage
	if err := k.checkDeprecatedAPIs(ctx, status); err != nil {
		return nil, err
	}

	// Check logging components
	if err := k.checkLoggingStatus(ctx, status); err != nil {
		return nil, err
	}

	// Check networking
	if err := k.checkNetworkingStatus(ctx, &status.NetworkingStatus); err != nil {
		return nil, err
	}

	// Check load balancers and ingress
	if err := k.checkLoadBalancerStatus(ctx, &status.LoadBalancerStatus); err != nil {
		return nil, err
	}

	// Check scheduling and resources
	if err := k.checkSchedulingStatus(ctx, &status.SchedulingStatus); err != nil {
		return nil, err
	}

	// Check authentication and authorization
	if err := k.checkAuthStatus(ctx, &status.AuthStatus); err != nil {
		return nil, err
	}

	// Check node health
	if err := k.checkNodeStatus(ctx, &status.NodeStatus); err != nil {
		return nil, err
	}

	return status, nil
}

func (k *KubeClient) checkVersionMismatch(ctx context.Context, status *ClusterHealthStatus) error {
	nodes, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, node := range nodes.Items {
		version := node.Status.NodeInfo.KubeletVersion
		status.NodeVersions[version] = append(status.NodeVersions[version], node.Name)
	}

	return nil
}

func (k *KubeClient) checkDeprecatedAPIs(ctx context.Context, status *ClusterHealthStatus) error {
	// Check deployments for deprecated API versions
	deployments, err := k.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, deploy := range deployments.Items {
		// Example check for deprecated API versions in annotations
		if _, hasDeprecated := deploy.Annotations["deprecated.kubernetes.io"]; hasDeprecated {
			status.DeprecatedAPIs = append(status.DeprecatedAPIs, 
				fmt.Sprintf("Deployment %s/%s uses deprecated APIs", deploy.Namespace, deploy.Name))
		}
	}

	return nil
}

func (k *KubeClient) checkLoggingStatus(ctx context.Context, status *ClusterHealthStatus) error {
	// Check FluentBit status
	fluentBitPods, err := k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "app=fluent-bit",
	})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	
	for _, pod := range fluentBitPods.Items {
		status.LoggingStatus.FluentBitStatus = append(status.LoggingStatus.FluentBitStatus, PodStatus{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Message:   pod.Status.Message,
		})
	}

	// Check CloudWatch agent status
	cwPods, err := k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "app=cloudwatch-agent",
	})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	for _, pod := range cwPods.Items {
		status.LoggingStatus.CloudWatchStatus = append(status.LoggingStatus.CloudWatchStatus, PodStatus{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Message:   pod.Status.Message,
		})
	}

	// Check metrics-server status
	metricsPods, err := k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "k8s-app=metrics-server",
	})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	for _, pod := range metricsPods.Items {
		status.LoggingStatus.MetricsServerStatus = append(status.LoggingStatus.MetricsServerStatus, PodStatus{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Message:   pod.Status.Message,
		})
	}

	return nil
}

func (k *KubeClient) checkNetworkingStatus(ctx context.Context, status *NetworkingStatus) error {
	// Check CNI plugin status
	cniPods, err := k.clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "k8s-app=aws-node",
	})
	if err != nil {
		return err
	}

	for _, pod := range cniPods.Items {
		status.CNIStatus = append(status.CNIStatus, PodStatus{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Message:   pod.Status.Message,
		})
	}

	// Check CoreDNS status
	coreDNSPods, err := k.clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "k8s-app=kube-dns",
	})
	if err != nil {
		return err
	}

	for _, pod := range coreDNSPods.Items {
		status.CoreDNSStatus = append(status.CoreDNSStatus, PodStatus{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Message:   pod.Status.Message,
		})
	}

	return nil
}

func (k *KubeClient) checkLoadBalancerStatus(ctx context.Context, status *LoadBalancerStatus) error {
	// Check services of type LoadBalancer
	services, err := k.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, svc := range services.Items {
		if svc.Spec.Type == corev1.ServiceTypeLoadBalancer && len(svc.Status.LoadBalancer.Ingress) == 0 {
			status.PendingServices = append(status.PendingServices, 
				fmt.Sprintf("%s/%s", svc.Namespace, svc.Name))
		}
	}

	// Check ingress controllers
	ingresses, err := k.clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ing := range ingresses.Items {
		ingressStatus := IngressStatus{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Status:    "Pending",
		}

		if len(ing.Status.LoadBalancer.Ingress) > 0 {
			ingressStatus.Status = "Ready"
		}

		status.IngressStatus = append(status.IngressStatus, ingressStatus)
	}

	return nil
}

func (k *KubeClient) checkSchedulingStatus(ctx context.Context, status *SchedulingStatus) error {
	pods, err := k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Pending",
	})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		issue := PodSchedulingIssue{
			Pod:       pod.Name,
			Namespace: pod.Namespace,
		}

		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
				issue.Reason = cond.Message
				break
			}
		}

		status.PendingPods = append(status.PendingPods, issue)
	}

	return nil
}

func (k *KubeClient) checkAuthStatus(ctx context.Context, status *AuthStatus) error {
	// Check IRSA setup
	sa, err := k.clientset.CoreV1().ServiceAccounts("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, account := range sa.Items {
		if annotations := account.GetAnnotations(); annotations != nil {
			if role, exists := annotations["eks.amazonaws.com/role-arn"]; exists {
				// Verify if pods using this SA can access AWS resources
				pods, err := k.clientset.CoreV1().Pods(account.Namespace).List(ctx, metav1.ListOptions{
					FieldSelector: fmt.Sprintf("spec.serviceAccountName=%s", account.Name),
				})
				if err != nil {
					continue
				}

				for _, pod := range pods.Items {
					if pod.Status.Phase == corev1.PodRunning {
						// Check pod logs for AWS API errors
						logs, err := k.GetPodLogs(ctx, pod.Namespace, pod.Name, "")
						if err != nil {
							continue
						}
						if strings.Contains(logs, "AccessDenied") || strings.Contains(logs, "UnauthorizedOperation") {
							status.IRSAIssues = append(status.IRSAIssues,
								fmt.Sprintf("Pod %s/%s using SA %s with role %s has AWS access issues",
									pod.Namespace, pod.Name, account.Name, role))
						}
					}
				}
			}
		}
	}

	return nil
}

func (k *KubeClient) checkNodeStatus(ctx context.Context, status *NodeStatus) error {
	nodes, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, node := range nodes.Items {
		isReady := false
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				isReady = condition.Status == corev1.ConditionTrue
				break
			}
		}

		if !isReady {
			status.NotReady = append(status.NotReady, node.Name)
			// Check node conditions for bootstrap issues
			for _, condition := range node.Status.Conditions {
				if condition.Status == corev1.ConditionFalse {
					status.BootstrapIssues = append(status.BootstrapIssues,
						fmt.Sprintf("Node %s: %s - %s", node.Name, condition.Type, condition.Message))
				}
			}
		}
	}

	return nil
}
