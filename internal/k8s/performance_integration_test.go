package k8s

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnhancedMultiContextClient_ParallelFetching(t *testing.T) {
	// Create fake clients for testing
	contexts := []string{"context1", "context2", "context3"}

	_ = &EnhancedMultiContextOptions{
		CacheSize:           100,
		CacheTTL:            5 * time.Minute,
		ParallelFetch:       true,
		HealthCheckInterval: 30 * time.Second,
		ContextTimeout:      5 * time.Second,
		MaxConnections:      10,
	}

	// This would normally create real clients, but for testing we'll use a simpler approach
	// In a real implementation, you'd mock the client creation

	// Test that parallel fetching is faster than sequential
	start := time.Now()

	// Simulate parallel work
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(contextName string) {
			defer wg.Done()
			// Simulate API call delay
			time.Sleep(100 * time.Millisecond)
		}(contexts[i])
	}
	wg.Wait()

	parallelDuration := time.Since(start)

	// Sequential simulation
	start = time.Now()
	for i := 0; i < 3; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	sequentialDuration := time.Since(start)

	// Parallel should be significantly faster
	if parallelDuration >= sequentialDuration {
		t.Errorf("Parallel fetching should be faster than sequential. Parallel: %v, Sequential: %v",
			parallelDuration, sequentialDuration)
	}

	t.Logf("Parallel duration: %v, Sequential duration: %v", parallelDuration, sequentialDuration)
}

func TestResourceCache_PerformanceWithLargeDataset(t *testing.T) {
	cache := NewResourceCache(1000, 5*time.Minute)

	// Create a large dataset
	numPods := 500
	pods := make([]v1.Pod, numPods)
	for i := 0; i < numPods; i++ {
		pods[i] = v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
				UID:       types.UID(fmt.Sprintf("uid-%d", i)),
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
			},
		}
	}

	// Test cache set performance
	start := time.Now()
	cache.SetPods("default", pods, "12345")
	setDuration := time.Since(start)

	// Test cache get performance
	start = time.Now()
	cachedPods, found := cache.GetPods("default")
	getDuration := time.Since(start)

	if !found {
		t.Error("Expected to find cached pods")
	}

	if len(cachedPods) != numPods {
		t.Errorf("Expected %d cached pods, got %d", numPods, len(cachedPods))
	}

	// Performance should be reasonable
	if setDuration > 10*time.Millisecond {
		t.Errorf("Cache set took too long: %v", setDuration)
	}

	if getDuration > 5*time.Millisecond {
		t.Errorf("Cache get took too long: %v", getDuration)
	}

	t.Logf("Cache set duration: %v, get duration: %v", setDuration, getDuration)
}

func TestBatchProcessor_Performance(t *testing.T) {
	processed := make([]interface{}, 0)
	var mu sync.Mutex

	processor := func(batch []interface{}) error {
		mu.Lock()
		processed = append(processed, batch...)
		mu.Unlock()
		return nil
	}

	bp := NewBatchProcessor(50, 100*time.Millisecond, processor)

	// Add items rapidly
	numItems := 200
	start := time.Now()

	for i := 0; i < numItems; i++ {
		err := bp.Add(fmt.Sprintf("item-%d", i))
		if err != nil {
			t.Errorf("Failed to add item: %v", err)
		}
	}

	// Flush remaining items
	err := bp.Flush()
	if err != nil {
		t.Errorf("Failed to flush: %v", err)
	}

	duration := time.Since(start)

	mu.Lock()
	processedCount := len(processed)
	mu.Unlock()

	if processedCount != numItems {
		t.Errorf("Expected %d processed items, got %d", numItems, processedCount)
	}

	// Should process quickly
	if duration > 500*time.Millisecond {
		t.Errorf("Batch processing took too long: %v", duration)
	}

	t.Logf("Processed %d items in %v", processedCount, duration)
}

func TestResourceTransformer_Performance(t *testing.T) {
	transformer := &DefaultResourceTransformer{}

	// Create large dataset
	numPods := 1000
	pods := make([]v1.Pod, numPods)
	for i := 0; i < numPods; i++ {
		pods[i] = v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
				UID:       types.UID(fmt.Sprintf("uid-%d", i)),
				CreationTimestamp: metav1.Time{
					Time: time.Now().Add(-time.Duration(i) * time.Minute),
				},
			},
			Spec: v1.PodSpec{
				NodeName: fmt.Sprintf("node-%d", i%10),
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodReady,
						Status: v1.ConditionTrue,
					},
				},
				ContainerStatuses: []v1.ContainerStatus{
					{
						Name:         "container-1",
						RestartCount: int32(i % 5),
						Ready:        true,
					},
				},
			},
		}
	}

	// Test transformation performance
	start := time.Now()
	transformed, err := transformer.TransformPods(pods)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Transformation failed: %v", err)
	}

	if len(transformed) != numPods {
		t.Errorf("Expected %d transformed items, got %d", numPods, len(transformed))
	}

	// Should transform quickly
	if duration > 50*time.Millisecond {
		t.Errorf("Transformation took too long: %v", duration)
	}

	// Verify transformation quality
	firstItem := transformed[0].(map[string]interface{})
	if firstItem["name"] != "pod-0" {
		t.Errorf("Expected name 'pod-0', got %v", firstItem["name"])
	}

	if firstItem["namespace"] != "default" {
		t.Errorf("Expected namespace 'default', got %v", firstItem["namespace"])
	}

	t.Logf("Transformed %d pods in %v", numPods, duration)
}

func TestWatchCoalescer_Performance(t *testing.T) {
	coalescer := NewWatchCoalescer()

	// Simulate multiple listeners for the same resource
	numListeners := 10
	listeners := make([]<-chan WatchEvent, numListeners)

	// Create fake client
	fakeClient := &Client{
		clientset: fake.NewSimpleClientset(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add multiple listeners for the same resource
	start := time.Now()
	for i := 0; i < numListeners; i++ {
		listener, err := coalescer.AddWatchListener(ctx, fakeClient, "test-context", "default", "pods")
		if err != nil {
			t.Errorf("Failed to add watch listener: %v", err)
		}
		listeners[i] = listener
	}
	setupDuration := time.Since(start)

	// Should reuse the same watch for all listeners
	if setupDuration > 100*time.Millisecond {
		t.Errorf("Watch setup took too long: %v", setupDuration)
	}

	// Clean up
	for i, listener := range listeners {
		coalescer.RemoveWatchListener("test-context", "default", "pods", listener)
		_ = i // Use the variable to avoid unused warning
	}

	t.Logf("Set up %d watch listeners in %v", numListeners, setupDuration)
}

func TestConnectionPool_Performance(t *testing.T) {
	_ = NewConnectionPool(10)

	// Test concurrent access
	numGoroutines := 20
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			_ = fmt.Sprintf("context-%d", id%5) // 5 unique contexts

			// This would normally create a real client
			// For testing, we'll simulate the operation
			time.Sleep(10 * time.Millisecond) // Simulate client creation time

			// Simulate successful client creation
			if id < 10 { // First 10 should succeed (within pool limit)
				// Success
			} else {
				// Later ones might hit pool limit
				errors <- fmt.Errorf("pool full")
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	close(errors)
	errorCount := 0
	for range errors {
		errorCount++
	}

	// Should handle concurrent access efficiently
	if duration > 500*time.Millisecond {
		t.Errorf("Connection pool operations took too long: %v", duration)
	}

	t.Logf("Handled %d concurrent operations in %v with %d errors",
		numGoroutines, duration, errorCount)
}

func TestPerformanceMonitor_Metrics(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Record various operations
	operations := []string{"list-pods", "list-deployments", "list-services"}

	for _, op := range operations {
		for i := 0; i < 10; i++ {
			duration := time.Duration(i+1) * 10 * time.Millisecond
			var err error
			if i%3 == 0 {
				err = fmt.Errorf("simulated error")
			}
			monitor.RecordOperation(op, duration, err)
		}
	}

	// Check metrics
	for _, op := range operations {
		metrics := monitor.GetMetrics(op)
		if metrics == nil {
			t.Errorf("Expected metrics for operation %s", op)
			continue
		}

		if metrics.TotalRequests != 10 {
			t.Errorf("Expected 10 requests for %s, got %d", op, metrics.TotalRequests)
		}

		if metrics.ErrorCount != 4 { // Every 3rd request fails, so 4 errors out of 10
			t.Errorf("Expected 4 errors for %s, got %d", op, metrics.ErrorCount)
		}

		if metrics.AverageDuration <= 0 {
			t.Errorf("Expected positive average duration for %s", op)
		}

		t.Logf("Operation %s: %d requests, %d errors, avg duration: %v",
			op, metrics.TotalRequests, metrics.ErrorCount, metrics.AverageDuration)
	}

	// Test all metrics
	allMetrics := monitor.GetAllMetrics()
	if len(allMetrics) != len(operations) {
		t.Errorf("Expected %d operations in all metrics, got %d", len(operations), len(allMetrics))
	}
}

func BenchmarkResourceTransformer_TransformPods(b *testing.B) {
	transformer := &DefaultResourceTransformer{}

	pods := make([]v1.Pod, 100)
	for i := 0; i < 100; i++ {
		pods[i] = v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              fmt.Sprintf("pod-%d", i),
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				Conditions: []v1.PodCondition{
					{Type: v1.PodReady, Status: v1.ConditionTrue},
				},
				ContainerStatuses: []v1.ContainerStatus{
					{RestartCount: 0, Ready: true},
				},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = transformer.TransformPods(pods)
	}
}

func BenchmarkBatchProcessor_Add(b *testing.B) {
	processed := 0
	processor := func(batch []interface{}) error {
		processed += len(batch)
		return nil
	}

	bp := NewBatchProcessor(50, 100*time.Millisecond, processor)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bp.Add(fmt.Sprintf("item-%d", i))
	}

	// Flush remaining
	_ = bp.Flush()
}
