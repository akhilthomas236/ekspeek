# ekspeek

A powerful CLI tool for debugging AWS EKS clusters. It provides comprehensive debugging capabilities for EKS components, Karpenter, IRSA, networking, and more.

## Features

### Cluster Management
- EKS cluster inspection and management
- Node group administration and scaling
- Multi-cluster configuration handling
- Detailed cluster health monitoring

### Resource Management
- Complete resource health checks and monitoring
- Real-time CPU and memory utilization tracking
- Storage management (EFS, PVC, StorageClasses)
- Pod lifecycle and container diagnostics
- Failed pod analysis and log aggregation

### Autoscaling
- Cluster Autoscaler diagnostics and troubleshooting
- Karpenter integration and management
  - Provisioner configuration and status
  - Node pool management and scaling
  - Real-time scaling event analysis
  - Unschedulable pod detection and resolution

### Security and Identity
- IRSA (IAM Roles for Service Accounts) validation
- Cross-account access verification
- WebIdentity token validation
- TLS and certificate management
- Security compliance scanning

### Monitoring and Diagnostics
- CloudWatch metrics integration
  - API throttling analysis
  - Performance metrics collection
  - Resource utilization tracking
- Real-time log streaming and analysis
- Event tracking and correlation
- Network diagnostics and troubleshooting

### Storage and Data Management
- EFS CSI driver status monitoring
- PVC lifecycle management
- StorageClass configuration and validation
- Volume status and health checks

## Installation

```bash
go install github.com/yourusername/ekspeek/cmd/ekspeek@latest
```

## Usage

### Cluster Commands
```bash
# Cluster Management
ekspeek eks list                             # List all EKS clusters
ekspeek eks describe <cluster-name>          # Show detailed cluster information
ekspeek eks list-nodegroups <cluster-name>   # List all nodegroups in cluster
ekspeek eks describe-nodegroup <name>        # Show detailed nodegroup information

# Health and Diagnostics
ekspeek debug efs <cluster-name>             # Debug EFS CSI driver status
ekspeek debug pvc <cluster-name>             # Debug PVC status
ekspeek debug pods <cluster-name>            # Debug pod status and show failed pods
ekspeek debug resources <cluster-name>       # Show cluster resource usage

# IRSA and Identity
ekspeek debug irsa <pod-name>               # Debug IRSA configuration
ekspeek debug cross-account <cluster-name>   # Debug cross-account access

# Autoscaling Management
ekspeek debug autoscaler <cluster-name>      # Debug Cluster Autoscaler
ekspeek debug karpenter <cluster-name>       # Debug Karpenter configuration
ekspeek debug karpenter nodes               # List Karpenter-managed nodes
ekspeek debug karpenter pending             # Show pending pods for Karpenter

# Network and Security
ekspeek debug network <cluster-name>         # Debug network configuration
ekspeek debug egress <pod-name>             # Debug pod egress traffic
ekspeek debug tls <cluster-name>            # Debug TLS and certificates

# Performance and API
ekspeek debug throttling <cluster-name>      # Check API throttling
ekspeek debug performance <cluster-name>     # Check cluster performance
ekspeek debug metrics <cluster-name>         # Show detailed resource metrics

# Security and Compliance
ekspeek debug security <cluster-name>        # Run security compliance checks
ekspeek debug rbac <cluster-name>           # Analyze RBAC configuration
```

### Common Options

```bash
# Namespace filter for relevant commands
--namespace, -n           # Specify namespace (default: all namespaces)

# Additional options
--logs                    # Show logs for failed pods (with pods command)
```

## Requirements

### Software Dependencies
- Go 1.24 or higher
- AWS CLI v2.x
- kubectl 1.24+
- Docker (optional, for container builds)

### AWS Setup
- AWS credentials configured with appropriate permissions:
  - EKS cluster access (eks:*)
  - CloudWatch metrics access (cloudwatch:*)
  - IAM role inspection (iam:*)
  - EFS access (elasticfilesystem:*)
- AWS CLI configured with:
  ```bash
  aws configure
  ```
- EKS cluster access configured:
  ```bash
  aws eks update-kubeconfig --name your-cluster
  ```

### Kubernetes Setup
- kubectl installed and configured
- Metrics Server deployed in cluster
- CloudWatch agent (for enhanced metrics)
- Proper RBAC permissions

### Optional Components
- Cluster Autoscaler (for scaling diagnostics)
- Karpenter (for advanced node management)
- AWS Load Balancer Controller
- EFS CSI Driver
````markdown
## AWS IAM Permissions

The tool requires the following AWS IAM permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "eks:*",
                "cloudwatch:GetMetricData",
                "cloudwatch:ListMetrics",
                "iam:GetRole",
                "iam:GetRolePolicy",
                "elasticfilesystem:DescribeFileSystems",
                "elasticfilesystem:DescribeMountTargets"
            ],
            "Resource": "*"
        }
    ]
}
```

## Kubernetes RBAC

For cluster debugging, the tool needs a ServiceAccount with the following permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["*"]
  verbs: ["get", "list"]
```

## Examples

### Cluster and Node Management
```bash
# List and inspect clusters
ekspeek eks list
ekspeek eks describe my-cluster

# Node group management
ekspeek eks list-nodegroups my-cluster
ekspeek eks describe-nodegroup my-cluster my-nodegroup
```

### Resource Monitoring
```bash
# Check cluster-wide resource usage
ekspeek debug resources my-cluster
ekspeek debug metrics my-cluster --namespace kube-system

# Storage management
ekspeek debug efs my-cluster
ekspeek debug pvc my-cluster --namespace my-namespace
```

### Autoscaling Management
```bash
# Cluster Autoscaler diagnostics
ekspeek debug autoscaler my-cluster

# Karpenter management
ekspeek debug karpenter my-cluster
ekspeek debug karpenter nodes               # List managed nodes
ekspeek debug karpenter pending            # Show pending workloads
```

### Security and Identity
```bash
# IRSA validation
ekspeek debug irsa my-pod
ekspeek debug irsa --namespace my-namespace --all-pods

# Security analysis
ekspeek debug security my-cluster --compliance pci-dss
ekspeek debug rbac my-cluster --show-violations
```

### Performance Analysis
```bash
# API and metrics analysis
ekspeek debug throttling my-cluster
ekspeek debug performance my-cluster --detailed
ekspeek debug metrics my-cluster --resource-type node

# Log analysis
ekspeek debug pods my-cluster --show-logs --tail 100
ekspeek debug autoscaler my-cluster --events-only
```

Each command provides detailed output with:
- Current state analysis
- Historical trends where applicable
- Actionable recommendations
- Related resource information

## Troubleshooting

### Common Issues

1. **AWS Credentials**
   ```bash
   export AWS_PROFILE=my-profile
   export AWS_REGION=us-west-2
   ```

2. **Kubeconfig**
   ```bash
   aws eks update-kubeconfig --name my-cluster
   ```

3. **Permission Errors**
   - Verify AWS IAM permissions
   - Check Kubernetes RBAC settings
   - Ensure proper role assumption for cross-account access

4. **Metric Collection**
   - Confirm CloudWatch agent installation
   - Verify metrics retention period
   - Check IAM permissions for CloudWatch

### Error Messages

- `Error: cluster name is required`
  - Ensure you provide the cluster name as an argument
  - Example: `ekspeek debug efs my-cluster`

- `Error: service account missing IAM role annotation`
  - Check service account configuration
  - Verify IRSA setup in the cluster

- `Error: unable to get CloudWatch metrics`
  - Verify AWS credentials
  - Check CloudWatch IAM permissions
  - Confirm correct region setting

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/ekspeek
cd ekspeek

# Build the binary
go build -o ekspeek ./cmd/ekspeek

# Run tests
go test ./...
```

### Release Process

The project uses GitHub Actions for automated releases. The process includes:

1. Continuous Integration
   - Automated testing on pull requests
   - Code quality checks
   - Security scanning

2. Automated Releases
   - Triggered on merges to main branch
   - Semantic versioning
   - Multi-platform binary builds
   - Automated changelog generation

3. Binary Distribution
   - Pre-built binaries for multiple platforms:
     - Linux (amd64, arm64)
     - macOS (amd64, arm64)
     - Windows (amd64)
   - Docker images
   - Homebrew formula

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run integration tests
go test -tags=integration ./...
```

### Project Structure

```
ekspeek/
├── cmd/                    # Command line interface
├── internal/              # Private application code
├── pkg/                   # Public packages
│   ├── aws/              # AWS client and operations
│   ├── k8s/              # Kubernetes operations
│   ├── eks/              # EKS specific functionality
│   └── common/           # Shared utilities
├── docs/                 # Documentation
└── examples/             # Usage examples
```

## Contributing

Contributions are welcome! Here are some ways you can contribute:

- Report bugs
- Suggest new features
- Add new debugging capabilities
- Improve documentation
- Submit pull requests

Please ensure your pull requests:
1. Include detailed description of changes
2. Update relevant documentation
3. Add/update tests as needed
4. Follow existing code style

## License

MIT License
