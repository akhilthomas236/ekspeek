package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

var clusterName string

// ClientConfig holds the configuration for the AWS client
type ClientConfig struct {
	Profile string
	Region  string
}

// Client is the struct that holds the AWS services clients
type Client struct {
	EKSClient       *eks.Client
	CloudWatchClient *cloudwatch.Client
	IAMClient       *iam.Client
}

// NewClient creates a new AWS client
func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}
	if cfg.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(cfg.Profile))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	return &Client{
		EKSClient:       eks.NewFromConfig(awsCfg),
		CloudWatchClient: cloudwatch.NewFromConfig(awsCfg),
		IAMClient:       iam.NewFromConfig(awsCfg),
	}, nil
}

// ValidateNodeGroupsConfig validates the configuration of node groups
func (c *Client) ValidateNodeGroupsConfig(ctx context.Context, clusterName string) error {
	input := &eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterName),
	}

	result, err := c.EKSClient.ListNodegroups(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to list nodegroups: %w", err)
	}

	for _, ng := range result.Nodegroups {
		descInput := &eks.DescribeNodegroupInput{
			ClusterName:   aws.String(clusterName),
			NodegroupName: aws.String(ng),
		}

		desc, err := c.EKSClient.DescribeNodegroup(ctx, descInput)
		if err != nil {
			return fmt.Errorf("failed to describe nodegroup %s: %w", ng, err)
		}

		// Check for common misconfiguration issues
		if desc.Nodegroup.ScalingConfig.MinSize == nil || desc.Nodegroup.ScalingConfig.MaxSize == nil {
			return fmt.Errorf("nodegroup %s has invalid scaling configuration", ng)
		}

		if *desc.Nodegroup.ScalingConfig.MinSize > *desc.Nodegroup.ScalingConfig.MaxSize {
			return fmt.Errorf("nodegroup %s has min size greater than max size", ng)
		}
	}

	return nil
}

// ListClusters lists all EKS clusters in the current region
func (c *Client) ListClusters(ctx context.Context) ([]string, error) {
	input := &eks.ListClustersInput{}
	result, err := c.EKSClient.ListClusters(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	return result.Clusters, nil
}

// DescribeCluster gets detailed information about an EKS cluster
func (c *Client) DescribeCluster(ctx context.Context, clusterName string) (*eks.DescribeClusterOutput, error) {
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	result, err := c.EKSClient.DescribeCluster(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster %s: %w", clusterName, err)
	}

	return result, nil
}

// VerifyIAMRoleTrust checks if the IAM role trust relationship is configured correctly
func VerifyIAMRoleTrust(roleARN string) error {
	client := iam.NewFromConfig(aws.Config{})

	// Extract role name from ARN
	roleName := extractRoleNameFromARN(roleARN)

	input := &iam.GetRolePolicyInput{
		RoleName: aws.String(roleName),
	}

	resp, err := client.GetRolePolicy(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to get role policy: %w", err)
	}

	// Validate trust relationship
	if !validateTrustPolicy(*resp.PolicyDocument) {
		return fmt.Errorf("invalid trust relationship for role %s", roleName)
	}

	return nil
}

// ThrottlingMetrics holds the throttling metrics for EKS
type ThrottlingMetrics struct {
	APICallCount    float64
	ThrottledCalls  float64
	ErrorRate       float64
	Timestamp       time.Time
}

// PerformanceMetrics holds the performance metrics for EKS
type PerformanceMetrics struct {
	CPUUtilization    float64
	MemoryUtilization float64
	NetworkIn         float64
	NetworkOut        float64
	Timestamp         time.Time
}

// GetEKSThrottlingMetrics retrieves API throttling metrics from CloudWatch
func GetEKSThrottlingMetrics(ctx context.Context, clusterName string) ([]ThrottlingMetrics, error) {
	client := cloudwatch.NewFromConfig(aws.Config{})

	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	input := &cloudwatch.GetMetricDataInput{
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("apiCalls"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String("AWS/EKS"),
						MetricName: aws.String("APICallCount"),
						Dimensions: []types.Dimension{
							{
								Name:  aws.String("ClusterName"),
								Value: aws.String(clusterName),
							},
						},
					},
					Period: aws.Int32(300), // 5 minute periods
					Stat:   aws.String("Sum"),
				},
			},
			{
				Id: aws.String("throttles"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String("AWS/EKS"),
						MetricName: aws.String("Throttles"),
						Dimensions: []types.Dimension{
							{
								Name:  aws.String("ClusterName"),
								Value: aws.String(clusterName),
							},
						},
					},
					Period: aws.Int32(300),
					Stat:   aws.String("Sum"),
				},
			},
		},
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	output, err := client.GetMetricData(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get CloudWatch metrics: %w", err)
	}

	metrics := make([]ThrottlingMetrics, 0)
	for i := range output.MetricDataResults[0].Timestamps {
		metric := ThrottlingMetrics{
			APICallCount:   output.MetricDataResults[0].Values[i],
			ThrottledCalls: output.MetricDataResults[1].Values[i],
			Timestamp:      output.MetricDataResults[0].Timestamps[i],
		}
		if metric.APICallCount > 0 {
			metric.ErrorRate = (metric.ThrottledCalls / metric.APICallCount) * 100
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// GetEKSPerformanceMetrics retrieves cluster performance metrics from CloudWatch
func GetEKSPerformanceMetrics(ctx context.Context, clusterName string) ([]PerformanceMetrics, error) {
	client := cloudwatch.NewFromConfig(aws.Config{})

	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	input := &cloudwatch.GetMetricDataInput{
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("cpu"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String("AWS/EKS"),
						MetricName: aws.String("cluster_cpu_utilization"),
						Dimensions: []types.Dimension{
							{
								Name:  aws.String("ClusterName"),
								Value: aws.String(clusterName),
							},
						},
					},
					Period: aws.Int32(300),
					Stat:   aws.String("Average"),
				},
			},
			{
				Id: aws.String("memory"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String("AWS/EKS"),
						MetricName: aws.String("cluster_memory_utilization"),
						Dimensions: []types.Dimension{
							{
								Name:  aws.String("ClusterName"),
								Value: aws.String(clusterName),
							},
						},
					},
					Period: aws.Int32(300),
					Stat:   aws.String("Average"),
				},
			},
		},
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	output, err := client.GetMetricData(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get CloudWatch metrics: %w", err)
	}

	metrics := make([]PerformanceMetrics, 0)
	for i := 0; i < len(output.MetricDataResults[0].Timestamps); i++ {
		metric := PerformanceMetrics{
			CPUUtilization:    output.MetricDataResults[0].Values[i],
			MemoryUtilization: output.MetricDataResults[1].Values[i],
			Timestamp:         output.MetricDataResults[0].Timestamps[i],
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// Helper functions
func extractRoleNameFromARN(arn string) string {
	// ARN format: arn:aws:iam::123456789012:role/role-name
	parts := strings.Split(arn, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

func validateTrustPolicy(policy string) bool {
	// Simple validation - check if the policy contains required elements
	requiredElements := []string{
		"\"Service\"", "\"eks.amazonaws.com\"",
		"\"Action\"", "\"sts:AssumeRole\"",
		"\"Effect\"", "\"Allow\"",
	}

	for _, element := range requiredElements {
		if !strings.Contains(policy, element) {
			return false
		}
	}

	return true
}
