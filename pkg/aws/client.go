package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
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
	EKSClient        *eks.Client
	EC2Client        *ec2.Client
	CloudWatchClient *cloudwatch.Client
	IAMClient        *iam.Client
}

// NATGatewayInfo contains information about a NAT gateway
type NATGatewayInfo struct {
    NatGatewayId *string
    State        string
    SubnetId     *string
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
		EKSClient:        eks.NewFromConfig(awsCfg),
		EC2Client:        ec2.NewFromConfig(awsCfg),
		CloudWatchClient: cloudwatch.NewFromConfig(awsCfg),
		IAMClient:        iam.NewFromConfig(awsCfg),
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

// ThrottlingMetrics represents AWS API throttling metrics
type ThrottlingMetrics struct {
	Service        string
	Operation      string
	Count          int64
	Period         string
	ThrottledCalls float64
	ErrorRate      float64
}

// PerformanceMetrics holds the performance metrics for EKS
type PerformanceMetrics struct {
	CPUUtilization    float64
	MemoryUtilization float64
	NetworkIn         float64
	NetworkOut        float64
	Timestamp         time.Time
}

// GetEKSThrottlingMetrics retrieves throttling metrics for EKS API calls
func (c *Client) GetEKSThrottlingMetrics(ctx context.Context, startTime, endTime time.Time) ([]ThrottlingMetrics, error) {
	input := &cloudwatch.GetMetricDataInput{
		MetricDataQueries: []cloudwatchtypes.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &cloudwatchtypes.MetricStat{
					Metric: &cloudwatchtypes.Metric{
						Namespace:  aws.String("AWS/EKS"),
						MetricName: aws.String("ThrottledRequestCount"),
						Dimensions: []cloudwatchtypes.Dimension{
							{
								Name:  aws.String("Service"),
								Value: aws.String("eks"),
							},
						},
					},
					Period: aws.Int32(300),
					Stat:   aws.String("Sum"),
				},
			},
		},
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
	}

	output, err := c.CloudWatchClient.GetMetricData(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get throttling metrics: %w", err)
	}

	var metrics []ThrottlingMetrics
	if len(output.MetricDataResults) > 0 && len(output.MetricDataResults[0].Values) > 0 {
		throttledCalls := output.MetricDataResults[0].Values[0]
		metrics = append(metrics, ThrottlingMetrics{
			Service:        "eks",
			Operation:      "all",
			Count:         int64(throttledCalls),
			Period:        "5m",
			ThrottledCalls: throttledCalls,
			ErrorRate:      (throttledCalls / 100.0) * 100.0, // Convert to percentage
		})
	}

	return metrics, nil
}

// GetEKSPerformanceMetrics retrieves cluster performance metrics from CloudWatch
func (c *Client) GetEKSPerformanceMetrics(ctx context.Context, clusterName string) ([]PerformanceMetrics, error) {
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	input := &cloudwatch.GetMetricDataInput{
		MetricDataQueries: []cloudwatchtypes.MetricDataQuery{
			{
				Id: aws.String("cpu"),
				MetricStat: &cloudwatchtypes.MetricStat{
					Metric: &cloudwatchtypes.Metric{
						Namespace:  aws.String("AWS/EKS"),
						MetricName: aws.String("cluster_cpu_utilization"),
						Dimensions: []cloudwatchtypes.Dimension{
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
				MetricStat: &cloudwatchtypes.MetricStat{
					Metric: &cloudwatchtypes.Metric{
						Namespace:  aws.String("AWS/EKS"),
						MetricName: aws.String("cluster_memory_utilization"),
						Dimensions: []cloudwatchtypes.Dimension{
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

	output, err := c.CloudWatchClient.GetMetricData(ctx, input)
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

// VPCInfo contains information about VPC configuration
type VPCInfo struct {
	VPCID          string
	SubnetID       string
	SecurityGroups []string
}

func (c *Client) GetVPCInfo(ctx context.Context, nodeID string) (*VPCInfo, error) {
	// Implementation to get VPC information
	return &VPCInfo{
		VPCID:          "vpc-example",
		SubnetID:       "subnet-example",
		SecurityGroups: []string{"sg-example"},
	}, nil
}

func (c *Client) GetControlPlaneMetrics(ctx context.Context, clusterName string) (*ControlPlaneMetrics, error) {
	// Implementation to get control plane metrics
	return nil, nil
}

// ControlPlaneMetrics contains metrics for the control plane
type ControlPlaneMetrics struct {
	APIServerLatencyP99 string
	EtcdLatencyP99     string
	RequestThroughput  float64
}

// GetClusterNodegroups gets detailed information about all nodegroups in a cluster
func (c *Client) GetClusterNodegroups(ctx context.Context, clusterName string) ([]*ekstypes.Nodegroup, error) {
	input := &eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterName),
	}

	result, err := c.EKSClient.ListNodegroups(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodegroups: %w", err)
	}

	var nodegroups []*ekstypes.Nodegroup
	for _, ng := range result.Nodegroups {
		descInput := &eks.DescribeNodegroupInput{
			ClusterName:   aws.String(clusterName),
			NodegroupName: aws.String(ng),
		}

		desc, err := c.EKSClient.DescribeNodegroup(ctx, descInput)
		if err != nil {
			return nil, fmt.Errorf("failed to describe nodegroup %s: %w", ng, err)
		}
		nodegroups = append(nodegroups, desc.Nodegroup)
	}

	return nodegroups, nil
}

// GetClusterPerformanceMetrics retrieves performance metrics for the cluster from CloudWatch
func (c *Client) GetClusterPerformanceMetrics(ctx context.Context, clusterName string) (map[string]float64, error) {
	// Define the metrics to collect
	metrics := []struct {
		Name      string
		Namespace string
		Stat      string
	}{
		{"pod_cpu_utilization", "ContainerInsights", "Average"},
		{"pod_memory_utilization", "ContainerInsights", "Average"},
		{"node_cpu_utilization", "ContainerInsights", "Average"},
		{"node_memory_utilization", "ContainerInsights", "Average"},
	}

	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	result := make(map[string]float64)

	for _, m := range metrics {
		input := &cloudwatch.GetMetricDataInput{
			MetricDataQueries: []cloudwatchtypes.MetricDataQuery{
				{
					Id: aws.String("m1"),
					MetricStat: &cloudwatchtypes.MetricStat{
						Metric: &cloudwatchtypes.Metric{
							Namespace:  aws.String(m.Namespace),
							MetricName: aws.String(m.Name),
							Dimensions: []cloudwatchtypes.Dimension{
								{
									Name:  aws.String("ClusterName"),
									Value: aws.String(clusterName),
								},
							},
						},
						Period: aws.Int32(300),
						Stat:   aws.String(m.Stat),
					},
				},
			},
			StartTime: aws.Time(startTime),
			EndTime:   aws.Time(endTime),
		}

		output, err := c.CloudWatchClient.GetMetricData(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get metric %s: %w", m.Name, err)
		}

		if len(output.MetricDataResults) > 0 && len(output.MetricDataResults[0].Values) > 0 {
			result[m.Name] = output.MetricDataResults[0].Values[0]
		}
	}

	return result, nil
}

// GetAddons gets detailed information about all addons in a cluster
func (c *Client) GetAddons(ctx context.Context, clusterName string) ([]*ekstypes.Addon, error) {
	input := &eks.ListAddonsInput{
		ClusterName: aws.String(clusterName),
	}

	result, err := c.EKSClient.ListAddons(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list addons: %w", err)
	}

	var addons []*ekstypes.Addon
	for _, a := range result.Addons {
		descInput := &eks.DescribeAddonInput{
			ClusterName: aws.String(clusterName),
			AddonName:   aws.String(a),
		}

		desc, err := c.EKSClient.DescribeAddon(ctx, descInput)
		if err != nil {
			return nil, fmt.Errorf("failed to describe addon %s: %w", a, err)
		}
		addons = append(addons, desc.Addon)
	}

	return addons, nil
}

// ListAddons lists all addons in a cluster
func (c *Client) ListAddons(ctx context.Context, clusterName string) ([]string, error) {
    input := &eks.ListAddonsInput{
        ClusterName: aws.String(clusterName),
    }

    result, err := c.EKSClient.ListAddons(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to list addons: %w", err)
    }

    return result.Addons, nil
}

// DescribeAddon gets detailed information about an addon
func (c *Client) DescribeAddon(ctx context.Context, clusterName, addonName string) (*eks.DescribeAddonOutput, error) {
    input := &eks.DescribeAddonInput{
        AddonName:   aws.String(addonName),
        ClusterName: aws.String(clusterName),
    }

    result, err := c.EKSClient.DescribeAddon(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to describe addon %s: %w", addonName, err)
    }

    return result, nil
}

// GetNATGateways gets NAT gateways in a VPC
func (c *Client) GetNATGateways(ctx context.Context, vpcID string) ([]*NATGatewayInfo, error) {
    input := &ec2.DescribeNatGatewaysInput{
        Filter: []ec2types.Filter{
            {
                Name:   aws.String("vpc-id"),
                Values: []string{vpcID},
            },
        },
    }

    result, err := c.EC2Client.DescribeNatGateways(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to describe NAT gateways: %w", err)
    }

    var gateways []*NATGatewayInfo
    for _, ng := range result.NatGateways {
        gateways = append(gateways, &NATGatewayInfo{
            NatGatewayId: ng.NatGatewayId,
            State:        string(ng.State),
            SubnetId:     ng.SubnetId,
        })
    }

    return gateways, nil
}

// GetSecurityGroupEgressRules gets egress rules for a security group
func (c *Client) GetSecurityGroupEgressRules(ctx context.Context, securityGroupID string) ([]ec2types.SecurityGroupRule, error) {
    input := &ec2.DescribeSecurityGroupRulesInput{
        Filters: []ec2types.Filter{
            {
                Name:   aws.String("group-id"),
                Values: []string{securityGroupID},
            },
            {
                Name:   aws.String("is-egress"),
                Values: []string{"true"},
            },
        },
    }

    result, err := c.EC2Client.DescribeSecurityGroupRules(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to describe security group rules: %w", err)
    }

    return result.SecurityGroupRules, nil
}

// GetRouteTables gets route tables in a VPC
func (c *Client) GetRouteTables(ctx context.Context, vpcID string) ([]ec2types.RouteTable, error) {
    input := &ec2.DescribeRouteTablesInput{
        Filters: []ec2types.Filter{
            {
                Name:   aws.String("vpc-id"),
                Values: []string{vpcID},
            },
        },
    }

    result, err := c.EC2Client.DescribeRouteTables(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to describe route tables: %w", err)
    }

    return result.RouteTables, nil
}

// ValidateSecurityGroupAccess checks if a security group can be accessed from another account
func (c *Client) ValidateSecurityGroupAccess(ctx context.Context, securityGroupID string) error {
    input := &ec2.DescribeSecurityGroupsInput{
        GroupIds: []string{securityGroupID},
    }

    result, err := c.EC2Client.DescribeSecurityGroups(ctx, input)
    if err != nil {
        return fmt.Errorf("failed to describe security group: %w", err)
    }

    if len(result.SecurityGroups) == 0 {
        return fmt.Errorf("security group %s not found", securityGroupID)
    }

    sg := result.SecurityGroups[0]

    // Check for cross-account references in ingress rules
    for _, rule := range sg.IpPermissions {
        for _, group := range rule.UserIdGroupPairs {
            if group.UserId != nil && *group.UserId != *sg.OwnerId {
                // Found a cross-account reference, validate if the account has permission
                iamInput := &iam.GetRoleInput{
                    RoleName: aws.String(extractRoleNameFromARN(*group.UserId)),
                }
                _, err := c.IAMClient.GetRole(ctx, iamInput)
                if err != nil {
                    return fmt.Errorf("cross-account access issue: %w", err)
                }
            }
        }
    }

    return nil
}

// GetSecurityAnalysis analyzes security settings of the cluster and nodegroups
func (c *Client) GetSecurityAnalysis(ctx context.Context, clusterName string) (map[string]string, error) {
	findings := make(map[string]string)

	// Get cluster details
	cluster, err := c.DescribeCluster(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}

	// Check cluster encryption
	if cluster.Cluster.EncryptionConfig == nil {
		findings["cluster_encryption"] = "WARNING: Cluster encryption is not enabled"
	} else {
		findings["cluster_encryption"] = "OK: Cluster encryption is enabled"
	}

	// Check endpoint access
	if cluster.Cluster.ResourcesVpcConfig.EndpointPublicAccess {
		findings["endpoint_access"] = "WARNING: Public endpoint access is enabled"
	} else {
		findings["endpoint_access"] = "OK: Public endpoint access is disabled"
	}

	// Check logging
	if cluster.Cluster.Logging != nil && len(cluster.Cluster.Logging.ClusterLogging) > 0 {
		findings["logging"] = "OK: Cluster logging is configured"
	} else {
		findings["logging"] = "WARNING: Cluster logging is not configured"
	}

	// Check nodegroups
	nodegroups, err := c.GetClusterNodegroups(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodegroups: %w", err)
	}

	for _, ng := range nodegroups {
		ngName := *ng.NodegroupName

		// Check remote access
		if ng.RemoteAccess != nil && len(ng.RemoteAccess.SourceSecurityGroups) == 0 {
			findings[fmt.Sprintf("nodegroup_%s_remote_access", ngName)] = 
				"WARNING: Nodegroup remote access is not restricted by security groups"
		}

		// Check IAM roles
		if ng.NodeRole != nil {
			findings[fmt.Sprintf("nodegroup_%s_iam", ngName)] = "OK: Nodegroup has IAM role configured"
		} else {
			findings[fmt.Sprintf("nodegroup_%s_iam", ngName)] = "WARNING: Nodegroup IAM role not found"
		}
	}

	return findings, nil
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

// ListNodegroups lists all nodegroups in a cluster
func (c *Client) ListNodegroups(ctx context.Context, clusterName string) ([]string, error) {
    input := &eks.ListNodegroupsInput{
        ClusterName: aws.String(clusterName),
    }

    result, err := c.EKSClient.ListNodegroups(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to list nodegroups: %w", err)
    }

    return result.Nodegroups, nil
}

// DescribeNodegroup gets detailed information about a nodegroup
func (c *Client) DescribeNodegroup(ctx context.Context, clusterName, nodegroupName string) (*eks.DescribeNodegroupOutput, error) {
    input := &eks.DescribeNodegroupInput{
        ClusterName:   aws.String(clusterName),
        NodegroupName: aws.String(nodegroupName),
    }

    result, err := c.EKSClient.DescribeNodegroup(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to describe nodegroup %s: %w", nodegroupName, err)
    }

    return result, nil
}
