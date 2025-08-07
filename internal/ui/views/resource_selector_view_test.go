package views

import (
	"testing"

	"github.com/HamStudy/kubewatch/internal/components/dropdown"
	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

func TestResourceSelectorView_BasicFunctionality(t *testing.T) {
	view := NewResourceSelectorView()

	// Test initial state
	if view.IsOpen() {
		t.Error("Resource selector should be closed initially")
	}

	// Test opening
	view.Open()
	if !view.IsOpen() {
		t.Error("Resource selector should be open after calling Open()")
	}

	// Test closing
	view.Close()
	if view.IsOpen() {
		t.Error("Resource selector should be closed after calling Close()")
	}
}

func TestResourceSelectorView_SetCurrentResourceType(t *testing.T) {
	view := NewResourceSelectorView()

	// Set current resource type
	view.SetCurrentResourceType(core.ResourceTypeDeployment)

	// Get selected option
	selectedOption := view.GetSelectedOption()
	if selectedOption.Value != core.ResourceTypeDeployment {
		t.Errorf("Expected selected resource type to be %v, got %v",
			core.ResourceTypeDeployment, selectedOption.Value)
	}
}

func TestResourceSelectorView_Navigation(t *testing.T) {
	view := NewResourceSelectorView()
	view.Open()

	// Test down navigation
	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	selectedOption := view.GetSelectedOption()
	if selectedOption.Value != core.ResourceTypeDeployment {
		t.Errorf("Expected selected resource type to be %v after down navigation, got %v",
			core.ResourceTypeDeployment, selectedOption.Value)
	}

	// Test up navigation (should wrap to last item)
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	selectedOption = view.GetSelectedOption()
	if selectedOption.Value != core.ResourceTypePod {
		t.Errorf("Expected selected resource type to be %v after up navigation, got %v",
			core.ResourceTypePod, selectedOption.Value)
	}
}

func TestResourceSelectorView_Selection(t *testing.T) {
	view := NewResourceSelectorView()
	view.Open()

	// Navigate to second option
	view.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Select the option
	_, cmd := view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Check that dropdown is closed
	if view.IsOpen() {
		t.Error("Resource selector should be closed after selection")
	}

	// Check that command returns SelectedMsg
	if cmd == nil {
		t.Error("Expected command to be returned")
	} else {
		msg := cmd()
		if selectedMsg, ok := msg.(dropdown.SelectedMsg); ok {
			if selectedMsg.Option.Value != core.ResourceTypeDeployment {
				t.Errorf("Expected selected resource type to be %v, got %v",
					core.ResourceTypeDeployment, selectedMsg.Option.Value)
			}
		} else {
			t.Error("Expected dropdown.SelectedMsg")
		}
	}
}

func TestResourceSelectorView_Cancel(t *testing.T) {
	view := NewResourceSelectorView()
	view.Open()

	// Cancel the selection
	_, cmd := view.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Check that dropdown is closed
	if view.IsOpen() {
		t.Error("Resource selector should be closed after cancel")
	}

	// Check that command returns CancelledMsg
	if cmd == nil {
		t.Error("Expected command to be returned")
	} else {
		msg := cmd()
		if _, ok := msg.(dropdown.CancelledMsg); !ok {
			t.Error("Expected dropdown.CancelledMsg")
		}
	}
}

func TestResourceSelectorView_ViewRendering(t *testing.T) {
	view := NewResourceSelectorView()

	// View should be empty when closed
	viewStr := view.View()
	if viewStr != "" {
		t.Error("View should be empty when resource selector is closed")
	}

	// Open and check view is not empty
	view.Open()
	viewStr = view.View()
	if viewStr == "" {
		t.Error("View should not be empty when resource selector is open")
	}
}

func TestResourceSelectorView_AllResourceTypes(t *testing.T) {
	view := NewResourceSelectorView()

	expectedTypes := []core.ResourceType{
		core.ResourceTypePod,
		core.ResourceTypeDeployment,
		core.ResourceTypeStatefulSet,
		core.ResourceTypeService,
		core.ResourceTypeIngress,
		core.ResourceTypeConfigMap,
		core.ResourceTypeSecret,
	}

	// Test that all resource types are available
	for i, expectedType := range expectedTypes {
		view.SetCurrentResourceType(expectedType)
		selectedOption := view.GetSelectedOption()
		if selectedOption.Value != expectedType {
			t.Errorf("Resource type %d: expected %v, got %v",
				i, expectedType, selectedOption.Value)
		}
	}
}
