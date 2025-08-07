package views

import (
	"strings"
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

func TestResourceSelectorView_CenteredModal(t *testing.T) {
	view := NewResourceSelectorView()
	view.SetSize(100, 50) // Large screen size
	view.Open()
	
	viewStr := view.View()
	if viewStr == "" {
		t.Error("View should not be empty when open")
	}
	
	// The view should be centered - check for proper spacing
	lines := strings.Split(viewStr, "\n")
	
	// Find lines with actual content (not just empty lines)
	var contentLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			contentLines = append(contentLines, line)
		}
	}
	
	if len(contentLines) == 0 {
		t.Error("Should have content lines in the modal")
	}
	
	// Check that the modal is centered by verifying leading whitespace
	hasLeadingSpace := false
	minLeadingSpaces := 999
	for _, line := range contentLines {
		leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))
		if leadingSpaces > 0 {
			hasLeadingSpace = true
			if leadingSpaces < minLeadingSpaces {
				minLeadingSpaces = leadingSpaces
			}
		}
	}
	
	if !hasLeadingSpace {
		t.Error("Modal should be centered with leading whitespace")
	}
	
	// The modal should have reasonable centering (at least a few spaces)
	if minLeadingSpaces < 5 {
		t.Errorf("Modal should be more centered, only has %d leading spaces", minLeadingSpaces)
	}
}

func TestResourceSelectorView_AutoSizedModal(t *testing.T) {
	view := NewResourceSelectorView()
	view.SetSize(100, 50) // Large screen size
	view.Open()
	
	viewStr := view.View()
	if viewStr == "" {
		t.Error("View should not be empty when open")
	}
	
	// Find the actual modal content (trimmed lines)
	lines := strings.Split(viewStr, "\n")
	var contentLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "╭") && !strings.HasPrefix(trimmed, "╰") {
			// Skip border lines, focus on actual content
			contentLines = append(contentLines, trimmed)
		}
	}
	
	if len(contentLines) == 0 {
		t.Error("Should have content lines")
	}
	
	// Check that the modal content is reasonably sized
	maxContentWidth := 0
	for _, line := range contentLines {
		if len(line) > maxContentWidth {
			maxContentWidth = len(line)
		}
	}
	
	// The modal should be auto-sized - not too wide, not too narrow
	if maxContentWidth < 20 {
		t.Errorf("Modal content appears too narrow (%d chars)", maxContentWidth)
	}
	
	// Focus on content width, not border width (due to lipgloss quirks)
	if maxContentWidth > 40 {
		t.Errorf("Modal content appears too wide (%d chars), expected auto-sized modal", maxContentWidth)
	}
	
	// Verify that the modal contains the expected resource types
	viewContent := strings.Join(contentLines, " ")
	expectedTypes := []string{"Pods", "Deployments", "Services"}
	for _, expectedType := range expectedTypes {
		if !strings.Contains(viewContent, expectedType) {
			t.Errorf("Modal should contain resource type '%s'", expectedType)
		}
	}
}

func TestResourceSelectorView_LeftAlignedContent(t *testing.T) {
	view := NewResourceSelectorView()
	view.Open()
	
	viewStr := view.View()
	if viewStr == "" {
		t.Error("View should not be empty when open")
	}
	
	// The content within the modal should be left-aligned
	lines := strings.Split(viewStr, "\n")
	
	// Look for option lines (they should contain resource type names)
	optionLines := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Pods") || 
		   strings.Contains(trimmed, "Deployments") || 
		   strings.Contains(trimmed, "Services") {
			optionLines = append(optionLines, trimmed)
		}
	}
	
	if len(optionLines) == 0 {
		t.Error("Should find option lines with resource types")
	}
	
	// Options should be left-aligned within their container
	// Check that the content appears properly formatted
	for _, line := range optionLines {
		if len(strings.TrimLeft(line, " \t")) == 0 {
			t.Error("Option lines should have actual content, not just whitespace")
		}
		
		// Check that the line contains the expected format (with border characters)
		if !strings.Contains(line, "│") {
			t.Error("Option lines should be properly formatted with borders")
		}
	}
}

func TestResourceSelectorView_ImprovedLayout(t *testing.T) {
	view := NewResourceSelectorView()
	view.SetSize(100, 50)
	view.Open()
	
	viewStr := view.View()
	if viewStr == "" {
		t.Error("View should not be empty when open")
	}
	
	// This test verifies the key improvements:
	// 1. Modal is centered (not full-width)
	// 2. Modal is auto-sized (content-based width)
	// 3. Content is left-aligned within the modal
	
	lines := strings.Split(viewStr, "\n")
	
	// Find content lines (excluding empty lines)
	var contentLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			contentLines = append(contentLines, line)
		}
	}
	
	if len(contentLines) == 0 {
		t.Error("Should have content lines")
	}
	
	// Verify centering: content should not start at position 0
	firstContentLine := contentLines[0]
	if !strings.HasPrefix(firstContentLine, " ") {
		t.Error("Modal should be centered, not left-aligned to screen edge")
	}
	
	// Verify auto-sizing: modal should not be full-width
	// Check that the content area (excluding centering padding) is reasonable
	// Find a content line (not border line)
	var contentLine string
	for _, line := range contentLines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "│") && !strings.HasPrefix(trimmed, "╭") && !strings.HasPrefix(trimmed, "╰") {
			contentLine = trimmed
			break
		}
	}
	if contentLine == "" {
		t.Error("Should find content line with actual text")
		return
	}
	if len(contentLine) > 50 {
		t.Errorf("Modal content appears too wide (%d chars), expected auto-sized", len(contentLine))
	}
	
	// Verify content contains expected resource types
	fullContent := strings.Join(contentLines, " ")
	expectedTypes := []string{"Select Resource Type", "Pods", "Deployments", "Services"}
	for _, expectedType := range expectedTypes {
		if !strings.Contains(fullContent, expectedType) {
			t.Errorf("Modal should contain '%s'", expectedType)
		}
	}
	
	// Verify the modal is not using the old full-width approach
	// (The old approach would have lines close to 100 characters)
	maxLineLength := 0
	for _, line := range contentLines {
		if len(line) > maxLineLength {
			maxLineLength = len(line)
		}
	}
	
	// The modal should be significantly smaller than full screen width
	if maxLineLength > 95 {
		t.Errorf("Modal appears to be full-width (%d chars), expected centered modal", maxLineLength)
	}
}
