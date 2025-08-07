package core

import (
	"testing"
)

// TestNewStateInitializesCurrentContext tests that NewState properly initializes
// the CurrentContext field from the config
func TestNewStateInitializesCurrentContext(t *testing.T) {
	tests := []struct {
		name            string
		configContext   string
		expectedContext string
	}{
		{
			name:            "Context from config is set in state",
			configContext:   "cluster1",
			expectedContext: "cluster1",
		},
		{
			name:            "Empty context from config",
			configContext:   "",
			expectedContext: "",
		},
		{
			name:            "Context with special characters",
			configContext:   "prod-cluster-us-west-2",
			expectedContext: "prod-cluster-us-west-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				CurrentContext:    tt.configContext,
				CurrentNamespace:  "default",
				RefreshInterval:   2,
				LogTailLines:      100,
				MaxResourcesShown: 500,
				ColorScheme:       "default",
			}

			state := NewState(config)

			if state.CurrentContext != tt.expectedContext {
				t.Errorf("Expected CurrentContext %q, got %q", tt.expectedContext, state.CurrentContext)
			}
		})
	}
}

// TestContextBugFix tests the specific bug that was reported:
// The UI was showing the wrong context because state.CurrentContext was not initialized
func TestContextBugFix(t *testing.T) {
	// Simulate the bug scenario:
	// User runs: kubewatch --context=cluster1
	// Config gets CurrentContext set to "cluster1"
	// But state.CurrentContext was not being initialized, so UI showed empty or wrong context

	config := &Config{
		CurrentContext:    "cluster1", // This comes from CLI flag processing
		CurrentNamespace:  "default",
		RefreshInterval:   2,
		LogTailLines:      100,
		MaxResourcesShown: 500,
		ColorScheme:       "default",
	}

	state := NewState(config)

	// Before the fix, state.CurrentContext would be empty
	// After the fix, it should be "cluster1"
	if state.CurrentContext != "cluster1" {
		t.Errorf("BUG: state.CurrentContext should be 'cluster1', got '%s'", state.CurrentContext)
		t.Error("This is the bug that caused the UI to show the wrong context")
	}

	// Verify that the UI would now display the correct context
	// (This simulates what resource_view.go line 1146 does)
	displayedContext := state.CurrentContext
	if displayedContext != "cluster1" {
		t.Errorf("UI would display wrong context: expected 'cluster1', got '%s'", displayedContext)
	}
}

// TestStateContextConsistency tests that the context is consistent throughout the state
func TestStateContextConsistency(t *testing.T) {
	config := &Config{
		CurrentContext:    "test-context",
		CurrentNamespace:  "test-namespace",
		RefreshInterval:   2,
		LogTailLines:      100,
		MaxResourcesShown: 500,
		ColorScheme:       "default",
	}

	state := NewState(config)

	// Verify all context-related fields are properly initialized
	if state.CurrentContext != "test-context" {
		t.Errorf("CurrentContext not set correctly: expected 'test-context', got '%s'", state.CurrentContext)
	}

	if state.CurrentNamespace != "test-namespace" {
		t.Errorf("CurrentNamespace not set correctly: expected 'test-namespace', got '%s'", state.CurrentNamespace)
	}

	// Verify that the config reference is maintained
	if state.config != config {
		t.Error("Config reference not maintained in state")
	}

	if state.config.CurrentContext != "test-context" {
		t.Errorf("Config context not preserved: expected 'test-context', got '%s'", state.config.CurrentContext)
	}
}
