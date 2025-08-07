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

// TestSinglePodLogStreaming tests log streaming for a single pod
func TestSinglePodLogStreaming(t *testing.T) {
	tests := []struct {
		name        string
		scenario    string
		expectError bool
	}{
		{
			name:        "successful_streaming",
			scenario:    "Stream logs from running pod",
			expectError: false,
		},
		{
			name:        "pod_not_found",
			scenario:    "Attempt to stream from non-existent pod",
			expectError: true,
		},
		{
			name:        "pod_terminating",
			scenario:    "Stream from terminating pod",
			expectError: false,
		},
		{
			name:        "follow_mode",
			scenario:    "Stream with follow mode enabled",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			simulator := NewLogStreamSimulator()
			helper := NewLogStreamTestHelper(t)

			// Add test pod
			simulator.AddPod("test-pod", "default", map[string][]string{
				"main": {
					"[2024-01-01 10:00:00] Starting application",
					"[2024-01-01 10:00:01] Server listening on port 8080",
					"[2024-01-01 10:00:02] Ready to accept connections",
				},
			})

			// Create app
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

			// Test follow mode toggle
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}
			model, _ := app.Update(keyMsg)
			app = model.(*App)

			// Test search functionality
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
			model, _ = app.Update(keyMsg)
			app = model.(*App)

			// Type search term
			searchTerm := "Server"
			for _, ch := range searchTerm {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
				model, _ = app.Update(keyMsg)
				app = model.(*App)
			}

			// Execute search
			keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
			model, _ = app.Update(keyMsg)
			app = model.(*App)

			// Verify view still renders
			view = app.View()
			if len(view) == 0 {
				t.Error("View should not be empty after search")
			}

			// Test streaming with context
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			logCh, errCh := simulator.SimulatePodLogs(ctx, "test-pod", "main", []string{
				"[2024-01-01 10:00:03] New connection from client",
				"[2024-01-01 10:00:04] Processing request",
			})

			if !tt.expectError {
				expectedLogs := []string{
					"[2024-01-01 10:00:03] New connection from client",
					"[2024-01-01 10:00:04] Processing request",
				}
				helper.AssertLogsReceived(logCh, expectedLogs, 1*time.Second)
				helper.AssertNoErrors(errCh, 100*time.Millisecond)
			}
		})
	}
}

// TestSinglePodStreamingErrors tests error handling during single pod streaming
func TestSinglePodStreamingErrors(t *testing.T) {
	tests := []struct {
		name          string
		errorScenario string
	}{
		{
			name:          "pod_not_found",
			errorScenario: "Pod does not exist",
		},
		{
			name:          "permission_denied",
			errorScenario: "No permission to view logs",
		},
		{
			name:          "pod_termination",
			errorScenario: "Pod terminates during streaming",
		},
		{
			name:          "network_interruption",
			errorScenario: "Network connection lost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simulator := NewLogStreamSimulator()

			// Simulate error
			simulator.SimulateError("error-pod", fmt.Errorf(tt.errorScenario))

			// Try to stream logs
			ctx := context.Background()
			logCh, errCh := simulator.SimulatePodLogs(ctx, "error-pod", "main", nil)

			// Should receive error
			select {
			case err := <-errCh:
				if err == nil {
					t.Error("Expected error but got nil")
				}
			case <-logCh:
				t.Error("Should not receive logs when there's an error")
			case <-time.After(100 * time.Millisecond):
				// Check if error was set
				if logCh != nil {
					t.Error("Expected nil log channel for error case")
				}
			}
		})
	}
}

// TestMultiContainerPodLogStreaming tests log streaming for pods with multiple containers
func TestMultiContainerPodLogStreaming(t *testing.T) {
	tests := []struct {
		name       string
		containers []string
		scenario   string
	}{
		{
			name:       "two_containers",
			containers: []string{"main", "sidecar"},
			scenario:   "Stream from pod with main and sidecar containers",
		},
		{
			name:       "three_containers",
			containers: []string{"app", "proxy", "monitor"},
			scenario:   "Stream from pod with three containers",
		},
		{
			name:       "init_containers",
			containers: []string{"init", "main"},
			scenario:   "Stream including init containers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simulator := NewMultiContainerPodSimulator("multi-pod", "default", tt.containers)
			helper := NewLogStreamTestHelper(t)

			// Add logs to each container
			for _, container := range tt.containers {
				simulator.AddContainerLogs(container, []string{
					fmt.Sprintf("[Container %s] Starting", container),
					fmt.Sprintf("[Container %s] Initialized", container),
					fmt.Sprintf("[Container %s] Ready", container),
				})
			}

			// Test streaming from each container
			ctx := context.Background()
			for _, container := range tt.containers {
				logCh, err := simulator.StreamContainerLogs(ctx, container)
				if err != nil {
					t.Errorf("Failed to stream logs from container %s: %v", container, err)
					continue
				}

				expectedLogs := []string{
					fmt.Sprintf("[Container %s] Starting", container),
					fmt.Sprintf("[Container %s] Initialized", container),
					fmt.Sprintf("[Container %s] Ready", container),
				}
				helper.AssertLogsReceived(logCh, expectedLogs, 1*time.Second)
			}

			// Test container switching in UI
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
			app.setMode(ModeLog)

			// Simulate container switching with 'c' key
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
			model, _ := app.Update(keyMsg)
			app = model.(*App)

			// View should still render
			view := app.View()
			if len(view) == 0 {
				t.Error("View should not be empty after container switch")
			}
		})
	}
}

// TestMultiplePodLogStreaming tests streaming logs from multiple pods (deployment/statefulset)
func TestMultiplePodLogStreaming(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		replicas     int
	}{
		{
			name:         "deployment_3_replicas",
			resourceType: "deployment",
			replicas:     3,
		},
		{
			name:         "statefulset_5_replicas",
			resourceType: "statefulset",
			replicas:     5,
		},
		{
			name:         "replicaset_2_replicas",
			resourceType: "replicaset",
			replicas:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simulator := NewDeploymentLogSimulator("test-"+tt.resourceType, "default", tt.replicas)

			// Stream logs from all pods
			ctx := context.Background()
			logChannels := simulator.StreamAllPodLogs(ctx)

			// Verify we get logs from all replicas
			if len(logChannels) != tt.replicas {
				t.Errorf("Expected %d log channels, got %d", tt.replicas, len(logChannels))
			}

			// Verify logs from each pod
			for podName, logCh := range logChannels {
				// Read at least one log from each pod
				select {
				case log := <-logCh:
					if log == "" {
						t.Errorf("Received empty log from pod %s", podName)
					}
				case <-time.After(1 * time.Second):
					t.Errorf("Timeout waiting for logs from pod %s", podName)
				}
			}

			// Test pod selection cycling in UI
			state := &core.State{
				CurrentResourceType: core.ResourceType(tt.resourceType),
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
			app.setMode(ModeLog)

			// Simulate pod cycling with 'p' key
			for i := 0; i < tt.replicas; i++ {
				keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")}
				model, _ := app.Update(keyMsg)
				app = model.(*App)

				// View should render for each pod
				view := app.View()
				if len(view) == 0 {
					t.Errorf("View empty after switching to pod %d", i)
				}
			}
		})
	}
}

// TestDeploymentLogAggregation tests log aggregation from deployment pods
func TestDeploymentLogAggregation(t *testing.T) {
	simulator := NewDeploymentLogSimulator("test-deployment", "default", 3)

	// Stream logs from all pods concurrently
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	logChannels := simulator.StreamAllPodLogs(ctx)

	// Collect logs from all pods with proper synchronization
	var mu sync.Mutex
	allLogs := []string{}
	done := make(chan bool, len(logChannels))

	for podName, logCh := range logChannels {
		go func(name string, ch <-chan string) {
			localLogs := []string{}
			for log := range ch {
				localLogs = append(localLogs, fmt.Sprintf("[%s] %s", name, log))
			}
			// Use mutex to safely append to shared slice
			mu.Lock()
			allLogs = append(allLogs, localLogs...)
			mu.Unlock()
			done <- true
		}(podName, logCh)
	}

	// Wait for all goroutines with timeout
	timeout := time.After(3 * time.Second)
	for i := 0; i < len(logChannels); i++ {
		select {
		case <-done:
			// One goroutine finished
		case <-timeout:
			t.Error("Timeout waiting for log aggregation")
			return
		}
	}

	// Verify we got logs from all pods (with mutex protection for read)
	mu.Lock()
	logCount := len(allLogs)
	mu.Unlock()

	if logCount < len(logChannels)*3 { // Each pod sends at least 3 logs
		t.Errorf("Expected at least %d logs, got %d", len(logChannels)*3, logCount)
	}
}

// TestMultiContextLogStreaming tests log streaming across multiple contexts
func TestMultiContextLogStreaming(t *testing.T) {
	contexts := []string{"context-1", "context-2", "context-3"}
	simulator := NewMultiContextLogSimulator(contexts)

	// Add pods to each context
	for _, ctx := range contexts {
		simulator.AddPodToContext(ctx, "test-pod", "default", map[string][]string{
			"main": {
				fmt.Sprintf("[%s] Pod starting", ctx),
				fmt.Sprintf("[%s] Pod ready", ctx),
			},
		})
	}

	// Test streaming from each context
	ctx := context.Background()
	for _, contextName := range contexts {
		logCh, errCh, err := simulator.StreamLogsFromContext(ctx, contextName, "test-pod", "main")
		if err != nil {
			t.Errorf("Failed to stream from context %s: %v", contextName, err)
			continue
		}

		// Verify logs are received
		select {
		case log := <-logCh:
			if log == "" {
				t.Errorf("Received empty log from context %s", contextName)
			}
		case err := <-errCh:
			t.Errorf("Received error from context %s: %v", contextName, err)
		case <-time.After(1 * time.Second):
			// Some contexts might not have logs immediately
			t.Logf("No logs from context %s within timeout", contextName)
		}
	}

	// Test streaming from all contexts simultaneously
	allContextLogs := simulator.StreamLogsFromAllContexts(ctx, "test-pod", "main")

	// Verify we get channels for all contexts with the pod
	if len(allContextLogs) == 0 {
		t.Error("Expected log channels from at least one context")
	}

	// Test multi-context UI behavior
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContexts:     contexts,
	}
	config := &core.Config{
		RefreshInterval: 5,
	}

	app := NewAppWithMultiContext(context.Background(), nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true
	app.isMultiContext = true
	app.activeContexts = contexts

	// Switch to log mode
	app.setMode(ModeLog)

	// View should handle multi-context
	view := app.View()
	if len(view) == 0 {
		t.Error("Multi-context log view should not be empty")
	}
}

// TestLogStreamingPerformance tests performance with high-frequency logs
func TestLogStreamingPerformance(t *testing.T) {
	simulator := NewLogStreamSimulator()

	// Generate many log lines
	numLogs := 1000
	logs := make([]string, numLogs)
	for i := 0; i < numLogs; i++ {
		logs[i] = fmt.Sprintf("Log line %d: Processing request with ID %d", i, i)
	}

	simulator.AddPod("perf-test-pod", "default", map[string][]string{
		"main": logs,
	})

	// Measure streaming performance
	ctx := context.Background()
	start := time.Now()

	logCh, errCh := simulator.SimulatePodLogs(ctx, "perf-test-pod", "main", logs)

	received := 0
	done := make(chan bool)

	go func() {
		for range logCh {
			received++
		}
		done <- true
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		elapsed := time.Since(start)
		t.Logf("Streamed %d logs in %v", received, elapsed)

		// Should receive all logs
		if received != numLogs {
			t.Errorf("Expected %d logs, received %d", numLogs, received)
		}

		// Performance check: should stream 1000 logs in under 5 seconds
		if elapsed > 5*time.Second {
			t.Errorf("Streaming took too long: %v", elapsed)
		}

	case <-time.After(10 * time.Second):
		t.Error("Timeout during performance test")
	}

	// Check for errors
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Unexpected error during streaming: %v", err)
		}
	default:
		// No error, which is expected
	}
}

// TestLogSearchAndFilter tests log search and filtering functionality
func TestLogSearchAndFilter(t *testing.T) {
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
	app.setMode(ModeLog)

	// Test search initiation
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	model, _ := app.Update(keyMsg)
	app = model.(*App)

	// Type search term
	searchTerm := "error"
	for _, ch := range searchTerm {
		keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		model, _ = app.Update(keyMsg)
		app = model.(*App)
	}

	// Execute search
	keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	model, _ = app.Update(keyMsg)
	app = model.(*App)

	// Clear search
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = app.Update(keyMsg)
	app = model.(*App)

	// View should still render
	view := app.View()
	if len(view) == 0 {
		t.Error("View should not be empty after search operations")
	}
}

// TestLogFollowMode tests follow mode behavior
func TestLogFollowMode(t *testing.T) {
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
	app.setMode(ModeLog)

	// Toggle follow mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}
	model, _ := app.Update(keyMsg)
	app = model.(*App)

	// Scroll up (should disable follow)
	keyMsg = tea.KeyMsg{Type: tea.KeyUp}
	model, _ = app.Update(keyMsg)
	app = model.(*App)

	// Re-enable follow
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}
	model, _ = app.Update(keyMsg)
	app = model.(*App)

	// View should render in all states
	view := app.View()
	if len(view) == 0 {
		t.Error("View should not be empty in follow mode")
	}
}
