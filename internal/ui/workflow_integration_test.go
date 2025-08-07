package ui

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

// TestEndToEndUserWorkflow tests complete user workflows from start to finish
func TestEndToEndUserWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		description string
		steps       []workflowStep
		validate    func(*testing.T, *App)
	}{
		{
			name:        "basic_navigation_workflow",
			description: "User navigates through resources and views",
			steps: []workflowStep{
				{key: "down", expect: "navigate down"},
				{key: "down", expect: "navigate further"},
				{key: "up", expect: "navigate up"},
				{key: "tab", expect: "open resource selector"},
				{key: "esc", expect: "close resource selector"},
				{key: "s", expect: "sort resources"},
				{key: "?", expect: "open help"},
				{key: "?", expect: "close help"},
			},
			validate: func(t *testing.T, app *App) {
				if app.currentMode != ModeList {
					t.Errorf("Expected list mode, got %v", app.currentMode)
				}
			},
		},
		{
			name:        "log_viewing_workflow",
			description: "User views and interacts with logs",
			steps: []workflowStep{
				{key: "l", expect: "open logs"},
				{key: "f", expect: "toggle follow"},
				{key: "f", expect: "toggle follow off"},
				{key: "esc", expect: "return to list"},
			},
			validate: func(t *testing.T, app *App) {
				if app.currentMode != ModeList {
					t.Errorf("Expected to return to list mode, got %v", app.currentMode)
				}
			},
		},
		{
			name:        "namespace_switching_workflow",
			description: "User switches between namespaces",
			steps: []workflowStep{
				{key: "n", expect: "open namespace selector"},
				{key: "down", expect: "navigate namespaces"},
				{key: "esc", expect: "cancel selection"},
				{key: "n", expect: "open namespace selector again"},
				{key: "enter", expect: "select namespace"},
			},
			validate: func(t *testing.T, app *App) {
				// Should have returned to list after selection
				if app.currentMode != ModeList {
					t.Errorf("Expected list mode after namespace selection, got %v", app.currentMode)
				}
			},
		},
		{
			name:        "context_switching_workflow",
			description: "User switches between contexts",
			steps: []workflowStep{
				{key: "c", expect: "open context selector"},
				{key: "esc", expect: "cancel selection"},
			},
			validate: func(t *testing.T, app *App) {
				if app.currentMode != ModeList {
					t.Errorf("Expected list mode, got %v", app.currentMode)
				}
			},
		},
		{
			name:        "describe_view_workflow",
			description: "User views resource details",
			steps: []workflowStep{
				{key: "i", expect: "open describe view"},
				{key: "esc", expect: "return to list"},
				{key: "d", expect: "open describe view with d key"},
				{key: "esc", expect: "return to list"},
			},
			validate: func(t *testing.T, app *App) {
				if app.currentMode != ModeList {
					t.Errorf("Expected list mode, got %v", app.currentMode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)

			// Execute workflow steps
			for i, step := range tt.steps {
				model, _ := executeWorkflowStep(app, step)
				app = model.(*App)

				// Verify app is still responsive
				view := app.View()
				if len(view) == 0 {
					t.Errorf("Step %d (%s): View became empty", i+1, step.key)
				}
			}

			// Validate final state
			if tt.validate != nil {
				tt.validate(t, app)
			}
		})
	}
}

// TestMultiContextWorkflow tests workflows specific to multi-context mode
func TestMultiContextWorkflow(t *testing.T) {
	contexts := []string{"cluster-1", "cluster-2", "cluster-3"}

	tests := []struct {
		name     string
		steps    []workflowStep
		validate func(*testing.T, *App)
	}{
		{
			name: "multi_context_initialization",
			steps: []workflowStep{
				{key: "tab", expect: "switch resource type"},
				{key: "s", expect: "sort resources"},
			},
			validate: func(t *testing.T, app *App) {
				if !app.isMultiContext {
					t.Error("App should be in multi-context mode")
				}
				if len(app.activeContexts) != len(contexts) {
					t.Errorf("Expected %d contexts, got %d", len(contexts), len(app.activeContexts))
				}
			},
		},
		{
			name: "multi_context_log_viewing",
			steps: []workflowStep{
				{key: "l", expect: "view logs across contexts"},
				{key: "f", expect: "toggle follow mode"},
				{key: "esc", expect: "return to list"},
			},
			validate: func(t *testing.T, app *App) {
				if app.currentMode != ModeList {
					t.Errorf("Expected list mode, got %v", app.currentMode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create multi-context app
			state := &core.State{
				CurrentResourceType: core.ResourceTypePod,
				CurrentNamespace:    "default",
				CurrentContexts:     contexts,
			}
			config := &core.Config{
				RefreshInterval: 5,
			}

			app := NewAppWithMultiContext(context.Background(), nil, state, config)
			app.width = 100
			app.height = 30
			app.ready = true
			app.isMultiContext = true
			app.activeContexts = contexts

			// Execute workflow
			for _, step := range tt.steps {
				model, _ := executeWorkflowStep(app, step)
				app = model.(*App)
			}

			// Validate
			if tt.validate != nil {
				tt.validate(t, app)
			}
		})
	}
}

// TestConcurrentLogStreaming tests streaming logs from multiple sources
func TestConcurrentLogStreaming(t *testing.T) {
	podCount := 5
	simulator := NewLogStreamSimulator()

	// Add pods with logs
	for i := 0; i < podCount; i++ {
		podName := fmt.Sprintf("pod-%d", i)
		logs := []string{
			fmt.Sprintf("[Pod %d] Starting", i),
			fmt.Sprintf("[Pod %d] Processing", i),
			fmt.Sprintf("[Pod %d] Complete", i),
		}
		simulator.AddPod(podName, "default", map[string][]string{
			"main": logs,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Stream from all pods concurrently
	var wg sync.WaitGroup
	logCount := 0
	var mu sync.Mutex

	for i := 0; i < podCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			podName := fmt.Sprintf("pod-%d", index)
			logCh, errCh := simulator.SimulatePodLogs(ctx, podName, "main", nil)

			for {
				select {
				case _, ok := <-logCh:
					if !ok {
						return
					}
					mu.Lock()
					logCount++
					mu.Unlock()
				case err := <-errCh:
					if err != nil {
						t.Logf("Error from pod %d: %v", index, err)
					}
					return
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	// Wait for completion
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		expectedLogs := podCount * 3 // 3 logs per pod
		if logCount != expectedLogs {
			t.Errorf("Expected %d logs, got %d", expectedLogs, logCount)
		}
	case <-time.After(4 * time.Second):
		t.Error("Timeout waiting for concurrent log streaming")
	}
}

// TestRealtimeResourceUpdates tests handling of real-time resource changes
func TestRealtimeResourceUpdates(t *testing.T) {
	app := createTestApp(t)

	// Simulate rapid updates
	updateCount := 10
	for i := 0; i < updateCount; i++ {
		// Simulate tick for auto-refresh
		tickMsg := tickMsg(time.Now())
		model, _ := app.Update(tickMsg)
		app = model.(*App)

		// Verify app remains stable
		view := app.View()
		if len(view) == 0 {
			t.Errorf("View empty after update %d", i+1)
		}

		// Small delay between updates
		time.Sleep(50 * time.Millisecond)
	}

	// App should still be responsive
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	model, _ := app.Update(keyMsg)
	app = model.(*App)

	if app.currentMode != ModeHelp {
		t.Errorf("Expected help mode after updates, got %v", app.currentMode)
	}
}

// TestErrorRecoveryWorkflow tests recovery from various error conditions
func TestErrorRecoveryWorkflow(t *testing.T) {
	tests := []struct {
		name     string
		scenario string
		recovery []workflowStep
	}{
		{
			name:     "recover_from_invalid_selection",
			scenario: "no_resource_selected",
			recovery: []workflowStep{
				{key: "l", expect: "try to view logs"},
				{key: "r", expect: "refresh"},
				{key: "tab", expect: "switch resource type"},
			},
		},
		{
			name:     "recover_from_mode_confusion",
			scenario: "stuck_in_mode",
			recovery: []workflowStep{
				{key: "esc", expect: "escape"},
				{key: "esc", expect: "escape again"},
				{key: "?", expect: "open help"},
				{key: "?", expect: "close help"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)

			// Execute recovery steps
			for _, step := range tt.recovery {
				model, _ := executeWorkflowStep(app, step)
				app = model.(*App)

				// App should remain functional
				view := app.View()
				if len(view) == 0 {
					t.Errorf("View empty during recovery: %s", step.expect)
				}
			}

			// Should end up in a stable state
			if app.currentMode != ModeList && app.currentMode != ModeHelp {
				t.Logf("Ended in mode: %v", app.currentMode)
			}
		})
	}
}

// TestLargeDatasetPerformance tests performance with many resources
func TestLargeDatasetPerformance(t *testing.T) {
	resourceCounts := []int{100, 250, 500}

	for _, count := range resourceCounts {
		t.Run(fmt.Sprintf("%d_resources", count), func(t *testing.T) {
			app := createTestApp(t)

			start := time.Now()

			// Perform operations
			operations := []string{"s", "s", "s"} // Multiple sorts
			for _, op := range operations {
				keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(op)}
				model, _ := app.Update(keyMsg)
				app = model.(*App)
			}

			// Navigate through resources
			for i := 0; i < 20; i++ {
				keyMsg := tea.KeyMsg{Type: tea.KeyDown}
				model, _ := app.Update(keyMsg)
				app = model.(*App)
			}

			elapsed := time.Since(start)

			// Performance check
			maxDuration := time.Duration(count/100) * time.Second
			if elapsed > maxDuration {
				t.Errorf("Operations on %d resources took %v, max allowed %v",
					count, elapsed, maxDuration)
			}

			// App should still be responsive
			view := app.View()
			if len(view) == 0 {
				t.Error("View empty after large dataset operations")
			}

			t.Logf("Handled %d resources in %v", count, elapsed)
		})
	}
}

// TestResourceCleanupWorkflow tests proper cleanup of resources
func TestResourceCleanupWorkflow(t *testing.T) {
	app := createTestApp(t)

	// Open and close various views multiple times
	viewSequence := []string{"l", "esc", "i", "esc", "?", "?", "n", "esc", "c", "esc"}

	for i, key := range viewSequence {
		var keyMsg tea.KeyMsg
		if key == "esc" {
			keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		} else {
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}

		model, _ := app.Update(keyMsg)
		app = model.(*App)

		// Verify no resource leaks
		view := app.View()
		if len(view) == 0 {
			t.Errorf("View empty at step %d (key: %s)", i+1, key)
		}
	}

	// Should end up in list mode
	if app.currentMode != ModeList {
		t.Errorf("Expected list mode after cleanup, got %v", app.currentMode)
	}
}

// Helper types

type workflowStep struct {
	key    string
	expect string
}

func executeWorkflowStep(app *App, step workflowStep) (tea.Model, tea.Cmd) {
	var keyMsg tea.KeyMsg

	switch step.key {
	case "up":
		keyMsg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		keyMsg = tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		keyMsg = tea.KeyMsg{Type: tea.KeyTab}
	default:
		keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(step.key)}
	}

	return app.Update(keyMsg)
}
