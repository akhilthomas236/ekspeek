package cmd

import (
	"context"
	"fmt"
	"strings"

	"ekspeek/pkg/aws"
	"ekspeek/pkg/k8s"
	"ekspeek/pkg/common/logger"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getKubeClient is a helper function to create a new KubeClient
func getKubeClient(clusterName string) (*k8s.KubeClient, error) {
	ctx := context.Background()

	// Update kubeconfig
	logger.Info("Updating kubeconfig for cluster %s", clusterName)
	if err := k8s.UpdateKubeconfig(ctx, clusterName, region); err != nil {
		return nil, err
	}

	return k8s.CreateKubeClient()
}

func NewDebugCommand() *cobra.Command {
	debugCmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug EKS cluster components and resources",
		Long:  `Commands for debugging EKS cluster components including EFS CSI driver, PVCs, pods, and other resources`,
	}

	debugCmd.AddCommand(
		newDebugEFSCommand(),
		newDebugPVCCommand(),
		newDebugPodsCommand(),
		newDebugResourcesCommand(),
		newHealthCheckCommand(),
		newDebugIRSACommand(),
		newDebugAutoscalerCommand(),
		newDebugThrottlingCommand(),
		newDebugNetworkingCommand(),
		newDebugEgressCommand(),
		newDebugCrossAccountCommand(),
		newDebugTLSCommand(),
		newDebugPerformanceCommand(),
		newDebugSecurityCommand(),
		newDebugKarpenterCommand(),
	)

	return debugCmd
}

func newDebugEFSCommand() *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:   "efs [cluster-name]",
		Short: "Debug EFS CSI driver status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()

			// Update kubeconfig
			logger.Info("Updating kubeconfig for cluster %s", clusterName)
			if err := k8s.UpdateKubeconfig(ctx, clusterName, region); err != nil {
				return err
			}

			// Create kubernetes client
			kubeClient, err := k8s.CreateKubeClient()
			if err != nil {
				return err
			}

			// Get EFS CSI driver status
			logger.Info("Checking EFS CSI driver status...")
			pods, err := kubeClient.GetEFSCSIStatus(ctx)
			if err != nil {
				return err
			}

			if len(pods) == 0 {
				logger.Warning("No EFS CSI driver pods found. Is the driver installed?")
				return nil
			}

			logger.Success("Found %d EFS CSI driver pods:", len(pods))
			for _, pod := range pods {
				status := "Healthy"
				if pod.Phase != corev1.PodRunning {
					status = fmt.Sprintf("Unhealthy (%s)", pod.Phase)
				}
				fmt.Printf("Pod: %s\nStatus: %s\nNode: %s\n\n", pod.Name, status, pod.NodeName)
			}

			return nil
		},
	}

	return cmd
}

func newDebugPVCCommand() *cobra.Command {
	var (
		clusterName string
		namespace   string
	)

	cmd := &cobra.Command{
		Use:   "pvc [cluster-name]",
		Short: "Debug PVC status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()

			// Update kubeconfig
			logger.Info("Updating kubeconfig for cluster %s", clusterName)
			if err := k8s.UpdateKubeconfig(ctx, clusterName, region); err != nil {
				return err
			}

			// Create kubernetes client
			kubeClient, err := k8s.CreateKubeClient()
			if err != nil {
				return err
			}

			// Get PVC status
			logger.Info("Checking PVC status...")
			pvcs, err := kubeClient.GetPVCStatus(ctx, namespace)
			if err != nil {
				return err
			}

			if len(pvcs) == 0 {
				logger.Info("No PVCs found in %s", namespace)
				return nil
			}

			logger.Success("Found %d PVCs:", len(pvcs))
			for _, pvc := range pvcs {
				fmt.Printf("Name: %s\nNamespace: %s\nStatus: %s\nVolume: %s\nStorage Class: %s\nCapacity: %s\n\n",
					pvc.Name,
					pvc.Namespace,
					pvc.Status.Phase,
					pvc.Spec.VolumeName,
					*pvc.Spec.StorageClassName,
					pvc.Status.Capacity.Storage().String())
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to check PVCs in (default is all namespaces)")
	return cmd
}

func newDebugPodsCommand() *cobra.Command {
	var (
		clusterName string
		namespace   string
		showLogs    bool
	)

	cmd := &cobra.Command{
		Use:   "pods [cluster-name]",
		Short: "Debug pod status and show failed pods",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()

			// Update kubeconfig
			logger.Info("Updating kubeconfig for cluster %s", clusterName)
			if err := k8s.UpdateKubeconfig(ctx, clusterName, region); err != nil {
				return err
			}

			// Create kubernetes client
			kubeClient, err := k8s.CreateKubeClient()
			if err != nil {
				return err
			}

			// Get failed pods
			logger.Info("Checking for failed pods...")
			pods, err := kubeClient.GetFailedPods(ctx, namespace)
			if err != nil {
				return err
			}

			if len(pods) == 0 {
				logger.Success("No failed pods found!")
				return nil
			}

			logger.Warning("Found %d failed pods:", len(pods))
			for _, pod := range pods {
				fmt.Printf("\nPod: %s\nNamespace: %s\nStatus: %s\nMessage: %s\n",
					pod.Name,
					pod.Namespace,
					pod.Status,
					pod.Message)

				if showLogs {
					logs, err := kubeClient.GetPodLogs(ctx, pod.Namespace, pod.Name, "")
					if err != nil {
						logger.Warning("Failed to get logs for pod %s: %v", pod.Name, err)
						continue
					}
					fmt.Printf("\nLogs:\n%s\n", strings.TrimSpace(logs))
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to check pods in (default is all namespaces)")
	cmd.Flags().BoolVar(&showLogs, "logs", false, "Show logs for failed pods")
	return cmd
}

func newDebugResourcesCommand() *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:   "resources [cluster-name]",
		Short: "Show cluster resource usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()

			// Update kubeconfig
			logger.Info("Updating kubeconfig for cluster %s", clusterName)
			if err := k8s.UpdateKubeconfig(ctx, clusterName, region); err != nil {
				return err
			}

			// Create kubernetes client
			kubeClient, err := k8s.CreateKubeClient()
			if err != nil {
				return err
			}

			// Get cluster resources
			logger.Info("Gathering cluster resource usage...")
			resources, err := kubeClient.GetClusterResources(ctx)
			if err != nil {
				return err
			}

			fmt.Printf("\nCluster Resource Usage:\n")
			fmt.Printf("CPU Usage: %.2f%% (%.2f/%.2f cores)\n",
				resources.CPUPercentage,
				float64(resources.AllocatedCPU)/1000,
				float64(resources.TotalCPU)/1000)
			fmt.Printf("Memory Usage: %.2f%% (%.2f/%.2f GB)\n",
				resources.MemPercentage,
				float64(resources.AllocatedMemory)/(1024*1024*1024),
				float64(resources.TotalMemory)/(1024*1024*1024))

			return nil
		},
	}

	return cmd
}

func newDebugIRSACommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "irsa [pod-name]",
		Short: "Debug IRSA (IAM Roles for Service Accounts) issues",
		Long: `Debug IRSA related issues including:
- WebIdentity token validation
- IAM role trust relationships
- Service account annotations
- Pod identity configuration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("pod name is required")
			}
			podName := args[0]
			ctx := context.Background()

			// Create kubernetes client
			kubeClient, err := k8s.CreateKubeClient()
			if err != nil {
				return err
			}

			// 1. Check pod's service account
			saName, err := kubeClient.GetPodServiceAccount(ctx, "default", podName)
			if err != nil {
				return err
			}

			// Get service account
			sa, err := kubeClient.Clientset.CoreV1().ServiceAccounts("default").Get(ctx, saName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// 2. Validate service account annotations
			roleARN, exists := sa.Annotations["eks.amazonaws.com/role-arn"]
			if !exists {
				return fmt.Errorf("service account %s is missing IAM role annotation", sa.Name)
			}

			// 3. Verify trust relationship
			if err := aws.VerifyIAMRoleTrust(roleARN); err != nil {
				return err
			}

			// 4. Check WebIdentity token mounting
			if err := kubeClient.ValidatePodWebIdentityToken(ctx, "default", podName); err != nil {
				return err
			}

			fmt.Printf("✅ IRSA configuration for pod %s is valid\n", podName)
			return nil
		},
	}

	return cmd
}

func newDebugAutoscalerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "autoscaler [cluster-name]",
		Short: "Debug Cluster Autoscaler issues",
		Long: `Debug EKS Cluster Autoscaler issues including:
- Scaling events and decisions
- Node group configuration
- ASG settings
- Pending pods analysis`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			// Create k8s client
			kubeClient, err := getKubeClient(clusterName)
			if err != nil {
				return fmt.Errorf("failed to create kubernetes client: %w", err)
			}

			// Create AWS client
			awsClient, err := aws.NewClient(ctx, aws.ClientConfig{
				Profile: profile,
				Region:  region,
			})
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			// 1. Get Cluster Autoscaler pod
			caPod, err := kubeClient.GetClusterAutoscalerPod(ctx)
			if err != nil {
				return err
			}
			logger.Success("✅ Found Cluster Autoscaler pod: %s/%s", caPod.Namespace, caPod.Name)

			// 2. Check Cluster Autoscaler logs
			logs, err := kubeClient.GetPodLogs(ctx, caPod.Namespace, caPod.Name, "")
			if err != nil {
				return err
			}

			// Analyze logs for common issues
			if strings.Contains(logs, "FailedToUpdateNodeGroupSize") {
				logger.Warning("❌ Node group size update failures detected")
			}
			if strings.Contains(logs, "NoScaleUpGroups") {
				logger.Warning("❌ Unable to scale up - no node groups available")
			}
			if !strings.Contains(logs, "Started CA") {
				logger.Warning("❌ Cluster Autoscaler may not be properly initialized")
			}

			// 3. Analyze scaling events
			events, err := kubeClient.GetScalingEvents(ctx)
			if err != nil {
				return err
			}

			// Process scaling events
			for _, event := range events {
				if event.Type == corev1.EventTypeWarning {
					logger.Warning("❌ Scaling event warning: %s", event.Message)
				} else {
					logger.Info("ℹ️ Scaling event: %s", event.Message)
				}
			}

			// 4. Check node groups configuration
			if err := awsClient.ValidateNodeGroupsConfig(ctx, clusterName); err != nil {
				logger.Warning("❌ Node group configuration issue: %s", err)
			} else {
				logger.Success("✅ Node group configuration is valid")
			}

			// 5. Analyze unschedulable pods
			if err := kubeClient.AnalyzeUnschedulablePods(ctx); err != nil {
				logger.Warning("❌ Issues with unschedulable pods: %s", err)
			}

			logger.Success("✅ Cluster Autoscaler diagnostics completed")
			return nil
		},
	}

	return cmd
}

func newDebugThrottlingCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "throttling",
		Short: "Debug API throttling issues",
		Long:  "Analyze control plane API throttling and provide recommendations",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implementation to check API throttling metrics and logs
			return nil
		},
	}
	return cmd
}

func newDebugNetworkingCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "networking [pod-name]",
		Short: "Debug pod networking issues",
		Long:  "Analyze pod networking, including Multus and custom CNI configurations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("pod name is required")
			}
			// Implementation to check pod networking configuration
			return nil
		},
	}
	return cmd
}

func newDebugEgressCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "egress [pod-name]",
		Short: "Debug pod egress traffic issues",
		Long:  "Analyze pod egress traffic and security group configurations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("pod name is required")
			}
			// Implementation to check egress configuration
			return nil
		},
	}
	return cmd
}

func newDebugCrossAccountCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cross-account",
		Short: "Debug cross-account access issues",
		Long:  "Analyze cross-account IAM roles and resource policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implementation to check cross-account configurations
			return nil
		},
	}
	return cmd
}

func newDebugTLSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tls",
		Short: "Debug TLS and certificate issues",
		Long:  "Analyze TLS configurations and certificate validity",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implementation to check TLS and certificates
			return nil
		},
	}
	return cmd
}

func newDebugPerformanceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "performance",
		Short: "Debug performance and scaling issues",
		Long:  "Analyze cluster performance metrics and scaling behavior",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implementation to check performance metrics
			return nil
		},
	}
	return cmd
}

func newDebugSecurityCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Check security and compliance",
		Long:  "Analyze cluster security configuration and compliance status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Implementation to check security configurations
			return nil
		},
	}
	return cmd
}

func newDebugKarpenterCommand() *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:   "karpenter [cluster-name]",
		Short: "Debug Karpenter issues",
		Long:  "Debug Karpenter provisioner configuration, node states, and scaling decisions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()

			// Update kubeconfig
			logger.Info("Updating kubeconfig for cluster %s", clusterName)
			if err := k8s.UpdateKubeconfig(ctx, clusterName, region); err != nil {
				return err
			}

			// Create kubernetes client
			kubeClient, err := getKubeClient(clusterName)
			if err != nil {
				return err
			}

			// Check Karpenter deployment status
			logger.Info("Checking Karpenter deployment status...")
			status, err := kubeClient.GetKarpenterStatus(ctx)
			if err != nil {
				return err
			}

			if !status.IsDeployed {
				logger.Warning("Karpenter is not deployed in the cluster")
				return nil
			}

			// Get Karpenter provisioners
			logger.Info("Checking Karpenter provisioners...")
			provisioners, err := kubeClient.GetKarpenterProvisioners(ctx)
			if err != nil {
				return err
			}

			logger.Success("Found %d Karpenter provisioners:", len(provisioners))
			for _, p := range provisioners {
				fmt.Printf("\nProvisioner: %s\n", p.Name)
				fmt.Printf("Requirements:\n  CPU: %s\n  Memory: %s\n",
					p.Requirements.CPU,
					p.Requirements.Memory)
				fmt.Printf("Limits:\n  CPU: %s\n  Memory: %s\n",
					p.Limits.CPU,
					p.Limits.Memory)
			}

			// Get Karpenter node pools
			logger.Info("Checking Karpenter managed nodes...")
			nodes, err := kubeClient.GetKarpenterNodes(ctx)
			if err != nil {
				return err
			}

			logger.Success("Found %d Karpenter managed nodes:", len(nodes))
			for _, node := range nodes {
				fmt.Printf("\nNode: %s\n", node.Name)
				fmt.Printf("Instance Type: %s\n", node.InstanceType)
				fmt.Printf("Capacity:\n  CPU: %s\n  Memory: %s\n",
					node.Capacity.CPU,
					node.Capacity.Memory)
				fmt.Printf("Usage:\n  CPU: %.2f%%\n  Memory: %.2f%%\n",
					node.Usage.CPUPercent,
					node.Usage.MemoryPercent)
			}

			// Check pending pods that Karpenter should handle
			logger.Info("Checking for pending pods...")
			pendingPods, err := kubeClient.GetKarpenterPendingPods(ctx)
			if err != nil {
				return err
			}

			if len(pendingPods) > 0 {
				logger.Warning("Found %d pending pods that Karpenter should handle:", len(pendingPods))
				for _, pod := range pendingPods {
					fmt.Printf("\nPod: %s/%s\n", pod.Namespace, pod.Name)
					fmt.Printf("Requirements:\n  CPU: %s\n  Memory: %s\n",
						pod.Requirements.CPU,
						pod.Requirements.Memory)
					fmt.Printf("Status: %s\n", pod.Status)
				}
			} else {
				logger.Success("No pending pods found that need Karpenter provisioning")
			}

			return nil
		},
	}

	return cmd
}
