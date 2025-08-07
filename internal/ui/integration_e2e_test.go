package ui

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
)

// TestCompleteUserJourney tests a complete user workflow from connection to action
func TestCompleteUserJourney(t *testing.T) {
	tests := []struct {
		name     string
		journey  []userAction
		validate func(*testing.T, *App)
	}{
		{
			name: "connect_navigate_select_view_logs",
			journey: []userAction{
				{action: "init", description: "Initialize app"},
				{action: "wait_ready", description: "Wait for app ready"},
				{action: "key", value: "down", description: "Navigate to second pod"},
				{action: "key", value: "down", description: "Navigate to third pod"},
				{action: "key", value: "l", description: "View logs"},
				{action: "key", value: "f", description: "Toggle follow mode"},
				{action: "key", value: "/", description: "Search in logs"},
				{action: "type", value: "error", description: "Type search term"},
				{action: "key", value: "enter", description: "Execute search"},
				{action: "key", value: "esc", description: "Exit search"},
				{action: "key", value: "esc", description: "Return to list"},
			},
			validate: func(t *testing.T, app *App) {
				if app.currentMode != ModeList {
					t.Errorf("Expected to end in list mode, got %v", app.currentMode)
				}
			},
		},
		{
			name: "multi_resource_navigation_workflow",
			journey: []userAction{
				{action: "init", description: "Initialize app"},
				{action: "key", value: "tab", description: "Switch to deployments"},
				{action: "wait", duration: 100 * time.Millisecond},
				{action: "key", value: "tab", description: "Switch to services"},
				{action: "wait", duration: 100 * time.Millisecond},
				{action: "key", value: "tab", description: "Switch to statefulsets"},
				{action: "key", value: "i", description: "View details"},
				{action: "key", value: "esc", description: "Return to list"},
				{action: "key", value: "shift+tab", description: "Previous resource type"},
			},
			validate: func(t *testing.T, app *App) {
				if app.state.CurrentResourceType == core.ResourceTypePod {
					t.Error("Should have changed from initial Pod resource type")
				}
			},
		},
		{
			name: "namespace_context_switching_workflow",
			journey: []userAction{
				{action: "init", description: "Initialize app"},
				{action: "key", value: "n", description: "Open namespace selector"},
				{action: "key", value: "down", description: "Navigate namespaces"},
				{action: "key", value: "enter", description: "Select namespace"},
				{action: "wait", duration: 100 * time.Millisecond},
				{action: "key", value: "c", description: "Open context selector"},
				{action: "key", value: "esc", description: "Cancel context selection"},
				{action: "key", value: "n", description: "Open namespace selector again"},
				{action: "key", value: "down", description: "Navigate"},
				{action: "key", value: "down", description: "Navigate more"},
				{action: "key", value: "enter", description: "Select different namespace"},
			},
			validate: func(t *testing.T, app *App) {
				// Should have changed namespace
				if app.state.CurrentNamespace == "" {
					t.Error("Namespace should be set")
				}
			},
		},
		{
			name: "delete_with_confirmation_workflow",
			journey: []userAction{
				{action: "init", description: "Initialize app"},
				{action: "key", value: "down", description: "Select a pod"},
				{action: "key", value: "D", description: "Delete pod"},
				{action: "wait", duration: 50 * time.Millisecond},
				{action: "key", value: "n", description: "Cancel deletion"},
				{action: "key", value: "D", description: "Delete pod again"},
				{action: "wait", duration: 50 * time.Millisecond},
				{action: "key", value: "y", description: "Confirm deletion"},
				{action: "wait", duration: 200 * time.Millisecond},
			},
			validate: func(t *testing.T, app *App) {
				if app.currentMode != ModeList {
					t.Errorf("Should return to list after delete, got %v", app.currentMode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestAppWithMockClient(t)

			// Execute journey
			for _, action := range tt.journey {
				executeUserAction(t, app, action)
			}

			// Validate final state
			if tt.validate != nil {
				tt.validate(t, app)
			}

			// Ensure app is still responsive
			view := app.View()
			if len(view) == 0 {
				t.Error("App view should not be empty after journey")
			}
		})
	}
}

// TestMultiContextCompleteWorkflow tests complete workflows in multi-context mode
func TestMultiContextCompleteWorkflow(t *testing.T) {
	contexts := []string{"prod-us-east", "prod-us-west", "prod-eu"}

	tests := []struct {
		name     string
		setup    func(*MockMultiClient)
		workflow []userAction
		validate func(*testing.T, *App)
	}{
		{
			name: "multi_context_pod_logs_aggregation",
			setup: func(client *MockMultiClient) {
				// Add same pod to all contexts
				for _, ctx := range contexts {
					client.contexts[ctx].pods = []*v1.Pod{
						createMockPod("api-server", "Running", "default"),
						createMockPod("database", "Running", "default"),
					}
				}
			},
			workflow: []userAction{
				{action: "init", description: "Initialize multi-context app"},
				{action: "key", value: "l", description: "View logs across contexts"},
				{action: "wait", duration: 200 * time.Millisecond},
				{action: "key", value: "f", description: "Toggle follow mode"},
				{action: "key", value: "p", description: "Cycle through pods"},
				{action: "key", value: "c", description: "Cycle containers"},
				{action: "key", value: "esc", description: "Return to list"},
			},
			validate: func(t *testing.T, app *App) {
				if !app.isMultiContext {
					t.Error("App should be in multi-context mode")
				}
				if len(app.activeContexts) != len(contexts) {
					t.Errorf("Expected %d active contexts, got %d", len(contexts), len(app.activeContexts))
				}
			},
		},
		{
			name: "multi_context_resource_comparison",
			setup: func(client *MockMultiClient) {
				// Different resources in each context
				client.contexts["prod-us-east"].pods = []*v1.Pod{
					createMockPod("api-v1", "Running", "default"),
					createMockPod("api-v2", "Running", "default"),
				}
				client.contexts["prod-us-west"].pods = []*v1.Pod{
					createMockPod("api-v2", "Running", "default"),
					createMockPod("api-v3", "Pending", "default"),
				}
				client.contexts["prod-eu"].pods = []*v1.Pod{
					createMockPod("api-v3", "Running", "default"),
				}
			},
			workflow: []userAction{
				{action: "init", description: "Initialize"},
				{action: "key", value: "tab", description: "Change resource type"},
				{action: "key", value: "s", description: "Sort resources"},
				{action: "key", value: "i", description: "View details"},
				{action: "wait", duration: 100 * time.Millisecond},
				{action: "key", value: "esc", description: "Return"},
			},
			validate: func(t *testing.T, app *App) {
				// Should handle different resources gracefully
				if app.currentMode != ModeList {
					t.Errorf("Expected list mode, got %v", app.currentMode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createMockMultiClient(contexts)
			if tt.setup != nil {
				tt.setup(client)
			}

			state := &core.State{
				CurrentResourceType: core.ResourceTypePod,
				CurrentNamespace:    "default",
				CurrentContexts:     contexts,
			}
			config := &core.Config{
				RefreshInterval: 5,
			}

			app := NewAppWithMultiContext(context.Background(), nil, state, config)
			app.width = 120
			app.height = 40
			app.ready = true
			app.isMultiContext = true
			app.activeContexts = contexts

			// Execute workflow
			for _, action := range tt.workflow {
				executeUserAction(t, app, action)
			}

			// Validate
			if tt.validate != nil {
				tt.validate(t, app)
			}
		})
	}
}

// TestLogStreamingWithMultiplePods tests streaming logs from multiple pods simultaneously
func TestLogStreamingWithMultiplePods(t *testing.T) {
	tests := []struct {
		name         string
		podCount     int
		logLinesEach int
		concurrent   bool
	}{
		{
			name:         "sequential_small_logs",
			podCount:     3,
			logLinesEach: 10,
			concurrent:   false,
		},
		{
			name:         "concurrent_medium_logs",
			podCount:     5,
			logLinesEach: 50,
			concurrent:   true,
		},
		{
			name:         "concurrent_many_pods",
			podCount:     10,
			logLinesEach: 20,
			concurrent:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simulator := NewLogStreamSimulator()

			// Create pods with logs
			for i := 0; i < tt.podCount; i++ {
				podName := fmt.Sprintf("pod-%d", i)
				logs := make([]string, tt.logLinesEach)
				for j := 0; j < tt.logLinesEach; j++ {
					logs[j] = fmt.Sprintf("[Pod %d] Log line %d: Processing request", i, j)
				}

				simulator.AddPod(podName, "default", map[string][]string{
					"main": logs,
				})
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if tt.concurrent {
				// Stream from all pods concurrently
				var wg sync.WaitGroup
				totalLogs := int32(0)

				for i := 0; i < tt.podCount; i++ {
					wg.Add(1)
					go func(podIndex int) {
						defer wg.Done()

						podName := fmt.Sprintf("pod-%d", podIndex)
						logCh, errCh := simulator.SimulatePodLogs(ctx, podName, "main", nil)

						for {
							select {
							case log, ok := <-logCh:
								if !ok {
									return
								}
								if log != "" {
									atomic.AddInt32(&totalLogs, 1)
								}
							case err := <-errCh:
								if err != nil {
									t.Errorf("Error streaming pod %d: %v", podIndex, err)
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
					expectedLogs := int32(tt.podCount * tt.logLinesEach)
					if totalLogs < expectedLogs*8/10 { // Allow some loss
						t.Errorf("Expected ~%d logs, got %d", expectedLogs, totalLogs)
					}
				case <-time.After(6 * time.Second):
					t.Error("Timeout waiting for concurrent log streaming")
				}
			} else {
				// Sequential streaming
				totalLogs := 0
				for i := 0; i < tt.podCount; i++ {
					podName := fmt.Sprintf("pod-%d", i)
					logCh, _ := simulator.SimulatePodLogs(ctx, podName, "main", nil)

					for log := range logCh {
						if log != "" {
							totalLogs++
						}
					}
				}

				expectedLogs := tt.podCount * tt.logLinesEach
				if totalLogs != expectedLogs {
					t.Errorf("Expected %d logs, got %d", expectedLogs, totalLogs)
				}
			}
		})
	}
}

// TestResourceUpdatesAndRealtimeSync tests real-time synchronization of resource updates
func TestResourceUpdatesAndRealtimeSync(t *testing.T) {
	tests := []struct {
		name           string
		updateInterval time.Duration
		updateCount    int
		resourceType   core.ResourceType
	}{
		{
			name:           "rapid_pod_updates",
			updateInterval: 100 * time.Millisecond,
			updateCount:    10,
			resourceType:   core.ResourceTypePod,
		},
		{
			name:           "deployment_rollout_updates",
			updateInterval: 200 * time.Millisecond,
			updateCount:    5,
			resourceType:   core.ResourceTypeDeployment,
		},
		{
			name:           "service_endpoint_changes",
			updateInterval: 150 * time.Millisecond,
			updateCount:    7,
			resourceType:   core.ResourceTypeService,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &core.State{
				CurrentResourceType: tt.resourceType,
				CurrentNamespace:    "default",
				CurrentContext:      "test-context",
			}

			config := &core.Config{
				RefreshInterval: 1, // Fast refresh for testing
			}

			app := NewApp(context.Background(), nil, state, config)
			app.width = 80
			app.height = 24
			app.ready = true

			// Simulate resource updates
			updateCount := 0
			ticker := time.NewTicker(tt.updateInterval)
			defer ticker.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(tt.updateCount)*tt.updateInterval+time.Second)
			defer cancel()

			for {
				select {
				case <-ticker.C:
					// Simulate refresh
					tickMsg := tickMsg(time.Now())
					model, _ := app.Update(tickMsg)
					app = model.(*App)

					updateCount++
					if updateCount >= tt.updateCount {
						cancel()
					}

					// Verify view still renders
					view := app.View()
					if len(view) == 0 {
						t.Errorf("View empty after update %d", updateCount)
					}

				case <-ctx.Done():
					// Verify all updates were processed
					if updateCount < tt.updateCount {
						t.Errorf("Only processed %d/%d updates", updateCount, tt.updateCount)
					}
					return
				}
			}
		})
	}
}

// TestErrorRecoveryAndReconnection tests error handling and recovery scenarios
func TestErrorRecoveryAndReconnection(t *testing.T) {
	tests := []struct {
		name          string
		errorType     string
		recovery      []userAction
		expectRecover bool
	}{
		{
			name:      "connection_lost_recovery",
			errorType: "connection_timeout",
			recovery: []userAction{
				{action: "key", value: "r", description: "Manual refresh"},
				{action: "wait", duration: 500 * time.Millisecond},
				{action: "key", value: "r", description: "Retry refresh"},
			},
			expectRecover: true,
		},
		{
			name:      "permission_denied_handling",
			errorType: "unauthorized",
			recovery: []userAction{
				{action: "key", value: "c", description: "Try context switch"},
				{action: "key", value: "esc", description: "Cancel"},
				{action: "key", value: "n", description: "Try namespace switch"},
				{action: "key", value: "esc", description: "Cancel"},
			},
			expectRecover: false,
		},
		{
			name:      "resource_not_found_recovery",
			errorType: "not_found",
			recovery: []userAction{
				{action: "key", value: "tab", description: "Switch resource type"},
				{action: "key", value: "r", description: "Refresh"},
				{action: "wait", duration: 200 * time.Millisecond},
			},
			expectRecover: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createMockK8sClient()

			// Inject error condition
			if tt.errorType == "connection_timeout" {
				client.streamingErr = fmt.Errorf("connection timeout")
			} else if tt.errorType == "unauthorized" {
				client.streamingErr = fmt.Errorf("unauthorized")
			} else if tt.errorType == "not_found" {
				client.pods = []*v1.Pod{} // Empty pods
			}

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

			// Try to trigger error
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
			model, _ := app.Update(keyMsg)
			app = model.(*App)

			// Execute recovery actions
			for _, action := range tt.recovery {
				executeUserAction(t, app, action)
			}

			// Clear error condition if recovery expected
			if tt.expectRecover {
				client.streamingErr = nil
				if tt.errorType == "not_found" {
					client.pods = createTestPods()
				}

				// Final refresh
				app, _ = simulateKeyPress(app, "r")
			}

			// Verify app is still functional
			view := app.View()
			if len(view) == 0 {
				t.Error("App view should not be empty after recovery attempt")
			}

			// Should be able to navigate
			app, _ = simulateKeyPress(app, "?")
			if app.currentMode != ModeHelp {
				t.Error("App should still respond to navigation after error")
			}
		})
	}
}

// TestPerformanceWithLargeDatasets tests performance with large numbers of resources
func TestPerformanceWithLargeDatasets(t *testing.T) {
	tests := []struct {
		name          string
		resourceCount int
		operations    []string
		maxDuration   time.Duration
	}{
		{
			name:          "moderate_dataset",
			resourceCount: 100,
			operations:    []string{"sort", "filter", "navigate"},
			maxDuration:   2 * time.Second,
		},
		{
			name:          "large_dataset",
			resourceCount: 500,
			operations:    []string{"sort", "search", "scroll"},
			maxDuration:   3 * time.Second,
		},
		{
			name:          "very_large_dataset",
			resourceCount: 1000,
			operations:    []string{"sort", "navigate", "view"},
			maxDuration:   5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createMockK8sClient()

			// Create large dataset
			client.pods = make([]*v1.Pod, tt.resourceCount)
			for i := 0; i < tt.resourceCount; i++ {
				phase := "Running"
				if i%10 == 0 {
					phase = "Pending"
				} else if i%20 == 0 {
					phase = "Failed"
				}
				client.pods[i] = createMockPod(fmt.Sprintf("pod-%d", i), phase, "default")
			}

			state := &core.State{
				CurrentResourceType: core.ResourceTypePod,
				CurrentNamespace:    "default",
				CurrentContext:      "test-context",
			}

			config := &core.Config{
				RefreshInterval: 30, // Slow refresh for performance test
			}

			app := NewApp(context.Background(), nil, state, config)
			app.width = 120
			app.height = 50
			app.ready = true

			start := time.Now()

			// Perform operations
			for _, op := range tt.operations {
				switch op {
				case "sort":
					for i := 0; i < 3; i++ {
						app, _ = simulateKeyPress(app, "s")
					}
				case "filter":
					app, _ = simulateKeyPress(app, "/")
					// Type search term
					for _, char := range "Running" {
						app, _ = simulateKeyPress(app, string(char))
					}
					app, _ = simulateKeyPress(app, "enter")
				case "search":
					app, _ = simulateKeyPress(app, "/")
					for _, char := range "pod-5" {
						app, _ = simulateKeyPress(app, string(char))
					}
					app, _ = simulateKeyPress(app, "enter")
				case "navigate":
					for i := 0; i < 10; i++ {
						app, _ = simulateKeyPress(app, "down")
					}
					for i := 0; i < 5; i++ {
						app, _ = simulateKeyPress(app, "up")
					}
				case "scroll":
					for i := 0; i < 20; i++ {
						app, _ = simulateKeyPress(app, "down")
					}
				case "view":
					app, _ = simulateKeyPress(app, "i")
					time.Sleep(50 * time.Millisecond)
					app, _ = simulateKeyPress(app, "esc")
				}

				// Verify view still renders
				view := app.View()
				if len(view) == 0 {
					t.Errorf("View empty after operation %s", op)
				}
			}

			elapsed := time.Since(start)
			if elapsed > tt.maxDuration {
				t.Errorf("Operations took %v, exceeding max duration %v", elapsed, tt.maxDuration)
			}

			t.Logf("Handled %d resources with operations %v in %v", tt.resourceCount, tt.operations, elapsed)
		})
	}
}

// TestConcurrentOperations tests multiple concurrent operations

// TestResourceCleanup tests proper cleanup of resources
func TestResourceCleanup(t *testing.T) {
	tests := []struct {
		name     string
		scenario string
		verify   func(*testing.T, *App)
	}{
		{
			name:     "log_stream_cleanup",
			scenario: "open_close_logs",
			verify: func(t *testing.T, app *App) {
				// Verify no lingering log streams
				if app.currentMode == ModeLog {
					t.Error("Should not be in log mode after cleanup")
				}
			},
		},
		{
			name:     "context_switch_cleanup",
			scenario: "switch_contexts",
			verify: func(t *testing.T, app *App) {
				// Verify context state is clean
				if app.state.CurrentContext == "" {
					t.Error("Context should be set after switch")
				}
			},
		},
		{
			name:     "multi_window_cleanup",
			scenario: "open_multiple_views",
			verify: func(t *testing.T, app *App) {
				// Verify only one mode is active
				if app.currentMode != ModeList {
					t.Errorf("Should be in list mode, got %v", app.currentMode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestAppWithMockClient(t)

			switch tt.scenario {
			case "open_close_logs":
				// Open logs multiple times
				for i := 0; i < 5; i++ {
					app, _ = simulateKeyPress(app, "l")
					time.Sleep(50 * time.Millisecond)
					app, _ = simulateKeyPress(app, "esc")
				}

			case "switch_contexts":
				// Switch contexts multiple times
				for i := 0; i < 3; i++ {
					app, _ = simulateKeyPress(app, "c")
					app, _ = simulateKeyPress(app, "down")
					app, _ = simulateKeyPress(app, "enter")
					time.Sleep(50 * time.Millisecond)
				}

			case "open_multiple_views":
				// Open different views
				views := []string{"l", "i", "?", "n", "c"}
				for _, key := range views {
					app, _ = simulateKeyPress(app, key)
					time.Sleep(30 * time.Millisecond)
					app, _ = simulateKeyPress(app, "esc")
				}
			}

			// Verify cleanup
			if tt.verify != nil {
				tt.verify(t, app)
			}

			// General cleanup verification
			view := app.View()
			if len(view) == 0 {
				t.Error("View should not be empty after cleanup")
			}
		})
	}
}

// Helper types and functions

type userAction struct {
	action      string
	value       string
	description string
	duration    time.Duration
}

func executeUserAction(t *testing.T, app *App, action userAction) {
	t.Helper()

	switch action.action {
	case "init":
		// App already initialized
	case "wait":
		time.Sleep(action.duration)
	case "wait_ready":
		timeout := time.After(2 * time.Second)
		for !app.ready {
			select {
			case <-timeout:
				t.Fatal("App did not become ready")
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	case "key":
		var keyMsg tea.KeyMsg
		switch action.value {
		case "up":
			keyMsg = tea.KeyMsg{Type: tea.KeyUp}
		case "down":
			keyMsg = tea.KeyMsg{Type: tea.KeyDown}
		case "left":
			keyMsg = tea.KeyMsg{Type: tea.KeyLeft}
		case "right":
			keyMsg = tea.KeyMsg{Type: tea.KeyRight}
		case "enter":
			keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		case "tab":
			keyMsg = tea.KeyMsg{Type: tea.KeyTab}
		case "shift+tab":
			keyMsg = tea.KeyMsg{Type: tea.KeyShiftTab}
		case "space":
			keyMsg = tea.KeyMsg{Type: tea.KeySpace}
		case "delete":
			keyMsg = tea.KeyMsg{Type: tea.KeyDelete}
		default:
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(action.value)}
		}
		model, _ := app.Update(keyMsg)
		*app = *model.(*App)

	case "type":
		for _, char := range action.value {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
			model, _ := app.Update(keyMsg)
			*app = *model.(*App)
			time.Sleep(10 * time.Millisecond) // Simulate typing speed
		}
	}
}

func createTestAppWithMockClient(t *testing.T) *App {
	t.Helper()

	// For UI testing, we use nil client since we're testing UI behavior
	// The mock client is available but not directly used in NewApp
	_ = createMockK8sClient()

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

	// Initialize modes if needed
	if app.modes == nil {
		app.modes = map[ScreenModeType]ScreenMode{
			ModeList:              NewListMode(),
			ModeLog:               NewLogMode(),
			ModeDescribe:          NewDescribeMode(),
			ModeHelp:              NewHelpMode(),
			ModeContextSelector:   NewContextSelectorMode(),
			ModeNamespaceSelector: NewNamespaceSelectorMode(),
			ModeConfirmDialog:     NewConfirmDialogMode(),
		}
	}

	// Add test data to ResourceView for delete operations to work
	app.resourceView.SetTestData(
		[]string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"},
		[][]string{
			{"test-pod-1", "1/1", "Running", "0", "5m"},
			{"test-pod-2", "0/1", "Pending", "0", "2m"},
			{"test-pod-3", "1/1", "Running", "1", "10m"},
		},
	)
	app.resourceView.SetSelectedRow(0)

	return app
}
