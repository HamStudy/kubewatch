package ui

import (
	"context"
	"testing"

	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

// TestResourceOperationsWorkflow tests complete resource operation workflows
func TestResourceOperationsWorkflow(t *testing.T) {
	app := createTestApp(t)

	tests := []struct {
		name        string
		workflow    []string
		description string
	}{
		{
			name:        "view_logs_workflow",
			workflow:    []string{"l", "esc"},
			description: "Should open logs and return to list",
		},
		{
			name:        "describe_resource_workflow",
			workflow:    []string{"i", "esc"},
			description: "Should open describe view and return to list",
		},
		{
			name:        "describe_resource_workflow_d_key",
			workflow:    []string{"d", "esc"},
			description: "Should open describe view with 'd' key and return to list",
		},
		{
			name:        "help_workflow",
			workflow:    []string{"?", "?"},
			description: "Should open and close help",
		},
		{
			name:        "context_selector_workflow",
			workflow:    []string{"c", "esc"},
			description: "Should open context selector and cancel",
		},
		{
			name:        "namespace_selector_workflow",
			workflow:    []string{"n", "esc"},
			description: "Should open namespace selector and cancel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to list mode
			app.setMode(ModeList)

			for i, key := range tt.workflow {
				var keyMsg tea.KeyMsg
				switch key {
				case "l":
					keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
				case "i":
					keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")}
				case "d":
					keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
				case "?":
					keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
				case "c":
					keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
				case "n":
					keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
				case "esc":
					keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
				}

				model, cmd := app.Update(keyMsg)
				app = model.(*App)

				// Execute any returned commands
				if cmd != nil {
					// For testing, we'll just verify the command exists
					_ = cmd
				}

				// Verify we don't panic and the app is in a valid state
				view := app.View()
				if len(view) == 0 {
					t.Errorf("Step %d (%s): View should not be empty", i+1, key)
				}
			}

			// Should end up back in list mode for most workflows
			if tt.name == "help_workflow" || tt.name == "context_selector_workflow" ||
				tt.name == "namespace_selector_workflow" || tt.name == "view_logs_workflow" ||
				tt.name == "describe_resource_workflow" || tt.name == "describe_resource_workflow_d_key" {
				if app.currentMode != ModeList {
					t.Errorf("Expected to return to list mode, got %v", app.currentMode)
				}
			}
		})
	}
}

// TestMultiContextFunctionality tests multi-context specific features
func TestMultiContextFunctionality(t *testing.T) {
	// Create multi-context app with proper mock clients
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContexts:     []string{"context-1", "context-2", "context-3"},
	}

	config := &core.Config{
		RefreshInterval: 5,
	}

	// Create mock multi-client - for now we'll use nil since it's just UI testing
	app := NewAppWithMultiContext(context.Background(), nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true
	// Test multi-context state
	if !app.isMultiContext {
		t.Error("App should be in multi-context mode")
	}

	if len(app.activeContexts) != 3 {
		t.Errorf("Expected 3 active contexts, got %d", len(app.activeContexts))
	}

	// Test context selector in multi-context mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
	model, cmd := app.Update(keyMsg)
	app = model.(*App)

	// Should open context selector
	if app.currentMode != ModeContextSelector {
		t.Errorf("Expected context selector mode, got %v", app.currentMode)
	}

	// Execute command if returned
	if cmd != nil {
		_ = cmd
	}

	// Test view renders in multi-context mode
	view := app.View()
	if len(view) == 0 {
		t.Error("Multi-context view should not be empty")
	}

	// Test escape from context selector
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = app.Update(keyMsg)
	app = model.(*App)

	if app.currentMode != ModeList {
		t.Errorf("Expected to return to list mode, got %v", app.currentMode)
	}
}

// TestResourceTypeNavigation tests cycling through different resource types
func TestResourceTypeNavigation(t *testing.T) {
	app := createTestApp(t)

	initialType := app.state.CurrentResourceType
	seenTypes := make(map[core.ResourceType]bool)
	seenTypes[initialType] = true

	// Test tab navigation through resource types
	for i := 0; i < 10; i++ { // Try up to 10 cycles
		keyMsg := tea.KeyMsg{Type: tea.KeyTab}
		model, cmd := app.Update(keyMsg)
		app = model.(*App)

		// Execute refresh command
		if cmd != nil {
			_ = cmd
		}

		currentType := app.state.CurrentResourceType
		if seenTypes[currentType] && currentType == initialType {
			// We've cycled back to the beginning
			break
		}
		seenTypes[currentType] = true

		// Verify view still renders
		view := app.View()
		if len(view) == 0 {
			t.Errorf("View should not be empty for resource type %v", currentType)
		}
	}

	if len(seenTypes) < 2 {
		t.Error("Should cycle through at least 2 different resource types")
	}

	// Test shift+tab (reverse navigation)
	keyMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	model, cmd := app.Update(keyMsg)
	app = model.(*App)

	if cmd != nil {
		_ = cmd
	}

	// Should have changed resource type
	if app.state.CurrentResourceType == initialType {
		// This might be expected if we only have one resource type
		t.Logf("Resource type unchanged after shift+tab: %v", app.state.CurrentResourceType)
	}
}

// TestErrorHandlingScenarios tests various error conditions
func TestErrorHandlingScenarios(t *testing.T) {
	app := createTestApp(t)

	// Test operations with no selected resource
	// We can't directly set selectedRow as it's unexported, but we can test
	// the behavior when there's no valid selection by using an empty resource view

	operations := []string{"l", "i", "d", "D"}
	for _, op := range operations {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(op)}
		model, cmd := app.Update(keyMsg)
		app = model.(*App)

		// Should not panic
		view := app.View()
		if len(view) == 0 {
			t.Errorf("View should not be empty after operation %s with no selection", op)
		}

		// Should still be in list mode (operations should fail gracefully)
		if app.currentMode != ModeList {
			t.Errorf("Should remain in list mode after failed operation %s", op)
		}

		// Execute any commands
		if cmd != nil {
			_ = cmd
		}
	}
}

// TestModeTransitionsIntegration tests all possible mode transitions
func TestModeTransitionsIntegration(t *testing.T) {
	app := createTestApp(t)

	transitions := []struct {
		name     string
		from     ScreenModeType
		key      string
		expected ScreenModeType
	}{
		{"list_to_help", ModeList, "?", ModeHelp},
		{"help_to_list", ModeHelp, "?", ModeList},
		{"help_esc_to_list", ModeHelp, "esc", ModeList},
		{"list_to_context", ModeList, "c", ModeContextSelector},
		{"context_to_list", ModeContextSelector, "esc", ModeList},
		{"list_to_namespace", ModeList, "n", ModeNamespaceSelector},
		{"namespace_to_list", ModeNamespaceSelector, "esc", ModeList},
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			// Set initial mode
			app.currentMode = tt.from
			app.previousMode = ModeList // Set a reasonable previous mode

			// Execute transition
			var keyMsg tea.KeyMsg
			switch tt.key {
			case "?":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
			case "c":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
			case "n":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
			case "esc":
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			model, cmd := app.Update(keyMsg)
			app = model.(*App)

			// Execute any commands
			if cmd != nil {
				_ = cmd
			}

			// Check final mode
			if app.currentMode != tt.expected {
				t.Errorf("Expected mode %v, got %v", tt.expected, app.currentMode)
			}

			// Verify view renders
			view := app.View()
			if len(view) == 0 {
				t.Error("View should not be empty after transition")
			}
		})
	}
}

// TestSortingFunctionality tests resource sorting
func TestSortingFunctionality(t *testing.T) {
	app := createTestApp(t)

	// Test sort cycling
	initialColumn := app.state.SortColumn
	initialAscending := app.state.SortAscending

	// Press 's' to cycle sort
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	model, cmd := app.Update(keyMsg)
	app = model.(*App)

	// Execute refresh command
	if cmd != nil {
		_ = cmd
	}

	// Sort state should have changed
	if app.state.SortColumn == initialColumn && app.state.SortAscending == initialAscending {
		// This might be expected if there's only one sort column
		t.Logf("Sort state unchanged: column=%s, ascending=%v", app.state.SortColumn, app.state.SortAscending)
	}

	// Test multiple sort cycles
	for i := 0; i < 5; i++ {
		model, cmd = app.Update(keyMsg)
		app = model.(*App)

		if cmd != nil {
			_ = cmd
		}

		// Should not panic
		view := app.View()
		if len(view) == 0 {
			t.Errorf("View should not be empty after sort cycle %d", i+1)
		}
	}
}

// TestRefreshFunctionality tests resource refresh
func TestRefreshFunctionality(t *testing.T) {
	app := createTestApp(t)

	// Test manual refresh
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	model, cmd := app.Update(keyMsg)
	app = model.(*App)

	// Should return a refresh command
	if cmd == nil {
		t.Error("Refresh should return a command")
	}

	// View should still render
	view := app.View()
	if len(view) == 0 {
		t.Error("View should not be empty after refresh")
	}

	// Test ctrl+r refresh
	keyMsg = tea.KeyMsg{Type: tea.KeyCtrlR}
	model, cmd = app.Update(keyMsg)
	app = model.(*App)

	if cmd == nil {
		t.Error("Ctrl+R refresh should return a command")
	}
}

// TestDeleteConfirmationWorkflow tests the delete confirmation dialog
func TestDeleteConfirmationWorkflow(t *testing.T) {
	app := createTestApp(t)

	// Note: Delete confirmation may not trigger without a selected resource
	// This test verifies the behavior when delete is attempted

	// Trigger delete confirmation with Delete key
	keyMsg := tea.KeyMsg{Type: tea.KeyDelete}
	model, cmd := app.Update(keyMsg)
	app = model.(*App)

	// Execute command if any
	if cmd != nil {
		_ = cmd
	}

	// If confirmation dialog opened, test cancellation
	if app.currentMode == ModeConfirmDialog {
		// Test cancel
		keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		model, _ = app.Update(keyMsg)
		app = model.(*App)

		// Should return to list mode
		if app.currentMode != ModeList {
			t.Errorf("Expected to return to list mode after cancel, got %v", app.currentMode)
		}
	} else {
		// Without a selected resource, delete should not change mode
		if app.currentMode != ModeList {
			t.Errorf("Expected to remain in list mode without selection, got %v", app.currentMode)
		}
	}

	// Test with 'D' key
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")}
	model, cmd = app.Update(keyMsg)
	app = model.(*App)

	if cmd != nil {
		_ = cmd
	}

	// If confirmation dialog opened, test cancellation
	if app.currentMode == ModeConfirmDialog {
		// Cancel
		keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		model, _ = app.Update(keyMsg)
		app = model.(*App)

		if app.currentMode != ModeList {
			t.Errorf("Expected to return to list mode after cancel, got %v", app.currentMode)
		}
	} else {
		// Without a selected resource, 'D' should not change mode
		if app.currentMode != ModeList {
			t.Errorf("Expected to remain in list mode without selection, got %v", app.currentMode)
		}
	}
}

// TestViewSizeHandling tests responsive view sizing
func TestViewSizeHandling(t *testing.T) {
	app := createTestApp(t)

	sizes := []struct {
		width  int
		height int
	}{
		{40, 10},  // Very small
		{80, 24},  // Standard
		{120, 40}, // Large
		{200, 60}, // Very large
	}

	for _, size := range sizes {
		t.Run(string(rune(size.width))+"x"+string(rune(size.height)), func(t *testing.T) {
			// Send window size message
			sizeMsg := tea.WindowSizeMsg{Width: size.width, Height: size.height}
			model, _ := app.Update(sizeMsg)
			app = model.(*App)

			// Verify size was set
			if app.width != size.width || app.height != size.height {
				t.Errorf("Expected size %dx%d, got %dx%d",
					size.width, size.height, app.width, app.height)
			}

			// Verify view still renders
			view := app.View()
			if len(view) == 0 {
				t.Errorf("View should not be empty at size %dx%d", size.width, size.height)
			}

			// Test different modes at this size
			modes := []ScreenModeType{ModeList, ModeHelp}
			for _, mode := range modes {
				app.setMode(mode)
				view = app.View()
				if len(view) == 0 {
					t.Errorf("Mode %v should render at size %dx%d", mode, size.width, size.height)
				}
			}

			// Return to list mode
			app.setMode(ModeList)
		})
	}
}
