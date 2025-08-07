package dropdown

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDropdownBasicFunctionality(t *testing.T) {
	options := []Option{
		{Label: "Option 1", Value: "value1"},
		{Label: "Option 2", Value: "value2"},
		{Label: "Option 3", Value: "value3"},
	}

	model := New(options)

	// Test initial state
	if model.IsOpen() {
		t.Error("Dropdown should be closed initially")
	}

	if model.GetSelectedIndex() != 0 {
		t.Error("Initial selected index should be 0")
	}

	selectedOption := model.GetSelectedOption()
	if selectedOption.Label != "Option 1" || selectedOption.Value != "value1" {
		t.Error("Initial selected option should be the first option")
	}
}

func TestDropdownNavigation(t *testing.T) {
	options := []Option{
		{Label: "Option 1", Value: "value1"},
		{Label: "Option 2", Value: "value2"},
		{Label: "Option 3", Value: "value3"},
	}

	model := New(options)
	model.Open()

	// Test down navigation
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if model.GetSelectedIndex() != 1 {
		t.Errorf("Expected selected index 1, got %d", model.GetSelectedIndex())
	}

	// Test up navigation
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	if model.GetSelectedIndex() != 0 {
		t.Errorf("Expected selected index 0, got %d", model.GetSelectedIndex())
	}

	// Test wrap around (up from first item)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	if model.GetSelectedIndex() != 2 {
		t.Errorf("Expected selected index 2 (wrap around), got %d", model.GetSelectedIndex())
	}

	// Test wrap around (down from last item)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if model.GetSelectedIndex() != 0 {
		t.Errorf("Expected selected index 0 (wrap around), got %d", model.GetSelectedIndex())
	}
}

func TestDropdownSelection(t *testing.T) {
	options := []Option{
		{Label: "Option 1", Value: "value1"},
		{Label: "Option 2", Value: "value2"},
		{Label: "Option 3", Value: "value3"},
	}

	model := New(options)
	model.Open()

	// Navigate to second option
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Select the option
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Check that dropdown is closed
	if model.IsOpen() {
		t.Error("Dropdown should be closed after selection")
	}

	// Check that command returns SelectedMsg
	if cmd == nil {
		t.Error("Expected command to be returned")
	} else {
		msg := cmd()
		if selectedMsg, ok := msg.(SelectedMsg); ok {
			if selectedMsg.Index != 1 {
				t.Errorf("Expected selected index 1, got %d", selectedMsg.Index)
			}
			if selectedMsg.Option.Label != "Option 2" {
				t.Errorf("Expected selected option 'Option 2', got '%s'", selectedMsg.Option.Label)
			}
		} else {
			t.Error("Expected SelectedMsg")
		}
	}
}

func TestDropdownCancel(t *testing.T) {
	options := []Option{
		{Label: "Option 1", Value: "value1"},
		{Label: "Option 2", Value: "value2"},
	}

	model := New(options)
	model.Open()

	// Cancel the dropdown
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Check that dropdown is closed
	if model.IsOpen() {
		t.Error("Dropdown should be closed after cancel")
	}

	// Check that command returns CancelledMsg
	if cmd == nil {
		t.Error("Expected command to be returned")
	} else {
		msg := cmd()
		if _, ok := msg.(CancelledMsg); !ok {
			t.Error("Expected CancelledMsg")
		}
	}
}

func TestDropdownSetSelectedValue(t *testing.T) {
	options := []Option{
		{Label: "Option 1", Value: "value1"},
		{Label: "Option 2", Value: "value2"},
		{Label: "Option 3", Value: "value3"},
	}

	model := New(options)

	// Set selected value
	model.SetSelectedValue("value2")

	if model.GetSelectedIndex() != 1 {
		t.Errorf("Expected selected index 1, got %d", model.GetSelectedIndex())
	}

	selectedOption := model.GetSelectedOption()
	if selectedOption.Value != "value2" {
		t.Errorf("Expected selected value 'value2', got '%v'", selectedOption.Value)
	}
}

func TestDropdownIgnoresKeysWhenClosed(t *testing.T) {
	options := []Option{
		{Label: "Option 1", Value: "value1"},
		{Label: "Option 2", Value: "value2"},
	}

	model := New(options)
	// Don't open the dropdown

	initialIndex := model.GetSelectedIndex()

	// Try to navigate
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Index should not change
	if model.GetSelectedIndex() != initialIndex {
		t.Error("Dropdown should ignore keys when closed")
	}
}

func TestDropdownView(t *testing.T) {
	options := []Option{
		{Label: "Option 1", Value: "value1"},
		{Label: "Option 2", Value: "value2"},
	}

	model := New(options)

	// View should be empty when closed
	view := model.View()
	if view != "" {
		t.Error("View should be empty when dropdown is closed")
	}

	// Open dropdown
	model.Open()
	view = model.View()
	if view == "" {
		t.Error("View should not be empty when dropdown is open")
	}

	// View should contain option labels
	if !containsString(view, "Option 1") || !containsString(view, "Option 2") {
		t.Error("View should contain option labels")
	}
}

func TestDropdownWithTitle(t *testing.T) {
	options := []Option{
		{Label: "Option 1", Value: "value1"},
	}

	model := New(options)
	model.SetTitle("Select Resource Type")
	model.Open()

	view := model.View()
	if !containsString(view, "Select Resource Type") {
		t.Error("View should contain title")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
