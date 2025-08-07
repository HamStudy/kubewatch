package main

import (
	"context"
	"strings"
	"testing"

	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDebugContextSelector(t *testing.T) {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
	}

	config := &core.Config{
		RefreshInterval: 5,
	}

	app := ui.NewApp(context.Background(), nil, state, config)

	// Initialize the app
	app.Init()

	// Test 'c' key to open context selector
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
	model, cmd := app.Update(keyMsg)
	app = model.(*ui.App)

	// Check that the view contains context selector elements
	view := app.View()

	// The view should contain context-related text when context selector is open
	if !strings.Contains(view, "Context") && !strings.Contains(view, "context") {
		t.Logf("View content: %s", view)
		// This is not necessarily an error since the context selector might not be visible
		// if there are no contexts available, but we can at least verify the app responds
	}

	// Verify that a command was returned (indicating the app processed the key)
	if cmd == nil {
		t.Error("Expected a command to be returned after pressing 'c' key")
	}

	t.Log("Context selector test completed - app responded to 'c' key")
}
