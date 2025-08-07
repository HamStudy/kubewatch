package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ContextView displays available Kubernetes contexts for selection
type ContextView struct {
	contexts         []string
	selectedContexts map[string]bool // For multi-select
	currentIndex     int
	width            int
	height           int
	searchQuery      string
	SearchMode       bool
	multiSelect      bool            // Toggle between single and multi-select mode
	loading          bool            // Show loading indicator
	loadingContexts  map[string]bool // Track which contexts are loading
	showingInfo      bool            // Whether currently showing context info
	infoContext      string          // Context name for which info is being shown
}

// NewContextView creates a new context selector view
func NewContextView(contexts []string, currentContexts []string) *ContextView {
	selectedMap := make(map[string]bool)
	for _, ctx := range currentContexts {
		selectedMap[ctx] = true
	}

	return &ContextView{
		contexts:         contexts,
		selectedContexts: selectedMap,
		currentIndex:     0,
		multiSelect:      len(currentContexts) > 1,
		loading:          false,
		loadingContexts:  make(map[string]bool),
	}
}

// Init initializes the view
func (v *ContextView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *ContextView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height

	case tea.KeyMsg:
		if v.SearchMode {
			switch msg.Type {
			case tea.KeyEscape:
				v.SearchMode = false
				v.searchQuery = ""
			case tea.KeyEnter:
				v.SearchMode = false
			case tea.KeyBackspace:
				if len(v.searchQuery) > 0 {
					v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
				}
			default:
				if msg.Type == tea.KeyRunes {
					v.searchQuery += string(msg.Runes)
				}
			}
			// Block all other keys while in search mode
			return v, nil
		}

		// Handle special keys first
		if msg.Type == tea.KeySpace {
			// Toggle selection for current context
			visibleContexts := v.getVisibleContexts()
			if v.currentIndex < len(visibleContexts) {
				ctx := visibleContexts[v.currentIndex]
				if v.multiSelect {
					v.selectedContexts[ctx] = !v.selectedContexts[ctx]
				} else {
					// Single select - clear others and select current
					v.selectedContexts = make(map[string]bool)
					v.selectedContexts[ctx] = true
				}
			}
			return v, nil
		}

		switch msg.String() {
		case "up", "k":
			if v.currentIndex > 0 {
				v.currentIndex--
			}
			v.ensureValidIndex()
		case "down", "j":
			visibleContexts := v.getVisibleContexts()
			if v.currentIndex < len(visibleContexts)-1 {
				v.currentIndex++
			}
			v.ensureValidIndex()
		case "a":
			// Select/deselect all (in multi-select mode)
			if v.multiSelect {
				allSelected := len(v.selectedContexts) == len(v.contexts)
				v.selectedContexts = make(map[string]bool)
				if !allSelected {
					for _, ctx := range v.contexts {
						v.selectedContexts[ctx] = true
					}
				}
			}
		case "m":
			// Toggle multi-select mode
			v.multiSelect = !v.multiSelect
			if !v.multiSelect && len(v.selectedContexts) > 1 {
				// Keep only current selection in single mode
				if v.currentIndex < len(v.contexts) {
					ctx := v.contexts[v.currentIndex]
					v.selectedContexts = make(map[string]bool)
					v.selectedContexts[ctx] = true
				}
			}
		case "/":
			v.SearchMode = true
			v.searchQuery = ""
		case "i":
			// Toggle context info display
			if v.showingInfo {
				v.showingInfo = false
				v.infoContext = ""
			} else {
				visibleContexts := v.getVisibleContexts()
				if v.currentIndex < len(visibleContexts) {
					ctx := visibleContexts[v.currentIndex]
					v.showingInfo = true
					v.infoContext = ctx
				}
			}
		case "enter":
			// Confirm selection
			return v, nil
		case "esc", "q":
			// Cancel or exit info mode
			if v.showingInfo {
				v.showingInfo = false
				v.infoContext = ""
			}
			return v, nil
		}
	}
	return v, nil
}

// View renders the context selector
func (v *ContextView) View() string {
	// If showing info, render info view instead
	if v.showingInfo {
		return v.renderInfoView()
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	title := "Select Kubernetes Context(s)"
	if v.multiSelect {
		title += " [Multi-Select]"
	}

	var content strings.Builder
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n\n")

	// Search or filter display
	if v.SearchMode {
		searchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229"))
		content.WriteString(searchStyle.Render(fmt.Sprintf("Search: %s_", v.searchQuery)))
		content.WriteString("\n\n")
	}

	// Context list
	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)
	currentStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("237"))

	visibleContexts := v.getVisibleContexts()
	maxVisible := v.height - 10 // Leave room for header and help
	startIdx := 0
	if v.currentIndex >= maxVisible {
		startIdx = v.currentIndex - maxVisible + 1
	}

	for i := startIdx; i < len(visibleContexts) && i < startIdx+maxVisible; i++ {
		ctx := visibleContexts[i]
		line := ""

		// Selection indicator
		if v.selectedContexts[ctx] {
			line = "[✓] "
		} else {
			line = "[ ] "
		}

		// Add loading indicator if context is loading
		if v.loadingContexts[ctx] {
			line += ctx + " (loading...)"
		} else {
			line += ctx
		}

		// Apply styles
		if v.selectedContexts[ctx] {
			line = selectedStyle.Render(line)
		}
		// Fix: compare with the relative index in the visible list
		if i == v.currentIndex {
			line = currentStyle.Render(line)
		}

		content.WriteString(itemStyle.Render(line))
		content.WriteString("\n")
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(2)

	helpText := "↑↓: Navigate | Space: Toggle | Enter: Confirm | i: Info | Esc: Cancel"
	if v.multiSelect {
		helpText += " | a: All | m: Single-select"
	} else {
		helpText += " | m: Multi-select"
	}
	helpText += " | /: Search"

	content.WriteString(helpStyle.Render(helpText))

	// Center vertically but left-align content for better checkbox alignment
	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Left,
		lipgloss.Center,
		content.String(),
	)
}

// GetSelectedContexts returns the selected context names
func (v *ContextView) GetSelectedContexts() []string {
	var selected []string
	for ctx, isSelected := range v.selectedContexts {
		if isSelected {
			selected = append(selected, ctx)
		}
	}
	return selected
}

// filterContexts filters the context list based on search query
func (v *ContextView) filterContexts() {
	// Reset current index to ensure it's valid for filtered results
	v.currentIndex = 0
}

// getVisibleContexts returns contexts that match the search filter
func (v *ContextView) getVisibleContexts() []string {
	if v.searchQuery == "" {
		return v.contexts
	}

	var filtered []string
	query := strings.ToLower(v.searchQuery)
	for _, ctx := range v.contexts {
		if strings.Contains(strings.ToLower(ctx), query) {
			filtered = append(filtered, ctx)
		}
	}
	return filtered
}

// SetSize updates the view size
func (v *ContextView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// showContextInfo shows detailed information about a context
func (v *ContextView) showContextInfo(contextName string) tea.Cmd {
	return func() tea.Msg {
		return ContextInfoMsg{ContextName: contextName}
	}
}

// ContextInfoMsg is sent when context info should be displayed
type ContextInfoMsg struct {
	ContextName string
}

// ensureValidIndex ensures currentIndex is within bounds of visible contexts
func (v *ContextView) ensureValidIndex() {
	visibleContexts := v.getVisibleContexts()
	if len(visibleContexts) == 0 {
		v.currentIndex = 0
		return
	}
	if v.currentIndex >= len(visibleContexts) {
		v.currentIndex = len(visibleContexts) - 1
	}
	if v.currentIndex < 0 {
		v.currentIndex = 0
	}
}

// SetLoading sets the loading state for the entire view
func (v *ContextView) SetLoading(loading bool) {
	v.loading = loading
}

// SetContextLoading sets the loading state for a specific context
func (v *ContextView) SetContextLoading(contextName string, loading bool) {
	if loading {
		v.loadingContexts[contextName] = true
	} else {
		delete(v.loadingContexts, contextName)
	}
}

// IsLoading returns whether the view is in loading state
func (v *ContextView) IsLoading() bool {
	return v.loading
}

// renderInfoView renders the context information view
func (v *ContextView) renderInfoView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		MarginBottom(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(2)

	var content strings.Builder
	content.WriteString(titleStyle.Render(fmt.Sprintf("Context Information: %s", v.infoContext)))
	content.WriteString("\n\n")

	// Basic context information
	content.WriteString(infoStyle.Render("Context Name: " + v.infoContext))
	content.WriteString("\n")

	// Add some basic info (in a real implementation, this would come from kubectl config)
	content.WriteString(infoStyle.Render("Type: Kubernetes Context"))
	content.WriteString("\n")

	if v.selectedContexts[v.infoContext] {
		content.WriteString(infoStyle.Render("Status: Currently Selected"))
	} else {
		content.WriteString(infoStyle.Render("Status: Available"))
	}
	content.WriteString("\n\n")

	// Note about functionality
	content.WriteString(infoStyle.Render("Note: This is a basic info display. In a full implementation,"))
	content.WriteString("\n")
	content.WriteString(infoStyle.Render("this would show cluster details, server URL, namespace, etc."))
	content.WriteString("\n")

	// Help text
	content.WriteString(helpStyle.Render("i: Close Info | Esc: Back to Context List"))

	// Center vertically but left-align content
	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Left,
		lipgloss.Center,
		content.String(),
	)
}
