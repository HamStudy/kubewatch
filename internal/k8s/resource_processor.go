package k8s

import (
	"context"
	"fmt"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

// ResourceFilter provides smart filtering at the K8s level
type ResourceFilter struct {
	LabelSelector string
	FieldSelector string
	Namespace     string
	Limit         int64
	Continue      string
}

// BatchProcessor handles batch processing of resource updates
type BatchProcessor struct {
	batchSize    int
	flushTimeout time.Duration
	processor    func([]interface{}) error
	buffer       []interface{}
	mu           sync.Mutex
	timer        *time.Timer
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, flushTimeout time.Duration, processor func([]interface{}) error) *BatchProcessor {
	return &BatchProcessor{
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
		processor:    processor,
		buffer:       make([]interface{}, 0, batchSize),
	}
}

// Add adds an item to the batch
func (bp *BatchProcessor) Add(item interface{}) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.buffer = append(bp.buffer, item)

	// Start timer on first item
	if len(bp.buffer) == 1 {
		bp.timer = time.AfterFunc(bp.flushTimeout, bp.flush)
	}

	// Flush if batch is full
	if len(bp.buffer) >= bp.batchSize {
		return bp.flushLocked()
	}

	return nil
}

// flush flushes the current batch
func (bp *BatchProcessor) flush() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.flushLocked()
}

// flushLocked flushes the current batch (must be called with lock held)
func (bp *BatchProcessor) flushLocked() error {
	if len(bp.buffer) == 0 {
		return nil
	}

	// Stop timer if running
	if bp.timer != nil {
		bp.timer.Stop()
		bp.timer = nil
	}

	// Process batch
	batch := make([]interface{}, len(bp.buffer))
	copy(batch, bp.buffer)
	bp.buffer = bp.buffer[:0]

	return bp.processor(batch)
}

// Flush manually flushes any pending items
func (bp *BatchProcessor) Flush() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.flushLocked()
}

// OptimizedResourceClient provides optimized resource operations
type OptimizedResourceClient struct {
	client         *Client
	cache          *ResourceCache
	transformer    ResourceTransformer
	batchProcessor *BatchProcessor
}

// ResourceTransformer transforms raw K8s resources into optimized formats
type ResourceTransformer interface {
	TransformPods(pods []v1.Pod) ([]interface{}, error)
	TransformDeployments(deployments []appsv1.Deployment) ([]interface{}, error)
	TransformServices(services []v1.Service) ([]interface{}, error)
	TransformConfigMaps(configmaps []v1.ConfigMap) ([]interface{}, error)
	TransformSecrets(secrets []v1.Secret) ([]interface{}, error)
	TransformStatefulSets(statefulsets []appsv1.StatefulSet) ([]interface{}, error)
	TransformIngresses(ingresses []networkingv1.Ingress) ([]interface{}, error)
}

// DefaultResourceTransformer provides default transformations
type DefaultResourceTransformer struct{}

// TransformPods transforms pods into a lightweight format
func (t *DefaultResourceTransformer) TransformPods(pods []v1.Pod) ([]interface{}, error) {
	result := make([]interface{}, len(pods))
	for i, pod := range pods {
		result[i] = map[string]interface{}{
			"name":      pod.Name,
			"namespace": pod.Namespace,
			"status":    string(pod.Status.Phase),
			"ready":     isPodReady(&pod),
			"restarts":  getPodRestartCount(&pod),
			"age":       time.Since(pod.CreationTimestamp.Time),
			"node":      pod.Spec.NodeName,
			"uid":       string(pod.UID),
		}
	}
	return result, nil
}

// TransformDeployments transforms deployments
func (t *DefaultResourceTransformer) TransformDeployments(deployments []appsv1.Deployment) ([]interface{}, error) {
	result := make([]interface{}, len(deployments))
	for i, deployment := range deployments {
		result[i] = map[string]interface{}{
			"name":      deployment.Name,
			"namespace": deployment.Namespace,
			"ready":     fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, deployment.Status.Replicas),
			"upToDate":  deployment.Status.UpdatedReplicas,
			"available": deployment.Status.AvailableReplicas,
			"age":       time.Since(deployment.CreationTimestamp.Time),
			"uid":       string(deployment.UID),
		}
	}
	return result, nil
}

// TransformServices transforms services
func (t *DefaultResourceTransformer) TransformServices(services []v1.Service) ([]interface{}, error) {
	result := make([]interface{}, len(services))
	for i, service := range services {
		result[i] = map[string]interface{}{
			"name":      service.Name,
			"namespace": service.Namespace,
			"type":      string(service.Spec.Type),
			"clusterIP": service.Spec.ClusterIP,
			"ports":     getServicePorts(&service),
			"age":       time.Since(service.CreationTimestamp.Time),
			"uid":       string(service.UID),
		}
	}
	return result, nil
}

// TransformConfigMaps transforms configmaps
func (t *DefaultResourceTransformer) TransformConfigMaps(configmaps []v1.ConfigMap) ([]interface{}, error) {
	result := make([]interface{}, len(configmaps))
	for i, cm := range configmaps {
		result[i] = map[string]interface{}{
			"name":      cm.Name,
			"namespace": cm.Namespace,
			"data":      len(cm.Data),
			"age":       time.Since(cm.CreationTimestamp.Time),
			"uid":       string(cm.UID),
		}
	}
	return result, nil
}

// TransformSecrets transforms secrets
func (t *DefaultResourceTransformer) TransformSecrets(secrets []v1.Secret) ([]interface{}, error) {
	result := make([]interface{}, len(secrets))
	for i, secret := range secrets {
		result[i] = map[string]interface{}{
			"name":      secret.Name,
			"namespace": secret.Namespace,
			"type":      string(secret.Type),
			"data":      len(secret.Data),
			"age":       time.Since(secret.CreationTimestamp.Time),
			"uid":       string(secret.UID),
		}
	}
	return result, nil
}

// TransformStatefulSets transforms statefulsets
func (t *DefaultResourceTransformer) TransformStatefulSets(statefulsets []appsv1.StatefulSet) ([]interface{}, error) {
	result := make([]interface{}, len(statefulsets))
	for i, sts := range statefulsets {
		result[i] = map[string]interface{}{
			"name":      sts.Name,
			"namespace": sts.Namespace,
			"ready":     fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, sts.Status.Replicas),
			"age":       time.Since(sts.CreationTimestamp.Time),
			"uid":       string(sts.UID),
		}
	}
	return result, nil
}

// TransformIngresses transforms ingresses
func (t *DefaultResourceTransformer) TransformIngresses(ingresses []networkingv1.Ingress) ([]interface{}, error) {
	result := make([]interface{}, len(ingresses))
	for i, ingress := range ingresses {
		result[i] = map[string]interface{}{
			"name":      ingress.Name,
			"namespace": ingress.Namespace,
			"hosts":     getIngressHosts(&ingress),
			"address":   getIngressAddress(&ingress),
			"age":       time.Since(ingress.CreationTimestamp.Time),
			"uid":       string(ingress.UID),
		}
	}
	return result, nil
}

// Helper functions for transformations

func isPodReady(pod *v1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodReady {
			return condition.Status == v1.ConditionTrue
		}
	}
	return false
}

func getPodRestartCount(pod *v1.Pod) int32 {
	var restarts int32
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restarts += containerStatus.RestartCount
	}
	return restarts
}

func getServicePorts(service *v1.Service) []string {
	ports := make([]string, len(service.Spec.Ports))
	for i, port := range service.Spec.Ports {
		if port.Name != "" {
			ports[i] = fmt.Sprintf("%s:%d/%s", port.Name, port.Port, port.Protocol)
		} else {
			ports[i] = fmt.Sprintf("%d/%s", port.Port, port.Protocol)
		}
	}
	return ports
}

func getIngressHosts(ingress *networkingv1.Ingress) []string {
	hosts := make([]string, 0)
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	return hosts
}

func getIngressAddress(ingress *networkingv1.Ingress) string {
	if len(ingress.Status.LoadBalancer.Ingress) > 0 {
		if ingress.Status.LoadBalancer.Ingress[0].IP != "" {
			return ingress.Status.LoadBalancer.Ingress[0].IP
		}
		if ingress.Status.LoadBalancer.Ingress[0].Hostname != "" {
			return ingress.Status.LoadBalancer.Ingress[0].Hostname
		}
	}
	return ""
}

// NewOptimizedResourceClient creates a new optimized resource client
func NewOptimizedResourceClient(client *Client, cache *ResourceCache, transformer ResourceTransformer) *OptimizedResourceClient {
	if transformer == nil {
		transformer = &DefaultResourceTransformer{}
	}

	return &OptimizedResourceClient{
		client:      client,
		cache:       cache,
		transformer: transformer,
	}
}

// ListPodsOptimized lists pods with smart filtering and caching
func (orc *OptimizedResourceClient) ListPodsOptimized(ctx context.Context, namespace string, filter *ResourceFilter) ([]interface{}, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("pods:%s", namespace)
	if filter != nil {
		cacheKey = fmt.Sprintf("pods:%s:%s:%s", namespace, filter.LabelSelector, filter.FieldSelector)
	}

	if cached, found := orc.cache.GetPods(cacheKey); found {
		return orc.transformer.TransformPods(cached)
	}

	// Build list options
	listOpts := metav1.ListOptions{}
	if filter != nil {
		listOpts.LabelSelector = filter.LabelSelector
		listOpts.FieldSelector = filter.FieldSelector
		listOpts.Limit = filter.Limit
		listOpts.Continue = filter.Continue
	}

	// Fetch from API
	podList, err := orc.client.clientset.CoreV1().Pods(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Cache results
	orc.cache.SetPods(cacheKey, podList.Items, podList.ResourceVersion)

	// Transform and return
	return orc.transformer.TransformPods(podList.Items)
}

// ListDeploymentsOptimized lists deployments with optimizations
func (orc *OptimizedResourceClient) ListDeploymentsOptimized(ctx context.Context, namespace string, filter *ResourceFilter) ([]interface{}, error) {
	cacheKey := fmt.Sprintf("deployments:%s", namespace)
	if filter != nil {
		cacheKey = fmt.Sprintf("deployments:%s:%s:%s", namespace, filter.LabelSelector, filter.FieldSelector)
	}

	if cached, found := orc.cache.GetDeployments(cacheKey); found {
		return orc.transformer.TransformDeployments(cached)
	}

	listOpts := metav1.ListOptions{}
	if filter != nil {
		listOpts.LabelSelector = filter.LabelSelector
		listOpts.FieldSelector = filter.FieldSelector
		listOpts.Limit = filter.Limit
		listOpts.Continue = filter.Continue
	}

	deploymentList, err := orc.client.clientset.AppsV1().Deployments(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	orc.cache.SetDeployments(cacheKey, deploymentList.Items, deploymentList.ResourceVersion)

	return orc.transformer.TransformDeployments(deploymentList.Items)
}

// ListServicesOptimized lists services with optimizations
func (orc *OptimizedResourceClient) ListServicesOptimized(ctx context.Context, namespace string, filter *ResourceFilter) ([]interface{}, error) {
	cacheKey := fmt.Sprintf("services:%s", namespace)
	if filter != nil {
		cacheKey = fmt.Sprintf("services:%s:%s:%s", namespace, filter.LabelSelector, filter.FieldSelector)
	}

	if cached, found := orc.cache.GetServices(cacheKey); found {
		return orc.transformer.TransformServices(cached)
	}

	listOpts := metav1.ListOptions{}
	if filter != nil {
		listOpts.LabelSelector = filter.LabelSelector
		listOpts.FieldSelector = filter.FieldSelector
		listOpts.Limit = filter.Limit
		listOpts.Continue = filter.Continue
	}

	serviceList, err := orc.client.clientset.CoreV1().Services(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	orc.cache.SetServices(cacheKey, serviceList.Items, serviceList.ResourceVersion)

	return orc.transformer.TransformServices(serviceList.Items)
}

// SmartResourceFilter provides intelligent filtering based on resource type
type SmartResourceFilter struct {
	client *Client
}

// NewSmartResourceFilter creates a new smart resource filter
func NewSmartResourceFilter(client *Client) *SmartResourceFilter {
	return &SmartResourceFilter{client: client}
}

// FilterPodsForDeployment returns a filter for pods belonging to a deployment
func (srf *SmartResourceFilter) FilterPodsForDeployment(deploymentName string) *ResourceFilter {
	return &ResourceFilter{
		LabelSelector: fmt.Sprintf("app=%s", deploymentName),
	}
}

// FilterPodsForStatefulSet returns a filter for pods belonging to a statefulset
func (srf *SmartResourceFilter) FilterPodsForStatefulSet(statefulSetName string) *ResourceFilter {
	return &ResourceFilter{
		LabelSelector: fmt.Sprintf("app=%s", statefulSetName),
	}
}

// FilterPodsByNode returns a filter for pods on a specific node
func (srf *SmartResourceFilter) FilterPodsByNode(nodeName string) *ResourceFilter {
	return &ResourceFilter{
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", nodeName).String(),
	}
}

// FilterPodsByPhase returns a filter for pods in a specific phase
func (srf *SmartResourceFilter) FilterPodsByPhase(phase v1.PodPhase) *ResourceFilter {
	return &ResourceFilter{
		FieldSelector: fields.OneTermEqualSelector("status.phase", string(phase)).String(),
	}
}

// FilterServicesByType returns a filter for services of a specific type
func (srf *SmartResourceFilter) FilterServicesByType(serviceType v1.ServiceType) *ResourceFilter {
	return &ResourceFilter{
		FieldSelector: fields.OneTermEqualSelector("spec.type", string(serviceType)).String(),
	}
}

// FilterByLabels returns a filter for resources with specific labels
func (srf *SmartResourceFilter) FilterByLabels(labelMap map[string]string) *ResourceFilter {
	selector := labels.SelectorFromSet(labelMap)
	return &ResourceFilter{
		LabelSelector: selector.String(),
	}
}

// ResourceTypeOptimizer provides resource-specific optimizations
type ResourceTypeOptimizer struct {
	client *Client
}

// NewResourceTypeOptimizer creates a new resource type optimizer
func NewResourceTypeOptimizer(client *Client) *ResourceTypeOptimizer {
	return &ResourceTypeOptimizer{client: client}
}

// OptimizeForLargeDatasets applies optimizations for handling large datasets
func (rto *ResourceTypeOptimizer) OptimizeForLargeDatasets(resourceType string, namespace string) *ResourceFilter {
	switch resourceType {
	case "pods":
		// For pods, limit to running pods first
		return &ResourceFilter{
			FieldSelector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning)).String(),
			Limit:         500, // Reasonable limit for UI
		}
	case "deployments":
		// For deployments, no special filtering needed as they're typically fewer
		return &ResourceFilter{
			Limit: 100,
		}
	case "services":
		// Services are typically few, no special optimization
		return &ResourceFilter{
			Limit: 200,
		}
	case "configmaps":
		// ConfigMaps can be numerous, limit them
		return &ResourceFilter{
			Limit: 300,
		}
	case "secrets":
		// Secrets can be numerous, limit them
		return &ResourceFilter{
			Limit: 300,
		}
	default:
		return &ResourceFilter{
			Limit: 100,
		}
	}
}

// OptimizeForRealtimeUpdates provides optimizations for real-time scenarios
func (rto *ResourceTypeOptimizer) OptimizeForRealtimeUpdates(resourceType string) *ResourceFilter {
	// For real-time updates, we want minimal data transfer
	return &ResourceFilter{
		Limit: 50, // Smaller batches for faster updates
	}
}

// PerformanceMonitor tracks resource processing performance
type PerformanceMonitor struct {
	metrics map[string]*OperationMetrics
	mu      sync.RWMutex
}

// OperationMetrics tracks metrics for a specific operation
type OperationMetrics struct {
	TotalRequests   int64
	TotalDuration   time.Duration
	AverageDuration time.Duration
	LastRequest     time.Time
	ErrorCount      int64
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: make(map[string]*OperationMetrics),
	}
}

// RecordOperation records metrics for an operation
func (pm *PerformanceMonitor) RecordOperation(operation string, duration time.Duration, err error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.metrics[operation] == nil {
		pm.metrics[operation] = &OperationMetrics{}
	}

	metrics := pm.metrics[operation]
	metrics.TotalRequests++
	metrics.TotalDuration += duration
	metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.TotalRequests)
	metrics.LastRequest = time.Now()

	if err != nil {
		metrics.ErrorCount++
	}
}

// GetMetrics returns metrics for an operation
func (pm *PerformanceMonitor) GetMetrics(operation string) *OperationMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if metrics, exists := pm.metrics[operation]; exists {
		// Return a copy to avoid race conditions
		return &OperationMetrics{
			TotalRequests:   metrics.TotalRequests,
			TotalDuration:   metrics.TotalDuration,
			AverageDuration: metrics.AverageDuration,
			LastRequest:     metrics.LastRequest,
			ErrorCount:      metrics.ErrorCount,
		}
	}

	return nil
}

// GetAllMetrics returns all recorded metrics
func (pm *PerformanceMonitor) GetAllMetrics() map[string]*OperationMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*OperationMetrics)
	for k, v := range pm.metrics {
		result[k] = &OperationMetrics{
			TotalRequests:   v.TotalRequests,
			TotalDuration:   v.TotalDuration,
			AverageDuration: v.AverageDuration,
			LastRequest:     v.LastRequest,
			ErrorCount:      v.ErrorCount,
		}
	}

	return result
}
