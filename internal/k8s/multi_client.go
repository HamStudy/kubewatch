package k8s

import (
	"context"
	"fmt"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// MultiContextClient manages multiple Kubernetes clients for different contexts
type MultiContextClient struct {
	clients  map[string]*Client // context name -> client
	contexts []string           // list of context names
	mu       sync.RWMutex
}

// ResourceWithContext wraps a resource with its context information
type ResourceWithContext struct {
	Context  string
	Resource interface{}
}

// NewMultiContextClient creates a client that can work with multiple contexts
func NewMultiContextClient(contextNames []string) (*MultiContextClient, error) {
	if len(contextNames) == 0 {
		return nil, fmt.Errorf("no contexts specified")
	}

	mc := &MultiContextClient{
		clients:  make(map[string]*Client),
		contexts: contextNames,
	}

	// Load kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	for _, contextName := range contextNames {
		// Set context override for this specific client
		configOverrides.CurrentContext = contextName

		// Create client for this context
		client, err := NewClientWithContext(loadingRules, configOverrides)
		if err != nil {
			return nil, fmt.Errorf("failed to create client for context %s: %w", contextName, err)
		}

		mc.clients[contextName] = client
	}

	return mc, nil
}

// NewClientWithContext creates a client for a specific context
func NewClientWithContext(loadingRules *clientcmd.ClientConfigLoadingRules, overrides *clientcmd.ConfigOverrides) (*Client, error) {
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		overrides,
	)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	return NewClientFromConfig(config)
}

// GetContexts returns the list of active contexts
func (mc *MultiContextClient) GetContexts() []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return append([]string{}, mc.contexts...)
}

// GetClient returns the client for a specific context
func (mc *MultiContextClient) GetClient(context string) (*Client, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	client, ok := mc.clients[context]
	if !ok {
		return nil, fmt.Errorf("no client for context %s", context)
	}
	return client, nil
}

// ListPodsAllContexts returns pods from all contexts with context information
func (mc *MultiContextClient) ListPodsAllContexts(ctx context.Context, namespace string) ([]PodWithContext, error) {
	var allPods []PodWithContext
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(mc.contexts))

	for _, contextName := range mc.contexts {
		wg.Add(1)
		go func(ctxName string) {
			defer wg.Done()

			client, err := mc.GetClient(ctxName)
			if err != nil {
				errChan <- fmt.Errorf("context %s: %w", ctxName, err)
				return
			}

			pods, err := client.ListPods(ctx, namespace)
			if err != nil {
				errChan <- fmt.Errorf("context %s: %w", ctxName, err)
				return
			}

			mu.Lock()
			for _, pod := range pods {
				allPods = append(allPods, PodWithContext{
					Context: ctxName,
					Pod:     pod,
				})
			}
			mu.Unlock()
		}(contextName)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		// Return partial results with error
		return allPods, fmt.Errorf("errors from %d contexts: %v", len(errs), errs)
	}

	return allPods, nil
}

// PodWithContext wraps a pod with its context
type PodWithContext struct {
	Context string
	Pod     v1.Pod
}

// DeploymentWithContext wraps a deployment with its context
type DeploymentWithContext struct {
	Context    string
	Deployment appsv1.Deployment
}

// ListDeploymentsAllContexts returns deployments from all contexts
func (mc *MultiContextClient) ListDeploymentsAllContexts(ctx context.Context, namespace string) ([]DeploymentWithContext, error) {
	var allDeployments []DeploymentWithContext
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(mc.contexts))

	for _, contextName := range mc.contexts {
		wg.Add(1)
		go func(ctxName string) {
			defer wg.Done()

			client, err := mc.GetClient(ctxName)
			if err != nil {
				errChan <- fmt.Errorf("context %s: %w", ctxName, err)
				return
			}

			deployments, err := client.ListDeployments(ctx, namespace)
			if err != nil {
				errChan <- fmt.Errorf("context %s: %w", ctxName, err)
				return
			}

			mu.Lock()
			for _, deployment := range deployments {
				allDeployments = append(allDeployments, DeploymentWithContext{
					Context:    ctxName,
					Deployment: deployment,
				})
			}
			mu.Unlock()
		}(contextName)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return allDeployments, fmt.Errorf("errors from %d contexts: %v", len(errs), errs)
	}

	return allDeployments, nil
}

// ServiceWithContext wraps a service with its context
type ServiceWithContext struct {
	Context string
	Service v1.Service
}

// ConfigMapWithContext wraps a ConfigMap with its context
type ConfigMapWithContext struct {
	Context   string
	ConfigMap v1.ConfigMap
}

// SecretWithContext wraps a Secret with its context
type SecretWithContext struct {
	Context string
	Secret  v1.Secret
}

// IngressWithContext wraps an Ingress with its context
type IngressWithContext struct {
	Context string
	Ingress networkingv1.Ingress
}

// StatefulSetWithContext wraps a StatefulSet with its context
type StatefulSetWithContext struct {
	Context     string
	StatefulSet appsv1.StatefulSet
}

// NamespaceWithContext wraps a namespace with its context
type NamespaceWithContext struct {
	Context   string
	Namespace v1.Namespace
}

// ListNamespacesAllContexts returns namespaces from all contexts with context information
func (mc *MultiContextClient) ListNamespacesAllContexts(ctx context.Context) ([]NamespaceWithContext, error) {
	var allNamespaces []NamespaceWithContext
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(mc.contexts))

	for _, contextName := range mc.contexts {
		wg.Add(1)
		go func(ctxName string) {
			defer wg.Done()

			client, err := mc.GetClient(ctxName)
			if err != nil {
				errChan <- fmt.Errorf("context %s: %w", ctxName, err)
				return
			}

			namespaces, err := client.ListNamespaces(ctx)
			if err != nil {
				errChan <- fmt.Errorf("context %s: %w", ctxName, err)
				return
			}

			mu.Lock()
			for _, namespace := range namespaces {
				allNamespaces = append(allNamespaces, NamespaceWithContext{
					Context:   ctxName,
					Namespace: namespace,
				})
			}
			mu.Unlock()
		}(contextName)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		// Return partial results with error
		return allNamespaces, fmt.Errorf("errors from %d contexts: %v", len(errs), errs)
	}

	return allNamespaces, nil
}

// GetUniqueNamespaces returns unique namespace names from all contexts
func (mc *MultiContextClient) GetUniqueNamespaces(ctx context.Context) ([]v1.Namespace, error) {
	namespacesWithContext, err := mc.ListNamespacesAllContexts(ctx)
	if err != nil {
		return nil, err
	}

	// Create a map to deduplicate namespaces by name
	uniqueNamespaces := make(map[string]v1.Namespace)
	for _, nsWithCtx := range namespacesWithContext {
		uniqueNamespaces[nsWithCtx.Namespace.Name] = nsWithCtx.Namespace
	}

	// Convert back to slice
	var result []v1.Namespace
	for _, ns := range uniqueNamespaces {
		result = append(result, ns)
	}

	return result, nil
}

// GetAvailableContexts returns all available contexts from kubeconfig
func GetAvailableContexts() ([]string, string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := loadingRules.Load()
	if err != nil {
		return nil, "", err
	}

	var contexts []string
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}

	return contexts, config.CurrentContext, nil
}
