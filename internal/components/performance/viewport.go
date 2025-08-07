package performance

import (
	"sync"
	"time"
)

// ViewportManager handles viewport-based virtualization for large datasets
type ViewportManager struct {
	// Viewport configuration
	viewportHeight int
	bufferSize     int // Extra rows to render outside viewport

	// Data management
	totalItems   int
	visibleStart int
	visibleEnd   int
	renderStart  int
	renderEnd    int

	// Performance tracking
	lastScrollTime time.Time
	scrollVelocity float64

	// Caching
	renderCache map[int]string
	cacheSize   int
	cacheMutex  sync.RWMutex

	// Callbacks
	onViewportChange func(start, end int)
	onScrollEnd      func()
}

// NewViewportManager creates a new viewport manager
func NewViewportManager(viewportHeight, bufferSize int) *ViewportManager {
	return &ViewportManager{
		viewportHeight: viewportHeight,
		bufferSize:     bufferSize,
		renderCache:    make(map[int]string),
		cacheSize:      1000, // Cache up to 1000 rendered items
	}
}

// SetTotalItems updates the total number of items
func (vm *ViewportManager) SetTotalItems(count int) {
	vm.totalItems = count
	vm.updateViewport()
}

// SetViewportHeight updates the viewport height
func (vm *ViewportManager) SetViewportHeight(height int) {
	vm.viewportHeight = height
	vm.updateViewport()
}

// ScrollTo scrolls to a specific item index
func (vm *ViewportManager) ScrollTo(index int) {
	if index < 0 {
		index = 0
	}
	if index >= vm.totalItems {
		index = vm.totalItems - 1
	}

	vm.visibleStart = index
	vm.updateViewport()
	vm.trackScrolling()
}

// ScrollBy scrolls by a relative amount
func (vm *ViewportManager) ScrollBy(delta int) {
	newStart := vm.visibleStart + delta
	vm.ScrollTo(newStart)
}

// GetVisibleRange returns the currently visible item range
func (vm *ViewportManager) GetVisibleRange() (start, end int) {
	return vm.visibleStart, vm.visibleEnd
}

// GetRenderRange returns the range that should be rendered (including buffer)
func (vm *ViewportManager) GetRenderRange() (start, end int) {
	return vm.renderStart, vm.renderEnd
}

// ShouldRenderItem returns whether an item should be rendered
func (vm *ViewportManager) ShouldRenderItem(index int) bool {
	return index >= vm.renderStart && index <= vm.renderEnd
}

// IsItemVisible returns whether an item is currently visible
func (vm *ViewportManager) IsItemVisible(index int) bool {
	return index >= vm.visibleStart && index <= vm.visibleEnd
}

// updateViewport recalculates viewport boundaries
func (vm *ViewportManager) updateViewport() {
	// Calculate visible range
	vm.visibleEnd = vm.visibleStart + vm.viewportHeight - 1
	if vm.visibleEnd >= vm.totalItems {
		vm.visibleEnd = vm.totalItems - 1
	}

	// Calculate render range with buffer
	vm.renderStart = vm.visibleStart - vm.bufferSize
	if vm.renderStart < 0 {
		vm.renderStart = 0
	}

	vm.renderEnd = vm.visibleEnd + vm.bufferSize
	if vm.renderEnd >= vm.totalItems {
		vm.renderEnd = vm.totalItems - 1
	}

	// Notify callback
	if vm.onViewportChange != nil {
		vm.onViewportChange(vm.visibleStart, vm.visibleEnd)
	}
}

// trackScrolling tracks scrolling performance
func (vm *ViewportManager) trackScrolling() {
	now := time.Now()
	if !vm.lastScrollTime.IsZero() {
		timeDelta := now.Sub(vm.lastScrollTime).Seconds()
		if timeDelta > 0 {
			// Calculate scroll velocity (items per second)
			vm.scrollVelocity = 1.0 / timeDelta
		}
	}
	vm.lastScrollTime = now

	// Schedule scroll end detection
	lastScrollTime := vm.lastScrollTime
	go func() {
		time.Sleep(100 * time.Millisecond)
		if time.Since(lastScrollTime) >= 100*time.Millisecond {
			if vm.onScrollEnd != nil {
				vm.onScrollEnd()
			}
		}
	}()
}

// CacheRenderedItem caches a rendered item
func (vm *ViewportManager) CacheRenderedItem(index int, content string) {
	vm.cacheMutex.Lock()
	defer vm.cacheMutex.Unlock()

	// Implement LRU cache eviction if needed
	if len(vm.renderCache) >= vm.cacheSize {
		// Simple eviction: remove items outside render range
		for i := range vm.renderCache {
			if i < vm.renderStart || i > vm.renderEnd {
				delete(vm.renderCache, i)
				break
			}
		}
	}

	vm.renderCache[index] = content
}

// GetCachedItem retrieves a cached rendered item
func (vm *ViewportManager) GetCachedItem(index int) (string, bool) {
	vm.cacheMutex.RLock()
	defer vm.cacheMutex.RUnlock()

	content, exists := vm.renderCache[index]
	return content, exists
}

// ClearCache clears the render cache
func (vm *ViewportManager) ClearCache() {
	vm.cacheMutex.Lock()
	defer vm.cacheMutex.Unlock()

	vm.renderCache = make(map[int]string)
}

// SetOnViewportChange sets the viewport change callback
func (vm *ViewportManager) SetOnViewportChange(callback func(start, end int)) {
	vm.onViewportChange = callback
}

// SetOnScrollEnd sets the scroll end callback
func (vm *ViewportManager) SetOnScrollEnd(callback func()) {
	vm.onScrollEnd = callback
}

// GetScrollVelocity returns the current scroll velocity
func (vm *ViewportManager) GetScrollVelocity() float64 {
	return vm.scrollVelocity
}

// GetStats returns performance statistics
func (vm *ViewportManager) GetStats() map[string]interface{} {
	vm.cacheMutex.RLock()
	defer vm.cacheMutex.RUnlock()

	return map[string]interface{}{
		"total_items":     vm.totalItems,
		"visible_start":   vm.visibleStart,
		"visible_end":     vm.visibleEnd,
		"render_start":    vm.renderStart,
		"render_end":      vm.renderEnd,
		"viewport_height": vm.viewportHeight,
		"buffer_size":     vm.bufferSize,
		"cache_size":      len(vm.renderCache),
		"scroll_velocity": vm.scrollVelocity,
	}
}
