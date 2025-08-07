package ui

import (
	"context"
	"testing"

	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDebugKeyHandling(t *testing.T) {
	// Create a test app
	config := &core.Config{
		RefreshInterval: 30,
	}
	state := core.NewState(config)
	app := NewApp(context.Background(), nil, state, config)

	// Test the mode system directly
	listMode := NewListMode()
	bindings := listMode.GetKeyBindings()

	// Check if tab binding exists
	if tabBinding, exists := bindings["tab"]; exists {
		t.Logf("Tab binding found: %+v", tabBinding)
	} else {
		t.Error("Tab binding not found in list mode")
	}

	// Test key matching
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	handled, cmd := listMode.HandleKey(tabMsg, app)

	t.Logf("Tab key handled: %v, command: %v", handled, cmd != nil)
	t.Logf("App mode after handling: %v", app.currentMode)

	// Execute the command if it exists
	if cmd != nil {
		msg := cmd()
		t.Logf("Command executed, returned message: %T", msg)
	}

	t.Logf("App mode after command execution: %v", app.currentMode)

	if !handled {
		t.Error("Tab key should be handled by list mode")
	}
}
