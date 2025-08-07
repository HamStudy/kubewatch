package k8s

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/tools/clientcmd"
)

// ContextHealth represents the health status of a context
type ContextHealth struct {
	Context      string
	Healthy      bool
	LastCheck    time.Time
	Error        error
	ResponseTime time.Duration
}

// EnhancedMultiContextClient provides optimized multi-context operations
type EnhancedMultiContextClient struct {
	clients       map[string]*Client
	contexts      []string
	cache         *ResourceCache
	watchManager  *WatchCoalescer
	connPool      *ConnectionPool
	healthChecker *HealthChecker
	mu            sync.RWMutex

	// Performance settings
	parallelFetch       bool
	healthCheckInterval time.Duration
	contextTimeout      time.Duration
}

// EnhancedMultiContextOptions contains configuration for the enhanced client
type EnhancedMultiContextOptions struct {
	CacheSize           int
	CacheTTL            time.Duration
	ParallelFetch       bool
	HealthCheckInterval time.Duration
	ContextTimeout      time.Duration
	MaxConnections      int
}

// DefaultEnhancedMultiContextOptions returns default options
func DefaultEnhancedMultiContextOptions() *EnhancedMultiContextOptions {
	return &EnhancedMultiContextOptions{
		CacheSize:           1000,
		CacheTTL:            30 * time.Second,
		ParallelFetch:       true,
		HealthCheckInterval: 60 * time.Second,
		ContextTimeout:      10 * time.Second,
		MaxConnections:      50,
	}
}

// NewEnhancedMultiContextClient creates an optimized multi-context client
func NewEnhancedMultiContextClient(contextNames []string, opts *EnhancedMultiContextOptions) (*EnhancedMultiContextClient, error) {
	if len(contextNames) == 0 {
		return nil, fmt.Errorf("no contexts specified")
	}

	if opts == nil {
		opts = DefaultEnhancedMultiContextOptions()
	}

	// Create connection pool
	connPool := NewConnectionPool(opts.MaxConnections)

	emc := &EnhancedMultiContextClient{
		clients:             make(map[string]*Client),
		contexts:            contextNames,
		cache:               NewResourceCache(opts.CacheSize, opts.CacheTTL),
		watchManager:        NewWatchCoalescer(),
		connPool:            connPool,
		parallelFetch:       opts.ParallelFetch,
		healthCheckInterval: opts.HealthCheckInterval,
		contextTimeout:      opts.ContextTimeout,
	}

	// Initialize health checker
	emc.healthChecker = NewHealthChecker(connPool, opts.HealthCheckInterval, opts.ContextTimeout)

	// Load kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	// Initialize clients
	for _, contextName := range contextNames {
		configOverrides.CurrentContext = contextName

		client, err := NewClientWithContext(loadingRules, configOverrides)
		if err != nil {
			// Don't fail completely, just log and continue
			continue
		}

		emc.clients[contextName] = client
	}

	return emc, nil
}

// StartHealthMonitoring starts background health monitoring
func (emc *EnhancedMultiContextClient) StartHealthMonitoring(ctx context.Context) {
	go emc.healthChecker.Start(ctx)

	// Start cache cleanup routine
	go emc.cache.StartCleanupRoutine(ctx, 5*time.Minute)
}

// GetContexts returns the list of active contexts
func (emc *EnhancedMultiContextClient) GetContexts() []string {
	emc.mu.RLock()
	defer emc.mu.RUnlock()
	return append([]string{}, emc.contexts...)
}

// GetHealthyContexts returns only healthy contexts
func (emc *EnhancedMultiContextClient) GetHealthyContexts() []string {
	emc.mu.RLock()
	defer emc.mu.RUnlock()

	healthy := make([]string, 0, len(emc.contexts))
	for _, context := range emc.contexts {
		if _, exists := emc.clients[context]; exists {
			healthy = append(healthy, context)
		}
	}
	return healthy
}

// GetClient returns the client for a specific context
func (emc *EnhancedMultiContextClient) GetClient(context string) (*Client, error) {
	emc.mu.RLock()
	defer emc.mu.RUnlock()

	client, ok := emc.clients[context]
	if !ok {
		return nil, fmt.Errorf("no client for context %s", context)
	}
	return client, nil
}

// CheckContextHealth checks the health of a specific context
func (emc *EnhancedMultiContextClient) CheckContextHealth(ctx context.Context, contextName string) *ContextHealth {
	start := time.Now()
	health := &ContextHealth{
		Context:   contextName,
		LastCheck: start,
	}

	client, err := emc.GetClient(contextName)
	if err != nil {
		health.Healthy = false
		health.Error = err
		return health
	}

	// Create timeout context
	healthCtx, cancel := context.WithTimeout(ctx, emc.contextTimeout)
	defer cancel()

	// Simple health check - try to list namespaces
	_, err = client.ListNamespaces(healthCtx)
	health.ResponseTime = time.Since(start)

	if err != nil {
		health.Healthy = false
		health.Error = err
	} else {
		health.Healthy = true
	}

	return health
}

// ListPodsAllContextsOptimized returns pods from all contexts with caching and parallel fetching
func (emc *EnhancedMultiContextClient) ListPodsAllContextsOptimized(ctx context.Context, namespace string) ([]PodWithContext, error) {
	if !emc.parallelFetch {
		return emc.listPodsSequential(ctx, namespace)
	}

	return emc.listPodsParallel(ctx, namespace)
}

// listPodsParallel fetches pods from all contexts in parallel
func (emc *EnhancedMultiContextClient) listPodsParallel(ctx context.Context, namespace string) ([]PodWithContext, error) {
	type result struct {
		pods []PodWithContext
		err  error
	}

	resultChan := make(chan result, len(emc.contexts))
	var wg sync.WaitGroup

	for _, contextName := range emc.contexts {
		wg.Add(1)
		go func(ctxName string) {
			defer wg.Done()

			// Check cache first
			if cachedPods, found := emc.cache.GetPods(fmt.Sprintf("%s:%s", ctxName, namespace)); found {
				pods := make([]PodWithContext, len(cachedPods))
				for i, pod := range cachedPods {
					pods[i] = PodWithContext{
						Context: ctxName,
						Pod:     pod,
					}
				}
				resultChan <- result{pods: pods, err: nil}
				return
			}

			client, err := emc.GetClient(ctxName)
			if err != nil {
				resultChan <- result{err: fmt.Errorf("context %s: %w", ctxName, err)}
				return
			}

			// Create timeout context for this operation
			fetchCtx, cancel := context.WithTimeout(ctx, emc.contextTimeout)
			defer cancel()

			pods, err := client.ListPods(fetchCtx, namespace)
			if err != nil {
				resultChan <- result{err: fmt.Errorf("context %s: %w", ctxName, err)}
				return
			}

			// Cache the results
			emc.cache.SetPods(fmt.Sprintf("%s:%s", ctxName, namespace), pods, "")

			// Convert to PodWithContext
			podsWithContext := make([]PodWithContext, len(pods))
			for i, pod := range pods {
				podsWithContext[i] = PodWithContext{
					Context: ctxName,
					Pod:     pod,
				}
			}

			resultChan <- result{pods: podsWithContext, err: nil}
		}(contextName)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var allPods []PodWithContext
	var errors []error

	for res := range resultChan {
		if res.err != nil {
			errors = append(errors, res.err)
		} else {
			allPods = append(allPods, res.pods...)
		}
	}

	// Return partial results even if some contexts failed
	var err error
	if len(errors) > 0 {
		err = fmt.Errorf("errors from %d contexts: %v", len(errors), errors)
	}

	return allPods, err
}

// listPodsSequential fetches pods from contexts sequentially
func (emc *EnhancedMultiContextClient) listPodsSequential(ctx context.Context, namespace string) ([]PodWithContext, error) {
	var allPods []PodWithContext
	var errors []error

	for _, contextName := range emc.contexts {
		// Check cache first
		if cachedPods, found := emc.cache.GetPods(fmt.Sprintf("%s:%s", contextName, namespace)); found {
			for _, pod := range cachedPods {
				allPods = append(allPods, PodWithContext{
					Context: contextName,
					Pod:     pod,
				})
			}
			continue
		}

		client, err := emc.GetClient(contextName)
		if err != nil {
			errors = append(errors, fmt.Errorf("context %s: %w", contextName, err))
			continue
		}

		// Create timeout context for this operation
		fetchCtx, cancel := context.WithTimeout(ctx, emc.contextTimeout)

		pods, err := client.ListPods(fetchCtx, namespace)
		cancel()

		if err != nil {
			errors = append(errors, fmt.Errorf("context %s: %w", contextName, err))
			continue
		}

		// Cache the results
		emc.cache.SetPods(fmt.Sprintf("%s:%s", contextName, namespace), pods, "")

		for _, pod := range pods {
			allPods = append(allPods, PodWithContext{
				Context: contextName,
				Pod:     pod,
			})
		}
	}

	var err error
	if len(errors) > 0 {
		err = fmt.Errorf("errors from %d contexts: %v", len(errors), errors)
	}

	return allPods, err
}

// ListDeploymentsAllContextsOptimized returns deployments from all contexts with optimizations
func (emc *EnhancedMultiContextClient) ListDeploymentsAllContextsOptimized(ctx context.Context, namespace string) ([]DeploymentWithContext, error) {
	if !emc.parallelFetch {
		return emc.listDeploymentsSequential(ctx, namespace)
	}

	return emc.listDeploymentsParallel(ctx, namespace)
}

// listDeploymentsParallel fetches deployments from all contexts in parallel
func (emc *EnhancedMultiContextClient) listDeploymentsParallel(ctx context.Context, namespace string) ([]DeploymentWithContext, error) {
	type result struct {
		deployments []DeploymentWithContext
		err         error
	}

	resultChan := make(chan result, len(emc.contexts))
	var wg sync.WaitGroup

	for _, contextName := range emc.contexts {
		wg.Add(1)
		go func(ctxName string) {
			defer wg.Done()

			// Check cache first
			if cachedDeployments, found := emc.cache.GetDeployments(fmt.Sprintf("%s:%s", ctxName, namespace)); found {
				deployments := make([]DeploymentWithContext, len(cachedDeployments))
				for i, deployment := range cachedDeployments {
					deployments[i] = DeploymentWithContext{
						Context:    ctxName,
						Deployment: deployment,
					}
				}
				resultChan <- result{deployments: deployments, err: nil}
				return
			}

			client, err := emc.GetClient(ctxName)
			if err != nil {
				resultChan <- result{err: fmt.Errorf("context %s: %w", ctxName, err)}
				return
			}

			fetchCtx, cancel := context.WithTimeout(ctx, emc.contextTimeout)
			defer cancel()

			deployments, err := client.ListDeployments(fetchCtx, namespace)
			if err != nil {
				resultChan <- result{err: fmt.Errorf("context %s: %w", ctxName, err)}
				return
			}

			// Cache the results
			emc.cache.SetDeployments(fmt.Sprintf("%s:%s", ctxName, namespace), deployments, "")

			deploymentsWithContext := make([]DeploymentWithContext, len(deployments))
			for i, deployment := range deployments {
				deploymentsWithContext[i] = DeploymentWithContext{
					Context:    ctxName,
					Deployment: deployment,
				}
			}

			resultChan <- result{deployments: deploymentsWithContext, err: nil}
		}(contextName)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allDeployments []DeploymentWithContext
	var errors []error

	for res := range resultChan {
		if res.err != nil {
			errors = append(errors, res.err)
		} else {
			allDeployments = append(allDeployments, res.deployments...)
		}
	}

	var err error
	if len(errors) > 0 {
		err = fmt.Errorf("errors from %d contexts: %v", len(errors), errors)
	}

	return allDeployments, err
}

// listDeploymentsSequential fetches deployments sequentially
func (emc *EnhancedMultiContextClient) listDeploymentsSequential(ctx context.Context, namespace string) ([]DeploymentWithContext, error) {
	var allDeployments []DeploymentWithContext
	var errors []error

	for _, contextName := range emc.contexts {
		// Check cache first
		if cachedDeployments, found := emc.cache.GetDeployments(fmt.Sprintf("%s:%s", contextName, namespace)); found {
			for _, deployment := range cachedDeployments {
				allDeployments = append(allDeployments, DeploymentWithContext{
					Context:    contextName,
					Deployment: deployment,
				})
			}
			continue
		}

		client, err := emc.GetClient(contextName)
		if err != nil {
			errors = append(errors, fmt.Errorf("context %s: %w", contextName, err))
			continue
		}

		fetchCtx, cancel := context.WithTimeout(ctx, emc.contextTimeout)

		deployments, err := client.ListDeployments(fetchCtx, namespace)
		cancel()

		if err != nil {
			errors = append(errors, fmt.Errorf("context %s: %w", contextName, err))
			continue
		}

		// Cache the results
		emc.cache.SetDeployments(fmt.Sprintf("%s:%s", contextName, namespace), deployments, "")

		for _, deployment := range deployments {
			allDeployments = append(allDeployments, DeploymentWithContext{
				Context:    contextName,
				Deployment: deployment,
			})
		}
	}

	var err error
	if len(errors) > 0 {
		err = fmt.Errorf("errors from %d contexts: %v", len(errors), errors)
	}

	return allDeployments, err
}

// InvalidateCache invalidates cache for a specific namespace across all contexts
func (emc *EnhancedMultiContextClient) InvalidateCache(namespace string) {
	for _, context := range emc.contexts {
		emc.cache.InvalidateNamespace(fmt.Sprintf("%s:%s", context, namespace))
	}
}

// GetCacheMetrics returns cache performance metrics
func (emc *EnhancedMultiContextClient) GetCacheMetrics() *CacheMetrics {
	return emc.cache.GetMetrics()
}

// SetParallelFetch enables or disables parallel fetching
func (emc *EnhancedMultiContextClient) SetParallelFetch(enabled bool) {
	emc.mu.Lock()
	defer emc.mu.Unlock()
	emc.parallelFetch = enabled
}

// AddWatchListener adds a watch listener for a resource across contexts
func (emc *EnhancedMultiContextClient) AddWatchListener(ctx context.Context, namespace, resource string) (<-chan WatchEvent, error) {
	// For now, watch the first healthy context
	// In a full implementation, you might want to watch all contexts
	healthyContexts := emc.GetHealthyContexts()
	if len(healthyContexts) == 0 {
		return nil, fmt.Errorf("no healthy contexts available")
	}

	client, err := emc.GetClient(healthyContexts[0])
	if err != nil {
		return nil, err
	}

	return emc.watchManager.AddWatchListener(ctx, client, healthyContexts[0], namespace, resource)
}

// RemoveWatchListener removes a watch listener
func (emc *EnhancedMultiContextClient) RemoveWatchListener(contextName, namespace, resource string, listener <-chan WatchEvent) {
	emc.watchManager.RemoveWatchListener(contextName, namespace, resource, listener)
}

// Close cleans up resources
func (emc *EnhancedMultiContextClient) Close() {
	emc.cache.Clear()
	emc.connPool.Clear()
}
