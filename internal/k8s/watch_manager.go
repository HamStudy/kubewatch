package k8s

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/workqueue"
)

// WatchEvent represents a Kubernetes watch event with context
type WatchEvent struct {
	Type      watch.EventType
	Object    interface{}
	Context   string
	Namespace string
	Resource  string
	Timestamp time.Time
}

// WatchCoalescer coalesces multiple watch requests for the same resource
type WatchCoalescer struct {
	activeWatches map[string]*WatchRequest
	eventChan     chan WatchEvent
	mu            sync.RWMutex
}

// WatchRequest represents a watch request with metadata
type WatchRequest struct {
	Context   string
	Namespace string
	Resource  string
	Watcher   watch.Interface
	Listeners []chan WatchEvent
	mu        sync.RWMutex
}

// NewWatchCoalescer creates a new watch coalescer
func NewWatchCoalescer() *WatchCoalescer {
	return &WatchCoalescer{
		activeWatches: make(map[string]*WatchRequest),
		eventChan:     make(chan WatchEvent, 1000), // Buffered channel for high throughput
	}
}

// generateWatchKey creates a unique key for a watch request
func (wc *WatchCoalescer) generateWatchKey(context, namespace, resource string) string {
	return fmt.Sprintf("%s:%s:%s", context, namespace, resource)
}

// AddWatchListener adds a listener to an existing watch or creates a new one
func (wc *WatchCoalescer) AddWatchListener(ctx context.Context, client *Client, contextName, namespace, resource string) (<-chan WatchEvent, error) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	key := wc.generateWatchKey(contextName, namespace, resource)
	listenerChan := make(chan WatchEvent, 100)

	if req, exists := wc.activeWatches[key]; exists {
		// Add to existing watch
		req.mu.Lock()
		req.Listeners = append(req.Listeners, listenerChan)
		req.mu.Unlock()
		return listenerChan, nil
	}

	// Create new watch
	var watcher watch.Interface
	var err error

	switch resource {
	case "pods":
		watcher, err = client.WatchPods(ctx, namespace)
	case "deployments":
		watcher, err = client.WatchDeployments(ctx, namespace)
	case "services":
		watcher, err = client.WatchServices(ctx, namespace)
	case "configmaps":
		watcher, err = client.WatchConfigMaps(ctx, namespace)
	case "secrets":
		watcher, err = client.WatchSecrets(ctx, namespace)
	case "statefulsets":
		watcher, err = client.WatchStatefulSets(ctx, namespace)
	case "ingresses":
		watcher, err = client.WatchIngresses(ctx, namespace)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resource)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	req := &WatchRequest{
		Context:   contextName,
		Namespace: namespace,
		Resource:  resource,
		Watcher:   watcher,
		Listeners: []chan WatchEvent{listenerChan},
	}

	wc.activeWatches[key] = req

	// Start watching in background
	go wc.watchLoop(ctx, req)

	return listenerChan, nil
}

// watchLoop processes watch events and distributes them to listeners
func (wc *WatchCoalescer) watchLoop(ctx context.Context, req *WatchRequest) {
	defer func() {
		// Clean up when watch ends
		wc.mu.Lock()
		key := wc.generateWatchKey(req.Context, req.Namespace, req.Resource)
		delete(wc.activeWatches, key)
		wc.mu.Unlock()

		// Close all listener channels
		req.mu.Lock()
		for _, listener := range req.Listeners {
			close(listener)
		}
		req.mu.Unlock()

		// Stop the watcher
		req.Watcher.Stop()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-req.Watcher.ResultChan():
			if !ok {
				return
			}

			watchEvent := WatchEvent{
				Type:      event.Type,
				Object:    event.Object,
				Context:   req.Context,
				Namespace: req.Namespace,
				Resource:  req.Resource,
				Timestamp: time.Now(),
			}

			// Distribute to all listeners
			req.mu.RLock()
			for _, listener := range req.Listeners {
				select {
				case listener <- watchEvent:
				default:
					// Listener channel is full, skip this event
				}
			}
			req.mu.RUnlock()
		}
	}
}

// RemoveWatchListener removes a listener from a watch
func (wc *WatchCoalescer) RemoveWatchListener(contextName, namespace, resource string, listener <-chan WatchEvent) {
	wc.mu.RLock()
	key := wc.generateWatchKey(contextName, namespace, resource)
	req, exists := wc.activeWatches[key]
	wc.mu.RUnlock()

	if !exists {
		return
	}

	req.mu.Lock()
	defer req.mu.Unlock()

	// Remove listener from the list
	for i, l := range req.Listeners {
		if l == listener {
			req.Listeners = append(req.Listeners[:i], req.Listeners[i+1:]...)
			close(l)
			break
		}
	}

	// If no more listeners, stop the watch
	if len(req.Listeners) == 0 {
		wc.mu.Lock()
		delete(wc.activeWatches, key)
		wc.mu.Unlock()
		req.Watcher.Stop()
	}
}

// ReconnectionManager handles watch reconnections with exponential backoff
type ReconnectionManager struct {
	baseDelay     time.Duration
	maxDelay      time.Duration
	backoffFactor float64
	maxRetries    int
}

// NewReconnectionManager creates a new reconnection manager
func NewReconnectionManager() *ReconnectionManager {
	return &ReconnectionManager{
		baseDelay:     1 * time.Second,
		maxDelay:      30 * time.Second,
		backoffFactor: 2.0,
		maxRetries:    10,
	}
}

// RetryWithBackoff executes a function with exponential backoff
func (rm *ReconnectionManager) RetryWithBackoff(ctx context.Context, operation func() error) error {
	var lastErr error
	delay := rm.baseDelay

	for attempt := 0; attempt < rm.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := operation(); err != nil {
			lastErr = err

			// Wait before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Increase delay for next attempt
			delay = time.Duration(float64(delay) * rm.backoffFactor)
			if delay > rm.maxDelay {
				delay = rm.maxDelay
			}

			continue
		}

		// Success
		return nil
	}

	return fmt.Errorf("operation failed after %d attempts: %w", rm.maxRetries, lastErr)
}

// EventProcessor processes watch events with batching and rate limiting
type EventProcessor struct {
	eventQueue   workqueue.RateLimitingInterface
	batchSize    int
	batchTimeout time.Duration
	processor    func([]WatchEvent) error
	mu           sync.RWMutex
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(processor func([]WatchEvent) error) *EventProcessor {
	return &EventProcessor{
		eventQueue:   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		batchSize:    50,
		batchTimeout: 100 * time.Millisecond,
		processor:    processor,
	}
}

// AddEvent adds an event to the processing queue
func (ep *EventProcessor) AddEvent(event WatchEvent) {
	ep.eventQueue.Add(event)
}

// Start starts the event processing loop
func (ep *EventProcessor) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		go ep.worker(ctx)
	}
}

// worker processes events from the queue
func (ep *EventProcessor) worker(ctx context.Context) {
	batch := make([]WatchEvent, 0, ep.batchSize)
	timer := time.NewTimer(ep.batchTimeout)
	timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if len(batch) > 0 {
				ep.processBatch(batch)
				batch = batch[:0]
			}
		default:
			item, shutdown := ep.eventQueue.Get()
			if shutdown {
				return
			}

			if event, ok := item.(WatchEvent); ok {
				batch = append(batch, event)

				if len(batch) == 1 {
					timer.Reset(ep.batchTimeout)
				}

				if len(batch) >= ep.batchSize {
					timer.Stop()
					ep.processBatch(batch)
					batch = batch[:0]
				}
			}

			ep.eventQueue.Done(item)
		}
	}
}

// processBatch processes a batch of events
func (ep *EventProcessor) processBatch(events []WatchEvent) {
	if err := ep.processor(events); err != nil {
		// Log error or handle as needed
		for _, event := range events {
			ep.eventQueue.AddRateLimited(event)
		}
	}
}

// Stop stops the event processor
func (ep *EventProcessor) Stop() {
	ep.eventQueue.ShutDown()
}

// ConnectionPool manages a pool of Kubernetes clients for different contexts
type ConnectionPool struct {
	clients map[string]*Client
	mu      sync.RWMutex
	maxSize int
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int) *ConnectionPool {
	return &ConnectionPool{
		clients: make(map[string]*Client),
		maxSize: maxSize,
	}
}

// GetClient gets or creates a client for a context
func (cp *ConnectionPool) GetClient(contextName, kubeconfig string) (*Client, error) {
	cp.mu.RLock()
	if client, exists := cp.clients[contextName]; exists {
		cp.mu.RUnlock()
		return client, nil
	}
	cp.mu.RUnlock()

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := cp.clients[contextName]; exists {
		return client, nil
	}

	// Check pool size
	if len(cp.clients) >= cp.maxSize {
		return nil, fmt.Errorf("connection pool is full (max size: %d)", cp.maxSize)
	}

	// Create new client
	opts := &ClientOptions{
		Context: contextName,
	}

	client, err := NewClientWithOptions(kubeconfig, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for context %s: %w", contextName, err)
	}

	cp.clients[contextName] = client
	return client, nil
}

// RemoveClient removes a client from the pool
func (cp *ConnectionPool) RemoveClient(contextName string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	delete(cp.clients, contextName)
}

// Clear removes all clients from the pool
func (cp *ConnectionPool) Clear() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.clients = make(map[string]*Client)
}

// Size returns the current pool size
func (cp *ConnectionPool) Size() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return len(cp.clients)
}

// HealthChecker monitors the health of Kubernetes connections
type HealthChecker struct {
	pool     *ConnectionPool
	interval time.Duration
	timeout  time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(pool *ConnectionPool, interval, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		pool:     pool,
		interval: interval,
		timeout:  timeout,
	}
}

// Start starts the health checking routine
func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkHealth(ctx)
		}
	}
}

// checkHealth checks the health of all clients in the pool
func (hc *HealthChecker) checkHealth(ctx context.Context) {
	hc.pool.mu.RLock()
	clients := make(map[string]*Client)
	for k, v := range hc.pool.clients {
		clients[k] = v
	}
	hc.pool.mu.RUnlock()

	for contextName, client := range clients {
		go func(name string, c *Client) {
			healthCtx, cancel := context.WithTimeout(ctx, hc.timeout)
			defer cancel()

			// Simple health check - try to list namespaces
			_, err := c.ListNamespaces(healthCtx)
			if err != nil {
				// Remove unhealthy client from pool
				hc.pool.RemoveClient(name)
			}
		}(contextName, client)
	}
}
