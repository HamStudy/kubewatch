package ui

import (
	"context"
	"testing"

	"github.com/HamStudy/kubewatch/internal/components/dropdown"
	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

func TestResourceSelectorIntegration(t *testing.T) {
	// Create a test app
	config := &core.Config{
		RefreshInterval: 30,
	}
	state := core.NewState(config)
	app := NewApp(context.Background(), nil, state, config)

	// Test opening resource selector
	cmd := app.openResourceSelector()
	if cmd != nil {
		t.Error("openResourceSelector should return nil command")
	}

	if app.currentMode != ModeResourceSelector {
		t.Errorf("Expected mode to be ModeResourceSelector, got %v", app.currentMode)
	}

	if !app.resourceSelectorView.IsOpen() {
		t.Error("Resource selector view should be open")
	}

	// Test that the current resource type is set correctly
	selectedOption := app.resourceSelectorView.GetSelectedOption()
	if selectedOption.Value != state.CurrentResourceType {
		t.Errorf("Expected selected resource type to be %v, got %v",
			state.CurrentResourceType, selectedOption.Value)
	}
}

func TestResourceSelectorSelection(t *testing.T) {
	// Create a test app
	config := &core.Config{
		RefreshInterval: 30,
	}
	state := core.NewState(config)
	app := NewApp(context.Background(), nil, state, config)

	// Open resource selector
	app.openResourceSelector()

	// Simulate selecting a different resource type
	selectedMsg := dropdown.SelectedMsg{
		Option: dropdown.Option{
			Label: "Deployments",
			Value: core.ResourceTypeDeployment,
		},
		Index: 1,
	}

	// Process the selection message
	model, cmd := app.Update(selectedMsg)
	app = model.(*App)

	// Check that the resource type was changed
	if state.CurrentResourceType != core.ResourceTypeDeployment {
		t.Errorf("Expected resource type to be %v, got %v",
			core.ResourceTypeDeployment, state.CurrentResourceType)
	}

	// Check that mode was switched back to list
	if app.currentMode != ModeList {
		t.Errorf("Expected mode to be ModeList, got %v", app.currentMode)
	}

	// Check that refresh command was returned
	if cmd == nil {
		t.Error("Expected refresh command to be returned")
	}
}

func TestResourceSelectorCancel(t *testing.T) {
	// Create a test app
	config := &core.Config{
		RefreshInterval: 30,
	}
	state := core.NewState(config)
	originalResourceType := state.CurrentResourceType
	app := NewApp(context.Background(), nil, state, config)

	// Open resource selector
	app.openResourceSelector()

	// Simulate cancelling the selection
	cancelMsg := dropdown.CancelledMsg{}

	// Process the cancel message
	model, cmd := app.Update(cancelMsg)
	app = model.(*App)

	// Check that the resource type was not changed
	if state.CurrentResourceType != originalResourceType {
		t.Errorf("Expected resource type to remain %v, got %v",
			originalResourceType, state.CurrentResourceType)
	}

	// Check that mode was switched back to list
	if app.currentMode != ModeList {
		t.Errorf("Expected mode to be ModeList, got %v", app.currentMode)
	}

	// Check that no command was returned
	if cmd != nil {
		t.Error("Expected no command to be returned on cancel")
	}
}

func TestTabKeyOpensResourceSelector(t *testing.T) {
	// Create a test app
	config := &core.Config{
		RefreshInterval: 30,
	}
	state := core.NewState(config)
	app := NewApp(context.Background(), nil, state, config)

	// Simulate pressing Tab key
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}

	// Process the key message
	model, cmd := app.Update(tabMsg)
	app = model.(*App)

	// Check that resource selector was opened
	if app.currentMode != ModeResourceSelector {
		t.Errorf("Expected mode to be ModeResourceSelector, got %v", app.currentMode)
	}

	// Check that resource selector view exists and is open
	if app.resourceSelectorView == nil {
		t.Error("Resource selector view should not be nil")
	} else if !app.resourceSelectorView.IsOpen() {
		t.Error("Resource selector view should be open")
	}

	// Check that no command was returned (opening is synchronous)
	if cmd != nil {
		t.Error("Expected no command to be returned when opening resource selector")
	}
}

func TestShiftTabKeyOpensResourceSelector(t *testing.T) {
	// Create a test app
	config := &core.Config{
		RefreshInterval: 30,
	}
	state := core.NewState(config)
	app := NewApp(context.Background(), nil, state, config)

	// Simulate pressing Shift+Tab key
	shiftTabMsg := tea.KeyMsg{Type: tea.KeyShiftTab}

	// Process the key message
	model, cmd := app.Update(shiftTabMsg)
	app = model.(*App)

	// Check that resource selector was opened
	if app.currentMode != ModeResourceSelector {
		t.Errorf("Expected mode to be ModeResourceSelector, got %v", app.currentMode)
	}

	// Check that resource selector view exists and is open
	if app.resourceSelectorView == nil {
		t.Error("Resource selector view should not be nil")
	} else if !app.resourceSelectorView.IsOpen() {
		t.Error("Resource selector view should be open")
	}

	// Check that no command was returned (opening is synchronous)
	if cmd != nil {
		t.Error("Expected no command to be returned when opening resource selector")
	}
}
