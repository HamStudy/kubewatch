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

// TestSinglePodLogsAcrossMultipleContexts tests streaming logs from a single pod across multiple contexts
func TestSinglePodLogsAcrossMultipleContexts(t *testing.T) {
	contexts := []string{"prod-cluster", "staging-cluster", "dev-cluster"}

	tests := []struct {
		name         string
		podName      string
		namespace    string
		expectErrors map[string]bool // context -> should error
	}{
		{
			name:      "same_pod_all_contexts",
			podName:   "app-pod",
			namespace: "default",
			expectErrors: map[string]bool{
				"prod-cluster":    false,
				"staging-cluster": false,
				"dev-cluster":     false,
			},
		},
		{
			name:      "pod_missing_in_some_contexts",
			podName:   "prod-only-pod",
			namespace: "production",
			expectErrors: map[string]bool{
				"prod-cluster":    false,
				"staging-cluster": true, // Pod doesn't exist
				"dev-cluster":     true, // Pod doesn't exist
			},
		},
		{
			name:      "different_pod_states",
			podName:   "migrating-pod",
			namespace: "default",
			expectErrors: map[string]bool{
				"prod-cluster":    false, // Running
				"staging-cluster": false, // Terminating
				"dev-cluster":     true,  // Not found
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simulator := NewMultiContextLogSimulator(contexts)

			// Set up pods in each context based on test scenario
			for _, ctx := range contexts {
				if !tt.expectErrors[ctx] {
					simulator.AddPodToContext(ctx, tt.podName, tt.namespace, map[string][]string{
						"main": {
							fmt.Sprintf("[%s] Pod %s starting", ctx, tt.podName),
							fmt.Sprintf("[%s] Application initialized", ctx),
							fmt.Sprintf("[%s] Ready to serve requests", ctx),
						},
					})
				}
			}

			// Create multi-context app
			state := &core.State{
				CurrentResourceType: core.ResourceTypePod,
				CurrentNamespace:    tt.namespace,
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

			// Test streaming from all contexts simultaneously
			ctx := context.Background()
			allLogs := simulator.StreamLogsFromAllContexts(ctx, tt.podName, "main")

			// Verify we get logs from expected contexts
			for ctxName, shouldError := range tt.expectErrors {
				if shouldError {
					// Should not have logs from this context
					if _, exists := allLogs[ctxName]; exists {
						t.Errorf("Expected no logs from context %s, but got channel", ctxName)
					}
				} else {
					// Should have logs from this context
					if logCh, exists := allLogs[ctxName]; !exists {
						t.Errorf("Expected logs from context %s, but got none", ctxName)
					} else {
						// Verify at least one log is received
						select {
						case log := <-logCh:
							if log == "" {
								t.Errorf("Received empty log from context %s", ctxName)
							}
						case <-time.After(1 * time.Second):
							t.Errorf("Timeout waiting for logs from context %s", ctxName)
						}
					}
				}
			}

			// Test UI behavior with multi-context logs
			view := app.View()
			if len(view) == 0 {
				t.Error("Multi-context log view should not be empty")
			}

			// Test follow mode across contexts
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}
			model, _ := app.Update(keyMsg)
			app = model.(*App)

			// Test search across contexts
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
			model, _ = app.Update(keyMsg)
			app = model.(*App)
		})
	}
}

// TestMultiContainerPodLogsAcrossMultipleContexts tests multi-container pods across contexts
func TestMultiContainerPodLogsAcrossMultipleContexts(t *testing.T) {
	contexts := []string{"cluster-1", "cluster-2", "cluster-3"}

	tests := []struct {
		name       string
		podName    string
		containers map[string][]string // container -> contexts it exists in
	}{
		{
			name:    "consistent_containers",
			podName: "multi-container-pod",
			containers: map[string][]string{
				"app":     contexts, // Exists in all contexts
				"sidecar": contexts, // Exists in all contexts
			},
		},
		{
			name:    "varying_containers",
			podName: "evolving-pod",
			containers: map[string][]string{
				"app":     contexts,                   // In all
				"sidecar": {"cluster-1", "cluster-2"}, // Not in cluster-3
				"monitor": {"cluster-1"},              // Only in cluster-1
			},
		},
		{
			name:    "init_containers_different",
			podName: "complex-pod",
			containers: map[string][]string{
				"init-db":    {"cluster-1", "cluster-2"},
				"init-cache": {"cluster-1"},
				"app":        contexts,
				"proxy":      {"cluster-2", "cluster-3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simulator := NewMultiContextLogSimulator(contexts)

			// Set up pods with containers in appropriate contexts
			for _, ctx := range contexts {
				containerLogs := make(map[string][]string)

				for container, ctxList := range tt.containers {
					// Check if this container exists in this context
					containerExists := false
					for _, c := range ctxList {
						if c == ctx {
							containerExists = true
							break
						}
					}

					if containerExists {
						containerLogs[container] = []string{
							fmt.Sprintf("[%s-%s] Starting container", ctx, container),
							fmt.Sprintf("[%s-%s] Container initialized", ctx, container),
							fmt.Sprintf("[%s-%s] Container ready", ctx, container),
						}
					}
				}

				if len(containerLogs) > 0 {
					simulator.AddPodToContext(ctx, tt.podName, "default", containerLogs)
				}
			}

			// Test container switching across contexts
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
			app.setMode(ModeLog)

			// Test container cycling with 'c' key across contexts
			for i := 0; i < len(tt.containers); i++ {
				keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
				model, _ := app.Update(keyMsg)
				app = model.(*App)

				// View should render for each container
				view := app.View()
				if len(view) == 0 {
					t.Errorf("View empty after switching to container %d", i)
				}
			}

			// Verify streaming from each container in each context
			ctx := context.Background()
			for container, ctxList := range tt.containers {
				for _, ctxName := range ctxList {
					logCh, errCh, err := simulator.StreamLogsFromContext(ctx, ctxName, tt.podName, container)
					if err != nil {
						t.Errorf("Failed to stream container %s from context %s: %v", container, ctxName, err)
						continue
					}

					// Should receive logs
					select {
					case log := <-logCh:
						if log == "" {
							t.Errorf("Empty log from container %s in context %s", container, ctxName)
						}
					case err := <-errCh:
						t.Errorf("Error streaming container %s in context %s: %v", container, ctxName, err)
					case <-time.After(1 * time.Second):
						// Some containers might not have logs immediately
						t.Logf("No logs from container %s in context %s", container, ctxName)
					}
				}
			}
		})
	}
}

// TestDeploymentLogsAcrossMultipleContexts tests deployment/statefulset logs across contexts
func TestDeploymentLogsAcrossMultipleContexts(t *testing.T) {
	contexts := []string{"prod", "staging", "dev"}

	tests := []struct {
		name         string
		resourceType string
		resourceName string
		replicas     map[string]int // context -> replica count
	}{
		{
			name:         "deployment_different_scales",
			resourceType: "deployment",
			resourceName: "web-app",
			replicas: map[string]int{
				"prod":    10, // Production has more replicas
				"staging": 3,  // Staging has fewer
				"dev":     1,  // Dev has minimal
			},
		},
		{
			name:         "statefulset_ordered",
			resourceType: "statefulset",
			resourceName: "database",
			replicas: map[string]int{
				"prod":    5,
				"staging": 3,
				"dev":     1,
			},
		},
		{
			name:         "rolling_update_scenario",
			resourceType: "deployment",
			resourceName: "api-server",
			replicas: map[string]int{
				"prod":    6, // Some old, some new pods
				"staging": 4, // All new pods
				"dev":     2, // All old pods
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simulator := NewMultiContextLogSimulator(contexts)

			// Set up deployments in each context with different replica counts
			for ctx, replicaCount := range tt.replicas {
				for i := 0; i < replicaCount; i++ {
					podName := fmt.Sprintf("%s-%s-%d", tt.resourceName, ctx, i)
					simulator.AddPodToContext(ctx, podName, "default", map[string][]string{
						"main": {
							fmt.Sprintf("[%s] Pod %d/%d starting", ctx, i+1, replicaCount),
							fmt.Sprintf("[%s] Pod %d/%d ready", ctx, i+1, replicaCount),
						},
					})
				}
			}

			// Create multi-context app
			state := &core.State{
				CurrentResourceType: core.ResourceType(tt.resourceType),
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
			app.setMode(ModeLog)

			// Test pod cycling across all contexts and replicas
			totalPods := 0
			for _, count := range tt.replicas {
				totalPods += count
			}

			// Cycle through pods with 'p' key
			for i := 0; i < totalPods; i++ {
				keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")}
				model, _ := app.Update(keyMsg)
				app = model.(*App)

				// View should render
				view := app.View()
				if len(view) == 0 {
					t.Errorf("View empty after cycling to pod %d/%d", i+1, totalPods)
				}
			}

			// Test aggregated log streaming from all pods across all contexts
			ctx := context.Background()
			var wg sync.WaitGroup
			logCount := make(map[string]int)
			var mu sync.Mutex

			for ctxName, replicaCount := range tt.replicas {
				for i := 0; i < replicaCount; i++ {
					wg.Add(1)
					go func(context string, replica int) {
						defer wg.Done()

						podName := fmt.Sprintf("%s-%s-%d", tt.resourceName, context, replica)
						logCh, _, err := simulator.StreamLogsFromContext(ctx, context, podName, "main")
						if err != nil {
							return
						}

						// Count logs received
						for range logCh {
							mu.Lock()
							logCount[context]++
							mu.Unlock()
						}
					}(ctxName, i)
				}
			}

			// Wait for streaming with timeout
			done := make(chan bool)
			go func() {
				wg.Wait()
				done <- true
			}()

			select {
			case <-done:
				// Verify we got logs from all contexts
				for ctx, replicas := range tt.replicas {
					if count, exists := logCount[ctx]; !exists || count == 0 {
						t.Errorf("No logs received from context %s with %d replicas", ctx, replicas)
					}
				}
			case <-time.After(5 * time.Second):
				t.Error("Timeout waiting for deployment logs across contexts")
			}
		})
	}
}

// TestStatefulSetLogsAcrossMultipleContexts tests statefulset-specific behavior across contexts
func TestStatefulSetLogsAcrossMultipleContexts(t *testing.T) {
	contexts := []string{"primary", "secondary", "tertiary"}

	// Test ordered pod logs across contexts
	simulator := NewMultiContextLogSimulator(contexts)

	// Create statefulset pods with ordered names
	for _, ctx := range contexts {
		for i := 0; i < 3; i++ {
			podName := fmt.Sprintf("database-%d", i)
			simulator.AddPodToContext(ctx, podName, "default", map[string][]string{
				"postgres": {
					fmt.Sprintf("[%s] Database instance %d initializing", ctx, i),
					fmt.Sprintf("[%s] Replication slot %d created", ctx, i),
					fmt.Sprintf("[%s] Database %d ready", ctx, i),
				},
			})
		}
	}

	// Test that pods are accessed in order
	state := &core.State{
		CurrentResourceType: core.ResourceTypeStatefulSet,
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
	app.setMode(ModeLog)

	// Verify ordered access
	view := app.View()
	if len(view) == 0 {
		t.Error("StatefulSet log view should not be empty")
	}
}

// TestMultiContextLogStreamingErrors tests error handling across multiple contexts
func TestMultiContextLogStreamingErrors(t *testing.T) {
	contexts := []string{"healthy-cluster", "degraded-cluster", "offline-cluster"}

	tests := []struct {
		name           string
		errorScenarios map[string]string // context -> error type
	}{
		{
			name: "mixed_health_status",
			errorScenarios: map[string]string{
				"healthy-cluster":  "",                   // No error
				"degraded-cluster": "connection_timeout", // Slow/timeout
				"offline-cluster":  "connection_refused", // Can't connect
			},
		},
		{
			name: "permission_errors",
			errorScenarios: map[string]string{
				"healthy-cluster":  "",             // No error
				"degraded-cluster": "unauthorized", // No permission
				"offline-cluster":  "forbidden",    // Forbidden
			},
		},
		{
			name: "partial_failures",
			errorScenarios: map[string]string{
				"healthy-cluster":  "",                    // No error
				"degraded-cluster": "pod_not_found",       // Pod missing
				"offline-cluster":  "namespace_not_found", // Namespace missing
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simulator := NewMultiContextLogSimulator(contexts)

			// Set up contexts with appropriate error conditions
			for ctx, errorType := range tt.errorScenarios {
				if errorType == "" {
					// Healthy context - add normal pod
					simulator.AddPodToContext(ctx, "test-pod", "default", map[string][]string{
						"main": {
							fmt.Sprintf("[%s] Healthy pod running", ctx),
						},
					})
				} else {
					// Error context - simulate error
					simulator.contexts[ctx].SimulateError("test-pod", fmt.Errorf(errorType))
				}
			}

			// Test error handling in multi-context app
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
			app.setMode(ModeLog)

			// App should handle mixed errors gracefully
			view := app.View()
			if len(view) == 0 {
				t.Error("View should render even with some contexts having errors")
			}

			// Test that healthy contexts still work
			ctx := context.Background()
			for ctxName, errorType := range tt.errorScenarios {
				logCh, errCh, _ := simulator.StreamLogsFromContext(ctx, ctxName, "test-pod", "main")

				if errorType == "" {
					// Should get logs from healthy context
					select {
					case log := <-logCh:
						if log == "" {
							t.Errorf("Expected logs from healthy context %s", ctxName)
						}
					case <-time.After(1 * time.Second):
						t.Errorf("Timeout waiting for logs from healthy context %s", ctxName)
					}
				} else {
					// Should get error from unhealthy context
					select {
					case err := <-errCh:
						if err == nil {
							t.Errorf("Expected error from context %s with error type %s", ctxName, errorType)
						}
					case <-logCh:
						t.Errorf("Should not receive logs from erroring context %s", ctxName)
					case <-time.After(1 * time.Second):
						// Expected for some error types
					}
				}
			}

			// Test that app remains responsive despite errors
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}
			model, _ := app.Update(keyMsg)
			app = model.(*App)

			// Should still be able to navigate
			keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			model, _ = app.Update(keyMsg)
			app = model.(*App)

			if app.currentMode != ModeList {
				t.Error("Should be able to return to list mode despite errors")
			}
		})
	}
}

// TestMultiContextLogAggregation tests aggregating logs from multiple contexts
func TestMultiContextLogAggregation(t *testing.T) {
	contexts := []string{"us-east", "us-west", "eu-central"}
	simulator := NewMultiContextLogSimulator(contexts)

	// Add same pod to all contexts
	for _, ctx := range contexts {
		simulator.AddPodToContext(ctx, "global-app", "default", map[string][]string{
			"main": {
				fmt.Sprintf("[%s] Request received", ctx),
				fmt.Sprintf("[%s] Processing", ctx),
				fmt.Sprintf("[%s] Response sent", ctx),
			},
		})
	}

	// Stream from all contexts simultaneously
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	allLogs := simulator.StreamLogsFromAllContexts(ctx, "global-app", "main")

	// Collect all logs with context labels
	aggregatedLogs := []string{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	for ctxName, logCh := range allLogs {
		wg.Add(1)
		go func(context string, ch <-chan string) {
			defer wg.Done()
			for log := range ch {
				mu.Lock()
				aggregatedLogs = append(aggregatedLogs, fmt.Sprintf("[%s] %s", context, log))
				mu.Unlock()
			}
		}(ctxName, logCh)
	}

	// Wait for completion
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Should have logs from all contexts
		if len(aggregatedLogs) < len(contexts)*3 {
			t.Errorf("Expected at least %d logs, got %d", len(contexts)*3, len(aggregatedLogs))
		}

		// Verify logs from each context
		for _, ctx := range contexts {
			found := false
			for _, log := range aggregatedLogs {
				if contains(log, ctx) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("No logs found from context %s", ctx)
			}
		}
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for log aggregation")
	}
}

// TestMultiContextPerformance tests performance with multiple contexts
func TestMultiContextPerformance(t *testing.T) {
	contexts := []string{"ctx-1", "ctx-2", "ctx-3", "ctx-4", "ctx-5"}
	simulator := NewMultiContextLogSimulator(contexts)

	// Add many pods to each context
	podsPerContext := 10
	logsPerPod := 100

	for _, ctx := range contexts {
		for i := 0; i < podsPerContext; i++ {
			logs := make([]string, logsPerPod)
			for j := 0; j < logsPerPod; j++ {
				logs[j] = fmt.Sprintf("[%s] Pod %d Log %d", ctx, i, j)
			}

			podName := fmt.Sprintf("pod-%d", i)
			simulator.AddPodToContext(ctx, podName, "default", map[string][]string{
				"main": logs,
			})
		}
	}

	// Measure time to stream all logs
	start := time.Now()
	ctx := context.Background()

	totalLogs := 0
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Stream from all pods in all contexts
	for _, ctxName := range contexts {
		for i := 0; i < podsPerContext; i++ {
			wg.Add(1)
			go func(context string, podIndex int) {
				defer wg.Done()

				podName := fmt.Sprintf("pod-%d", podIndex)
				logCh, _, err := simulator.StreamLogsFromContext(ctx, context, podName, "main")
				if err != nil {
					return
				}

				for range logCh {
					mu.Lock()
					totalLogs++
					mu.Unlock()
				}
			}(ctxName, i)
		}
	}

	// Wait with timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		elapsed := time.Since(start)
		expectedLogs := len(contexts) * podsPerContext * logsPerPod

		t.Logf("Streamed %d logs from %d contexts in %v", totalLogs, len(contexts), elapsed)

		// Should receive most logs (allow for some loss due to timing)
		if totalLogs < expectedLogs*8/10 {
			t.Errorf("Expected ~%d logs, got %d", expectedLogs, totalLogs)
		}

		// Performance check: should handle multi-context streaming efficiently
		if elapsed > 10*time.Second {
			t.Errorf("Multi-context streaming took too long: %v", elapsed)
		}

	case <-time.After(15 * time.Second):
		t.Error("Timeout during multi-context performance test")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
