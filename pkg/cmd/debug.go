package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ekspeek/pkg/aws"
	"ekspeek/pkg/k8s"
	"ekspeek/pkg/common/logger"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getKubeClient is a helper function to create a new KubeClient
func getKubeClient() (*k8s.KubeClient, error) {
	cfg := k8s.KubeClientConfig{
		KubeConfig: "",  // Use default location
		Context:    "",  // Use current context
	}
	return k8s.NewKubeClient(cfg)
}

// getAWSClient is a helper function to create a new AWS Client
func getAWSClient(ctx context.Context) (*aws.Client, error) {
	cfg := aws.ClientConfig{
		Profile: "",  // Use default profile
		Region:  region,
	}
	return aws.NewClient(ctx, cfg)
}

func NewDebugCommand() *cobra.Command {
	debugCmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug EKS cluster components and resources",
		Long:  `Commands for debugging EKS cluster components including performance metrics, security analysis, and other resources`,
	}

	debugCmd.AddCommand(
		newDebugPerformanceCommand(),
		newDebugSecurityCommand(),
		newDebugEFSCommand(),
		newDebugPVCCommand(),
		newDebugPodsCommand(),
		newDebugResourcesCommand(),
		newDebugIRSACommand(),
		newDebugAutoscalerCommand(),
		newDebugThrottlingCommand(),
		newDebugNetworkingCommand(),
		newDebugEgressCommand(),
		newDebugCrossAccountCommand(),
		newDebugTLSCommand(),
		newDebugKarpenterCommand(),
	)

	return debugCmd
}

func newDebugPerformanceCommand() *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:   "performance [cluster-name]",
		Short: "Debug cluster performance metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()

			// Create AWS client
			awsClient, err := getAWSClient(ctx)
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			// Get performance metrics
			logger.Info("Collecting performance metrics for cluster %s...", clusterName)
			metrics, err := awsClient.GetClusterPerformanceMetrics(ctx, clusterName)
			if err != nil {
				return fmt.Errorf("failed to get performance metrics: %w", err)
			}

			// Print metrics
			logger.Success("Cluster Performance Metrics:")
			for name, value := range metrics {
				logger.Info("  %s: %.2f", name, value)
			}

			return nil
		},
	}

	return cmd
}

func newDebugSecurityCommand() *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:   "security [cluster-name]",
		Short: "Analyze cluster security configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()

			// Create AWS client
			awsClient, err := getAWSClient(ctx)
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			// Get security analysis
			logger.Info("Analyzing security configuration for cluster %s...", clusterName)
			findings, err := awsClient.GetSecurityAnalysis(ctx, clusterName)
			if err != nil {
				return fmt.Errorf("failed to get security analysis: %w", err)
			}

			// Print findings
			logger.Success("Security Analysis Results:")
			for check, result := range findings {
				if strings.HasPrefix(result, "WARNING") {
					logger.Warning("  %s: %s", check, result)
				} else {
					logger.Info("  %s: %s", check, result)
				}
			}

			return nil
		},
	}

	return cmd
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

			// Create kubernetes client
			kubeClient, err := getKubeClient()
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

			// Create kubernetes client
			kubeClient, err := getKubeClient()
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

			// Create kubernetes client
			kubeClient, err := getKubeClient()
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

			// Create kubernetes client
			kubeClient, err := getKubeClient()
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
			kubeClient, err := getKubeClient()
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
			kubeClient, err := getKubeClient()
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
	var clusterName string

	cmd := &cobra.Command{
		Use:   "throttling [cluster-name]",
		Short: "Debug API throttling issues",
		Long:  "Analyze control plane API throttling and provide recommendations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()

			// Create AWS client
			awsClient, err := aws.NewClient(ctx, aws.ClientConfig{
				Profile: profile,
				Region:  region,
			})
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			// Get throttling metrics
			logger.Info("Fetching API throttling metrics for cluster %s...", clusterName)
			endTime := time.Now()
			startTime := endTime.Add(-1 * time.Hour)
			metrics, err := awsClient.GetEKSThrottlingMetrics(ctx, startTime, endTime)
			if err != nil {
				return fmt.Errorf("failed to get throttling metrics: %w", err)
			}

			if len(metrics) == 0 {
				logger.Info("No throttling metrics found in the last hour")
				return nil
			}

			// Analyze and display metrics
			var totalThrottles float64
			var maxErrorRate float64
			for _, m := range metrics {
				totalThrottles += m.ThrottledCalls
				if m.ErrorRate > maxErrorRate {
					maxErrorRate = m.ErrorRate
				}
			}

			// Display summary
			fmt.Printf("\nThrottling Analysis for cluster %s:\n", clusterName)
			fmt.Printf("Time period: Last hour\n")
			fmt.Printf("Total throttled calls: %.0f\n", totalThrottles)
			fmt.Printf("Maximum error rate: %.2f%%\n\n", maxErrorRate)

			// Provide recommendations
			if totalThrottles > 0 {
				logger.Warning("⚠️ API throttling detected:")
				if totalThrottles > 100 {
					logger.Warning("- High number of throttled calls (%.0f) indicates potential issues", totalThrottles)
				}
				if maxErrorRate > 5 {
					logger.Warning("- Error rate peaked at %.2f%% which is above recommended threshold (5%%)", maxErrorRate)
				}

				fmt.Printf("\nRecommendations:\n")
				fmt.Printf("1. Implement exponential backoff in your applications\n")
				fmt.Printf("2. Consider using client-side caching where appropriate\n")
				if maxErrorRate > 10 {
					fmt.Printf("3. Review applications making frequent API calls\n")
					fmt.Printf("4. Consider requesting a service quota increase\n")
				}
			} else {
				logger.Success("✅ No significant API throttling detected")
			}

			return nil
		},
	}
	return cmd
}

func newDebugNetworkingCommand() *cobra.Command {
	var (
		namespace   string
		podName     string
	)

	cmd := &cobra.Command{
		Use:   "networking [cluster-name] [pod-name]",
		Short: "Debug pod networking issues",
		Long:  "Analyze pod networking, including network policies, DNS, and connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("both cluster name and pod name are required")
			}
			podName = args[1]

			ctx := context.Background()

			// Create k8s client
			kubeClient, err := getKubeClient()
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

			// 1. Get pod details
			logger.Info("Getting pod networking details...")
			pod, err := kubeClient.GetPod(ctx, namespace, podName)
			if err != nil {
				return fmt.Errorf("failed to get pod %s: %w", podName, err)
			}

			fmt.Printf("\nPod Network Configuration:\n")
			fmt.Printf("Pod IP: %s\n", pod.Status.PodIP)
			fmt.Printf("Host IP: %s\n", pod.Status.HostIP)
			fmt.Printf("Node: %s\n\n", pod.Spec.NodeName)

			// 2. Check network policies
			logger.Info("Checking network policies...")
			policies, err := kubeClient.GetNetworkPolicies(ctx, pod.Namespace)
			if err != nil {
				logger.Warning("Failed to get network policies: %v", err)
			} else {
				if len(policies.Items) == 0 {
					logger.Warning("⚠️ No NetworkPolicies found in namespace %s", pod.Namespace)
				} else {
					logger.Success("Found %d NetworkPolicies:", len(policies.Items))
					for _, policy := range policies.Items {
						fmt.Printf("- %s\n", policy.Name)
						if len(policy.Spec.PodSelector.MatchLabels) > 0 {
							fmt.Printf("  Applies to pods with labels: %v\n", policy.Spec.PodSelector.MatchLabels)
						}
					}
				}
			}

			// 3. Check DNS resolution
			logger.Info("Testing DNS resolution...")
			success, err := kubeClient.TestPodDNS(ctx, pod.Namespace, pod.Name, "kubernetes.default.svc.cluster.local")
			if err != nil {
				logger.Warning("❌ DNS resolution test failed: %v", err)
			} else if !success {
				logger.Warning("❌ DNS resolution test failed")
			} else {
				logger.Success("✅ DNS resolution test passed")
			}

			// 4. Get VPC and subnet info
			logger.Info("Getting VPC networking details...")
			node, err := kubeClient.GetNode(ctx, pod.Spec.NodeName)
			if err != nil {
				logger.Warning("Failed to get node details: %v", err)
			} else {
				nodeID := node.Spec.ProviderID
				vpcInfo, err := awsClient.GetVPCInfo(ctx, nodeID)
				if err != nil {
					logger.Warning("Failed to get VPC info: %v", err)
				} else {
					fmt.Printf("\nVPC Configuration:\n")
					fmt.Printf("VPC ID: %s\n", vpcInfo.VPCID)
					fmt.Printf("Subnet ID: %s\n", vpcInfo.SubnetID)
					fmt.Printf("Security Groups: %v\n", vpcInfo.SecurityGroups)
				}
			}

			// 5. Check connectivity
			logger.Info("Testing pod connectivity...")
			if err := kubeClient.TestPodConnectivity(ctx, pod.Namespace, pod.Name, "default", "kubernetes"); err != nil {
				logger.Warning("❌ Connectivity test failed: %v", err)
			} else {
				logger.Success("✅ Pod connectivity test passed")
			}

			// 6. Provide recommendations
			fmt.Printf("\nRecommendations:\n")
			if policies == nil || len(policies.Items) == 0 {
				fmt.Printf("1. Consider implementing NetworkPolicies to secure pod communication\n")
			}
			mtuMap, err := kubeClient.CheckMTU(ctx)
			if err != nil {
				fmt.Printf("2. Review MTU settings: %v\n", err)
			} else if len(mtuMap) == 0 {
				fmt.Printf("2. Could not determine MTU settings\n")
			}
			if pod.Spec.HostNetwork {
				fmt.Printf("3. Pod is using host network - review if this is intended\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of the pod")
	return cmd
}

func newDebugEgressCommand() *cobra.Command {
	var (
		clusterName string
		namespace   string
	)

	cmd := &cobra.Command{
		Use:   "egress [cluster-name]",
		Short: "Debug pod egress traffic issues",
		Long: `Analyze pod egress traffic configuration including:
- Security group egress rules
- NAT gateway configuration
- Network policies
- VPC routing tables`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("cluster name is required")
			}
			clusterName = args[0]

			ctx := context.Background()

			// Create AWS client
			awsClient, err := aws.NewClient(ctx, aws.ClientConfig{
				Profile: profile,
				Region:  region,
			})
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			// Create Kubernetes client
			kubeClient, err := getKubeClient()
			if err != nil {
				return fmt.Errorf("failed to create kubernetes client: %w", err)
			}

			// 1. Get cluster VPC configuration
			logger.Info("Getting cluster VPC configuration...")
			cluster, err := awsClient.DescribeCluster(ctx, clusterName)
			if err != nil {
				return fmt.Errorf("failed to get cluster details: %w", err)
			}

			vpcConfig := cluster.Cluster.ResourcesVpcConfig
			if vpcConfig == nil {
				return fmt.Errorf("cluster VPC configuration not found")
			}

			// 2. Check NAT gateway configuration
			logger.Info("Checking NAT gateway configuration...")
			natGateways, err := awsClient.GetNATGateways(ctx, *vpcConfig.VpcId)
			if err != nil {
				logger.Warning("Failed to get NAT gateways: %v", err)
			} else if len(natGateways) == 0 {
				logger.Warning("❌ No NAT gateways found in the VPC")
			} else {
				logger.Success("✅ Found %d NAT gateways", len(natGateways))
				for _, ng := range natGateways {
					fmt.Printf("NAT Gateway: %s (State: %s)\n", *ng.NatGatewayId, ng.State)
				}
			}

			// 3. Check security group egress rules
			logger.Info("Checking security group egress rules...")
			for i := range vpcConfig.SecurityGroupIds {
				sgID := vpcConfig.SecurityGroupIds[i]
				rules, err := awsClient.GetSecurityGroupEgressRules(ctx, sgID)
				if err != nil {
					logger.Warning("Failed to get egress rules for %s: %v", sgID, err)
					continue
				}
				
				if len(rules) == 0 {
					logger.Warning("❌ No egress rules found for security group %s", sgID)
				} else {
					logger.Success("✅ Found %d egress rules for security group %s", len(rules), sgID)
					for _, rule := range rules {
						fmt.Printf("  - %s: %d -> %d\n", *rule.IpProtocol, *rule.FromPort, *rule.ToPort)
					}
				}
			}

			// 4. Check network policies
			logger.Info("Checking network policies...")
			policies, err := kubeClient.GetNetworkPolicies(ctx, namespace)
			if err != nil {
				logger.Warning("Failed to get network policies: %v", err)
			} else if policies != nil && len(policies.Items) == 0 {
				logger.Warning("❌ No network policies found")
			} else if policies != nil {
				logger.Success("✅ Found %d network policies", len(policies.Items))
				for _, policy := range policies.Items {
					fmt.Printf("\nPolicy: %s/%s\n", policy.Namespace, policy.Name)
					if len(policy.Spec.Egress) == 0 {
						fmt.Println("  - No egress rules (traffic blocked)")
					} else {
						for _, rule := range policy.Spec.Egress {
							fmt.Println("  - Egress rule:")
							for _, port := range rule.Ports {
								fmt.Printf("    Port: %s/%d\n", *port.Protocol, *port.Port)
							}
							for _, to := range rule.To {
								if to.IPBlock != nil {
									fmt.Printf("    CIDR: %s\n", to.IPBlock.CIDR)
									if len(to.IPBlock.Except) > 0 {
										fmt.Printf("    Except: %v\n", to.IPBlock.Except)
									}
								}
							}
						}
					}
				}
			}

			// 5. Check VPC route tables
			logger.Info("Checking VPC route tables...")
			routeTables, err := awsClient.GetRouteTables(ctx, *vpcConfig.VpcId)
			if err != nil {
				logger.Warning("Failed to get route tables: %v", err)
			} else {
				logger.Success("✅ Found %d route tables", len(routeTables))
				for _, rt := range routeTables {
					fmt.Printf("\nRoute Table: %s\n", *rt.RouteTableId)
					for _, route := range rt.Routes {
						if route.DestinationCidrBlock != nil {
							fmt.Printf("  %s -> ", *route.DestinationCidrBlock)
							switch {
							case route.GatewayId != nil:
								fmt.Printf("IGW: %s\n", *route.GatewayId)
							case route.NatGatewayId != nil:
								fmt.Printf("NAT: %s\n", *route.NatGatewayId)
							case route.VpcPeeringConnectionId != nil:
								fmt.Printf("VPC Peering: %s\n", *route.VpcPeeringConnectionId)
							default:
								fmt.Println("Other")
							}
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to check network policies in (default is all namespaces)")
	return cmd
}

func newDebugCrossAccountCommand() *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:   "cross-account [cluster-name]",
		Short: "Debug cross-account access issues",
		Long: `Analyze cross-account access configuration including:
- IAM role trust relationships
- Cross-account resource access policies
- Cross-account networking configuration
- Cross-account service permissions`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()

			// Create AWS client
			awsClient, err := aws.NewClient(ctx, aws.ClientConfig{
				Profile: profile,
				Region:  region,
			})
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}

			// Get cluster details
			logger.Info("Getting cluster details for %s...", clusterName)
			cluster, err := awsClient.DescribeCluster(ctx, clusterName)
			if err != nil {
				return fmt.Errorf("failed to get cluster details: %w", err)
			}

			// 1. Check cluster role trust relationships
			logger.Info("Checking cluster IAM role trust relationships...")
			roleARN := *cluster.Cluster.RoleArn
			if err := aws.VerifyIAMRoleTrust(roleARN); err != nil {
				logger.Warning("❌ Cluster role trust relationship issue: %v", err)
			} else {
				logger.Success("✅ Cluster role trust relationship is valid")
			}

			// 2. Check node role trust relationships
			logger.Info("Checking node IAM role trust relationships...")
			nodegroups, err := awsClient.ListNodegroups(ctx, clusterName)
			if err != nil {
				logger.Warning("Failed to list nodegroups: %v", err)
			} else {
				for _, ng := range nodegroups {
					ngDetails, err := awsClient.DescribeNodegroup(ctx, clusterName, ng)
					if err != nil {
						logger.Warning("Failed to get details for nodegroup %s: %v", ng, err)
						continue
					}
					
					if err := aws.VerifyIAMRoleTrust(*ngDetails.Nodegroup.NodeRole); err != nil {
						logger.Warning("❌ Node role trust relationship issue for %s: %v", ng, err)
					} else {
						logger.Success("✅ Node role trust relationship is valid for nodegroup %s", ng)
					}
				}
			}

			// 3. Check addon service accounts
			logger.Info("Checking addon service account configurations...")
			addons, err := awsClient.ListAddons(ctx, clusterName)
			if err != nil {
				logger.Warning("Failed to list addons: %v", err)
			} else {
				for _, addon := range addons {
					addonDetails, err := awsClient.DescribeAddon(ctx, clusterName, addon)
					if err != nil {
						logger.Warning("Failed to get details for addon %s: %v", addon, err)
						continue
					}

					if addonDetails.Addon.ServiceAccountRoleArn != nil {
						roleARN := *addonDetails.Addon.ServiceAccountRoleArn
						if err := aws.VerifyIAMRoleTrust(roleARN); err != nil {
							logger.Warning("❌ Addon role trust relationship issue for %s: %v", addon, err)
						} else {
							logger.Success("✅ Addon role trust relationship is valid for %s", addon)
						}
					}
				}
			}

			// 4. Check cross-account VPC access
			logger.Info("Checking cross-account VPC access...")
			vpcConfig := cluster.Cluster.ResourcesVpcConfig
			if vpcConfig != nil && len(vpcConfig.SecurityGroupIds) > 0 {
				for _, sgID := range vpcConfig.SecurityGroupIds {
					if err := awsClient.ValidateSecurityGroupAccess(ctx, sgID); err != nil {
						logger.Warning("❌ Security group access issue for %s: %v", sgID, err)
					} else {
						logger.Success("✅ Security group access is valid for %s", sgID)
					}
				}
			}

			return nil
		},
	}

	return cmd
}

func newDebugTLSCommand() *cobra.Command {
	var (
		namespace string
	)

	cmd := &cobra.Command{
		Use:   "tls [cluster-name]",
		Short: "Debug TLS and certificate issues",
		Long: `Analyze TLS configurations and certificate validity including:
- API server certificate
- Service certificates
- Ingress TLS certificates
- Certificate expiration dates
- Certificate chain validation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()

			// Create k8s client
			kubeClient, err := getKubeClient()
			if err != nil {
				return fmt.Errorf("failed to create kubernetes client: %w", err)
			}

			// 1. Check API server certificate
			logger.Info("Checking API server certificate...")
			apiCert, err := kubeClient.GetAPIServerCertificate(ctx)
			if err != nil {
				logger.Warning("Failed to get API server certificate: %v", err)
			} else {
				fmt.Printf("\nAPI Server Certificate:\n")
				fmt.Printf("Subject: %s\n", apiCert.Subject)
				fmt.Printf("Issuer: %s\n", apiCert.Issuer)
				fmt.Printf("Valid Until: %s\n", apiCert.NotAfter.Format("2006-01-02 15:04:05 MST"))

				// Check expiration
				daysUntilExpiry := time.Until(apiCert.NotAfter).Hours() / 24
				if daysUntilExpiry < 30 {
					logger.Warning("❌ API server certificate expires in %.0f days", daysUntilExpiry)
				} else {
					logger.Success("✅ API server certificate is valid for %.0f more days", daysUntilExpiry)
				}
			}

			// 2. Check Ingress TLS certificates
			logger.Info("\nChecking Ingress TLS certificates...")
			ingCerts, err := kubeClient.GetIngressTLSCertificates(ctx, namespace)
			if err != nil {
				logger.Warning("Failed to get Ingress certificates: %v", err)
			} else {
				if len(ingCerts) == 0 {
					logger.Info("No Ingress TLS certificates found")
				} else {
					fmt.Printf("\nIngress TLS Certificates:\n")
					for host, cert := range ingCerts {
						fmt.Printf("\nHost: %s\n", host)
						fmt.Printf("Subject: %s\n", cert.Subject)
						fmt.Printf("Issuer: %s\n", cert.Issuer)
						fmt.Printf("Valid Until: %s\n", cert.NotAfter.Format("2006-01-02 15:04:05 MST"))

						daysUntilExpiry := time.Until(cert.NotAfter).Hours() / 24
						if daysUntilExpiry < 30 {
							logger.Warning("❌ Certificate expires in %.0f days", daysUntilExpiry)
						} else {
							logger.Success("✅ Certificate is valid for %.0f more days", daysUntilExpiry)
						}
					}
				}
			}

			// 3. Check service certificates (for services with TLS)
			logger.Info("\nChecking service certificates...")
			svcCerts, err := kubeClient.GetServiceCertificates(ctx, namespace)
			if err != nil {
				logger.Warning("Failed to get service certificates: %v", err)
			} else {
				if len(svcCerts) == 0 {
					logger.Info("No service TLS certificates found")
				} else {
					fmt.Printf("\nService TLS Certificates:\n")
					for svc, cert := range svcCerts {
						fmt.Printf("\nService: %s\n", svc)
						fmt.Printf("Subject: %s\n", cert.Subject)
						fmt.Printf("Issuer: %s\n", cert.Issuer)
						fmt.Printf("Valid Until: %s\n", cert.NotAfter.Format("2006-01-02 15:04:05 MST"))

						daysUntilExpiry := time.Until(cert.NotAfter).Hours() / 24
						if daysUntilExpiry < 30 {
							logger.Warning("❌ Certificate expires in %.0f days", daysUntilExpiry)
						} else {
							logger.Success("✅ Certificate is valid for %.0f more days", daysUntilExpiry)
						}
					}
				}
			}

			// 4. Check certificate chain validity
			logger.Info("\nValidating certificate chains...")
			chainIssues, err := kubeClient.ValidateCertificateChains(ctx, namespace)
			if err != nil {
				logger.Warning("Failed to validate certificate chains: %v", err)
			} else {
				if len(chainIssues) == 0 {
					logger.Success("✅ All certificate chains are valid")
				} else {
					logger.Warning("Found certificate chain issues:")
					for resource, issue := range chainIssues {
						fmt.Printf("- %s: %s\n", resource, issue)
					}
				}
			}

			// 5. Provide recommendations
			fmt.Printf("\nRecommendations:\n")
			anyIssues := false

			if daysUntilExpiry := time.Until(apiCert.NotAfter).Hours() / 24; daysUntilExpiry < 90 {
				fmt.Printf("1. Plan to rotate API server certificate within %.0f days\n", daysUntilExpiry)
				anyIssues = true
			}

			for host, cert := range ingCerts {
				if daysUntilExpiry := time.Until(cert.NotAfter).Hours() / 24; daysUntilExpiry < 30 {
					fmt.Printf("2. Renew certificate for %s (expires in %.0f days)\n", host, daysUntilExpiry)
					anyIssues = true
				}
			}

			if len(chainIssues) > 0 {
				fmt.Printf("3. Fix certificate chain issues for identified resources\n")
				anyIssues = true
			}

			if !anyIssues {
				logger.Success("✅ No immediate TLS or certificate issues found")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to check certificates in (default is all namespaces)")
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

			// Create kubernetes client
			kubeClient, err := getKubeClient()
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
