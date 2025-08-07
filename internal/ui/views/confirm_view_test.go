package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmViewInitialization(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		message     string
		wantDefault bool
	}{
		{
			name:        "creates confirm view with title and message",
			title:       "Delete Resource",
			message:     "Are you sure you want to delete this pod?",
			wantDefault: false, // Should default to No
		},
		{
			name:        "handles empty strings",
			title:       "",
			message:     "",
			wantDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewConfirmView(tt.title, tt.message)

			if view == nil {
				t.Fatal("NewConfirmView returned nil")
			}

			if view.title != tt.title {
				t.Errorf("title = %q, want %q", view.title, tt.title)
			}

			if view.message != tt.message {
				t.Errorf("message = %q, want %q", view.message, tt.message)
			}

			if view.confirmed != tt.wantDefault {
				t.Errorf("confirmed = %v, want %v", view.confirmed, tt.wantDefault)
			}

			if view.confirmText != "Yes" {
				t.Errorf("confirmText = %q, want %q", view.confirmText, "Yes")
			}

			if view.cancelText != "No" {
				t.Errorf("cancelText = %q, want %q", view.cancelText, "No")
			}
		})
	}
}

func TestConfirmViewKeyHandling(t *testing.T) {
	tests := []struct {
		name          string
		keys          []string
		wantConfirmed bool
	}{
		{
			name:          "left arrow toggles selection",
			keys:          []string{"left"},
			wantConfirmed: true, // Starts at false, toggles to true
		},
		{
			name:          "right arrow toggles selection",
			keys:          []string{"right"},
			wantConfirmed: true,
		},
		{
			name:          "h key toggles selection",
			keys:          []string{"h"},
			wantConfirmed: true,
		},
		{
			name:          "l key toggles selection",
			keys:          []string{"l"},
			wantConfirmed: true,
		},
		{
			name:          "tab toggles selection",
			keys:          []string{"tab"},
			wantConfirmed: true,
		},
		{
			name:          "Y key confirms",
			keys:          []string{"Y"},
			wantConfirmed: true,
		},
		{
			name:          "y key confirms",
			keys:          []string{"y"},
			wantConfirmed: true,
		},
		{
			name:          "N key cancels",
			keys:          []string{"N"},
			wantConfirmed: false,
		},
		{
			name:          "n key cancels",
			keys:          []string{"n"},
			wantConfirmed: false,
		},
		{
			name:          "q key cancels",
			keys:          []string{"q"},
			wantConfirmed: false,
		},
		{
			name:          "multiple toggles",
			keys:          []string{"left", "right", "tab"},
			wantConfirmed: true, // false -> true -> false -> true
		},
		{
			name:          "enter key maintains state",
			keys:          []string{"enter"},
			wantConfirmed: false, // No change
		},
		{
			name:          "space key maintains state",
			keys:          []string{" "},
			wantConfirmed: false, // No change
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewConfirmView("Test", "Confirm?")

			for _, key := range tt.keys {
				var msg tea.KeyMsg
				switch key {
				case "left":
					msg = tea.KeyMsg{Type: tea.KeyLeft}
				case "right":
					msg = tea.KeyMsg{Type: tea.KeyRight}
				case "tab":
					msg = tea.KeyMsg{Type: tea.KeyTab}
				case "enter":
					msg = tea.KeyMsg{Type: tea.KeyEnter}
				case " ":
					msg = tea.KeyMsg{Type: tea.KeySpace}
				default:
					msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
				}

				model, _ := view.Update(msg)
				view = model.(*ConfirmView)
			}

			if view.confirmed != tt.wantConfirmed {
				t.Errorf("confirmed = %v, want %v", view.confirmed, tt.wantConfirmed)
			}
		})
	}
}

func TestConfirmViewRendering(t *testing.T) {
	tests := []struct {
		name            string
		title           string
		message         string
		confirmed       bool
		confirmText     string
		cancelText      string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:      "renders title and message",
			title:     "Delete Pod",
			message:   "Are you sure?",
			confirmed: false,
			wantContains: []string{
				"Delete Pod",
				"Are you sure?",
				"No",
				"Yes",
			},
		},
		{
			name:         "highlights Yes when confirmed",
			title:        "Delete",
			message:      "Confirm deletion",
			confirmed:    true,
			wantContains: []string{"Yes", "No"},
		},
		{
			name:         "highlights No when not confirmed",
			title:        "Delete",
			message:      "Confirm deletion",
			confirmed:    false,
			wantContains: []string{"Yes", "No"},
		},
		{
			name:         "shows custom button text",
			title:        "Custom",
			message:      "Test",
			confirmed:    false,
			confirmText:  "Accept",
			cancelText:   "Decline",
			wantContains: []string{"Accept", "Decline"},
		},
		{
			name:      "shows help text",
			title:     "Help",
			message:   "Test",
			confirmed: false,
			wantContains: []string{
				"Switch",
				"Select",
				"Confirm",
				"Cancel",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewConfirmView(tt.title, tt.message)
			view.confirmed = tt.confirmed
			view.SetSize(80, 24)

			if tt.confirmText != "" {
				view.SetConfirmText(tt.confirmText)
			}
			if tt.cancelText != "" {
				view.SetCancelText(tt.cancelText)
			}

			output := view.View()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output does not contain %q", want)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if strings.Contains(output, notWant) {
					t.Errorf("output should not contain %q", notWant)
				}
			}
		})
	}
}

func TestConfirmViewSetters(t *testing.T) {
	view := NewConfirmView("Test", "Message")

	// Test SetConfirmText
	view.SetConfirmText("Proceed")
	if view.confirmText != "Proceed" {
		t.Errorf("SetConfirmText failed: got %q, want %q", view.confirmText, "Proceed")
	}

	// Test SetCancelText
	view.SetCancelText("Abort")
	if view.cancelText != "Abort" {
		t.Errorf("SetCancelText failed: got %q, want %q", view.cancelText, "Abort")
	}

	// Test SetSize
	view.SetSize(100, 30)
	if view.width != 100 || view.height != 30 {
		t.Errorf("SetSize failed: got (%d, %d), want (100, 30)", view.width, view.height)
	}

	// Test IsConfirmed
	view.confirmed = true
	if !view.IsConfirmed() {
		t.Error("IsConfirmed should return true")
	}

	view.confirmed = false
	if view.IsConfirmed() {
		t.Error("IsConfirmed should return false")
	}
}

func TestConfirmViewInit(t *testing.T) {
	view := NewConfirmView("Test", "Message")
	cmd := view.Init()

	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

func TestConfirmViewEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*ConfirmView)
		update   tea.Msg
		validate func(*testing.T, *ConfirmView)
	}{
		{
			name:   "handles unknown key messages",
			setup:  func(v *ConfirmView) {},
			update: tea.KeyMsg{Type: tea.KeyF1},
			validate: func(t *testing.T, v *ConfirmView) {
				if v.confirmed != false {
					t.Error("unknown key should not change state")
				}
			},
		},
		{
			name:   "handles non-key messages",
			setup:  func(v *ConfirmView) {},
			update: tea.WindowSizeMsg{Width: 100, Height: 50},
			validate: func(t *testing.T, v *ConfirmView) {
				// Should not crash
			},
		},
		{
			name: "renders with zero size",
			setup: func(v *ConfirmView) {
				v.SetSize(0, 0)
			},
			update: nil,
			validate: func(t *testing.T, v *ConfirmView) {
				output := v.View()
				if output == "" {
					t.Error("should still render with zero size")
				}
			},
		},
		{
			name: "renders with very large size",
			setup: func(v *ConfirmView) {
				v.SetSize(10000, 10000)
			},
			update: nil,
			validate: func(t *testing.T, v *ConfirmView) {
				output := v.View()
				if output == "" {
					t.Error("should render with large size")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewConfirmView("Test", "Message")

			if tt.setup != nil {
				tt.setup(view)
			}

			if tt.update != nil {
				model, _ := view.Update(tt.update)
				view = model.(*ConfirmView)
			}

			if tt.validate != nil {
				tt.validate(t, view)
			}
		})
	}
}
