package views

import (
	"github.com/HamStudy/kubewatch/internal/components/dropdown"
	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	dropdownModel := dropdown.New(options)
	dropdownModel.SetTitle("Select Resource Type")
	dropdownModel.SetSize(25, 10)

	return &ResourceSelectorView{
		dropdown: dropdownModel,
		width:    25,
		height:   10,
	}
}

// SetSize sets the view dimensions
func (v *ResourceSelectorView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.dropdown.SetSize(width, height)
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

	// Center the dropdown on screen
	dropdownView := v.dropdown.View()

	// Calculate centering
	containerStyle := lipgloss.NewStyle().
		Width(v.width).
		Height(v.height).
		Align(lipgloss.Center, lipgloss.Center)

	return containerStyle.Render(dropdownView)
}

// ResourceSelectedMsg is sent when a resource type is selected
type ResourceSelectedMsg struct {
	ResourceType core.ResourceType
}

// ResourceSelectorCancelledMsg is sent when the selector is cancelled
type ResourceSelectorCancelledMsg struct{}
