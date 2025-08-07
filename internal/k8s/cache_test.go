package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResourceCache_BasicOperations(t *testing.T) {
	cache := NewResourceCache(100, 5*time.Minute)

	// Test pod caching
	pods := []v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-1",
				Namespace: "default",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-2",
				Namespace: "default",
			},
		},
	}

	// Set pods in cache
	cache.SetPods("default", pods, "12345")

	// Get pods from cache
	cachedPods, found := cache.GetPods("default")
	if !found {
		t.Error("Expected to find cached pods")
	}

	if len(cachedPods) != 2 {
		t.Errorf("Expected 2 cached pods, got %d", len(cachedPods))
	}

	if cachedPods[0].Name != "test-pod-1" {
		t.Errorf("Expected first pod name to be 'test-pod-1', got %s", cachedPods[0].Name)
	}
}

func TestResourceCache_Expiration(t *testing.T) {
	cache := NewResourceCache(100, 100*time.Millisecond)

	pods := []v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		},
	}

	// Set pods in cache
	cache.SetPods("default", pods, "12345")

	// Should find immediately
	_, found := cache.GetPods("default")
	if !found {
		t.Error("Expected to find cached pods immediately")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not find after expiration
	_, found = cache.GetPods("default")
	if found {
		t.Error("Expected cached pods to be expired")
	}
}

func TestResourceCache_LRUEviction(t *testing.T) {
	cache := NewResourceCache(2, 5*time.Minute) // Small cache size

	// Add first entry
	pods1 := []v1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "ns1"}}}
	cache.SetPods("ns1", pods1, "1")

	// Add second entry
	pods2 := []v1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "ns2"}}}
	cache.SetPods("ns2", pods2, "2")

	// Add third entry (should evict first)
	pods3 := []v1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod3", Namespace: "ns3"}}}
	cache.SetPods("ns3", pods3, "3")

	// First entry should be evicted
	_, found := cache.GetPods("ns1")
	if found {
		t.Error("Expected first entry to be evicted")
	}

	// Second and third should still be there
	_, found = cache.GetPods("ns2")
	if !found {
		t.Error("Expected second entry to still be cached")
	}

	_, found = cache.GetPods("ns3")
	if !found {
		t.Error("Expected third entry to still be cached")
	}
}

func TestResourceCache_Metrics(t *testing.T) {
	cache := NewResourceCache(100, 5*time.Minute)

	pods := []v1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default"}}}
	cache.SetPods("default", pods, "1")

	// Hit
	_, found := cache.GetPods("default")
	if !found {
		t.Error("Expected cache hit")
	}

	// Miss
	_, found = cache.GetPods("nonexistent")
	if found {
		t.Error("Expected cache miss")
	}

	metrics := cache.GetMetrics()
	if metrics.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", metrics.Hits)
	}

	if metrics.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics.Misses)
	}

	hitRatio := metrics.GetHitRatio()
	expectedRatio := 0.5
	if hitRatio != expectedRatio {
		t.Errorf("Expected hit ratio %.2f, got %.2f", expectedRatio, hitRatio)
	}
}

func TestResourceCache_InvalidateNamespace(t *testing.T) {
	cache := NewResourceCache(100, 5*time.Minute)

	// Add pods for different namespaces
	pods1 := []v1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "ns1"}}}
	pods2 := []v1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "ns2"}}}

	cache.SetPods("ns1", pods1, "1")
	cache.SetPods("ns2", pods2, "2")

	// Invalidate ns1
	cache.InvalidateNamespace("ns1")

	// ns1 should be gone
	_, found := cache.GetPods("ns1")
	if found {
		t.Error("Expected ns1 to be invalidated")
	}

	// ns2 should still be there
	_, found = cache.GetPods("ns2")
	if !found {
		t.Error("Expected ns2 to still be cached")
	}
}

func TestResourceCache_CleanupExpired(t *testing.T) {
	cache := NewResourceCache(100, 50*time.Millisecond)

	pods := []v1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default"}}}
	cache.SetPods("default", pods, "1")

	// Should be there initially
	_, found := cache.GetPods("default")
	if !found {
		t.Error("Expected to find cached pods")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup
	cache.CleanupExpired()

	// Should be gone after cleanup
	_, found = cache.GetPods("default")
	if found {
		t.Error("Expected expired entries to be cleaned up")
	}
}

func TestResourceCache_StartCleanupRoutine(t *testing.T) {
	cache := NewResourceCache(100, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cleanup routine
	cache.StartCleanupRoutine(ctx, 25*time.Millisecond)

	pods := []v1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default"}}}
	cache.SetPods("default", pods, "1")

	// Should be there initially
	_, found := cache.GetPods("default")
	if !found {
		t.Error("Expected to find cached pods")
	}

	// Wait for automatic cleanup
	time.Sleep(150 * time.Millisecond)

	// Should be gone after automatic cleanup
	_, found = cache.GetPods("default")
	if found {
		t.Error("Expected expired entries to be automatically cleaned up")
	}
}

func TestResourceCache_MultipleResourceTypes(t *testing.T) {
	cache := NewResourceCache(100, 5*time.Minute)

	// Test different resource types
	pods := []v1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default"}}}
	deployments := []appsv1.Deployment{{ObjectMeta: metav1.ObjectMeta{Name: "deploy", Namespace: "default"}}}
	services := []v1.Service{{ObjectMeta: metav1.ObjectMeta{Name: "svc", Namespace: "default"}}}

	cache.SetPods("default", pods, "1")
	cache.SetDeployments("default", deployments, "2")
	cache.SetServices("default", services, "3")

	// All should be retrievable
	cachedPods, found := cache.GetPods("default")
	if !found || len(cachedPods) != 1 {
		t.Error("Expected to find cached pods")
	}

	cachedDeployments, found := cache.GetDeployments("default")
	if !found || len(cachedDeployments) != 1 {
		t.Error("Expected to find cached deployments")
	}

	cachedServices, found := cache.GetServices("default")
	if !found || len(cachedServices) != 1 {
		t.Error("Expected to find cached services")
	}
}

func BenchmarkResourceCache_SetPods(b *testing.B) {
	cache := NewResourceCache(1000, 5*time.Minute)

	pods := make([]v1.Pod, 100)
	for i := 0; i < 100; i++ {
		pods[i] = v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.SetPods("default", pods, fmt.Sprintf("%d", i))
	}
}

func BenchmarkResourceCache_GetPods(b *testing.B) {
	cache := NewResourceCache(1000, 5*time.Minute)

	pods := make([]v1.Pod, 100)
	for i := 0; i < 100; i++ {
		pods[i] = v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
			},
		}
	}

	cache.SetPods("default", pods, "1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.GetPods("default")
	}
}

func BenchmarkResourceCache_ConcurrentAccess(b *testing.B) {
	cache := NewResourceCache(1000, 5*time.Minute)

	pods := make([]v1.Pod, 10)
	for i := 0; i < 10; i++ {
		pods[i] = v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
			},
		}
	}

	cache.SetPods("default", pods, "1")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cache.GetPods("default")
		}
	})
}
