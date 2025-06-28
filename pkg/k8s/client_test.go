package k8s

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

// Mock implementations
type mockEKSAPI interface {
	ListClusters(ctx context.Context, params *eks.ListClustersInput, optFns ...func(*eks.Options)) (*eks.ListClustersOutput, error)
	DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
	ListNodegroups(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error)
	DescribeNodegroup(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error)
}

type mockEKSClient struct {
	mockEKSAPI
	ListClustersFunc       func(ctx context.Context, params *eks.ListClustersInput, optFns ...func(*eks.Options)) (*eks.ListClustersOutput, error)
	DescribeClusterFunc    func(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
	ListNodegroupsFunc     func(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error)
	DescribeNodegroupFunc  func(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error)
}

func (m *mockEKSClient) ListClusters(ctx context.Context, params *eks.ListClustersInput, optFns ...func(*eks.Options)) (*eks.ListClustersOutput, error) {
	return m.ListClustersFunc(ctx, params, optFns...)
}

func (m *mockEKSClient) DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error) {
	return m.DescribeClusterFunc(ctx, params, optFns...)
}

func (m *mockEKSClient) ListNodegroups(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error) {
	return m.ListNodegroupsFunc(ctx, params, optFns...)
}

func (m *mockEKSClient) DescribeNodegroup(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error) {
	return m.DescribeNodegroupFunc(ctx, params, optFns...)
}

// Test cases
func TestListClusters(t *testing.T) {
	testCases := []struct {
		name          string
		mockResponse  *eks.ListClustersOutput
		mockError     error
		expectedLen   int
		expectError   bool
	}{
		{
			name: "Success - Multiple clusters",
			mockResponse: &eks.ListClustersOutput{
				Clusters: []string{"cluster1", "cluster2"},
			},
			mockError:   nil,
			expectedLen: 2,
			expectError: false,
		},
		{
			name: "Success - Empty list",
			mockResponse: &eks.ListClustersOutput{
				Clusters: []string{},
			},
			mockError:   nil,
			expectedLen: 0,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockEKS := &mockEKSClient{
				ListClustersFunc: func(ctx context.Context, params *eks.ListClustersInput, optFns ...func(*eks.Options)) (*eks.ListClustersOutput, error) {
					return tc.mockResponse, tc.mockError
				},
			}

			client := &Client{
				EKSClient: mockEKS,
			}

			clusters, err := client.ListClusters(context.Background())

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(clusters) != tc.expectedLen {
				t.Errorf("Expected %d clusters, got %d", tc.expectedLen, len(clusters))
			}
		})
	}
}

func TestValidateNodeGroupsConfig(t *testing.T) {
	testCases := []struct {
		name          string
		clusterName   string
		nodegroups    []string
		minSize       *int32
		maxSize       *int32
		expectError   bool
	}{
		{
			name:        "Valid configuration",
			clusterName: "test-cluster",
			nodegroups:  []string{"nodegroup1"},
			minSize:     aws.Int32(1),
			maxSize:     aws.Int32(3),
			expectError: false,
		},
		{
			name:        "Invalid min/max size",
			clusterName: "test-cluster",
			nodegroups:  []string{"nodegroup1"},
			minSize:     aws.Int32(5),
			maxSize:     aws.Int32(3),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockEKS := &mockEKSClient{
				ListNodegroupsFunc: func(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error) {
					return &eks.ListNodegroupsOutput{
						Nodegroups: tc.nodegroups,
					}, nil
				},
				DescribeNodegroupFunc: func(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error) {
					return &eks.DescribeNodegroupOutput{
						Nodegroup: &types.Nodegroup{
							ScalingConfig: &types.NodegroupScalingConfig{
								MinSize: tc.minSize,
								MaxSize: tc.maxSize,
							},
						},
					}, nil
				},
			}

			client := &Client{
				EKSClient: mockEKS,
			}

			err := client.ValidateNodeGroupsConfig(context.Background(), tc.clusterName)

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
