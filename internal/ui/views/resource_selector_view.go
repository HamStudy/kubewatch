package views

import (
	"strings"

	"github.com/HamStudy/kubewatch/internal/components/dropdown"
	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

// ResourceSelectorView provides a dropdown for selecting resource types
type ResourceSelectorView struct {
	dropdown dropdown.Model
	width    int
	height   int
}

// NewResourceSelectorView creates a new resource selector view
func NewResourceSelectorView() *ResourceSelectorView {
	// Create options for all resource types
	options := []dropdown.Option{
		{Label: "Pods", Value: core.ResourceTypePod},
		{Label: "Deployments", Value: core.ResourceTypeDeployment},
		{Label: "StatefulSets", Value: core.ResourceTypeStatefulSet},
		{Label: "Services", Value: core.ResourceTypeService},
		{Label: "Ingresses", Value: core.ResourceTypeIngress},
		{Label: "ConfigMaps", Value: core.ResourceTypeConfigMap},
		{Label: "Secrets", Value: core.ResourceTypeSecret},
	}

	// Calculate optimal width based on content
	maxLabelWidth := 0
	for _, option := range options {
		if len(option.Label) > maxLabelWidth {
			maxLabelWidth = len(option.Label)
		}
	}
	
	// Add padding for borders and selection indicators
	optimalWidth := maxLabelWidth + 8 // Account for borders, padding, and styling
	if optimalWidth < 25 {
		optimalWidth = 25 // Minimum width for good appearance
	}

	dropdownModel := dropdown.New(options)
	dropdownModel.SetTitle("Select Resource Type")
	dropdownModel.SetSize(optimalWidth, 10)

	return &ResourceSelectorView{
		dropdown: dropdownModel,
		width:    80,  // Screen width (will be set by app)
		height:   24,  // Screen height (will be set by app)
	}
}

// SetSize sets the view dimensions (screen size for centering)
func (v *ResourceSelectorView) SetSize(width, height int) {
	v.width = width
	v.height = height
	// Keep the dropdown at its optimal size - don't resize it
}

// SetCurrentResourceType sets the currently selected resource type
func (v *ResourceSelectorView) SetCurrentResourceType(resourceType core.ResourceType) {
	v.dropdown.SetSelectedValue(resourceType)
}

// Open opens the dropdown
func (v *ResourceSelectorView) Open() {
	v.dropdown.Open()
}

// Close closes the dropdown
func (v *ResourceSelectorView) Close() {
	v.dropdown.Close()
}

// IsOpen returns whether the dropdown is open
func (v *ResourceSelectorView) IsOpen() bool {
	return v.dropdown.IsOpen()
}

// GetSelectedOption returns the currently selected option
func (v *ResourceSelectorView) GetSelectedOption() dropdown.Option {
	return v.dropdown.GetSelectedOption()
}

// Init initializes the view
func (v *ResourceSelectorView) Init() tea.Cmd {
	return v.dropdown.Init()
}

// Update handles messages
func (v *ResourceSelectorView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	v.dropdown, cmd = v.dropdown.Update(msg)
	return v, cmd
}

// View renders the view
func (v *ResourceSelectorView) View() string {
	if !v.dropdown.IsOpen() {
		return ""
	}

	// Get the dropdown content
	dropdownView := v.dropdown.View()
	if dropdownView == "" {
		return ""
	}

	// Calculate the actual width of the dropdown content
	lines := strings.Split(dropdownView, "\n")
	dropdownWidth := 0
	for _, line := range lines {
		if len(line) > dropdownWidth {
			dropdownWidth = len(line)
		}
	}
	dropdownHeight := len(lines)

	// Calculate centering position
	leftPadding := (v.width - dropdownWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}
	
	topPadding := (v.height - dropdownHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Create the centered view
	var result strings.Builder
	
	// Add top padding
	for i := 0; i < topPadding; i++ {
		result.WriteString("\n")
	}
	
	// Add the dropdown with left padding
	for i, line := range lines {
		// Add left padding
		result.WriteString(strings.Repeat(" ", leftPadding))
		result.WriteString(line)
		
		// Add newline except for the last line
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// ResourceSelectedMsg is sent when a resource type is selected
type ResourceSelectedMsg struct {
	ResourceType core.ResourceType
}

// ResourceSelectorCancelledMsg is sent when the selector is cancelled
type ResourceSelectorCancelledMsg struct{}
