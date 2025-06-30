# ekspeek

A powerful CLI tool for debugging AWS EKS clusters. It provides comprehensive debugging capabilities for EKS clusters, nodegroups, EFS CSI drivers, PVCs, pods, and more. The tool simplifies cluster management and troubleshooting by providing an intuitive command-line interface.

## Data Collection Methods

### Kubernetes API Integration
The tool interacts with Kubernetes APIs to gather data:
- Uses the official Kubernetes Go client library
- Automatically updates kubeconfig for cluster access
- Handles API pagination for large result sets
- Respects RBAC permissions and namespace boundaries

### AWS Service Integration
Data is collected from various AWS services:
- EKS API for cluster and nodegroup information
- CloudWatch for metrics and performance data
- IAM for role and permission verification
- EFS API for storage system details

### Real-time Data Collection
For debug commands:
1. EFS CSI Driver Status:
   - Lists pods with EFS CSI driver labels
   - Checks pod status and health metrics
   - Monitors node assignments
   - Verifies mount points and volumes

2. PVC Analysis:
   - Queries PersistentVolumeClaim objects
   - Checks associated PersistentVolumes
   - Validates StorageClass configurations
   - Monitors capacity and usage metrics

3. Pod Diagnostics:
   - Lists pods with detailed states
   - Collects container status information
   - Retrieves pod events and conditions
   - Streams container logs when requested
   - Gathers resource usage metrics

4. Resource Monitoring:
   - Collects node-level metrics
   - Aggregates pod resource requests
   - Tracks quota usage and limits
   - Monitors cluster capacity

### Performance Data
Performance metrics are collected from:
- Kubernetes Metrics Server
- CloudWatch Container Insights
- Node-level resource statistics
- API server metrics

### Security Information
Security data is gathered by:
- Analyzing RBAC configurations
- Validating service account settings
- Checking network policies
- Scanning security contexts
- Validating TLS certificates

## Command Reference

### Global Flags
All commands support the following global flags:
- `--profile string`: AWS profile to use for authentication
- `--region string`: AWS region to use for operations
- `--debug`: Enable debug logging for verbose output

### Cluster Management Commands

#### `ekspeek list`
Lists all EKS clusters in the specified region.
- Usage: `ekspeek list`
- Output: Displays cluster names in the current region
- Example: `ekspeek list --region us-west-2`

#### `ekspeek describe [cluster-name]`
Shows detailed information about a specific EKS cluster.
- Usage: `ekspeek describe <cluster-name>`
- Output:
  - Cluster name
  - Kubernetes version
  - Status
  - API server endpoint
  - ARN
  - Creation timestamp
- Example: `ekspeek describe my-cluster`

#### `ekspeek list-nodegroups [cluster-name]`
Lists all nodegroups in a specified EKS cluster.
- Usage: `ekspeek list-nodegroups <cluster-name>`
- Output: Displays all nodegroup names in the cluster
- Example: `ekspeek list-nodegroups my-cluster`

#### `ekspeek describe-nodegroup [cluster-name] [nodegroup-name]`
Shows detailed information about a specific nodegroup.
- Usage: `ekspeek describe-nodegroup <cluster-name> <nodegroup-name>`
- Output: Detailed nodegroup configuration and status
- Example: `ekspeek describe-nodegroup my-cluster ng-1`

### Debug Commands

#### `ekspeek debug efs [cluster-name]`
Debug EFS CSI driver status and configuration.
- Usage: `ekspeek debug efs <cluster-name>`
- Checks:
  - EFS CSI driver pod status
  - Pod health status
  - Node assignment
- Example: `ekspeek debug efs my-cluster`

#### `ekspeek debug pvc [cluster-name]`
Analyze PVC status and configuration.
- Usage: `ekspeek debug pvc <cluster-name> [-n namespace]`
- Flags:
  - `-n, --namespace string`: Filter PVCs by namespace
- Output:
  - PVC name
  - Namespace
  - Status
  - Volume name
  - Storage class
  - Capacity
- Example: `ekspeek debug pvc my-cluster -n default`

#### `ekspeek debug pods [cluster-name]`
Debug pod status and investigate issues.
- Usage: `ekspeek debug pods <cluster-name>`
- Flags:
  - `--namespace, -n string`: Filter pods by namespace
  - `--logs`: Show logs for failed pods
- Checks:
  - Pod running status
  - Failed pods
  - Container states
  - Pod logs (with --logs flag)
- Example: `ekspeek debug pods my-cluster --logs`

#### `ekspeek debug resources [cluster-name]`
Analyze cluster resource utilization.
- Usage: `ekspeek debug resources <cluster-name>`
- Checks:
  - Node resource usage
  - Pod resource requests/limits
  - Resource quotas
  - Resource constraints
- Example: `ekspeek debug resources my-cluster`

#### `ekspeek debug irsa [cluster-name]`
Debug IAM Roles for Service Accounts (IRSA) configuration.
- Usage: `ekspeek debug irsa <cluster-name>`
- Checks:
  - IRSA configuration
  - IAM role validity
  - Service account configuration
  - Token mounting status
- Example: `ekspeek debug irsa my-cluster`

#### `ekspeek debug autoscaler [cluster-name]`
Debug cluster autoscaler configuration and behavior.
- Usage: `ekspeek debug autoscaler <cluster-name>`
- Checks:
  - Autoscaler status
  - Scaling events
  - Node group configuration
  - Scaling constraints
- Example: `ekspeek debug autoscaler my-cluster`

#### `ekspeek debug throttling [cluster-name]`
Monitor and debug API throttling issues.
- Usage: `ekspeek debug throttling <cluster-name>`
- Checks:
  - API request rates
  - Throttling events
  - Service quotas
  - API latency
- Example: `ekspeek debug throttling my-cluster`

#### `ekspeek debug network [cluster-name] [pod-name]`
Debug networking configuration and connectivity.
- Usage: `ekspeek debug network <cluster-name> <pod-name> [-n namespace]`
- Checks:
  - Pod network configuration
  - VPC and subnet details
  - Security groups
  - Network policies
  - DNS resolution
  - Pod connectivity tests
- Example: 
```bash
$ ekspeek debug network my-cluster web-app-pod -n default

Getting pod networking details...

Pod Network Configuration:
Pod IP: 10.0.15.123
Host IP: 192.168.1.100
Node: ip-192-168-1-100.ec2.internal

Checking network policies...
⚠️ No NetworkPolicies found in namespace default

Testing DNS resolution...
✅ DNS resolution test passed

Getting VPC networking details...
VPC Configuration:
VPC ID: vpc-0abc123def456789
Subnet ID: subnet-0123456789abcdef0
Security Groups: [sg-0987654321fedcba]

Testing pod connectivity...
✅ Pod connectivity test passed

Recommendations:
1. Consider implementing NetworkPolicies to secure pod communication
3. Pod is using host network - review if this is intended
```
The command performs comprehensive network diagnostics and provides actionable recommendations.
  - Load balancer configuration
- Example: `ekspeek debug network my-cluster`

#### `ekspeek debug egress [cluster-name]`
Debug egress network traffic and policies.
- Usage: `ekspeek debug egress <cluster-name>`
- Checks:
  - Egress rules
  - NAT gateway configuration
  - Security group rules
  - Network policies
- Example: `ekspeek debug egress my-cluster`

#### `ekspeek debug cross-account [cluster-name]`
Debug cross-account access and permissions.
- Usage: `ekspeek debug cross-account <cluster-name>`
- Checks:
  - IAM role trust relationships
  - Cross-account permissions
  - Resource access policies
- Example: `ekspeek debug cross-account my-cluster`

#### `ekspeek debug tls [cluster-name]`
Debug TLS certificates and configuration.
- Usage: `ekspeek debug tls <cluster-name> [-n namespace]`
- Checks:
  - API server certificate
  - Ingress TLS certificates
  - Service certificates
  - Certificate chains
  - Expiration dates
- Example:
```bash
$ ekspeek debug tls my-cluster

Checking API server certificate...

API Server Certificate:
Subject: CN=kube-apiserver
Issuer: CN=kubernetes
Valid Until: 2025-07-30 10:15:30 UTC
✅ API server certificate is valid for 89 more days

Checking Ingress TLS certificates...
Found 2 Ingress TLS certificates:

Host: api.example.com
Subject: CN=api.example.com
Issuer: CN=Let's Encrypt Authority X3
Valid Until: 2025-08-15 12:00:00 UTC
✅ Certificate is valid for 45 more days

Host: app.example.com
Subject: CN=app.example.com
Issuer: CN=Let's Encrypt Authority X3
Valid Until: 2025-07-15 12:00:00 UTC
❌ Certificate expires in 15 days

Checking service certificates...
Found 1 service TLS certificate:

Service: payment-service
Subject: CN=payment-service.default.svc
Issuer: CN=service-ca
Valid Until: 2025-09-30 00:00:00 UTC
✅ Certificate is valid for 92 more days

Validating certificate chains...
✅ All certificate chains are valid

Recommendations:
1. Plan to rotate API server certificate within 89 days
2. Renew certificate for app.example.com (expires in 15 days)
```
The command performs comprehensive certificate analysis and provides clear expiration warnings.

#### `ekspeek debug performance [cluster-name]`
Analyzes cluster performance metrics.
- API server latency
- etcd performance
- Node resource utilization
- Pod resource usage
- Control plane metrics

#### `ekspeek debug security [cluster-name]`
Performs security checks including:
- RBAC configurations
- Service account settings
- Network policies
- Pod security contexts
- Cluster role bindings

#### `ekspeek debug efs [cluster-name]`
Diagnoses EFS CSI driver issues:
- Driver pod status
- Mount point verification
- Volume status checks
- Storage provisioning

#### `ekspeek debug pvc [cluster-name]`
Analyzes Persistent Volume Claims:
- PVC status and events
- Storage class configuration
- Volume binding status
- Capacity and usage

#### `ekspeek debug pods [cluster-name]`
Investigates pod-related issues:
- Failed pod analysis
- Container status
- Resource constraints
- Pod events and logs

#### `ekspeek debug resources [cluster-name]`
Shows cluster resource utilization:
- Node capacity
- Pod resource requests
- Quota usage
- Available resources

#### `ekspeek debug irsa [cluster-name]`
Debugs IAM Roles for Service Accounts:
- IAM role validation
- Service account configuration
- Token mounting verification
- Permission checks

#### `ekspeek debug autoscaler [cluster-name]`
Diagnoses Cluster Autoscaler issues:
- Scaling events
- Node group configuration
- Scaling decisions
- Resource requirements

#### `ekspeek debug throttling [cluster-name]`
Analyzes API throttling:
- API call patterns
- Rate limiting issues
- Service quotas
- Throttling metrics

#### `ekspeek debug network [cluster-name] [pod-name]`
Troubleshoots networking issues:
- Pod connectivity
- DNS resolution
- Network policies
- MTU configuration

#### `ekspeek debug egress [cluster-name]`
Analyzes pod egress traffic:
- Security group rules
- NAT gateway configuration
- Routing tables
- Outbound connectivity

#### `ekspeek debug cross-account [cluster-name]`
Debugs cross-account access issues:
- IAM role trust relationships
- VPC access
- Security group permissions
- Cross-account resources

#### `ekspeek debug tls [cluster-name]`
Validates TLS configurations:
- Certificate validation
- Expiration checks
- Chain verification
- TLS endpoints

#### `ekspeek debug karpenter [cluster-name]`
Troubleshoots Karpenter autoscaling:
- Provisioner configuration
- Node pool status
- Scaling decisions
- Pending pods

## Features

### Comprehensive Cluster Management
- List and describe EKS clusters and nodegroups
- Monitor cluster health and performance
- Track resource utilization and constraints
- Multi-cluster configuration support

### Advanced Debugging Capabilities
- Pod lifecycle and container diagnostics
- Storage system validation (EFS, PVC)
- Network connectivity and policy verification
- Performance bottleneck detection
- Security compliance scanning

### Autoscaling Management
- Cluster Autoscaler diagnostics
- Karpenter configuration validation
- Scaling event analysis
- Node pool optimization

### Security & Identity
- IRSA validation and troubleshooting
- Cross-account access verification
- TLS certificate management
- RBAC configuration analysis

### Real-time Monitoring
- Resource utilization tracking
- API throttling detection
- Performance metrics collection
- Log streaming and analysis

## Installation

### Prerequisites
- Go 1.21 or higher
- AWS CLI v2
- kubectl 1.24+
- Proper AWS IAM permissions

### Install from Source
```bash
# Clone the repository
git clone https://github.com/yourusername/ekspeek.git
cd ekspeek

# Build and install
go install ./cmd/ekspeek
```

### Install Binary
```bash
go install github.com/yourusername/ekspeek/cmd/ekspeek@latest
```

### Manual Installation

1. Build the binary:
```bash
# Clone the repository
git clone https://github.com/yourusername/ekspeek.git
cd ekspeek

# Build the binary
go build -o bin/ekspeek ./cmd/ekspeek
```

2. Add to system PATH:

For Zsh (macOS default):
```bash
# Create bin directory if it doesn't exist
mkdir -p ~/bin

# Copy binary to bin directory
cp bin/ekspeek ~/bin/

# Add to PATH in .zshrc
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc

# Reload shell configuration
source ~/.zshrc
```

For Bash:
```bash
# Create bin directory if it doesn't exist
mkdir -p ~/bin

# Copy binary to bin directory
cp bin/ekspeek ~/bin/

# Add to PATH in .bash_profile (macOS) or .bashrc (Linux)
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bash_profile  # or ~/.bashrc

# Reload shell configuration
source ~/.bash_profile  # or source ~/.bashrc
```

3. Verify installation:
```bash
# Check if ekspeek is accessible
which ekspeek

# Verify version
ekspeek version
```

### System-wide Installation
To install the binary system-wide (requires sudo access):

```bash
# Copy to /usr/local/bin (recommended for macOS)
sudo cp bin/ekspeek /usr/local/bin/

# Or copy to /usr/bin (alternative location)
sudo cp bin/ekspeek /usr/bin/
```

## Usage

### Available Commands

```bash
# Global Flags (available for all commands)
  --profile string    # AWS profile to use
  --region string     # AWS region to use
  --debug            # Enable debug logging

# Cluster Management
ekspeek eks list                           # List all EKS clusters
ekspeek eks describe <cluster-name>        # Show detailed cluster information
ekspeek eks list-nodegroups <cluster-name> # List all nodegroups in cluster
ekspeek eks describe-nodegroup <cluster-name> <nodegroup-name>  # Show detailed nodegroup information

# Debug Commands
ekspeek debug efs <cluster-name>           # Debug EFS CSI driver status
ekspeek debug pvc <cluster-name>           # Debug PVC status
ekspeek debug pods <cluster-name>          # Debug pod status and show failed pods
ekspeek debug resources <cluster-name>     # Show cluster resource usage
ekspeek debug health <cluster-name>        # Run cluster health checks
ekspeek debug irsa <cluster-name>          # Debug IRSA configuration
ekspeek debug autoscaler <cluster-name>    # Debug cluster autoscaler
ekspeek debug throttling <cluster-name>    # Check API throttling
ekspeek debug network <cluster-name>       # Debug networking issues
ekspeek debug egress <cluster-name>        # Debug egress traffic
ekspeek debug cross-account <cluster-name> # Debug cross-account access
ekspeek debug tls <cluster-name>          # Debug TLS/certificate issues
ekspeek debug performance <cluster-name>   # Check cluster performance
ekspeek debug security <cluster-name>      # Run security checks

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

## Required Permissions

### AWS IAM Permissions

The tool requires an AWS IAM user or role with the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "eks:ListClusters",
                "eks:DescribeCluster",
                "eks:ListNodegroups",
                "eks:DescribeNodegroup",
                "cloudwatch:GetMetricData",
                "cloudwatch:ListMetrics",
                "cloudwatch:GetMetricStatistics",
                "iam:GetRole",
                "iam:GetRolePolicy",
                "iam:ListAttachedRolePolicies",
                "elasticfilesystem:DescribeFileSystems",
                "elasticfilesystem:DescribeMountTargets",
                "elasticfilesystem:DescribeMountTargetSecurityGroups"
            ],
            "Resource": "*"
        }
    ]
}
```

For cross-account functionality, additional trust relationships may be required.

### Kubernetes RBAC

For cluster debugging, create a ServiceAccount with these permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ekspeek-debug
rules:
- apiGroups: [""]
  resources: ["pods", "services", "nodes", "persistentvolumes", "persistentvolumeclaims"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "daemonsets", "statefulsets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["storage.k8s.io"]
  resources: ["storageclasses"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["nodes", "pods"]
  verbs: ["get", "list"]
```

Apply the role binding:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ekspeek-debug
subjects:
- kind: ServiceAccount
  name: ekspeek
  namespace: kube-system
roleRef:
  kind: ClusterRole
  name: ekspeek-debug
  apiGroup: rbac.authorization.k8s.io
```

## Common Error Messages and Solutions

### AWS Errors

1. "AccessDeniedException: User is not authorized to perform eks:ListClusters"
   - Solution: Verify AWS credentials and IAM permissions
   - Check: `aws sts get-caller-identity`

2. "ExpiredToken: The security token included in the request is expired"
   - Solution: Update AWS credentials
   - Check: `aws configure`

### Kubernetes Errors

1. "Unable to connect to the server: dial tcp: lookup <cluster>: no such host"
   - Solution: Update kubeconfig for the cluster
   - Run: `aws eks update-kubeconfig --name <cluster-name>`

2. "Error from server (Forbidden): pods is forbidden"
   - Solution: Verify RBAC permissions
   - Check: `kubectl auth can-i list pods`

### EFS Errors

1. "Failed to mount volume: mount failed"
   - Check EFS mount target security groups
   - Verify subnet connectivity
   - Validate IAM roles for EFS access

## Usage Examples

### Basic Cluster Operations
```bash
# List all clusters in a region
ekspeek list --region us-west-2

# Get detailed information about a cluster
ekspeek describe my-production-cluster

# List all nodegroups
ekspeek list-nodegroups my-production-cluster
```

### Debugging Storage Issues
```bash
# Check EFS CSI driver status
ekspeek debug efs my-cluster

# Investigate PVC issues in specific namespace
ekspeek debug pvc my-cluster -n application-ns

# Check all storage classes
ekspeek debug resources my-cluster
```

### Pod and Container Diagnostics
```bash
# Show failed pods with logs
ekspeek debug pods my-cluster --logs

# Check resource utilization
ekspeek debug resources my-cluster

# Debug networking for specific pods
ekspeek debug network my-cluster
```

### Security and Compliance Checks
```bash
# Verify IRSA configuration
ekspeek debug irsa my-cluster

# Run security compliance checks
ekspeek debug security my-cluster

# Check cross-account access
ekspeek debug cross-account my-cluster
```

### Autoscaling Management
```bash
# Debug cluster autoscaler
ekspeek debug autoscaler my-cluster

# Check Karpenter configuration
ekspeek debug karpenter my-cluster

# Monitor scaling events
ekspeek debug performance my-cluster
```

### Performance Analysis
```bash
# Check API throttling
ekspeek debug throttling my-cluster

# Monitor cluster performance
ekspeek debug performance my-cluster

# Analyze resource usage
ekspeek debug resources my-cluster
```

## Command Safety

All `ekspeek` commands are designed to be read-only operations, ensuring safe execution without any risk of modifying cluster state or resources. Here's how our commands operate:

### Read-Only Operations

All commands are limited to:
- Reading configurations and status
- Retrieving metrics and logs
- Performing validation checks
- Generating analysis reports
- Providing recommendations

### Command Categories

1. **List and Describe Commands**
   - `list clusters` - Reads cluster information
   - `describe cluster` - Reads cluster details
   - `list-nodegroups` - Reads nodegroup information
   - `describe-nodegroup` - Reads nodegroup details

2. **Debug Commands**
   - `debug pods` - Reads pod status and logs
   - `debug resources` - Reads cluster resource usage
   - `debug efs` - Reads EFS CSI driver status
   - `debug pvc` - Reads PVC status
   - `debug irsa` - Validates IRSA configuration
   - `debug autoscaler` - Reads autoscaler metrics and events
   - `debug throttling` - Reads API throttling metrics
   - `debug networking` - Reads network configuration
   - `debug tls` - Validates certificates
   - `debug performance` - Reads performance metrics
   - `debug security` - Performs security audits
   - `debug karpenter` - Reads Karpenter status

3. **Health Commands**
   - `health` - Reads cluster health metrics

### Safety Guarantees

- No create/update/delete operations on Kubernetes resources
- No AWS API calls that modify resources
- No write operations to cluster or infrastructure
- All commands focus purely on diagnostics and analysis
- Safe to use in production environments

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.
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
