package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ekspeek/pkg/k8s"
	"ekspeek/pkg/common/logger"

	"github.com/spf13/cobra"
)

// ClusterHealthCheckConfig contains the configuration for the health check command
type ClusterHealthCheckConfig struct {
	ExcludeComponents []string
	Namespace        string
	Timeout         time.Duration
}

func newClusterHealthCommand() *cobra.Command {
	var (
		clusterName string
		profile    string
		region     string
		cfg        ClusterHealthCheckConfig
	)

	cmd := &cobra.Command{
		Use:   "cluster-health [cluster-name]",
		Short: "Comprehensive health check for EKS cluster",
		Long: `Performs a thorough health check of the EKS cluster including:
  • Control Plane Status
    - API Server availability
    - Controller Manager health
    - Scheduler health
    - etcd cluster health

  • Core Components
    - CoreDNS functionality
    - kube-proxy status
    - CNI plugin health
    
  • Node Health
    - Node status and conditions
    - Kubelet health
    - System pods status
    - Resource utilization
    
  • Workload Health
    - Pod status across namespaces
    - Deployment status
    - StatefulSet health
    - DaemonSet status
    
  • Networking
    - DNS resolution
    - Network policy validation
    - Service endpoints health
    - Load balancer status
    
  • Storage
    - PVC/PV status
    - StorageClass availability
    - Volume health
    
  • Security
    - Certificate expiration
    - RBAC configuration
    - Pod security policies
    - Network policies
    
  • Logging & Monitoring
    - Metrics server status
    - Logging agent health
    - CloudWatch integration
    
  • Resource Utilization
    - CPU/Memory usage
    - Pod density
    - Resource quotas
    - Limit ranges`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()
			if cfg.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
				defer cancel()
			}

			// Create kubernetes client using default kubeconfig or KUBECONFIG env var
			kubeClient, err := getKubeClient()
			if err != nil {
				return fmt.Errorf("failed to create kubernetes client: %w", err)
			}

			logger.Info("Starting comprehensive cluster health check for %s...", clusterName)

			// Get cluster health status
			status, err := kubeClient.CheckClusterHealth(ctx)
			if err != nil {
				return fmt.Errorf("failed to check cluster health: %w", err)
			}

			// Print section headers in a more visible way
			fmt.Println("\n" + strings.Repeat("=", 80))
			fmt.Println("EKS CLUSTER HEALTH CHECK RESULTS")
			fmt.Println("Cluster: " + clusterName)
			fmt.Println("Time: " + time.Now().Format(time.RFC1123))
			fmt.Println(strings.Repeat("=", 80))

			// Control Plane Status
			if !contains(cfg.ExcludeComponents, "control-plane") {
				logger.Info("\n=== Control Plane Status ===")
				printControlPlaneStatus(status)
			}

			// Core Components Status
			if !contains(cfg.ExcludeComponents, "core") {
				logger.Info("\n=== Core Components Status ===")
				printCoreComponentsStatus(status)
			}

			// Node Health
			if !contains(cfg.ExcludeComponents, "nodes") {
				logger.Info("\n=== Node Health ===")
				printNodeStatus(status.NodeStatus)
			}

			// Workload Health
			if !contains(cfg.ExcludeComponents, "workloads") {
				logger.Info("\n=== Workload Health ===")
				printWorkloadStatus(status, cfg.Namespace)
			}

			// Networking Status
			if !contains(cfg.ExcludeComponents, "networking") {
				logger.Info("\n=== Networking Status ===")
				printNetworkingStatus(status.NetworkingStatus)
			}

			// Storage Status
			if !contains(cfg.ExcludeComponents, "storage") {
				logger.Info("\n=== Storage Status ===")
				printStorageStatus(status)
			}

			// Security Status
			if !contains(cfg.ExcludeComponents, "security") {
				logger.Info("\n=== Security Status ===")
				printSecurityStatus(status)
			}

			// Logging & Monitoring
			if !contains(cfg.ExcludeComponents, "logging") {
				logger.Info("\n=== Logging & Monitoring Status ===")
				printLoggingStatus(status.LoggingStatus)
			}

			// Resource Utilization
			if !contains(cfg.ExcludeComponents, "resources") {
				logger.Info("\n=== Resource Utilization ===")
				printResourceUtilization(status)
			}

			// Add summary section at the end
			fmt.Println("\n" + strings.Repeat("=", 80))
			fmt.Println("SUMMARY")
			fmt.Println(strings.Repeat("=", 80))
			printHealthSummary(status)

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&profile, "profile", "", "AWS profile to use")
	cmd.Flags().StringVar(&region, "region", "", "AWS region of the EKS cluster")
	cmd.Flags().StringSliceVar(&cfg.ExcludeComponents, "exclude", []string{},
		"Components to exclude from health check (comma-separated: control-plane,core,nodes,workloads,networking,storage,security,logging,resources)")
	cmd.Flags().StringVarP(&cfg.Namespace, "namespace", "n", "",
		"Namespace to check (default is all namespaces)")
	cmd.Flags().DurationVar(&cfg.Timeout, "timeout", 5*time.Minute,
		"Timeout for the health check (e.g. 5m, 1h)")

	return cmd
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, str) {
			return true
		}
	}
	return false
}

func printControlPlaneStatus(status *k8s.ClusterHealthStatus) {
	// Print control plane component status
	if len(status.NodeVersions) > 1 {
		logger.Warning("❌ Version mismatch detected:")
		for version, nodes := range status.NodeVersions {
			fmt.Printf("Version %s: %d nodes (%s)\n", version, len(nodes), strings.Join(nodes, ", "))
		}
	} else {
		logger.Success("✅ All nodes running same Kubernetes version")
	}
}

func printCoreComponentsStatus(status *k8s.ClusterHealthStatus) {
	// CoreDNS Status
	if len(status.NetworkingStatus.CoreDNSStatus) > 0 {
		fmt.Println("\nCoreDNS Status:")
		for _, pod := range status.NetworkingStatus.CoreDNSStatus {
			if pod.Status != "Running" {
				logger.Warning("❌ CoreDNS pod %s is %s: %s", pod.Name, pod.Status, pod.Message)
			} else {
				logger.Success("✅ CoreDNS pod %s is running", pod.Name)
			}
		}
	}

	// CNI Status
	if len(status.NetworkingStatus.CNIStatus) > 0 {
		fmt.Println("\nCNI Status:")
		for _, pod := range status.NetworkingStatus.CNIStatus {
			if pod.Status != "Running" {
				logger.Warning("❌ CNI pod %s is %s: %s", pod.Name, pod.Status, pod.Message)
			} else {
				logger.Success("✅ CNI pod %s is running", pod.Name)
			}
		}
	}
}

func printWorkloadStatus(status *k8s.ClusterHealthStatus, namespace string) {
	if len(status.SchedulingStatus.PendingPods) > 0 {
		logger.Warning("❌ Pods pending scheduling:")
		for _, pod := range status.SchedulingStatus.PendingPods {
			if namespace == "" || namespace == pod.Namespace {
				fmt.Printf("- %s/%s: %s\n", pod.Namespace, pod.Pod, pod.Reason)
			}
		}
	} else {
		logger.Success("✅ All pods are scheduled correctly")
	}

	// Add StatefulSet status
	if len(status.StatefulSetStatus) > 0 {
		fmt.Println("\nStatefulSet Status:")
		for _, sts := range status.StatefulSetStatus {
			if sts.ReadyReplicas != sts.DesiredReplicas {
				logger.Warning("❌ StatefulSet %s/%s: %d/%d replicas ready",
					sts.Namespace, sts.Name, sts.ReadyReplicas, sts.DesiredReplicas)
			} else {
				logger.Success("✅ StatefulSet %s/%s: %d/%d replicas ready",
					sts.Namespace, sts.Name, sts.ReadyReplicas, sts.DesiredReplicas)
			}
		}
	}

	// Add DaemonSet status
	if len(status.DaemonSetStatus) > 0 {
		fmt.Println("\nDaemonSet Status:")
		for _, ds := range status.DaemonSetStatus {
			if ds.NumberUnavailable > 0 {
				logger.Warning("❌ DaemonSet %s/%s: %d pods unavailable",
					ds.Namespace, ds.Name, ds.NumberUnavailable)
			} else {
				logger.Success("✅ DaemonSet %s/%s: all pods ready",
					ds.Namespace, ds.Name)
			}
		}
	}
}

func printStorageStatus(status *k8s.ClusterHealthStatus) {
	// Check PVC status
	if len(status.PVCStatus) > 0 {
		fmt.Println("\nPersistent Volume Claims:")
		for _, pvc := range status.PVCStatus {
			if pvc.Status.Phase != "Bound" {
				logger.Warning("❌ PVC %s/%s is %s", pvc.Namespace, pvc.Name, pvc.Status.Phase)
			} else {
				logger.Success("✅ PVC %s/%s is bound", pvc.Namespace, pvc.Name)
			}
		}
	}

	// Check StorageClass status
	if len(status.StorageClasses) > 0 {
		fmt.Println("\nStorage Classes:")
		for _, sc := range status.StorageClasses {
			if sc.DefaultClass {
				logger.Info("ℹ️ Default StorageClass: %s", sc.Name)
			} else {
				logger.Info("ℹ️ StorageClass: %s", sc.Name)
			}
		}
	} else {
		logger.Warning("⚠️ No StorageClasses found in cluster")
	}
}

func printSecurityStatus(status *k8s.ClusterHealthStatus) {
	if len(status.DeprecatedAPIs) > 0 {
		logger.Warning("❌ Deprecated API usage detected:")
		for _, api := range status.DeprecatedAPIs {
			fmt.Printf("- %s\n", api)
		}
	} else {
		logger.Success("✅ No deprecated API usage found")
	}

	if len(status.AuthStatus.IRSAIssues) > 0 {
		logger.Warning("\n❌ IRSA issues detected:")
		for _, issue := range status.AuthStatus.IRSAIssues {
			fmt.Printf("- %s\n", issue)
		}
	} else {
		logger.Success("✅ No IRSA issues detected")
	}

	if len(status.AuthStatus.RBACIssues) > 0 {
		logger.Warning("\n❌ RBAC issues detected:")
		for _, issue := range status.AuthStatus.RBACIssues {
			fmt.Printf("- %s\n", issue)
		}
	} else {
		logger.Success("✅ No RBAC issues detected")
	}
}

func printResourceUtilization(status *k8s.ClusterHealthStatus) {
	fmt.Printf("\nCluster Resource Usage by Node:\n")

	if len(status.SchedulingStatus.ResourceIssues) > 0 {
		fmt.Printf("\nNodes with Resource Pressure:\n")
		for _, issue := range status.SchedulingStatus.ResourceIssues {
			fmt.Printf("\nNode %s:\n", issue.NodeName)
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

func printHealthSummary(status *k8s.ClusterHealthStatus) {
	var (
		totalIssues    int
		criticalIssues int
	)

	// Count issues by category
	if len(status.NodeVersions) > 1 {
		criticalIssues++ // Version mismatch is critical
	}
	totalIssues += len(status.DeprecatedAPIs)
	totalIssues += len(status.AuthStatus.IRSAIssues)
	totalIssues += len(status.AuthStatus.RBACIssues)
	totalIssues += len(status.NodeStatus.NotReady)
	totalIssues += len(status.SchedulingStatus.PendingPods)
	totalIssues += len(status.LoadBalancerStatus.PendingServices)

	if criticalIssues > 0 {
		logger.Warning("Found %d critical issues that need immediate attention", criticalIssues)
	}
	
	if totalIssues > 0 {
		logger.Warning("Total issues found: %d", totalIssues)
		fmt.Println("\nRecommended actions:")
		if len(status.NodeVersions) > 1 {
			fmt.Println("1. Upgrade nodes to match control plane version")
		}
		if len(status.DeprecatedAPIs) > 0 {
			fmt.Println("2. Update applications using deprecated APIs")
		}
		if len(status.AuthStatus.IRSAIssues) > 0 {
			fmt.Println("3. Fix IRSA configuration issues")
		}
		if len(status.AuthStatus.RBACIssues) > 0 {
			fmt.Println("4. Review and fix RBAC issues")
		}
		if len(status.NodeStatus.NotReady) > 0 {
			fmt.Println("5. Investigate nodes in NotReady state")
		}
		if len(status.SchedulingStatus.PendingPods) > 0 {
			fmt.Println("6. Address pod scheduling issues")
		}
		if len(status.LoadBalancerStatus.PendingServices) > 0 {
			fmt.Println("7. Check LoadBalancer provisioning issues")
		}
	} else {
		logger.Success("No issues found - cluster is healthy!")
	}
}
