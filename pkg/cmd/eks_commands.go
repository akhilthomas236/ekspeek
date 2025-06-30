package cmd

import (
	"context"
	"fmt"

	"ekspeek/pkg/aws"
	"ekspeek/pkg/common/logger"
	"ekspeek/pkg/eks"

	"github.com/spf13/cobra"
)

// NewEKSCommand creates the root command and all its subcommands
func NewEKSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ekspeek",
		Short: "A tool for inspecting and managing EKS clusters",
		Long: `ekspeek is a command-line tool that helps you inspect and manage
your Amazon EKS clusters. It provides commands for listing clusters,
describing their configuration, and managing their components.`,
	}

	// Add global flags
	cmd.PersistentFlags().StringVar(&profile, "profile", "", "AWS profile to use")
	cmd.PersistentFlags().StringVar(&region, "region", "", "AWS region to use")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")

	// Add all subcommands
	cmd.AddCommand(
		NewListClustersCmd(),
		NewDescribeClusterCmd(),
		NewListNodegroupsCmd(),
		NewDescribeNodegroupCmd(),
		NewDebugCommand(),
		newClusterHealthCommand(),
	)

	return cmd
}

// NewListClustersCmd creates a command to list EKS clusters
func NewListClustersCmd() *cobra.Command {
	return newListClustersCmd()
}

// NewDescribeClusterCmd creates a command to describe an EKS cluster
func NewDescribeClusterCmd() *cobra.Command {
	return newDescribeClusterCmd()
}

// NewListNodegroupsCmd creates a command to list nodegroups
func NewListNodegroupsCmd() *cobra.Command {
	return newListNodegroupsCmd()
}

// NewDescribeNodegroupCmd creates a command to describe a nodegroup
func NewDescribeNodegroupCmd() *cobra.Command {
	return newDescribeNodegroupCmd()
}

func newListClustersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all EKS clusters",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := aws.NewClient(ctx, aws.ClientConfig{
				Profile: profile,
				Region:  region,
			})
			if err != nil {
				return err
			}

			handler := eks.NewHandler(client.EKSClient)
			clusters, err := handler.ListClusters(ctx)
			if err != nil {
				return err
			}

			if len(clusters) == 0 {
				logger.Info("No EKS clusters found in region %s", region)
				return nil
			}

			logger.Success("Found %d clusters:", len(clusters))
			for _, cluster := range clusters {
				fmt.Println(cluster)
			}

			return nil
		},
	}
}

func newDescribeClusterCmd() *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:   "describe [cluster-name]",
		Short: "Describe an EKS cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()
			client, err := aws.NewClient(ctx, aws.ClientConfig{
				Profile: profile,
				Region:  region,
			})
			if err != nil {
				return err
			}

			handler := eks.NewHandler(client.EKSClient)
			cluster, err := handler.DescribeCluster(ctx, clusterName)
			if err != nil {
				return err
			}

			// Print cluster details in a formatted way
			fmt.Printf("Name: %s\n", *cluster.Name)
			fmt.Printf("Version: %s\n", *cluster.Version)
			fmt.Printf("Status: %s\n", cluster.Status)
			fmt.Printf("Endpoint: %s\n", *cluster.Endpoint)
			fmt.Printf("ARN: %s\n", *cluster.Arn)
			fmt.Printf("Created: %s\n", cluster.CreatedAt.Format("2006-01-02 15:04:05"))
			
			return nil
		},
	}

	return cmd
}

func newListNodegroupsCmd() *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:   "list-nodegroups [cluster-name]",
		Short: "List all nodegroups in an EKS cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clusterName = args[0]
			}
			if clusterName == "" {
				return fmt.Errorf("cluster name is required")
			}

			ctx := context.Background()
			client, err := aws.NewClient(ctx, aws.ClientConfig{
				Profile: profile,
				Region:  region,
			})
			if err != nil {
				return err
			}

			handler := eks.NewHandler(client.EKSClient)
			nodegroups, err := handler.ListNodegroups(ctx, clusterName)
			if err != nil {
				return err
			}

			if len(nodegroups) == 0 {
				logger.Info("No nodegroups found in cluster %s", clusterName)
				return nil
			}

			logger.Success("Found %d nodegroups:", len(nodegroups))
			for _, ng := range nodegroups {
				fmt.Println(ng)
			}

			return nil
		},
	}

	return cmd
}

func newDescribeNodegroupCmd() *cobra.Command {
	var (
		clusterName   string
		nodegroupName string
	)

	cmd := &cobra.Command{
		Use:   "describe-nodegroup [cluster-name] [nodegroup-name]",
		Short: "Describe a nodegroup in an EKS cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("both cluster name and nodegroup name are required")
			}
			clusterName = args[0]
			nodegroupName = args[1]

			ctx := context.Background()
			client, err := aws.NewClient(ctx, aws.ClientConfig{
				Profile: profile,
				Region:  region,
			})
			if err != nil {
				return err
			}

			handler := eks.NewHandler(client.EKSClient)
			nodegroup, err := handler.DescribeNodegroup(ctx, clusterName, nodegroupName)
			if err != nil {
				return err
			}

			// Print nodegroup details in a formatted way
			fmt.Printf("Nodegroup Name: %s\n", *nodegroup.NodegroupName)
			fmt.Printf("Status: %s\n", nodegroup.Status)
			fmt.Printf("Cluster Name: %s\n", *nodegroup.ClusterName)
			fmt.Printf("Instance Types: %v\n", nodegroup.InstanceTypes)
			fmt.Printf("Desired Size: %d\n", nodegroup.ScalingConfig.DesiredSize)
			fmt.Printf("Min Size: %d\n", nodegroup.ScalingConfig.MinSize)
			fmt.Printf("Max Size: %d\n", nodegroup.ScalingConfig.MaxSize)
			fmt.Printf("Created: %s\n", nodegroup.CreatedAt.Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	return cmd
}
