package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"
)

// NamespaceView displays a list of namespaces for selection
type NamespaceView struct {
	namespaces       []v1.Namespace
	filteredItems    []v1.Namespace
	selectedIndex    int
	filter           string
	width            int
	height           int
	currentNamespace string
	loading          bool
	loadingMessage   string
}

// NewNamespaceView creates a new namespace selector view
func NewNamespaceView(namespaces []v1.Namespace, currentNamespace string) *NamespaceView {
	nv := &NamespaceView{
		namespaces:       namespaces,
		filteredItems:    namespaces,
		currentNamespace: currentNamespace,
	}

	// Add "all" option at the beginning
	allNs := v1.Namespace{}
	allNs.Name = "all"
	nv.namespaces = append([]v1.Namespace{allNs}, namespaces...)
	nv.filteredItems = nv.namespaces

	// Pre-select the current namespace
	if currentNamespace == "" || currentNamespace == "all" {
		nv.selectedIndex = 0
	} else {
		for i, ns := range nv.namespaces {
			if ns.Name == currentNamespace {
				nv.selectedIndex = i
				break
			}
		}
	}

	return nv
}

// NewNamespaceViewWithLoading creates a new namespace selector view in loading state
func NewNamespaceViewWithLoading(currentNamespace string, loadingMessage string) *NamespaceView {
	return &NamespaceView{
		namespaces:       []v1.Namespace{},
		filteredItems:    []v1.Namespace{},
		currentNamespace: currentNamespace,
		loading:          true,
		loadingMessage:   loadingMessage,
	}
}

// SetLoading sets the loading state
func (v *NamespaceView) SetLoading(loading bool, message string) {
	v.loading = loading
	v.loadingMessage = message
}

// SetNamespaces updates the namespaces and clears loading state
func (v *NamespaceView) SetNamespaces(namespaces []v1.Namespace) {
	v.loading = false
	v.loadingMessage = ""
	v.namespaces = namespaces

	// Add "all" option at the beginning
	allNs := v1.Namespace{}
	allNs.Name = "all"
	v.namespaces = append([]v1.Namespace{allNs}, namespaces...)
	v.filteredItems = v.namespaces

	// Pre-select the current namespace
	if v.currentNamespace == "" || v.currentNamespace == "all" {
		v.selectedIndex = 0
	} else {
		for i, ns := range v.namespaces {
			if ns.Name == v.currentNamespace {
				v.selectedIndex = i
				break
			}
		}
	}
}

// Init initializes the view
func (v *NamespaceView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *NamespaceView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if v.selectedIndex > 0 {
				v.selectedIndex--
			}
		case "down", "j":
			if v.selectedIndex < len(v.filteredItems)-1 {
				v.selectedIndex++
			}
		case "home":
			v.selectedIndex = 0
		case "end":
			v.selectedIndex = len(v.filteredItems) - 1
		case "pgup":
			v.selectedIndex -= 10
			if v.selectedIndex < 0 {
				v.selectedIndex = 0
			}
		case "pgdown":
			v.selectedIndex += 10
			if v.selectedIndex >= len(v.filteredItems) {
				v.selectedIndex = len(v.filteredItems) - 1
			}
		case "/":
			// Start filtering
			v.filter = ""
		case "backspace":
			if len(v.filter) > 0 {
				v.filter = v.filter[:len(v.filter)-1]
				v.applyFilter()
			}
		case "esc":
			// Clear filter
			v.filter = ""
			v.applyFilter()
		case "enter":
			// Selection made - will be handled by parent
			return v, nil
		case "q":
			// Cancel - will be handled by parent
			return v, nil
		case "n":
			// Only treat 'n' as cancel when not filtering
			if v.filter == "" {
				return v, nil
			}
			// Otherwise, add it to the filter
			v.filter += msg.String()
			v.applyFilter()
		default:
			// Add to filter if it's a printable character
			if len(msg.String()) == 1 && msg.String()[0] >= 32 && msg.String()[0] < 127 {
				v.filter += msg.String()
				v.applyFilter()
			}
		}
	}
	return v, nil
}

// View renders the namespace selector
func (v *NamespaceView) View() string {
	// Create styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(50).
		Height(20)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)

	currentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true)

	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Italic(true)

	// Build content
	var content strings.Builder

	title := "Select Namespace"
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n\n")

	// Show loading state
	if v.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)

		spinnerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))

		message := v.loadingMessage
		if message == "" {
			message = "Loading namespaces..."
		}

		content.WriteString(loadingStyle.Render(message))
		content.WriteString("\n\n")
		content.WriteString(spinnerStyle.Render("⠋ Fetching from contexts..."))
		content.WriteString("\n\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Please wait..."))

		// Center the popup
		return lipgloss.Place(
			v.width,
			v.height,
			lipgloss.Center,
			lipgloss.Center,
			borderStyle.Render(content.String()),
		)
	}

	// Show filter if active
	if v.filter != "" {
		content.WriteString(filterStyle.Render(fmt.Sprintf("Filter: %s", v.filter)))
		content.WriteString("\n\n")
	}

	// Calculate visible range (show 15 items)
	visibleItems := 15
	startIdx := 0
	endIdx := len(v.filteredItems)

	// Adjust viewport to keep selection visible
	if v.selectedIndex >= visibleItems {
		startIdx = v.selectedIndex - visibleItems/2
		if startIdx < 0 {
			startIdx = 0
		}
	}

	if endIdx > startIdx+visibleItems {
		endIdx = startIdx + visibleItems
	}

	// List namespaces
	for i := startIdx; i < endIdx && i < len(v.filteredItems); i++ {
		ns := v.filteredItems[i]
		line := ns.Name

		// Add status for special namespaces
		if ns.Name == "all" {
			line = "📁 All Namespaces"
		} else if ns.Name == v.currentNamespace {
			line = fmt.Sprintf("• %s", line)
		} else {
			line = fmt.Sprintf("  %s", line)
		}

		// Apply styling
		if i == v.selectedIndex {
			line = selectedStyle.Render(fmt.Sprintf("→ %s", line))
		} else if ns.Name == v.currentNamespace && ns.Name != "all" {
			line = currentStyle.Render(line)
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	// Add scroll indicator if needed
	if len(v.filteredItems) > visibleItems {
		scrollInfo := fmt.Sprintf("\n[%d-%d of %d]", startIdx+1, endIdx, len(v.filteredItems))
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(scrollInfo))
	}

	// Add help text
	helpText := "\n\n[↑↓/jk] Navigate  [/] Filter  [Enter] Select  [Esc/q/n] Cancel"
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(helpText))

	// Center the popup
	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		borderStyle.Render(content.String()),
	)
}

// SetSize updates the view size
func (v *NamespaceView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// GetSelectedNamespace returns the selected namespace name
func (v *NamespaceView) GetSelectedNamespace() string {
	if v.selectedIndex >= 0 && v.selectedIndex < len(v.filteredItems) {
		ns := v.filteredItems[v.selectedIndex]
		if ns.Name == "all" {
			return "" // Empty string means all namespaces
		}
		return ns.Name
	}
	return v.currentNamespace
}

// applyFilter filters the namespace list based on the current filter string
func (v *NamespaceView) applyFilter() {
	if v.filter == "" {
		v.filteredItems = v.namespaces
		return
	}

	v.filteredItems = []v1.Namespace{}
	filterLower := strings.ToLower(v.filter)

	for _, ns := range v.namespaces {
		if strings.Contains(strings.ToLower(ns.Name), filterLower) {
			v.filteredItems = append(v.filteredItems, ns)
		}
	}

	// Reset selection to first item if current selection is out of bounds
	if v.selectedIndex >= len(v.filteredItems) {
		v.selectedIndex = 0
	}
}
