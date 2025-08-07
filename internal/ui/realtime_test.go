package ui

import (
	"context"
	"testing"
	"time"

	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestAutoRefreshBehavior tests that the app automatically refreshes resources
func TestAutoRefreshBehavior(t *testing.T) {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
	}

	config := &core.Config{
		RefreshInterval: 1, // 1 second for testing
	}

	app := NewApp(context.Background(), nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true

	// Simulate initial load
	cmd := app.Init()

	// Should return a tick command for auto-refresh
	if cmd == nil {
		t.Error("Expected Init to return a tick command for auto-refresh")
	}

	// Simulate tick message
	tickMsg := tickMsg(time.Now())
	model, cmd := app.Update(tickMsg)
	app = model.(*App)

	// Should return both a refresh command and next tick
	if cmd == nil {
		t.Error("Expected tick to trigger refresh and schedule next tick")
	}

	// Verify app is still in list mode
	if app.currentMode != ModeList {
		t.Errorf("Expected to remain in list mode during refresh, got %v", app.currentMode)
	}
}

// TestResourceWatchingBehavior tests real-time resource updates
func TestResourceWatchingBehavior(t *testing.T) {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
	}

	config := &core.Config{
		RefreshInterval: 5,
	}

	// Create app with nil client for testing
	app := NewApp(context.Background(), nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true

	// Verify initial state
	view := app.View()
	if len(view) == 0 {
		t.Error("Initial view should not be empty")
	}

	// Simulate resource type change
	app.nextResourceType()

	// Verify view updates
	view = app.View()
	if len(view) == 0 {
		t.Error("View should not be empty after resource type change")
	}

	// Verify resource type changed
	if app.state.CurrentResourceType == core.ResourceTypePod {
		t.Error("Resource type should have changed from Pod")
	}
}

// TestLogStreamingBehavior tests log view behavior
func TestLogStreamingBehavior(t *testing.T) {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
	}

	config := &core.Config{
		RefreshInterval: 5,
	}

	app := NewApp(context.Background(), nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true

	// Switch to log mode
	app.setMode(ModeLog)

	// Verify log view is displayed
	view := app.View()
	if len(view) == 0 {
		t.Error("Log view should not be empty")
	}

	// Test follow mode toggle (f key)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}
	model, _ := app.Update(keyMsg)
	app = model.(*App)

	// View should still render
	view = app.View()
	if len(view) == 0 {
		t.Error("Log view should not be empty after follow toggle")
	}

	// Test search in logs (/ key)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	model, _ = app.Update(keyMsg)
	app = model.(*App)

	// View should still render
	view = app.View()
	if len(view) == 0 {
		t.Error("Log view should not be empty after search")
	}

	// First escape to exit search mode
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = app.Update(keyMsg)
	app = model.(*App)

	// Second escape to return to list
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = app.Update(keyMsg)
	app = model.(*App)

	// Should return to list mode
	if app.currentMode != ModeList {
		t.Errorf("Expected to return to list mode, got %v", app.currentMode)
	}
}

// TestStateSyncDuringUpdates tests that state remains consistent during updates
func TestStateSyncDuringUpdates(t *testing.T) {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
		SortColumn:          "name",
		SortAscending:       true,
	}

	config := &core.Config{
		RefreshInterval: 5,
	}

	app := NewApp(context.Background(), nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true

	// Store initial state
	initialNamespace := app.state.CurrentNamespace
	initialResourceType := app.state.CurrentResourceType
	initialSort := app.state.SortColumn

	// Simulate refresh
	tickMsg := tickMsg(time.Now())
	model, _ := app.Update(tickMsg)
	app = model.(*App)

	// Verify state is preserved
	if app.state.CurrentNamespace != initialNamespace {
		t.Errorf("Namespace changed during refresh: expected %s, got %s",
			initialNamespace, app.state.CurrentNamespace)
	}

	if app.state.CurrentResourceType != initialResourceType {
		t.Errorf("Resource type changed during refresh: expected %v, got %v",
			initialResourceType, app.state.CurrentResourceType)
	}

	if app.state.SortColumn != initialSort {
		t.Errorf("Sort column changed during refresh: expected %s, got %s",
			initialSort, app.state.SortColumn)
	}

	// Change namespace and verify it persists
	app.state.CurrentNamespace = "kube-system"

	// Another refresh
	model, _ = app.Update(tickMsg)
	app = model.(*App)

	if app.state.CurrentNamespace != "kube-system" {
		t.Errorf("Namespace not preserved after change: expected kube-system, got %s",
			app.state.CurrentNamespace)
	}
}

// TestAutoRefreshPauseDuringOperations tests that auto-refresh pauses during user operations
func TestAutoRefreshPauseDuringOperations(t *testing.T) {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
	}

	config := &core.Config{
		RefreshInterval: 1,
	}

	app := NewApp(context.Background(), nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true

	// Open help mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	model, _ := app.Update(keyMsg)
	app = model.(*App)

	if app.currentMode != ModeHelp {
		t.Errorf("Expected help mode, got %v", app.currentMode)
	}

	// Simulate tick while in help mode
	tickMsg := tickMsg(time.Now())
	model, cmd := app.Update(tickMsg)
	app = model.(*App)

	// Should still schedule next tick but view should remain in help
	if cmd == nil {
		t.Error("Expected tick to still be scheduled in help mode")
	}

	if app.currentMode != ModeHelp {
		t.Errorf("Mode changed during refresh in help: expected ModeHelp, got %v", app.currentMode)
	}
}

// TestConcurrentUpdateHandling tests handling of concurrent updates
func TestConcurrentUpdateHandling(t *testing.T) {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
	}

	config := &core.Config{
		RefreshInterval: 5,
	}

	app := NewApp(context.Background(), nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true

	// Simulate rapid key presses and refreshes
	updates := []tea.Msg{
		tea.KeyMsg{Type: tea.KeyTab},                       // Change resource type
		tickMsg(time.Now()),                                // Auto-refresh
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}, // Sort
		tickMsg(time.Now()),                                // Another refresh
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}, // Namespace selector
	}

	for i, msg := range updates {
		model, _ := app.Update(msg)
		app = model.(*App)

		// Verify app remains stable
		view := app.View()
		if len(view) == 0 {
			t.Errorf("View became empty after update %d", i)
		}
	}

	// App should end in namespace selector mode
	if app.currentMode != ModeNamespaceSelector {
		t.Errorf("Expected namespace selector mode after updates, got %v", app.currentMode)
	}
}

// Helper to create mock pod with more details
func createDetailedMockPod(name, phase, namespace string, ready bool) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			UID:               types.UID("uid-" + name),
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase(phase),
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:  "main",
					Ready: ready,
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{
							StartedAt: metav1.Time{Time: time.Now()},
						},
					},
				},
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "main", Image: "nginx:latest"},
			},
		},
	}
	return pod
}
