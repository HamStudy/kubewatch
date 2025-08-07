package performance

import (
	"sync"
	"testing"
	"time"
)

func TestViewportManagerCreation(t *testing.T) {
	vm := NewViewportManager(10, 2)

	if vm.viewportHeight != 10 {
		t.Errorf("Expected viewport height 10, got %d", vm.viewportHeight)
	}

	if vm.bufferSize != 2 {
		t.Errorf("Expected buffer size 2, got %d", vm.bufferSize)
	}

	if vm.renderCache == nil {
		t.Error("Expected render cache to be initialized")
	}
}

func TestViewportScrolling(t *testing.T) {
	vm := NewViewportManager(5, 1)
	vm.SetTotalItems(20)

	// Test initial state
	start, end := vm.GetVisibleRange()
	if start != 0 || end != 4 {
		t.Errorf("Expected initial visible range [0, 4], got [%d, %d]", start, end)
	}

	// Test scrolling to specific position
	vm.ScrollTo(10)
	start, end = vm.GetVisibleRange()
	if start != 10 || end != 14 {
		t.Errorf("Expected visible range [10, 14] after scroll, got [%d, %d]", start, end)
	}

	// Test scrolling by delta
	vm.ScrollBy(3)
	start, end = vm.GetVisibleRange()
	if start != 13 || end != 17 {
		t.Errorf("Expected visible range [13, 17] after scroll by 3, got [%d, %d]", start, end)
	}

	// Test scrolling beyond bounds
	vm.ScrollTo(100)
	start, end = vm.GetVisibleRange()
	if start != 19 || end != 19 {
		t.Errorf("Expected visible range [19, 19] after scroll beyond bounds, got [%d, %d]", start, end)
	}

	// Test scrolling to negative
	vm.ScrollTo(-5)
	start, end = vm.GetVisibleRange()
	if start != 0 || end != 4 {
		t.Errorf("Expected visible range [0, 4] after scroll to negative, got [%d, %d]", start, end)
	}
}

func TestViewportRenderRange(t *testing.T) {
	vm := NewViewportManager(5, 2)
	vm.SetTotalItems(20)

	// Test initial render range with buffer
	start, end := vm.GetRenderRange()
	if start != 0 || end != 6 {
		t.Errorf("Expected initial render range [0, 6], got [%d, %d]", start, end)
	}

	// Test render range after scrolling
	vm.ScrollTo(10)
	start, end = vm.GetRenderRange()
	if start != 8 || end != 16 {
		t.Errorf("Expected render range [8, 16] after scroll, got [%d, %d]", start, end)
	}

	// Test render range at end
	vm.ScrollTo(15)
	start, end = vm.GetRenderRange()
	if start != 13 || end != 19 {
		t.Errorf("Expected render range [13, 19] at end, got [%d, %d]", start, end)
	}
}

func TestViewportItemVisibility(t *testing.T) {
	vm := NewViewportManager(5, 1)
	vm.SetTotalItems(20)

	// Test initial visibility
	if !vm.IsItemVisible(0) {
		t.Error("Expected item 0 to be visible initially")
	}
	if !vm.IsItemVisible(4) {
		t.Error("Expected item 4 to be visible initially")
	}
	if vm.IsItemVisible(5) {
		t.Error("Expected item 5 to not be visible initially")
	}

	// Test render visibility
	if !vm.ShouldRenderItem(0) {
		t.Error("Expected item 0 to be rendered initially")
	}
	if !vm.ShouldRenderItem(5) {
		t.Error("Expected item 5 to be rendered (in buffer)")
	}
	if vm.ShouldRenderItem(7) {
		t.Error("Expected item 7 to not be rendered initially")
	}
}

func TestViewportCache(t *testing.T) {
	vm := NewViewportManager(5, 1)

	// Test caching
	vm.CacheRenderedItem(0, "rendered content 0")
	vm.CacheRenderedItem(1, "rendered content 1")

	// Test retrieval
	content, exists := vm.GetCachedItem(0)
	if !exists || content != "rendered content 0" {
		t.Errorf("Expected cached content 'rendered content 0', got '%s' (exists: %v)", content, exists)
	}

	// Test non-existent item
	_, exists = vm.GetCachedItem(99)
	if exists {
		t.Error("Expected non-existent item to not be cached")
	}

	// Test cache clearing
	vm.ClearCache()
	_, exists = vm.GetCachedItem(0)
	if exists {
		t.Error("Expected cache to be cleared")
	}
}

func TestViewportCallbacks(t *testing.T) {
	vm := NewViewportManager(10, 100)

	var viewportChanged bool
	var scrollEnded bool
	var mu sync.Mutex

	vm.SetOnViewportChange(func(start, end int) {
		mu.Lock()
		viewportChanged = true
		mu.Unlock()
	})

	vm.SetOnScrollEnd(func() {
		mu.Lock()
		scrollEnded = true
		mu.Unlock()
	})

	// Trigger viewport change
	vm.ScrollTo(10)

	mu.Lock()
	changed := viewportChanged
	mu.Unlock()

	if !changed {
		t.Error("Expected viewport change callback to be called")
	}

	// Wait for scroll end callback
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	ended := scrollEnded
	mu.Unlock()

	if !ended {
		t.Error("Expected scroll end callback to be called")
	}
}

func TestViewportStats(t *testing.T) {
	vm := NewViewportManager(5, 2)
	vm.SetTotalItems(100)
	vm.ScrollTo(20)

	stats := vm.GetStats()

	if stats["total_items"] != 100 {
		t.Errorf("Expected total_items 100, got %v", stats["total_items"])
	}

	if stats["visible_start"] != 20 {
		t.Errorf("Expected visible_start 20, got %v", stats["visible_start"])
	}

	if stats["visible_end"] != 24 {
		t.Errorf("Expected visible_end 24, got %v", stats["visible_end"])
	}

	if stats["viewport_height"] != 5 {
		t.Errorf("Expected viewport_height 5, got %v", stats["viewport_height"])
	}

	if stats["buffer_size"] != 2 {
		t.Errorf("Expected buffer_size 2, got %v", stats["buffer_size"])
	}
}

func TestViewportScrollVelocity(t *testing.T) {
	vm := NewViewportManager(5, 1)
	vm.SetTotalItems(100)

	// Initial velocity should be 0
	if vm.GetScrollVelocity() != 0 {
		t.Errorf("Expected initial scroll velocity 0, got %f", vm.GetScrollVelocity())
	}

	// Scroll and check velocity is tracked
	vm.ScrollTo(10)
	time.Sleep(10 * time.Millisecond)
	vm.ScrollTo(20)

	// Velocity should be greater than 0 after scrolling
	if vm.GetScrollVelocity() <= 0 {
		t.Errorf("Expected scroll velocity > 0 after scrolling, got %f", vm.GetScrollVelocity())
	}
}

func TestViewportHeightChange(t *testing.T) {
	vm := NewViewportManager(5, 1)
	vm.SetTotalItems(20)

	// Change viewport height
	vm.SetViewportHeight(10)

	start, end := vm.GetVisibleRange()
	if start != 0 || end != 9 {
		t.Errorf("Expected visible range [0, 9] after height change, got [%d, %d]", start, end)
	}
}

func TestViewportCacheEviction(t *testing.T) {
	vm := NewViewportManager(5, 1)
	vm.cacheSize = 3 // Set small cache size for testing

	// Fill cache beyond capacity
	vm.CacheRenderedItem(0, "content 0")
	vm.CacheRenderedItem(1, "content 1")
	vm.CacheRenderedItem(2, "content 2")
	vm.CacheRenderedItem(3, "content 3") // Should trigger eviction

	// Check that cache size is maintained
	if len(vm.renderCache) > vm.cacheSize {
		t.Errorf("Expected cache size <= %d, got %d", vm.cacheSize, len(vm.renderCache))
	}
}
