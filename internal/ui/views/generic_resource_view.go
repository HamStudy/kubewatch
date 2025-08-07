package views

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/HamStudy/kubewatch/internal/config/resource"
	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/template"
)

// GenericResourceView is a generic view that uses the template-driven configuration system
type GenericResourceView struct {
	// Core dependencies
	state          *core.State
	registry       *resource.Registry
	templateEngine *template.Engine
	dynamicClient  dynamic.Interface

	// Current state
	currentResourceType string
	currentDefinition   *resource.ResourceDefinition
	resources           []*unstructured.Unstructured
	filteredResources   []*unstructured.Unstructured

	// UI components
	table  table.Model
	width  int
	height int
	ready  bool

	// Sorting and filtering
	sortColumn int
	sortAsc    bool
	filterText string

	// Selection tracking
	selectedIndex int
	selectedItems map[string]bool

	// Styles
	baseStyle     lipgloss.Style
	headerStyle   lipgloss.Style
	selectedStyle lipgloss.Style
}

// NewGenericResourceView creates a new generic resource view
func NewGenericResourceView(state *core.State, registry *resource.Registry, engine *template.Engine, dynamicClient dynamic.Interface) (*GenericResourceView, error) {
	if state == nil {
		return nil, fmt.Errorf("state cannot be nil")
	}

	v := &GenericResourceView{
		state:          state,
		registry:       registry,
		templateEngine: engine,
		dynamicClient:  dynamicClient,
		resources:      make([]*unstructured.Unstructured, 0),
		selectedItems:  make(map[string]bool),
		sortAsc:        true,
	}

	// Initialize styles
	v.initStyles()

	// Initialize table
	v.initTable()

	return v, nil
}

// initStyles initializes the view styles
func (v *GenericResourceView) initStyles() {
	v.baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	v.headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57"))

	v.selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57"))
}

// initTable initializes the table component
func (v *GenericResourceView) initTable() {
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Status", Width: 15},
		{Title: "Namespace", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = v.headerStyle
	s.Selected = v.selectedStyle
	t.SetStyles(s)

	v.table = t
}

// RefreshResources fetches resources using the dynamic client
func (v *GenericResourceView) RefreshResources(ctx context.Context) error {
	if v.currentResourceType == "" {
		return nil
	}

	// Get resource definition from registry
	def := v.registry.GetByName(v.currentResourceType)
	if def == nil {
		return fmt.Errorf("resource type %s not found", v.currentResourceType)
	}
	v.currentDefinition = def

	// Build GVR from definition
	gvr := schema.GroupVersionResource{
		Group:    def.Spec.Kubernetes.Group,
		Version:  def.Spec.Kubernetes.Version,
		Resource: def.Spec.Kubernetes.Plural,
	}

	// Determine namespace scope
	var resourceInterface dynamic.ResourceInterface
	if def.Spec.Kubernetes.Namespaced {
		if v.state.CurrentNamespace == "" {
			// All namespaces
			resourceInterface = v.dynamicClient.Resource(gvr)
		} else {
			// Specific namespace
			resourceInterface = v.dynamicClient.Resource(gvr).Namespace(v.state.CurrentNamespace)
		}
	} else {
		// Cluster-scoped resource
		resourceInterface = v.dynamicClient.Resource(gvr)
	}

	// List resources
	list, err := resourceInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list %s: %w", v.currentResourceType, err)
	}

	// Convert to unstructured slice
	v.resources = make([]*unstructured.Unstructured, len(list.Items))
	for i := range list.Items {
		v.resources[i] = &list.Items[i]
	}

	// Apply sorting and filtering
	if err := v.sortResources(); err != nil {
		return err
	}

	// Update table
	return v.updateTable()
}

// renderColumns renders columns for a resource using templates
func (v *GenericResourceView) renderColumns(resource *unstructured.Unstructured, definition *resource.ResourceDefinition) ([]string, error) {
	if resource == nil || definition == nil {
		return nil, fmt.Errorf("resource and definition cannot be nil")
	}

	columns := make([]string, len(definition.Spec.Columns))

	for i, col := range definition.Spec.Columns {
		// Execute template for this column
		result, err := v.templateEngine.Execute(col.Template, resource.Object)
		if err != nil {
			// Handle template errors gracefully
			columns[i] = fmt.Sprintf("<%s>", err.Error())
		} else {
			columns[i] = result
		}
	}

	return columns, nil
}

// updateTable builds table rows using templates
func (v *GenericResourceView) updateTable() error {
	if v.currentDefinition == nil {
		return nil
	}

	// Build columns from definition
	columns := make([]table.Column, len(v.currentDefinition.Spec.Columns))
	for i, col := range v.currentDefinition.Spec.Columns {
		width := 20 // Default width
		if col.Width > 0 {
			width = col.Width
		}
		columns[i] = table.Column{
			Title: col.Name,
			Width: width,
		}
	}

	// Apply filter
	v.filteredResources = v.applyFilter()

	// Build rows
	rows := make([]table.Row, 0, len(v.filteredResources))
	for _, res := range v.filteredResources {
		cols, err := v.renderColumns(res, v.currentDefinition)
		if err != nil {
			// Skip resources that fail to render
			continue
		}
		rows = append(rows, cols)
	}

	// Update table
	v.table.SetColumns(columns)
	v.table.SetRows(rows)

	// Adjust table height based on available space
	if v.height > 5 {
		v.table.SetHeight(v.height - 5)
	}

	return nil
}

// sortResources sorts the resources based on current sort settings
func (v *GenericResourceView) sortResources() error {
	if v.currentDefinition == nil || len(v.resources) == 0 {
		return nil
	}

	// Render all columns for sorting
	type resourceWithColumns struct {
		resource *unstructured.Unstructured
		columns  []string
	}

	items := make([]resourceWithColumns, 0, len(v.resources))
	for _, res := range v.resources {
		cols, err := v.renderColumns(res, v.currentDefinition)
		if err != nil {
			continue
		}
		items = append(items, resourceWithColumns{
			resource: res,
			columns:  cols,
		})
	}

	// Sort by the selected column
	sort.Slice(items, func(i, j int) bool {
		if v.sortColumn >= len(items[i].columns) || v.sortColumn >= len(items[j].columns) {
			return false
		}

		val1 := items[i].columns[v.sortColumn]
		val2 := items[j].columns[v.sortColumn]

		if v.sortAsc {
			return val1 < val2
		}
		return val1 > val2
	})

	// Update resources with sorted order
	v.resources = make([]*unstructured.Unstructured, len(items))
	for i, item := range items {
		v.resources[i] = item.resource
	}

	return nil
}

// applyFilter filters resources based on filter text
func (v *GenericResourceView) applyFilter() []*unstructured.Unstructured {
	if v.filterText == "" {
		return v.resources
	}

	filtered := make([]*unstructured.Unstructured, 0)
	filterLower := strings.ToLower(v.filterText)

	for _, res := range v.resources {
		// Render columns for filtering
		cols, err := v.renderColumns(res, v.currentDefinition)
		if err != nil {
			continue
		}

		// Check if any column contains the filter text
		match := false
		for _, col := range cols {
			if strings.Contains(strings.ToLower(col), filterLower) {
				match = true
				break
			}
		}

		if match {
			filtered = append(filtered, res)
		}
	}

	return filtered
}

// SetResourceType sets the current resource type
func (v *GenericResourceView) SetResourceType(resourceType string) error {
	v.currentResourceType = resourceType

	// Get definition to validate
	def := v.registry.GetByName(resourceType)
	if def == nil {
		return fmt.Errorf("resource type %s not found", resourceType)
	}
	v.currentDefinition = def
	// Clear current resources
	v.resources = make([]*unstructured.Unstructured, 0)
	v.filteredResources = make([]*unstructured.Unstructured, 0)

	// Refresh resources
	return v.RefreshResources(context.Background())
}

// SetFilter sets the filter text
func (v *GenericResourceView) SetFilter(filter string) {
	v.filterText = filter
	v.updateTable()
}

// SetSort sets the sort column and direction
func (v *GenericResourceView) SetSort(column int, ascending bool) {
	v.sortColumn = column
	v.sortAsc = ascending
	v.sortResources()
	v.updateTable()
}

// GetSelectedResource returns the currently selected resource
func (v *GenericResourceView) GetSelectedResource() *unstructured.Unstructured {
	if v.selectedIndex >= 0 && v.selectedIndex < len(v.filteredResources) {
		return v.filteredResources[v.selectedIndex]
	}
	return nil
}

// Init initializes the view
func (v *GenericResourceView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *GenericResourceView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.table.SetWidth(msg.Width)
		if msg.Height > 5 {
			v.table.SetHeight(msg.Height - 5)
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyF5:
			// Refresh resources
			return v, func() tea.Msg {
				err := v.RefreshResources(context.Background())
				if err != nil {
					return ErrorMsg{Error: err}
				}
				return RefreshCompleteMsg{}
			}

		case tea.KeyUp, tea.KeyDown:
			v.table, cmd = v.table.Update(msg)
			v.selectedIndex = v.table.Cursor()

		case tea.KeyEnter:
			// Handle selection
			if res := v.GetSelectedResource(); res != nil {
				name, _, _ := unstructured.NestedString(res.Object, "metadata", "name")
				if v.selectedItems[name] {
					delete(v.selectedItems, name)
				} else {
					v.selectedItems[name] = true
				}
			}
		}

	case ResourceUpdateMsg:
		// Handle resource type change
		v.currentResourceType = msg.ResourceType
		return v, func() tea.Msg {
			err := v.SetResourceType(msg.ResourceType)
			if err != nil {
				return ErrorMsg{Error: err}
			}
			return RefreshCompleteMsg{}
		}

	default:
		v.table, cmd = v.table.Update(msg)
	}

	return v, cmd
}

// View renders the view
func (v *GenericResourceView) View() string {
	if !v.ready {
		return "Loading..."
	}

	// Build status line
	status := fmt.Sprintf("Resources: %d", len(v.filteredResources))
	if v.filterText != "" {
		status += fmt.Sprintf(" | Filter: %s", v.filterText)
	}
	if v.currentDefinition != nil {
		status += fmt.Sprintf(" | Type: %s", v.currentResourceType)
	}

	// Render table with status
	return lipgloss.JoinVertical(
		lipgloss.Left,
		status,
		v.baseStyle.Render(v.table.View()),
	)
}

// Message types
type ErrorMsg struct {
	Error error
}

type RefreshCompleteMsg struct{}

type ResourceUpdateMsg struct {
	ResourceType string
}
