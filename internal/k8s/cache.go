package k8s

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// CacheEntry represents a cached resource with metadata
type CacheEntry struct {
	Data      interface{}
	Timestamp time.Time
	Version   string // ResourceVersion for optimistic concurrency
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	Hits         int64
	Misses       int64
	Evictions    int64
	TotalEntries int64
	mu           sync.RWMutex
}

// GetHitRatio returns the cache hit ratio
func (m *CacheMetrics) GetHitRatio() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.Hits + m.Misses
	if total == 0 {
		return 0
	}
	return float64(m.Hits) / float64(total)
}

// RecordHit increments hit counter
func (m *CacheMetrics) RecordHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Hits++
}

// RecordMiss increments miss counter
func (m *CacheMetrics) RecordMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Misses++
}

// RecordEviction increments eviction counter
func (m *CacheMetrics) RecordEviction() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Evictions++
}

// SetTotalEntries sets the total entries count
func (m *CacheMetrics) SetTotalEntries(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalEntries = count
}

// ResourceCache provides bounded caching for different resource types
type ResourceCache struct {
	pods         map[string]*CacheEntry
	deployments  map[string]*CacheEntry
	services     map[string]*CacheEntry
	configmaps   map[string]*CacheEntry
	secrets      map[string]*CacheEntry
	statefulsets map[string]*CacheEntry
	ingresses    map[string]*CacheEntry

	maxSize int
	ttl     time.Duration
	metrics *CacheMetrics
	mu      sync.RWMutex

	// LRU tracking
	accessOrder []string
	accessMap   map[string]time.Time
}

// NewResourceCache creates a new bounded resource cache
func NewResourceCache(maxSize int, ttl time.Duration) *ResourceCache {
	return &ResourceCache{
		pods:         make(map[string]*CacheEntry),
		deployments:  make(map[string]*CacheEntry),
		services:     make(map[string]*CacheEntry),
		configmaps:   make(map[string]*CacheEntry),
		secrets:      make(map[string]*CacheEntry),
		statefulsets: make(map[string]*CacheEntry),
		ingresses:    make(map[string]*CacheEntry),
		maxSize:      maxSize,
		ttl:          ttl,
		metrics:      &CacheMetrics{},
		accessOrder:  make([]string, 0),
		accessMap:    make(map[string]time.Time),
	}
}

// generateKey creates a cache key for a resource
func (c *ResourceCache) generateKey(resourceType, namespace, name string) string {
	return fmt.Sprintf("%s:%s:%s", resourceType, namespace, name)
}

// isExpired checks if a cache entry has expired
func (c *ResourceCache) isExpired(entry *CacheEntry) bool {
	return time.Since(entry.Timestamp) > c.ttl
}

// evictLRU removes the least recently used entry
func (c *ResourceCache) evictLRU() {
	if len(c.accessOrder) == 0 {
		return
	}

	// Find the oldest accessed key
	oldestKey := c.accessOrder[0]
	oldestTime := c.accessMap[oldestKey]

	for _, key := range c.accessOrder {
		if accessTime, exists := c.accessMap[key]; exists && accessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = accessTime
		}
	}

	// Remove from all caches
	c.removeFromAllCaches(oldestKey)
	c.metrics.RecordEviction()
}

// removeFromAllCaches removes a key from all cache maps
func (c *ResourceCache) removeFromAllCaches(key string) {
	delete(c.pods, key)
	delete(c.deployments, key)
	delete(c.services, key)
	delete(c.configmaps, key)
	delete(c.secrets, key)
	delete(c.statefulsets, key)
	delete(c.ingresses, key)

	// Remove from access tracking
	delete(c.accessMap, key)
	for i, k := range c.accessOrder {
		if k == key {
			c.accessOrder = append(c.accessOrder[:i], c.accessOrder[i+1:]...)
			break
		}
	}
}

// updateAccess updates the access time for LRU tracking
func (c *ResourceCache) updateAccess(key string) {
	now := time.Now()
	c.accessMap[key] = now

	// Remove from current position and add to end
	for i, k := range c.accessOrder {
		if k == key {
			c.accessOrder = append(c.accessOrder[:i], c.accessOrder[i+1:]...)
			break
		}
	}
	c.accessOrder = append(c.accessOrder, key)
}

// ensureCapacity ensures cache doesn't exceed max size
func (c *ResourceCache) ensureCapacity() {
	totalEntries := len(c.pods) + len(c.deployments) + len(c.services) +
		len(c.configmaps) + len(c.secrets) + len(c.statefulsets) + len(c.ingresses)

	for totalEntries >= c.maxSize {
		c.evictLRU()
		totalEntries--
	}

	c.metrics.SetTotalEntries(int64(totalEntries))
}

// GetPods retrieves cached pods or returns nil if not found/expired
func (c *ResourceCache) GetPods(namespace string) ([]v1.Pod, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey("pods", namespace, "*")
	entry, exists := c.pods[key]

	if !exists || c.isExpired(entry) {
		c.metrics.RecordMiss()
		return nil, false
	}

	c.updateAccess(key)
	c.metrics.RecordHit()

	if pods, ok := entry.Data.([]v1.Pod); ok {
		return pods, true
	}

	return nil, false
}

// SetPods caches pods for a namespace
func (c *ResourceCache) SetPods(namespace string, pods []v1.Pod, resourceVersion string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ensureCapacity()

	key := c.generateKey("pods", namespace, "*")
	c.pods[key] = &CacheEntry{
		Data:      pods,
		Timestamp: time.Now(),
		Version:   resourceVersion,
	}

	c.updateAccess(key)
}

// GetDeployments retrieves cached deployments
func (c *ResourceCache) GetDeployments(namespace string) ([]appsv1.Deployment, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey("deployments", namespace, "*")
	entry, exists := c.deployments[key]

	if !exists || c.isExpired(entry) {
		c.metrics.RecordMiss()
		return nil, false
	}

	c.updateAccess(key)
	c.metrics.RecordHit()

	if deployments, ok := entry.Data.([]appsv1.Deployment); ok {
		return deployments, true
	}

	return nil, false
}

// SetDeployments caches deployments for a namespace
func (c *ResourceCache) SetDeployments(namespace string, deployments []appsv1.Deployment, resourceVersion string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ensureCapacity()

	key := c.generateKey("deployments", namespace, "*")
	c.deployments[key] = &CacheEntry{
		Data:      deployments,
		Timestamp: time.Now(),
		Version:   resourceVersion,
	}

	c.updateAccess(key)
}

// GetServices retrieves cached services
func (c *ResourceCache) GetServices(namespace string) ([]v1.Service, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey("services", namespace, "*")
	entry, exists := c.services[key]

	if !exists || c.isExpired(entry) {
		c.metrics.RecordMiss()
		return nil, false
	}

	c.updateAccess(key)
	c.metrics.RecordHit()

	if services, ok := entry.Data.([]v1.Service); ok {
		return services, true
	}

	return nil, false
}

// SetServices caches services for a namespace
func (c *ResourceCache) SetServices(namespace string, services []v1.Service, resourceVersion string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ensureCapacity()

	key := c.generateKey("services", namespace, "*")
	c.services[key] = &CacheEntry{
		Data:      services,
		Timestamp: time.Now(),
		Version:   resourceVersion,
	}

	c.updateAccess(key)
}

// GetConfigMaps retrieves cached configmaps
func (c *ResourceCache) GetConfigMaps(namespace string) ([]v1.ConfigMap, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey("configmaps", namespace, "*")
	entry, exists := c.configmaps[key]

	if !exists || c.isExpired(entry) {
		c.metrics.RecordMiss()
		return nil, false
	}

	c.updateAccess(key)
	c.metrics.RecordHit()

	if configmaps, ok := entry.Data.([]v1.ConfigMap); ok {
		return configmaps, true
	}

	return nil, false
}

// SetConfigMaps caches configmaps for a namespace
func (c *ResourceCache) SetConfigMaps(namespace string, configmaps []v1.ConfigMap, resourceVersion string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ensureCapacity()

	key := c.generateKey("configmaps", namespace, "*")
	c.configmaps[key] = &CacheEntry{
		Data:      configmaps,
		Timestamp: time.Now(),
		Version:   resourceVersion,
	}

	c.updateAccess(key)
}

// GetSecrets retrieves cached secrets
func (c *ResourceCache) GetSecrets(namespace string) ([]v1.Secret, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey("secrets", namespace, "*")
	entry, exists := c.secrets[key]

	if !exists || c.isExpired(entry) {
		c.metrics.RecordMiss()
		return nil, false
	}

	c.updateAccess(key)
	c.metrics.RecordHit()

	if secrets, ok := entry.Data.([]v1.Secret); ok {
		return secrets, true
	}

	return nil, false
}

// SetSecrets caches secrets for a namespace
func (c *ResourceCache) SetSecrets(namespace string, secrets []v1.Secret, resourceVersion string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ensureCapacity()

	key := c.generateKey("secrets", namespace, "*")
	c.secrets[key] = &CacheEntry{
		Data:      secrets,
		Timestamp: time.Now(),
		Version:   resourceVersion,
	}

	c.updateAccess(key)
}

// GetStatefulSets retrieves cached statefulsets
func (c *ResourceCache) GetStatefulSets(namespace string) ([]appsv1.StatefulSet, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey("statefulsets", namespace, "*")
	entry, exists := c.statefulsets[key]

	if !exists || c.isExpired(entry) {
		c.metrics.RecordMiss()
		return nil, false
	}

	c.updateAccess(key)
	c.metrics.RecordHit()

	if statefulsets, ok := entry.Data.([]appsv1.StatefulSet); ok {
		return statefulsets, true
	}

	return nil, false
}

// SetStatefulSets caches statefulsets for a namespace
func (c *ResourceCache) SetStatefulSets(namespace string, statefulsets []appsv1.StatefulSet, resourceVersion string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ensureCapacity()

	key := c.generateKey("statefulsets", namespace, "*")
	c.statefulsets[key] = &CacheEntry{
		Data:      statefulsets,
		Timestamp: time.Now(),
		Version:   resourceVersion,
	}

	c.updateAccess(key)
}

// GetIngresses retrieves cached ingresses
func (c *ResourceCache) GetIngresses(namespace string) ([]networkingv1.Ingress, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey("ingresses", namespace, "*")
	entry, exists := c.ingresses[key]

	if !exists || c.isExpired(entry) {
		c.metrics.RecordMiss()
		return nil, false
	}

	c.updateAccess(key)
	c.metrics.RecordHit()

	if ingresses, ok := entry.Data.([]networkingv1.Ingress); ok {
		return ingresses, true
	}

	return nil, false
}

// SetIngresses caches ingresses for a namespace
func (c *ResourceCache) SetIngresses(namespace string, ingresses []networkingv1.Ingress, resourceVersion string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ensureCapacity()

	key := c.generateKey("ingresses", namespace, "*")
	c.ingresses[key] = &CacheEntry{
		Data:      ingresses,
		Timestamp: time.Now(),
		Version:   resourceVersion,
	}

	c.updateAccess(key)
}

// InvalidateNamespace removes all cached entries for a namespace
func (c *ResourceCache) InvalidateNamespace(namespace string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove entries for this namespace
	keysToRemove := make([]string, 0)

	for key := range c.pods {
		if c.keyMatchesNamespace(key, namespace) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key := range c.deployments {
		if c.keyMatchesNamespace(key, namespace) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key := range c.services {
		if c.keyMatchesNamespace(key, namespace) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key := range c.configmaps {
		if c.keyMatchesNamespace(key, namespace) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key := range c.secrets {
		if c.keyMatchesNamespace(key, namespace) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key := range c.statefulsets {
		if c.keyMatchesNamespace(key, namespace) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key := range c.ingresses {
		if c.keyMatchesNamespace(key, namespace) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for _, key := range keysToRemove {
		c.removeFromAllCaches(key)
	}
}

// keyMatchesNamespace checks if a cache key matches a namespace
func (c *ResourceCache) keyMatchesNamespace(key, namespace string) bool {
	parts := strings.Split(key, ":")
	if len(parts) >= 2 {
		return parts[1] == namespace
	}
	return false
}

// Clear removes all cached entries
func (c *ResourceCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pods = make(map[string]*CacheEntry)
	c.deployments = make(map[string]*CacheEntry)
	c.services = make(map[string]*CacheEntry)
	c.configmaps = make(map[string]*CacheEntry)
	c.secrets = make(map[string]*CacheEntry)
	c.statefulsets = make(map[string]*CacheEntry)
	c.ingresses = make(map[string]*CacheEntry)

	c.accessOrder = make([]string, 0)
	c.accessMap = make(map[string]time.Time)

	c.metrics = &CacheMetrics{}
}

// GetMetrics returns cache performance metrics
func (c *ResourceCache) GetMetrics() *CacheMetrics {
	return c.metrics
}

// CleanupExpired removes expired entries from the cache
func (c *ResourceCache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	keysToRemove := make([]string, 0)

	// Check all cache types for expired entries
	for key, entry := range c.pods {
		if now.Sub(entry.Timestamp) > c.ttl {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key, entry := range c.deployments {
		if now.Sub(entry.Timestamp) > c.ttl {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key, entry := range c.services {
		if now.Sub(entry.Timestamp) > c.ttl {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key, entry := range c.configmaps {
		if now.Sub(entry.Timestamp) > c.ttl {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key, entry := range c.secrets {
		if now.Sub(entry.Timestamp) > c.ttl {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key, entry := range c.statefulsets {
		if now.Sub(entry.Timestamp) > c.ttl {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for key, entry := range c.ingresses {
		if now.Sub(entry.Timestamp) > c.ttl {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for _, key := range keysToRemove {
		c.removeFromAllCaches(key)
		c.metrics.RecordEviction()
	}
}

// StartCleanupRoutine starts a background goroutine to clean up expired entries
func (c *ResourceCache) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.CleanupExpired()
			}
		}
	}()
}
