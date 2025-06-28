package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"ekspeek/pkg/common/logger"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubeClient represents a Kubernetes client
type KubeClient struct {
	Clientset kubernetes.Interface
}

// NewKubeClient creates a new KubeClient
func NewKubeClient(clientset kubernetes.Interface) (*KubeClient, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes clientset cannot be nil")
	}
	return &KubeClient{
		Clientset: clientset,
	}, nil
}

// GetPodLogs retrieves logs for a specific pod
func (k *KubeClient) GetPodLogs(ctx context.Context, namespace, podName, containerName string) (string, error) {
	podLogOptions := corev1.PodLogOptions{
		Container: containerName,
	}

	req := k.Clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("failed to copy pod logs: %w", err)
	}

	return buf.String(), nil
}

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

// PodStatus represents the status of a pod
type PodStatus struct {
	Name      string
	Namespace string
	Status    string
	NodeName  string
	Phase     corev1.PodPhase
	Spec      corev1.PodSpec
	Message   string
	Requirements ResourceRequirements
}

// ResourceRequirements represents the compute resources required by a pod
type ResourceRequirements struct {
	CPU    string
	Memory string
}

// PVCStatus represents the status of a PVC
type PVCStatus struct {
	Name      string
	Namespace string
	Status    corev1.PersistentVolumeClaimStatus
	Spec      corev1.PersistentVolumeClaimSpec
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
	nodes, err := k.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
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
	deployments, err := k.Clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
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
	fluentBitPods, err := k.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
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
	cwPods, err := k.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
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
	metricsPods, err := k.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
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
	cniPods, err := k.Clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
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
	coreDNSPods, err := k.Clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
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
	services, err := k.Clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
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
	ingresses, err := k.Clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
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
	pods, err := k.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
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
	sa, err := k.Clientset.CoreV1().ServiceAccounts("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, account := range sa.Items {
		if annotations := account.GetAnnotations(); annotations != nil {
			if role, exists := annotations["eks.amazonaws.com/role-arn"]; exists {
				// Verify if pods using this SA can access AWS resources
				pods, err := k.Clientset.CoreV1().Pods(account.Namespace).List(ctx, metav1.ListOptions{
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
	nodes, err := k.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
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

// GetEFSCSIStatus checks the status of EFS CSI driver pods
func (k *KubeClient) GetEFSCSIStatus(ctx context.Context) ([]PodStatus, error) {
	pods, err := k.Clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "app=efs-csi-controller",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list EFS CSI pods: %w", err)
	}

	var status []PodStatus
	for _, pod := range pods.Items {
		podStatus := PodStatus{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
		}
		if pod.Status.Phase != corev1.PodRunning {
			podStatus.Message = "Pod is not in Running state"
		}
		status = append(status, podStatus)
	}

	return status, nil
}

// GetPVCStatus gets the status of all PVCs in the cluster
func (k *KubeClient) GetPVCStatus(ctx context.Context, namespace string) ([]*PVCStatus, error) {
	pvcs, err := k.Clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list PVCs: %w", err)
	}

	var pvcStatuses []*PVCStatus
	for _, pvc := range pvcs.Items {
		status := &PVCStatus{
			Name:      pvc.Name,
			Namespace: pvc.Namespace,
			Status:    pvc.Status,
			Spec:      pvc.Spec,
		}
		pvcStatuses = append(pvcStatuses, status)
	}

	return pvcStatuses, nil
}

// GetFailedPods returns a list of failed pods
func (k *KubeClient) GetFailedPods(ctx context.Context, namespace string) ([]PodStatus, error) {
	pods, err := k.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Failed",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list failed pods: %w", err)
	}

	var status []PodStatus
	for _, pod := range pods.Items {
		status = append(status, PodStatus{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Message:   pod.Status.Message,
			Phase:     pod.Status.Phase,
			NodeName:  pod.Spec.NodeName,
			Spec:      pod.Spec,
		})
	}

	return status, nil
}

// ClusterResources represents the resource usage in the cluster
type ClusterResources struct {
	TotalCPU        int64
	TotalMemory     int64
	AllocatedCPU    int64
	AllocatedMemory int64
	CPUPercentage   float64
	MemPercentage   float64
}	// GetClusterResources returns the current resource usage in the cluster
func (k *KubeClient) GetClusterResources(ctx context.Context) (*ClusterResources, error) {
	nodes, err := k.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	resources := &ClusterResources{}

	for _, node := range nodes.Items {
		cpu := node.Status.Capacity.Cpu().MilliValue()
		mem := node.Status.Capacity.Memory().Value()

		resources.TotalCPU += cpu
		resources.TotalMemory += mem

		pods, err := k.Clientset.CoreV1().Pods(corev1.NamespaceAll).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list pods for node %s: %w", node.Name, err)
		}

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				resources.AllocatedCPU += container.Resources.Requests.Cpu().MilliValue()
				resources.AllocatedMemory += container.Resources.Requests.Memory().Value()
			}
		}
	}

	if resources.TotalCPU > 0 {
		resources.CPUPercentage = float64(resources.AllocatedCPU) / float64(resources.TotalCPU) * 100
	}
	if resources.TotalMemory > 0 {
		resources.MemPercentage = float64(resources.AllocatedMemory) / float64(resources.TotalMemory) * 100
	}

	return resources, nil
}

// GetPodServiceAccount gets the service account for a pod
func (k *KubeClient) GetPodServiceAccount(ctx context.Context, namespace, podName string) (string, error) {
	pod, err := k.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}
	return pod.Spec.ServiceAccountName, nil
}

// ValidatePodWebIdentityToken validates the web identity token of a pod
func (k *KubeClient) ValidatePodWebIdentityToken(ctx context.Context, namespace, podName string) error {
	pod, err := k.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod: %w", err)
	}

	// Check if pod has service account token volume
	hasTokenVolume := false
	for _, volume := range pod.Spec.Volumes {
		if volume.Projected != nil {
			for _, source := range volume.Projected.Sources {
				if source.ServiceAccountToken != nil {
					hasTokenVolume = true
					break
				}
			}
		}
	}

	if !hasTokenVolume {
		return fmt.Errorf("pod does not have service account token volume mounted")
	}

	return nil
}

// GetKubeConfig returns the kubernetes config for the current context
func GetKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	return config, nil
}

// CreateKubeClient creates a new KubeClient with proper configuration
func CreateKubeClient() (*KubeClient, error) {
	config, err := GetKubeConfig()
	if err != nil {
		return nil, err
	}
	
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return NewKubeClient(clientset)
}

// GetClusterAutoscalerPod returns the cluster-autoscaler pod
func (k *KubeClient) GetClusterAutoscalerPod(ctx context.Context) (*corev1.Pod, error) {
	pods, err := k.Clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "app=cluster-autoscaler",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster-autoscaler pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no cluster-autoscaler pod found")
	}

	return &pods.Items[0], nil
}

// GetScalingEvents returns scaling-related events
func (k *KubeClient) GetScalingEvents(ctx context.Context) ([]corev1.Event, error) {
	events, err := k.Clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: "reason=TriggeredScaleUp,reason=ScalingReplicaSet",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list scaling events: %w", err)
	}

	return events.Items, nil
}

// KarpenterStatus represents the status of Karpenter
type KarpenterStatus struct {
	IsDeployed bool
	Status     string
}

// KarpenterProvisioner represents a Karpenter provisioner
type KarpenterProvisioner struct {
	Name         string
	Requirements struct {
		CPU    string
		Memory string
	}
	Limits struct {
		CPU    string
		Memory string
	}
}

// KarpenterNode represents a node managed by Karpenter
type KarpenterNode struct {
	Name         string
	InstanceType string
	Capacity     struct {
		CPU    string
		Memory string
	}
	Usage struct {
		CPUPercent    float64
		MemoryPercent float64
	}
}

// GetKarpenterStatus returns the status of Karpenter deployment
func (k *KubeClient) GetKarpenterStatus(ctx context.Context) (*KarpenterStatus, error) {
	deployment, err := k.Clientset.AppsV1().Deployments("karpenter").Get(ctx, "karpenter", metav1.GetOptions{})
	if err != nil {
		return &KarpenterStatus{IsDeployed: false}, nil
	}

	return &KarpenterStatus{
		IsDeployed: deployment.Status.AvailableReplicas > 0,
		Status:     fmt.Sprintf("%d/%d replicas available", deployment.Status.AvailableReplicas, deployment.Status.Replicas),
	}, nil
}

// GetKarpenterProvisioners returns all Karpenter provisioners
func (k *KubeClient) GetKarpenterProvisioners(ctx context.Context) ([]KarpenterProvisioner, error) {
	// This is a placeholder - you would need to implement the actual logic using Karpenter's CRDs
	return []KarpenterProvisioner{}, nil
}

// GetKarpenterNodes returns all nodes managed by Karpenter
func (k *KubeClient) GetKarpenterNodes(ctx context.Context) ([]KarpenterNode, error) {
	// This is a placeholder - you would need to implement the actual logic to identify Karpenter nodes
	return []KarpenterNode{}, nil
}

// GetKarpenterPendingPods returns pods that are pending and could be scheduled by Karpenter
func (k *KubeClient) GetKarpenterPendingPods(ctx context.Context) ([]PodStatus, error) {
	pods, err := k.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Pending",
	})
	if err != nil {
		return nil, err
	}

	var pendingPods []PodStatus
	for _, pod := range pods.Items {
		pendingPods = append(pendingPods, PodStatus{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Message:   pod.Status.Message,
		})
	}

	return pendingPods, nil
}

// AnalyzeUnschedulablePods analyzes pods that cannot be scheduled
func (k *KubeClient) AnalyzeUnschedulablePods(ctx context.Context) error {
	pods, err := k.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Pending",
	})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
				logger.Warning("Pod %s/%s is unschedulable: %s", pod.Namespace, pod.Name, cond.Message)
			}
		}
	}

	return nil
}
