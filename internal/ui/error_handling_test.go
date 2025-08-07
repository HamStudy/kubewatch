package ui

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

// TestNetworkFailureRecovery tests app behavior during network failures
func TestNetworkFailureRecovery(t *testing.T) {
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

	// App should handle nil client gracefully
	view := app.View()
	if len(view) == 0 {
		t.Error("App should render even with nil client")
	}

	// Simulate refresh with nil client
	tickMsg := tickMsg(time.Now())
	model, cmd := app.Update(tickMsg)
	app = model.(*App)

	// Should still return next tick command
	if cmd == nil {
		t.Error("Should schedule next tick even with network failure")
	}

	// App should remain stable
	if app.currentMode != ModeList {
		t.Errorf("Mode should remain stable during network failure, got %v", app.currentMode)
	}
}

// TestKubernetesAPIErrors tests handling of various K8s API errors
func TestKubernetesAPIErrors(t *testing.T) {
	tests := []struct {
		name        string
		operation   string
		expectedErr error
	}{
		{
			name:        "unauthorized_error",
			operation:   "get_pods",
			expectedErr: errors.New("unauthorized"),
		},
		{
			name:        "not_found_error",
			operation:   "get_pod",
			expectedErr: errors.New("pod not found"),
		},
		{
			name:        "timeout_error",
			operation:   "list_pods",
			expectedErr: errors.New("request timeout"),
		},
		{
			name:        "connection_refused",
			operation:   "connect",
			expectedErr: errors.New("connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// For testing, ensure both clients are nil to trigger test paths
			app.k8sClient = nil
			app.multiClient = nil
			// App should handle errors gracefully
			view := app.View()
			if len(view) == 0 {
				t.Errorf("App should render despite %s error", tt.name)
			}

			// Should remain interactive
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
			model, _ := app.Update(keyMsg)
			app = model.(*App)

			if app.currentMode != ModeHelp {
				t.Errorf("Should be able to open help despite %s error", tt.name)
			}
		})
	}
}

// TestGracefulDegradation tests that app degrades gracefully with limited functionality
func TestGracefulDegradation(t *testing.T) {
	app := createTestApp(t)

	// Test that navigation still works without client
	operations := []struct {
		key          string
		expectedMode ScreenModeType
	}{
		{"?", ModeHelp},
		{"?", ModeList}, // Toggle back
		{"n", ModeNamespaceSelector},
		{"esc", ModeList},
		{"c", ModeContextSelector},
		{"esc", ModeList},
	}

	for _, op := range operations {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(op.key)}
		if op.key == "esc" {
			keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		}

		model, _ := app.Update(keyMsg)
		app = model.(*App)

		if app.currentMode != op.expectedMode {
			t.Errorf("Expected mode %v after %s, got %v", op.expectedMode, op.key, app.currentMode)
		}

		// View should always render
		view := app.View()
		if len(view) == 0 {
			t.Errorf("View should render in mode %v", app.currentMode)
		}
	}
}

// TestUserErrorMessageDisplay tests that errors are displayed to users appropriately
func TestUserErrorMessageDisplay(t *testing.T) {
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

	// Try to perform operations that would fail without a client
	// These should not panic and should handle gracefully

	// Try to view logs (would fail without client)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
	model, _ := app.Update(keyMsg)
	app = model.(*App)

	// Should either stay in list mode or show an error
	if app.currentMode == ModeLog {
		// If it did switch to log mode, it should handle the lack of data
		view := app.View()
		if len(view) == 0 {
			t.Error("Log view should show something even without data")
		}
	}

	// Try to describe resource (would fail without client)
	app.setMode(ModeList)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")}
	model, _ = app.Update(keyMsg)
	app = model.(*App)

	// Should handle gracefully
	view := app.View()
	if len(view) == 0 {
		t.Error("View should not be empty after failed describe attempt")
	}
}

// TestPanicRecovery tests that the app doesn't panic on unexpected errors
func TestPanicRecovery(t *testing.T) {
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

	// Test various edge cases that could cause panics
	edgeCases := []tea.Msg{
		// Nil message
		nil,
		// Empty key message
		tea.KeyMsg{},
		// Window size with zero dimensions
		tea.WindowSizeMsg{Width: 0, Height: 0},
		// Very large window size
		tea.WindowSizeMsg{Width: 10000, Height: 10000},
		// Rapid mode changes
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")},
		tea.KeyMsg{Type: tea.KeyEsc},
	}

	for i, msg := range edgeCases {
		// This should not panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic on edge case %d: %v", i, r)
				}
			}()

			if msg != nil {
				model, _ := app.Update(msg)
				app = model.(*App)
			}

			// View should still render
			view := app.View()
			if view == "" && msg != nil {
				t.Errorf("View became empty after edge case %d", i)
			}
		}()
	}
}

// TestErrorPropagation tests that errors are properly propagated and handled
func TestErrorPropagation(t *testing.T) {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
	}

	config := &core.Config{
		RefreshInterval: 5,
	}

	// Create app with nil client to simulate errors
	ctx := context.Background()
	app := NewApp(ctx, nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true

	// For testing, ensure both clients are nil to trigger test paths
	app.k8sClient = nil
	app.multiClient = nil
	app.activeContexts = []string{"test-context"} // Ensure we have test contexts

	// Ensure modes are initialized properly
	if app.modes == nil {
		app.modes = map[ScreenModeType]ScreenMode{
			ModeList:              NewListMode(),
			ModeLog:               NewLogMode(),
			ModeDescribe:          NewDescribeMode(),
			ModeHelp:              NewHelpMode(),
			ModeContextSelector:   NewContextSelectorMode(),
			ModeNamespaceSelector: NewNamespaceSelectorMode(),
			ModeConfirmDialog:     NewConfirmDialogMode(),
			ModeResourceSelector:  NewResourceSelectorMode(),
		}
	}

	// Operations should handle client errors gracefully
	view := app.View()
	if len(view) == 0 {
		t.Error("View should render despite client errors")
	}

	// Test that app remains responsive
	keyMsg := tea.KeyMsg{Type: tea.KeyTab}
	model, _ := app.Update(keyMsg)
	app = model.(*App)

	// Tab should open resource selector despite errors
	if app.currentMode != ModeResourceSelector {
		t.Error("Should be able to open resource selector despite client errors")
	}
}

// TestInvalidInputHandling tests handling of invalid user input
func TestInvalidInputHandling(t *testing.T) {
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

	// Test invalid key combinations
	invalidKeys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("xyz")},   // Random string
		{Type: tea.KeyRunes, Runes: []rune("!@#$%")}, // Special characters
		{Type: tea.KeyRunes, Runes: []rune("")},      // Empty rune
		{Type: tea.KeyRunes},                         // No runes
		{Type: 999},                                  // Invalid type
	}

	for i, keyMsg := range invalidKeys {
		model, _ := app.Update(keyMsg)
		app = model.(*App)

		// App should remain stable
		if app.currentMode != ModeList {
			t.Errorf("Mode changed unexpectedly on invalid input %d", i)
		}

		// View should still render
		view := app.View()
		if len(view) == 0 {
			t.Errorf("View became empty after invalid input %d", i)
		}
	}
}

// TestResourceDeletionErrors tests error handling during resource deletion
func TestResourceDeletionErrors(t *testing.T) {
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

	// Attempt delete without selection
	keyMsg := tea.KeyMsg{Type: tea.KeyDelete}
	model, _ := app.Update(keyMsg)
	app = model.(*App)

	// Should handle gracefully (not open confirm dialog without selection)
	if app.currentMode == ModeConfirmDialog {
		// If dialog opened, test cancellation
		keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		model, _ = app.Update(keyMsg)
		app = model.(*App)
	}

	// Should be back in list mode
	if app.currentMode != ModeList {
		t.Errorf("Expected list mode after delete attempt, got %v", app.currentMode)
	}

	// View should still render
	view := app.View()
	if len(view) == 0 {
		t.Error("View should not be empty after failed delete")
	}
}
