package performance

import (
	"sync"
	"time"
)

// Debouncer handles debouncing of rapid updates to improve performance
type Debouncer struct {
	delay    time.Duration
	timer    *time.Timer
	callback func()
	mutex    sync.Mutex
	pending  bool
}

// NewDebouncer creates a new debouncer with the specified delay
func NewDebouncer(delay time.Duration, callback func()) *Debouncer {
	return &Debouncer{
		delay:    delay,
		callback: callback,
	}
}

// Trigger triggers the debounced function
func (d *Debouncer) Trigger() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.pending = true

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.delay, func() {
		d.mutex.Lock()
		defer d.mutex.Unlock()

		if d.pending {
			d.pending = false
			d.callback()
		}
	})
}

// Cancel cancels any pending debounced call
func (d *Debouncer) Cancel() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.pending = false
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}

// IsPending returns whether a call is pending
func (d *Debouncer) IsPending() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.pending
}

// SetDelay updates the debounce delay
func (d *Debouncer) SetDelay(delay time.Duration) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.delay = delay
}

// UpdateDebouncer handles debouncing of update operations
type UpdateDebouncer struct {
	debouncers map[string]*Debouncer
	mutex      sync.RWMutex
}

// NewUpdateDebouncer creates a new update debouncer
func NewUpdateDebouncer() *UpdateDebouncer {
	return &UpdateDebouncer{
		debouncers: make(map[string]*Debouncer),
	}
}

// Debounce debounces a function call by key
func (ud *UpdateDebouncer) Debounce(key string, delay time.Duration, callback func()) {
	ud.mutex.Lock()
	defer ud.mutex.Unlock()

	debouncer, exists := ud.debouncers[key]
	if !exists {
		debouncer = NewDebouncer(delay, callback)
		ud.debouncers[key] = debouncer
	} else {
		debouncer.callback = callback
		debouncer.SetDelay(delay)
	}

	debouncer.Trigger()
}

// Cancel cancels a debounced call by key
func (ud *UpdateDebouncer) Cancel(key string) {
	ud.mutex.RLock()
	debouncer, exists := ud.debouncers[key]
	ud.mutex.RUnlock()

	if exists {
		debouncer.Cancel()
	}
}

// CancelAll cancels all pending debounced calls
func (ud *UpdateDebouncer) CancelAll() {
	ud.mutex.RLock()
	defer ud.mutex.RUnlock()

	for _, debouncer := range ud.debouncers {
		debouncer.Cancel()
	}
}

// GetPendingCount returns the number of pending debounced calls
func (ud *UpdateDebouncer) GetPendingCount() int {
	ud.mutex.RLock()
	defer ud.mutex.RUnlock()

	count := 0
	for _, debouncer := range ud.debouncers {
		if debouncer.IsPending() {
			count++
		}
	}
	return count
}

// PerformanceMonitor tracks performance metrics
type PerformanceMonitor struct {
	metrics map[string]*Metric
	mutex   sync.RWMutex
}

// Metric represents a performance metric
type Metric struct {
	Name        string
	Count       int64
	TotalTime   time.Duration
	MinTime     time.Duration
	MaxTime     time.Duration
	LastTime    time.Duration
	LastUpdated time.Time
	Samples     []time.Duration
	MaxSamples  int
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: make(map[string]*Metric),
	}
}

// StartTimer starts timing an operation
func (pm *PerformanceMonitor) StartTimer(name string) func() {
	start := time.Now()
	return func() {
		pm.RecordDuration(name, time.Since(start))
	}
}

// RecordDuration records a duration for a metric
func (pm *PerformanceMonitor) RecordDuration(name string, duration time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	metric, exists := pm.metrics[name]
	if !exists {
		metric = &Metric{
			Name:       name,
			MinTime:    duration,
			MaxTime:    duration,
			MaxSamples: 100, // Keep last 100 samples
			Samples:    make([]time.Duration, 0, 100),
		}
		pm.metrics[name] = metric
	}

	// Update metric
	metric.Count++
	metric.TotalTime += duration
	metric.LastTime = duration
	metric.LastUpdated = time.Now()

	if duration < metric.MinTime {
		metric.MinTime = duration
	}
	if duration > metric.MaxTime {
		metric.MaxTime = duration
	}

	// Add to samples (with circular buffer)
	if len(metric.Samples) >= metric.MaxSamples {
		// Remove oldest sample
		metric.Samples = metric.Samples[1:]
	}
	metric.Samples = append(metric.Samples, duration)
}

// GetMetric returns a metric by name
func (pm *PerformanceMonitor) GetMetric(name string) *Metric {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	metric, exists := pm.metrics[name]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	return &Metric{
		Name:        metric.Name,
		Count:       metric.Count,
		TotalTime:   metric.TotalTime,
		MinTime:     metric.MinTime,
		MaxTime:     metric.MaxTime,
		LastTime:    metric.LastTime,
		LastUpdated: metric.LastUpdated,
		Samples:     append([]time.Duration(nil), metric.Samples...),
		MaxSamples:  metric.MaxSamples,
	}
}

// GetAllMetrics returns all metrics
func (pm *PerformanceMonitor) GetAllMetrics() map[string]*Metric {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	result := make(map[string]*Metric)
	for name, metric := range pm.metrics {
		result[name] = &Metric{
			Name:        metric.Name,
			Count:       metric.Count,
			TotalTime:   metric.TotalTime,
			MinTime:     metric.MinTime,
			MaxTime:     metric.MaxTime,
			LastTime:    metric.LastTime,
			LastUpdated: metric.LastUpdated,
			Samples:     append([]time.Duration(nil), metric.Samples...),
			MaxSamples:  metric.MaxSamples,
		}
	}
	return result
}

// AverageTime returns the average time for a metric
func (m *Metric) AverageTime() time.Duration {
	if m.Count == 0 {
		return 0
	}
	return m.TotalTime / time.Duration(m.Count)
}

// RecentAverageTime returns the average of recent samples
func (m *Metric) RecentAverageTime(sampleCount int) time.Duration {
	if len(m.Samples) == 0 {
		return 0
	}

	start := len(m.Samples) - sampleCount
	if start < 0 {
		start = 0
	}

	var total time.Duration
	count := 0
	for i := start; i < len(m.Samples); i++ {
		total += m.Samples[i]
		count++
	}

	if count == 0 {
		return 0
	}
	return total / time.Duration(count)
}

// Reset resets a metric
func (pm *PerformanceMonitor) Reset(name string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if metric, exists := pm.metrics[name]; exists {
		metric.Count = 0
		metric.TotalTime = 0
		metric.MinTime = 0
		metric.MaxTime = 0
		metric.LastTime = 0
		metric.Samples = metric.Samples[:0]
	}
}

// ResetAll resets all metrics
func (pm *PerformanceMonitor) ResetAll() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for _, metric := range pm.metrics {
		metric.Count = 0
		metric.TotalTime = 0
		metric.MinTime = 0
		metric.MaxTime = 0
		metric.LastTime = 0
		metric.Samples = metric.Samples[:0]
	}
}

// GetSummary returns a performance summary
func (pm *PerformanceMonitor) GetSummary() map[string]interface{} {
	metrics := pm.GetAllMetrics()
	summary := make(map[string]interface{})

	for name, metric := range metrics {
		summary[name] = map[string]interface{}{
			"count":        metric.Count,
			"total_time":   metric.TotalTime,
			"average_time": metric.AverageTime(),
			"min_time":     metric.MinTime,
			"max_time":     metric.MaxTime,
			"last_time":    metric.LastTime,
			"last_updated": metric.LastUpdated,
			"recent_avg":   metric.RecentAverageTime(10), // Last 10 samples
		}
	}

	return summary
}

// RateLimiter limits the rate of operations
type RateLimiter struct {
	rate     time.Duration
	lastCall time.Time
	mutex    sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate time.Duration) *RateLimiter {
	return &RateLimiter{
		rate: rate,
	}
}

// Allow returns whether an operation should be allowed
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	if now.Sub(rl.lastCall) >= rl.rate {
		rl.lastCall = now
		return true
	}
	return false
}

// Wait waits until the next operation is allowed
func (rl *RateLimiter) Wait() {
	rl.mutex.Lock()
	lastCall := rl.lastCall
	rate := rl.rate
	rl.mutex.Unlock()

	elapsed := time.Since(lastCall)
	if elapsed < rate {
		time.Sleep(rate - elapsed)
	}

	rl.mutex.Lock()
	rl.lastCall = time.Now()
	rl.mutex.Unlock()
}

// SetRate updates the rate limit
func (rl *RateLimiter) SetRate(rate time.Duration) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.rate = rate
}
