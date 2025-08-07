package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpViewInitialization(t *testing.T) {
	view := NewHelpView()

	if view == nil {
		t.Fatal("NewHelpView returned nil")
	}

	if view.contextMode != "resource" {
		t.Errorf("contextMode = %q, want %q", view.contextMode, "resource")
	}
}

func TestHelpViewSetContext(t *testing.T) {
	view := NewHelpView()

	// Set to logs context
	view.SetContext("logs")
	if view.contextMode != "logs" {
		t.Errorf("contextMode = %q, want %q", view.contextMode, "logs")
	}

	// Set back to resource context
	view.SetContext("resource")
	if view.contextMode != "resource" {
		t.Errorf("contextMode = %q, want %q", view.contextMode, "resource")
	}
}

func TestHelpViewInit(t *testing.T) {
	view := NewHelpView()
	cmd := view.Init()

	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

func TestHelpViewWindowResize(t *testing.T) {
	view := NewHelpView()

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	model, _ := view.Update(msg)
	view = model.(*HelpView)

	if view.width != 100 || view.height != 50 {
		t.Errorf("size = (%d, %d), want (100, 50)", view.width, view.height)
	}
}

func TestHelpViewResourceHelp(t *testing.T) {
	view := NewHelpView()
	view.SetContext("resource")
	view.width = 80
	view.height = 24

	output := view.View()

	expectedContent := []string{
		"KubeWatch TUI - Resource View Help",
		"Navigation",
		"Move up",
		"Move down",
		"Next resource type",
		"Previous resource type",
		"Change namespace",
		"Switch contexts",
		"Actions",
		"View logs",
		"Delete selected",
		"Manual refresh",
		"Cycle sort",
		"Toggle word wrap",
		"General",
		"Toggle help",
		"Quit",
		"Close dialog",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("resource help should contain %q", expected)
		}
	}
}

func TestHelpViewLogHelp(t *testing.T) {
	view := NewHelpView()
	view.SetContext("logs")
	view.width = 80
	view.height = 24

	output := view.View()

	expectedContent := []string{
		"KubeWatch TUI - Log View Help",
		"Navigation",
		"Scroll up",
		"Scroll down",
		"Page up",
		"Page down",
		"Jump to top",
		"Jump to bottom",
		"Log Controls",
		"Toggle follow mode",
		"Search in logs",
		"Cycle containers",
		"Cycle pods",
		"Clear log buffer",
		"General",
		"Close logs",
		"Toggle help",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("log help should contain %q", expected)
		}
	}
}

func TestHelpViewKeyBindings(t *testing.T) {
	tests := []struct {
		name         string
		context      string
		wantBindings []string
	}{
		{
			name:    "resource view bindings",
			context: "resource",
			wantBindings: []string{
				"↑/k",
				"↓/j",
				"Tab",
				"S-Tab",
				"n",
				"c",
				"Enter/l",
				"Del/D",
				"r",
				"s",
				"u",
				"?",
				"q",
				"Esc",
			},
		},
		{
			name:    "log view bindings",
			context: "logs",
			wantBindings: []string{
				"↑/k",
				"↓/j",
				"PgUp",
				"PgDn",
				"Home/g",
				"End/G",
				"f",
				"/",
				"c",
				"p",
				"C",
				"Esc/q",
				"?",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewHelpView()
			view.SetContext(tt.context)
			view.width = 80
			view.height = 24

			output := view.View()

			for _, binding := range tt.wantBindings {
				if !strings.Contains(output, binding) {
					t.Errorf("%s help should show binding %q", tt.context, binding)
				}
			}
		})
	}
}

func TestHelpViewRendering(t *testing.T) {
	view := NewHelpView()
	view.width = 80
	view.height = 24

	// Test resource mode
	view.SetContext("resource")
	output := view.View()
	if output == "" {
		t.Error("resource help should not be empty")
	}

	// Test logs mode
	view.SetContext("logs")
	output = view.View()
	if output == "" {
		t.Error("log help should not be empty")
	}

	// Test that output is centered
	if !strings.Contains(output, "KubeWatch TUI") {
		t.Error("help should contain title")
	}
}

func TestHelpViewSetModeHelp(t *testing.T) {
	view := NewHelpView()

	// This method is a placeholder for future functionality
	// Just ensure it doesn't crash
	view.SetModeHelp(struct{}{})

	// Should still render normally
	output := view.View()
	if output == "" {
		t.Error("should still render after SetModeHelp")
	}
}

func TestHelpViewUpdate(t *testing.T) {
	view := NewHelpView()

	// Test with non-window size message
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	model, cmd := view.Update(msg)

	if cmd != nil {
		t.Error("Update should return nil command for key messages")
	}

	if model != view {
		t.Error("Update should return the same view for non-window messages")
	}
}

func TestHelpViewEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*HelpView)
		validate func(*testing.T, *HelpView)
	}{
		{
			name: "renders with zero size",
			setup: func(v *HelpView) {
				v.width = 0
				v.height = 0
			},
			validate: func(t *testing.T, v *HelpView) {
				output := v.View()
				if output == "" {
					t.Error("should still render with zero size")
				}
			},
		},
		{
			name: "renders with very large size",
			setup: func(v *HelpView) {
				v.width = 10000
				v.height = 10000
			},
			validate: func(t *testing.T, v *HelpView) {
				output := v.View()
				if output == "" {
					t.Error("should render with large size")
				}
			},
		},
		{
			name: "handles unknown context",
			setup: func(v *HelpView) {
				v.contextMode = "unknown"
			},
			validate: func(t *testing.T, v *HelpView) {
				output := v.View()
				// Should default to resource help
				if !strings.Contains(output, "Resource View Help") {
					t.Error("should default to resource help for unknown context")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewHelpView()
			tt.setup(view)
			tt.validate(t, view)
		})
	}
}
