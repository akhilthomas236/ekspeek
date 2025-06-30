package cmd

import (
	"context"
	"fmt"
	"strings"

	"ekspeek/pkg/common/logger"
	"ekspeek/pkg/k8s"

	"github.com/spf13/cobra"
)

func newHealthCheckCommand() *cobra.Command {
	var (
		clusterName string
		components  []string
	)

	cmd := &cobra.Command{
		Use:   "health [cluster-name]",
		Short: "Perform comprehensive health check of the EKS cluster",
		Long: `Performs a comprehensive health check of the EKS cluster including:
- Version mismatches between control plane and node groups
- Deprecated API usage
- Logging and monitoring components
- Load balancer and ingress issues
- Networking and DNS issues
- Pod scheduling and resource issues
- Authentication and authorization issues
- Node group and worker node issues`,
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
			kubeClient, err := getKubeClient()
			if err != nil {
				return err
			}

			// Perform health check
			logger.Info("Performing comprehensive health check...")
			status, err := kubeClient.CheckClusterHealth(ctx)
			if err != nil {
				return err
			}

			// Print results based on components flag or all if none specified
			if len(components) == 0 {
				printFullHealthStatus(status)
			} else {
				printSelectedHealthStatus(status, components)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&components, "components", "c", []string{}, 
		"Comma-separated list of components to check (versions,apis,logging,network,lb,scheduling,auth,nodes)")
	return cmd
}

func printFullHealthStatus(status *k8s.ClusterHealthStatus) {
	// Version mismatches
	logger.Info("\n=== Version Status ===")
	if len(status.NodeVersions) > 1 {
		logger.Warning("❌ Version mismatch detected:")
		for version, nodes := range status.NodeVersions {
			fmt.Printf("Version %s: %d nodes (%s)\n", version, len(nodes), strings.Join(nodes, ", "))
		}
	} else {
		logger.Success("✅ All nodes running same Kubernetes version")
	}

	// Deprecated APIs
	logger.Info("\n=== API Status ===")
	if len(status.DeprecatedAPIs) > 0 {
		logger.Warning("❌ Deprecated API usage detected:")
		for _, api := range status.DeprecatedAPIs {
			fmt.Printf("- %s\n", api)
		}
	} else {
		logger.Success("✅ No deprecated API usage found")
	}

	// Logging Status
	logger.Info("\n=== Logging & Monitoring Status ===")
	printLoggingStatus(status.LoggingStatus)

	// Networking Status
	logger.Info("\n=== Networking Status ===")
	printNetworkingStatus(status.NetworkingStatus)

	// Load Balancer Status
	logger.Info("\n=== Load Balancer & Ingress Status ===")
	printLoadBalancerStatus(status.LoadBalancerStatus)

	// Scheduling Status
	logger.Info("\n=== Scheduling & Resource Status ===")
	printSchedulingStatus(status.SchedulingStatus)

	// Auth Status
	logger.Info("\n=== Authentication & Authorization Status ===")
	printAuthStatus(status.AuthStatus)

	// Node Status
	logger.Info("\n=== Node Status ===")
	printNodeStatus(status.NodeStatus)
}

func printSelectedHealthStatus(status *k8s.ClusterHealthStatus, components []string) {
	for _, component := range components {
		switch strings.ToLower(component) {
		case "versions":
			logger.Info("\n=== Version Status ===")
			for version, nodes := range status.NodeVersions {
				fmt.Printf("Version %s: %d nodes\n", version, len(nodes))
			}
		case "apis":
			logger.Info("\n=== API Status ===")
			for _, api := range status.DeprecatedAPIs {
				fmt.Printf("- %s\n", api)
			}
		case "logging":
			logger.Info("\n=== Logging & Monitoring Status ===")
			printLoggingStatus(status.LoggingStatus)
		case "network":
			logger.Info("\n=== Networking Status ===")
			printNetworkingStatus(status.NetworkingStatus)
		case "lb":
			logger.Info("\n=== Load Balancer & Ingress Status ===")
			printLoadBalancerStatus(status.LoadBalancerStatus)
		case "scheduling":
			logger.Info("\n=== Scheduling & Resource Status ===")
			printSchedulingStatus(status.SchedulingStatus)
		case "auth":
			logger.Info("\n=== Authentication & Authorization Status ===")
			printAuthStatus(status.AuthStatus)
		case "nodes":
			logger.Info("\n=== Node Status ===")
			printNodeStatus(status.NodeStatus)
		}
	}
}

func printLoggingStatus(status k8s.LoggingStatus) {
	// FluentBit Status
	if len(status.FluentBitStatus) > 0 {
		fmt.Println("\nFluentBit Status:")
		for _, pod := range status.FluentBitStatus {
			if pod.Status != "Running" {
				logger.Warning("❌ FluentBit pod %s is %s: %s", pod.Name, pod.Status, pod.Message)
			} else {
				logger.Success("✅ FluentBit pod %s is running", pod.Name)
			}
		}
	} else {
		logger.Warning("⚠️ FluentBit not detected in cluster")
	}

	// CloudWatch Status
	if len(status.CloudWatchStatus) > 0 {
		fmt.Println("\nCloudWatch Agent Status:")
		for _, pod := range status.CloudWatchStatus {
			if pod.Status != "Running" {
				logger.Warning("❌ CloudWatch pod %s is %s: %s", pod.Name, pod.Status, pod.Message)
			} else {
				logger.Success("✅ CloudWatch pod %s is running", pod.Name)
			}
		}
	} else {
		logger.Warning("⚠️ CloudWatch Agent not detected in cluster")
	}

	// Metrics Server Status
	if len(status.MetricsServerStatus) > 0 {
		fmt.Println("\nMetrics Server Status:")
		for _, pod := range status.MetricsServerStatus {
			if pod.Status != "Running" {
				logger.Warning("❌ Metrics Server pod %s is %s: %s", pod.Name, pod.Status, pod.Message)
			} else {
				logger.Success("✅ Metrics Server pod %s is running", pod.Name)
			}
		}
	} else {
		logger.Warning("⚠️ Metrics Server not detected in cluster")
	}
}

func printNetworkingStatus(status k8s.NetworkingStatus) {
	// CNI Status
	if len(status.CNIStatus) > 0 {
		fmt.Println("\nAWS CNI Status:")
		for _, pod := range status.CNIStatus {
			if pod.Status != "Running" {
				logger.Warning("❌ AWS CNI pod %s is %s: %s", pod.Name, pod.Status, pod.Message)
			} else {
				logger.Success("✅ AWS CNI pod %s is running", pod.Name)
			}
		}
	}

	// CoreDNS Status
	if len(status.CoreDNSStatus) > 0 {
		fmt.Println("\nCoreDNS Status:")
		for _, pod := range status.CoreDNSStatus {
			if pod.Status != "Running" {
				logger.Warning("❌ CoreDNS pod %s is %s: %s", pod.Name, pod.Status, pod.Message)
			} else {
				logger.Success("✅ CoreDNS pod %s is running", pod.Name)
			}
		}
	}

	// External Access & DNS Resolution
	if status.ExternalAccess {
		logger.Success("✅ External network access is working")
	} else {
		logger.Warning("❌ External network access issues detected")
	}

	if status.DNSResolution {
		logger.Success("✅ DNS resolution is working")
	} else {
		logger.Warning("❌ DNS resolution issues detected")
	}
}

func printLoadBalancerStatus(status k8s.LoadBalancerStatus) {
	if len(status.PendingServices) > 0 {
		logger.Warning("❌ Services pending LoadBalancer provisioning:")
		for _, svc := range status.PendingServices {
			fmt.Printf("- %s\n", svc)
		}
	} else {
		logger.Success("✅ All LoadBalancer services are provisioned")
	}

	if len(status.IngressStatus) > 0 {
		fmt.Println("\nIngress Status:")
		for _, ing := range status.IngressStatus {
			if ing.Status == "Ready" {
				logger.Success("✅ Ingress %s/%s is ready", ing.Namespace, ing.Name)
			} else {
				logger.Warning("❌ Ingress %s/%s is %s", ing.Namespace, ing.Name, ing.Status)
				for _, problem := range ing.Problems {
					fmt.Printf("  - %s\n", problem)
				}
			}
		}
	}
}

func printSchedulingStatus(status k8s.SchedulingStatus) {
	if len(status.PendingPods) > 0 {
		logger.Warning("❌ Pods pending scheduling:")
		for _, pod := range status.PendingPods {
			fmt.Printf("- %s/%s: %s\n", pod.Namespace, pod.Pod, pod.Reason)
		}
	} else {
		logger.Success("✅ All pods are scheduled")
	}

	if len(status.ResourceIssues) > 0 {
		logger.Warning("\n❌ Resource issues detected:")
		for _, issue := range status.ResourceIssues {
			fmt.Printf("Node %s:\n", issue.NodeName)
			fmt.Printf("  CPU: %.1f%% utilized (%.1f/%.1f cores)\n",
				issue.CPU.Utilization,
				float64(issue.CPU.Allocated)/1000,
				float64(issue.CPU.Capacity)/1000)
			fmt.Printf("  Memory: %.1f%% utilized (%.1f/%.1f GB)\n",
				issue.Memory.Utilization,
				float64(issue.Memory.Allocated)/(1024*1024*1024),
				float64(issue.Memory.Capacity)/(1024*1024*1024))
		}
	}
}

func printAuthStatus(status k8s.AuthStatus) {
	if len(status.IRSAIssues) > 0 {
		logger.Warning("❌ IRSA issues detected:")
		for _, issue := range status.IRSAIssues {
			fmt.Printf("- %s\n", issue)
		}
	} else {
		logger.Success("✅ No IRSA issues detected")
	}

	if len(status.RBACIssues) > 0 {
		logger.Warning("\n❌ RBAC issues detected:")
		for _, issue := range status.RBACIssues {
			fmt.Printf("- %s\n", issue)
		}
	} else {
		logger.Success("✅ No RBAC issues detected")
	}

	if len(status.IAMAuthIssues) > 0 {
		logger.Warning("\n❌ IAM authentication issues detected:")
		for _, issue := range status.IAMAuthIssues {
			fmt.Printf("- %s\n", issue)
		}
	} else {
		logger.Success("✅ No IAM authentication issues detected")
	}
}

func printNodeStatus(status k8s.NodeStatus) {
	if len(status.NotReady) > 0 {
		logger.Warning("❌ Nodes in NotReady state:")
		for _, node := range status.NotReady {
			fmt.Printf("- %s\n", node)
		}
	} else {
		logger.Success("✅ All nodes are Ready")
	}

	if len(status.ASGIssues) > 0 {
		logger.Warning("\n❌ Auto Scaling Group issues:")
		for _, issue := range status.ASGIssues {
			fmt.Printf("- %s\n", issue)
		}
	}

	if len(status.BootstrapIssues) > 0 {
		logger.Warning("\n❌ Node bootstrap issues:")
		for _, issue := range status.BootstrapIssues {
			fmt.Printf("- %s\n", issue)
		}
	}
}
