package k8s

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
)

// KubeClientConfig holds the configuration for the Kubernetes client
type KubeClientConfig struct {
	KubeConfig string
	Context    string
}

// KubeClient wraps the Kubernetes clientset and config
type KubeClient struct {
	Clientset *kubernetes.Clientset
	Config    *rest.Config
}

// NewKubeClient creates a new Kubernetes client
func NewKubeClient(cfg KubeClientConfig) (*KubeClient, error) {
	configPath := cfg.KubeConfig
	if configPath == "" {
		configPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from flags: %w", err)
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &KubeClient{
		Clientset: clientset,
		Config:    config,
	}, nil
}

// UpdateKubeconfig updates the kubeconfig file with EKS cluster info
func UpdateKubeconfig(ctx context.Context, clusterName, region string) error {
	// Get the AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create EKS client
	svc := eks.NewFromConfig(cfg)

	// Get cluster info
	input := &eks.DescribeClusterInput{
		Name: &clusterName,
	}

	result, err := svc.DescribeCluster(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to describe cluster: %w", err)
	}

	// Get kubeconfig file path
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	// Load existing kubeconfig
	kubeconfig, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		kubeconfig = api.NewConfig()
	}

	// Create cluster entry
	cluster := api.NewCluster()
	cluster.Server = *result.Cluster.Endpoint
	cluster.CertificateAuthorityData = []byte(*result.Cluster.CertificateAuthority.Data)

	// Create auth entry
	authInfo := api.NewAuthInfo()
	if token := os.Getenv("AWS_TOKEN"); token != "" {
		authInfo.Token = token
	}

	// Create context entry
	context := api.NewContext()
	context.Cluster = clusterName
	context.AuthInfo = clusterName

	// Add to kubeconfig
	kubeconfig.Clusters[clusterName] = cluster
	kubeconfig.AuthInfos[clusterName] = authInfo
	kubeconfig.Contexts[clusterName] = context
	kubeconfig.CurrentContext = clusterName

	// Write updated kubeconfig
	err = clientcmd.WriteToFile(*kubeconfig, kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	return nil
}

// GetNodes retrieves all nodes in the cluster
func (c *KubeClient) GetNodes(ctx context.Context) (*corev1.NodeList, error) {
	return c.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
}

// GetPods retrieves all pods in the specified namespace
func (c *KubeClient) GetPods(ctx context.Context, namespace string) (*corev1.PodList, error) {
	return c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
}

// GetServices retrieves all services in the specified namespace
func (c *KubeClient) GetServices(ctx context.Context, namespace string) (*corev1.ServiceList, error) {
	return c.Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
}

// GetIngresses retrieves all ingresses in the specified namespace
func (c *KubeClient) GetIngresses(ctx context.Context, namespace string) (*networkingv1.IngressList, error) {
	return c.Clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
}

// GetNamespaces retrieves all namespaces in the cluster
func (c *KubeClient) GetNamespaces(ctx context.Context) (*corev1.NamespaceList, error) {
	return c.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
}

// GetNode gets a node by name
func (c *KubeClient) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	return c.Clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
}

// GetPod gets a pod by name and namespace
func (c *KubeClient) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return c.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetNetworkPolicies gets all network policies in a namespace
func (c *KubeClient) GetNetworkPolicies(ctx context.Context, namespace string) (*networkingv1.NetworkPolicyList, error) {
	return c.Clientset.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
}

// TestPodDNS tests DNS resolution from a pod
func (c *KubeClient) TestPodDNS(ctx context.Context, namespace, podName, hostname string) (bool, error) {
	// Create a temporary pod to test DNS
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dns-test-",
			Namespace:    namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "dns-test",
					Image:   "busybox",
					Command: []string{"nslookup", hostname},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	pod, err := c.Clientset.CoreV1().Pods(namespace).Create(ctx, testPod, metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to create test pod: %w", err)
	}

	defer c.Clientset.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})

	// Wait for pod completion
	watch, err := c.Clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.SingleObject(pod.ObjectMeta))
	if err != nil {
		return false, fmt.Errorf("failed to watch test pod: %w", err)
	}
	defer watch.Stop()

	for event := range watch.ResultChan() {
		pod := event.Object.(*corev1.Pod)
		if pod.Status.Phase == corev1.PodSucceeded {
			return true, nil
		} else if pod.Status.Phase == corev1.PodFailed {
			return false, fmt.Errorf("DNS test failed")
		}
	}

	return false, fmt.Errorf("watch ended before pod completion")
}

// TestPodConnectivity tests network connectivity between pods
func (c *KubeClient) TestPodConnectivity(ctx context.Context, sourceNS, sourcePod, targetNS, targetPod string) error {
	// Get target pod IP
	targetPodObj, err := c.GetPod(ctx, targetNS, targetPod)
	if err != nil {
		return fmt.Errorf("failed to get target pod: %w", err)
	}

	targetIP := targetPodObj.Status.PodIP
	if targetIP == "" {
		return fmt.Errorf("target pod has no IP address")
	}

	// Create test pod
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "network-test-",
			Namespace:    sourceNS,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "network-test",
					Image:   "busybox",
					Command: []string{"wget", "-T", "5", "-O-", fmt.Sprintf("http://%s:80", targetIP)},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	pod, err := c.Clientset.CoreV1().Pods(sourceNS).Create(ctx, testPod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create test pod: %w", err)
	}

	defer c.Clientset.CoreV1().Pods(sourceNS).Delete(ctx, pod.Name, metav1.DeleteOptions{})

	// Wait for pod completion
	watch, err := c.Clientset.CoreV1().Pods(sourceNS).Watch(ctx, metav1.SingleObject(pod.ObjectMeta))
	if err != nil {
		return fmt.Errorf("failed to watch test pod: %w", err)
	}
	defer watch.Stop()

	for event := range watch.ResultChan() {
		pod := event.Object.(*corev1.Pod)
		if pod.Status.Phase == corev1.PodSucceeded {
			return nil
		} else if pod.Status.Phase == corev1.PodFailed {
			return fmt.Errorf("connectivity test failed")
		}
	}

	return fmt.Errorf("watch ended before pod completion")
}

// CheckMTU checks MTU settings on cluster nodes
func (c *KubeClient) CheckMTU(ctx context.Context) (map[string]int, error) {
	mtuByNode := make(map[string]int)

	nodes, err := c.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes.Items {
		// Create test pod on the node
		testPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "mtu-test-",
				Namespace:    "default",
			},
			Spec: corev1.PodSpec{
				NodeName: node.Name,
				Containers: []corev1.Container{
					{
						Name:    "mtu-test",
						Image:   "busybox",
						Command: []string{"cat", "/sys/class/net/eth0/mtu"},
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
			},
		}

		pod, err := c.Clientset.CoreV1().Pods("default").Create(ctx, testPod, metav1.CreateOptions{})
		if err != nil {
			continue
		}

		// Get pod logs
		var mtu int
		logs, err := c.GetPodLogs(ctx, "default", pod.Name, "")
		if err == nil {
			fmt.Sscanf(logs, "%d", &mtu)
			mtuByNode[node.Name] = mtu
		}

		c.Clientset.CoreV1().Pods("default").Delete(ctx, pod.Name, metav1.DeleteOptions{})
	}

	return mtuByNode, nil
}

// GetAPIServerCertificate gets the API server's TLS certificate
func (c *KubeClient) GetAPIServerCertificate(ctx context.Context) (*x509.Certificate, error) {
	config := c.Config
	host := config.Host

	if !strings.HasPrefix(host, "https://") {
		return nil, fmt.Errorf("API server URL is not HTTPS")
	}

	conn, err := tls.Dial("tcp", strings.TrimPrefix(host, "https://"), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API server: %w", err)
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found")
	}

	return certs[0], nil
}

// GetIngressTLSCertificates gets TLS certificates from all Ingress resources
func (c *KubeClient) GetIngressTLSCertificates(ctx context.Context, namespace string) (map[string]*x509.Certificate, error) {
	certs := make(map[string]*x509.Certificate)

	ingresses, err := c.GetIngresses(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	for _, ing := range ingresses.Items {
		for _, tls := range ing.Spec.TLS {
			secret, err := c.Clientset.CoreV1().Secrets(ing.Namespace).Get(ctx, tls.SecretName, metav1.GetOptions{})
			if err != nil {
				continue
			}

			if certBytes, ok := secret.Data["tls.crt"]; ok {
				cert, err := parseCertificate(certBytes)
				if err != nil {
					continue
				}

				for _, host := range tls.Hosts {
					certs[host] = cert
				}
			}
		}
	}

	return certs, nil
}

// GetServiceCertificates gets TLS certificates from all services with TLS
func (c *KubeClient) GetServiceCertificates(ctx context.Context, namespace string) (map[string]*x509.Certificate, error) {
	certs := make(map[string]*x509.Certificate)

	services, err := c.GetServices(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	for _, svc := range services.Items {
		for _, port := range svc.Spec.Ports {
			if port.Name == "https" || port.Port == 443 {
				// Check for TLS secret in annotations
				if secretName, ok := svc.Annotations["tls.secretName"]; ok {
					secret, err := c.Clientset.CoreV1().Secrets(svc.Namespace).Get(ctx, secretName, metav1.GetOptions{})
					if err != nil {
						continue
					}

					if certBytes, ok := secret.Data["tls.crt"]; ok {
						cert, err := parseCertificate(certBytes)
						if err != nil {
							continue
						}
						certs[svc.Name] = cert
					}
				}
			}
		}
	}

	return certs, nil
}

// ValidateCertificateChains validates the certificate chains for all TLS certificates
func (c *KubeClient) ValidateCertificateChains(ctx context.Context, namespace string) (map[string]string, error) {
	issues := make(map[string]string)

	// Check Ingress certificates
	ingressCerts, err := c.GetIngressTLSCertificates(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get ingress certificates: %w", err)
	}

	for host, cert := range ingressCerts {
		if err := validateCertChain(cert); err != nil {
			issues[fmt.Sprintf("ingress/%s", host)] = err.Error()
		}
	}

	// Check Service certificates
	svcCerts, err := c.GetServiceCertificates(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get service certificates: %w", err)
	}

	for svc, cert := range svcCerts {
		if err := validateCertChain(cert); err != nil {
			issues[fmt.Sprintf("service/%s", svc)] = err.Error()
		}
	}

	return issues, nil
}

func validateCertChain(cert *x509.Certificate) error {
	// Check expiration
	if time.Now().After(cert.NotAfter) {
		return fmt.Errorf("certificate has expired")
	}

	if time.Now().Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid")
	}

	// Basic chain validation
	if cert.IssuingCertificateURL == nil || len(cert.IssuingCertificateURL) == 0 {
		return fmt.Errorf("no issuing certificate URL found")
	}

	return nil
}

func parseCertificate(certBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}
