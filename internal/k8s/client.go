package k8s

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Client represents a Kubernetes client
type Client struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsclient.Clientset
	config        *rest.Config
}

// ClientOptions contains additional options for creating a Kubernetes client
type ClientOptions struct {
	Context              string
	Namespace            string
	User                 string
	Cluster              string
	ClientCertificate    string
	ClientKey            string
	CertificateAuthority string
	InsecureSkipVerify   bool
	Token                string
	TokenFile            string
	Impersonate          string
	ImpersonateGroups    []string
	ImpersonateUID       string
	Timeout              string
	CacheDir             string
}

// getPathSeparator returns the OS-specific path list separator
func getPathSeparator() string {
	if runtime.GOOS == "windows" {
		return ";"
	}
	return ":"
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfig string) (*Client, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

		// Handle KUBECONFIG environment variable with multiple paths
		if kubeconfig != "" {
			// Split by OS-specific separator to handle multiple paths
			separator := getPathSeparator()
			paths := strings.Split(kubeconfig, separator)

			// Filter out empty paths and trim whitespace
			var validPaths []string
			for _, path := range paths {
				trimmed := strings.TrimSpace(path)
				if trimmed != "" {
					validPaths = append(validPaths, trimmed)
				}
			}

			if len(validPaths) > 0 {
				loadingRules.Precedence = validPaths
			}
		} else {
			// Use default kubeconfig location if KUBECONFIG is not set
			if home, err := os.UserHomeDir(); err == nil && home != "" {
				defaultPath := filepath.Join(home, ".kube", "config")
				if _, err := os.Stat(defaultPath); err == nil {
					loadingRules.Precedence = []string{defaultPath}
				}
			}
		}

		// Create the client config
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules,
			configOverrides,
		)

		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create metrics client (optional - don't fail if metrics API is not available)
	metricsClient, _ := metricsclient.NewForConfig(config)

	return &Client{
		clientset:     clientset,
		metricsClient: metricsClient,
		config:        config,
	}, nil
}

// NewClientWithOptions creates a new Kubernetes client with additional options
func NewClientWithOptions(kubeconfig string, opts *ClientOptions) (*Client, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

		// Handle KUBECONFIG environment variable with multiple paths
		if kubeconfig != "" {
			// Split by OS-specific separator to handle multiple paths
			separator := getPathSeparator()
			paths := strings.Split(kubeconfig, separator)

			// Filter out empty paths and trim whitespace
			var validPaths []string
			for _, path := range paths {
				trimmed := strings.TrimSpace(path)
				if trimmed != "" {
					validPaths = append(validPaths, trimmed)
				}
			}

			if len(validPaths) > 0 {
				loadingRules.Precedence = validPaths
			}
		} else {
			// Use default kubeconfig location if KUBECONFIG is not set
			if home, err := os.UserHomeDir(); err == nil && home != "" {
				defaultPath := filepath.Join(home, ".kube", "config")
				if _, err := os.Stat(defaultPath); err == nil {
					loadingRules.Precedence = []string{defaultPath}
				}
			}
		}

		// Create config overrides from options
		configOverrides := &clientcmd.ConfigOverrides{}

		if opts != nil {
			// Set context override
			if opts.Context != "" {
				configOverrides.CurrentContext = opts.Context
			}

			// Set cluster overrides
			if opts.CertificateAuthority != "" {
				configOverrides.ClusterInfo.CertificateAuthority = opts.CertificateAuthority
			}
			if opts.InsecureSkipVerify {
				configOverrides.ClusterInfo.InsecureSkipTLSVerify = opts.InsecureSkipVerify
			}
			if opts.Cluster != "" {
				configOverrides.ClusterInfo.Server = opts.Cluster
			}

			// Set auth overrides
			if opts.ClientCertificate != "" {
				configOverrides.AuthInfo.ClientCertificate = opts.ClientCertificate
			}
			if opts.ClientKey != "" {
				configOverrides.AuthInfo.ClientKey = opts.ClientKey
			}
			if opts.Token != "" {
				configOverrides.AuthInfo.Token = opts.Token
			}
			if opts.TokenFile != "" {
				configOverrides.AuthInfo.TokenFile = opts.TokenFile
			}
			if opts.Impersonate != "" {
				configOverrides.AuthInfo.Impersonate = opts.Impersonate
			}
			if len(opts.ImpersonateGroups) > 0 {
				configOverrides.AuthInfo.ImpersonateGroups = opts.ImpersonateGroups
			}
			if opts.ImpersonateUID != "" {
				configOverrides.AuthInfo.ImpersonateUserExtra = map[string][]string{
					"uid": {opts.ImpersonateUID},
				}
			}

			// Set namespace override
			if opts.Namespace != "" {
				configOverrides.Context.Namespace = opts.Namespace
			}

			// Set timeout if specified
			if opts.Timeout != "" {
				configOverrides.Timeout = opts.Timeout
			}
		}

		// Create the client config
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules,
			configOverrides,
		)

		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build config: %w", err)
		}

		// Apply additional REST config settings
		if opts != nil && opts.Timeout != "" {
			// Parse and apply timeout to REST config
			// This would need proper duration parsing
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create metrics client (optional - don't fail if metrics API is not available)
	metricsClient, _ := metricsclient.NewForConfig(config)

	return &Client{
		clientset:     clientset,
		metricsClient: metricsClient,
		config:        config,
	}, nil
}

// GetNamespaces returns all namespaces
func (c *Client) GetNamespaces(ctx context.Context) ([]v1.Namespace, error) {
	list, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListNamespaces returns all namespaces
func (c *Client) ListNamespaces(ctx context.Context) ([]v1.Namespace, error) {
	list, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListPods returns pods in a namespace
func (c *Client) ListPods(ctx context.Context, namespace string) ([]v1.Pod, error) {
	list, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// WatchPods watches for pod changes
func (c *Client) WatchPods(ctx context.Context, namespace string) (watch.Interface, error) {
	return c.clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{})
}

// DeletePod deletes a pod
func (c *Client) DeletePod(ctx context.Context, namespace, name string) error {
	return c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// DeletePods deletes multiple pods
func (c *Client) DeletePods(ctx context.Context, namespace string, names []string) error {
	for _, name := range names {
		if err := c.DeletePod(ctx, namespace, name); err != nil {
			return err
		}
	}
	return nil
}

// GetPodLogs returns logs for a pod
// GetPodLogs returns a stream of pod logs
func (c *Client) GetPodLogs(ctx context.Context, namespace, pod, container string, follow bool, tailLines int64) (io.ReadCloser, error) {
	opts := &v1.PodLogOptions{
		Follow:    follow,
		TailLines: &tailLines,
	}
	if container != "" {
		opts.Container = container
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(pod, opts)
	return req.Stream(ctx)
}

// GetPodLogsWithOptions returns a stream of pod logs with more options
func (c *Client) GetPodLogsWithOptions(ctx context.Context, namespace, pod, container string, follow bool, tailLines int64, previous bool, sinceTime *time.Time, timestamps bool) (io.ReadCloser, error) {
	opts := &v1.PodLogOptions{
		Follow:     follow,
		TailLines:  &tailLines,
		Previous:   previous,
		Timestamps: timestamps,
	}
	if container != "" {
		opts.Container = container
	}
	if sinceTime != nil {
		opts.SinceTime = &metav1.Time{Time: *sinceTime}
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(pod, opts)
	return req.Stream(ctx)
}

// ListDeployments returns deployments in a namespace
func (c *Client) ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	list, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// WatchDeployments watches for deployment changes
func (c *Client) WatchDeployments(ctx context.Context, namespace string) (watch.Interface, error) {
	return c.clientset.AppsV1().Deployments(namespace).Watch(ctx, metav1.ListOptions{})
}

// DeleteDeployment deletes a deployment
func (c *Client) DeleteDeployment(ctx context.Context, namespace, name string) error {
	return c.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ListStatefulSets returns statefulsets in a namespace
func (c *Client) ListStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error) {
	list, err := c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// WatchStatefulSets watches for statefulset changes
func (c *Client) WatchStatefulSets(ctx context.Context, namespace string) (watch.Interface, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).Watch(ctx, metav1.ListOptions{})
}

// DeleteStatefulSet deletes a statefulset
func (c *Client) DeleteStatefulSet(ctx context.Context, namespace, name string) error {
	return c.clientset.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ListServices returns services in a namespace
func (c *Client) ListServices(ctx context.Context, namespace string) ([]v1.Service, error) {
	list, err := c.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// WatchServices watches for service changes
func (c *Client) WatchServices(ctx context.Context, namespace string) (watch.Interface, error) {
	return c.clientset.CoreV1().Services(namespace).Watch(ctx, metav1.ListOptions{})
}

// DeleteService deletes a service
func (c *Client) DeleteService(ctx context.Context, namespace, name string) error {
	return c.clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ListIngresses returns ingresses in a namespace
func (c *Client) ListIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error) {
	list, err := c.clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// WatchIngresses watches for ingress changes
func (c *Client) WatchIngresses(ctx context.Context, namespace string) (watch.Interface, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).Watch(ctx, metav1.ListOptions{})
}

// DeleteIngress deletes an ingress
func (c *Client) DeleteIngress(ctx context.Context, namespace, name string) error {
	return c.clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ListConfigMaps returns configmaps in a namespace
func (c *Client) ListConfigMaps(ctx context.Context, namespace string) ([]v1.ConfigMap, error) {
	list, err := c.clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// WatchConfigMaps watches for configmap changes
func (c *Client) WatchConfigMaps(ctx context.Context, namespace string) (watch.Interface, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Watch(ctx, metav1.ListOptions{})
}

// DeleteConfigMap deletes a configmap
func (c *Client) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	return c.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ListSecrets returns secrets in a namespace
func (c *Client) ListSecrets(ctx context.Context, namespace string) ([]v1.Secret, error) {
	list, err := c.clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// WatchSecrets watches for secret changes
func (c *Client) WatchSecrets(ctx context.Context, namespace string) (watch.Interface, error) {
	return c.clientset.CoreV1().Secrets(namespace).Watch(ctx, metav1.ListOptions{})
}

// DeleteSecret deletes a secret
func (c *Client) DeleteSecret(ctx context.Context, namespace, name string) error {
	return c.clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetPodsForDeployment returns all pods for a deployment
func (c *Client) GetPodsForDeployment(ctx context.Context, namespace, deploymentName string) ([]v1.Pod, error) {
	deployment, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	return pods.Items, nil
}

// GetPodsForStatefulSet returns pods belonging to a statefulset
func (c *Client) GetPodsForStatefulSet(ctx context.Context, namespace, statefulSetName string) ([]v1.Pod, error) {
	statefulSet, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	labelSelector := metav1.FormatLabelSelector(statefulSet.Spec.Selector)
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	return pods.Items, nil
}

// PodMetrics represents CPU and memory metrics for a pod
type PodMetrics struct {
	Name      string
	Namespace string
	CPU       string // in millicores (e.g., "100m")
	Memory    string // in bytes (e.g., "128Mi")
}

// formatCPU formats CPU value from millicores to a readable string
func formatCPU(milliCPU int64) string {
	if milliCPU == 0 {
		return "-"
	}

	if milliCPU < 1000 {
		// Less than 1 core: show as millicores
		return fmt.Sprintf("%dm", milliCPU)
	} else {
		// 1 core or more: show as cores without decimals
		cores := milliCPU / 1000
		return fmt.Sprintf("%d", cores)
	}
}

// formatMemory formats memory value from bytes to a readable string
func formatMemory(bytes int64) string {
	if bytes == 0 {
		return "-"
	}

	const (
		Ki = 1024
		Mi = 1024 * Ki
		Gi = 1024 * Mi
	)

	if bytes >= Gi {
		gb := bytes / Gi
		remainder := (bytes % Gi) / Mi
		// Show fractional GB if significant
		if remainder >= 100 && gb < 10 {
			return fmt.Sprintf("%d.%dGi", gb, remainder/100)
		}
		return fmt.Sprintf("%dGi", gb)
	} else if bytes >= Mi {
		mb := bytes / Mi
		return fmt.Sprintf("%dMi", mb)
	} else if bytes >= Ki {
		kb := bytes / Ki
		return fmt.Sprintf("%dKi", kb)
	}
	return fmt.Sprintf("%dB", bytes)
}

const (
	Ki = 1024
	Mi = 1024 * Ki
	Gi = 1024 * Mi
)

// GetPodMetrics returns metrics for pods in a namespace
func (c *Client) GetPodMetrics(ctx context.Context, namespace string) (map[string]*PodMetrics, error) {
	if c.metricsClient == nil {
		return nil, fmt.Errorf("metrics API not available")
	}

	metrics, err := c.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w", err)
	}

	result := make(map[string]*PodMetrics)
	for _, m := range metrics.Items {
		var totalCPU int64
		var totalMemory int64

		// Sum up container metrics
		for _, container := range m.Containers {
			if cpuQuantity, ok := container.Usage[v1.ResourceCPU]; ok {
				// CPU is in nanocores, convert to millicores
				totalCPU += cpuQuantity.MilliValue()
			}
			if memQuantity, ok := container.Usage[v1.ResourceMemory]; ok {
				totalMemory += memQuantity.Value()
			}
		}

		// Format CPU with appropriate precision
		cpu := formatCPU(totalCPU)

		// Format memory in human-readable format
		memory := formatMemory(totalMemory)

		result[m.Name] = &PodMetrics{
			Name:      m.Name,
			Namespace: m.Namespace,
			CPU:       cpu,
			Memory:    memory,
		}
	}
	return result, nil
}

// GetNodeMetrics returns metrics for nodes
func (c *Client) GetNodeMetrics(ctx context.Context) (map[string]*NodeMetrics, error) {
	if c.metricsClient == nil {
		return nil, fmt.Errorf("metrics API not available")
	}

	metrics, err := c.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %w", err)
	}

	result := make(map[string]*NodeMetrics)
	for _, m := range metrics.Items {
		var cpuMilli int64
		var memBytes int64

		if cpuQuantity, ok := m.Usage[v1.ResourceCPU]; ok {
			cpuMilli = cpuQuantity.MilliValue()
		}
		if memQuantity, ok := m.Usage[v1.ResourceMemory]; ok {
			memBytes = memQuantity.Value()
		}

		result[m.Name] = &NodeMetrics{
			Name:   m.Name,
			CPU:    formatCPU(cpuMilli),
			Memory: formatMemory(memBytes),
		}
	}
	return result, nil
}

// NodeMetrics represents CPU and memory metrics for a node
type NodeMetrics struct {
	Name   string
	CPU    string
	Memory string
}
