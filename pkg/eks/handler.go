package eks

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

// Handler handles EKS-related operations
type Handler struct {
	client *eks.Client
}

// NewHandler creates a new EKS handler
func NewHandler(client *eks.Client) *Handler {
	return &Handler{client: client}
}

// ListClusters returns a list of all EKS clusters in the region
func (h *Handler) ListClusters(ctx context.Context) ([]string, error) {
	input := &eks.ListClustersInput{}
	result, err := h.client.ListClusters(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	return result.Clusters, nil
}

// DescribeCluster returns detailed information about a specific cluster
func (h *Handler) DescribeCluster(ctx context.Context, clusterName string) (*types.Cluster, error) {
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}
	result, err := h.client.DescribeCluster(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster %s: %w", clusterName, err)
	}
	return result.Cluster, nil
}

// ListNodegroups returns a list of all nodegroups in a cluster
func (h *Handler) ListNodegroups(ctx context.Context, clusterName string) ([]string, error) {
	input := &eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterName),
	}
	result, err := h.client.ListNodegroups(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodegroups for cluster %s: %w", clusterName, err)
	}
	return result.Nodegroups, nil
}

// DescribeNodegroup returns detailed information about a specific nodegroup
func (h *Handler) DescribeNodegroup(ctx context.Context, clusterName, nodegroupName string) (*types.Nodegroup, error) {
	input := &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodegroupName),
	}
	result, err := h.client.DescribeNodegroup(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe nodegroup %s in cluster %s: %w", nodegroupName, clusterName, err)
	}
	return result.Nodegroup, nil
}

// GetNodegroupScaling returns the scaling configuration for a nodegroup
func (h *Handler) GetNodegroupScaling(ctx context.Context, clusterName, nodegroupName string) (*types.NodegroupScalingConfig, error) {
	nodegroup, err := h.DescribeNodegroup(ctx, clusterName, nodegroupName)
	if err != nil {
		return nil, err
	}
	return nodegroup.ScalingConfig, nil
}
