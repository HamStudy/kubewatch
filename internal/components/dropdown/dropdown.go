package dropdown

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Option represents a dropdown option
type Option struct {
	Label string
	Value interface{}
}

// Model represents the dropdown component
type Model struct {
	// Options
	options []Option

	// State
	selectedIndex int
	isOpen        bool
	width         int
	height        int

	// Styling
	selectedStyle   lipgloss.Style
	unselectedStyle lipgloss.Style
	borderStyle     lipgloss.Style
	titleStyle      lipgloss.Style

	// Configuration
	title       string
	placeholder string

	// Key bindings
	keyMap KeyMap
}

// KeyMap defines the key bindings for the dropdown
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Escape key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// New creates a new dropdown model
func New(options []Option) Model {
	return Model{
		options:         options,
		selectedIndex:   0,
		isOpen:          false,
		width:           30,
		height:          10,
		selectedStyle:   lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229")),
		unselectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
		borderStyle:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")),
		titleStyle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
		placeholder:     "Select an option...",
		keyMap:          DefaultKeyMap(),
	}
}

// SetOptions updates the dropdown options
func (m *Model) SetOptions(options []Option) {
	m.options = options
	if m.selectedIndex >= len(options) {
		m.selectedIndex = 0
	}
}

// SetTitle sets the dropdown title
func (m *Model) SetTitle(title string) {
	m.title = title
}

// SetPlaceholder sets the placeholder text
func (m *Model) SetPlaceholder(placeholder string) {
	m.placeholder = placeholder
}

// SetSize sets the dropdown dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Open opens the dropdown
func (m *Model) Open() {
	m.isOpen = true
}

// Close closes the dropdown
func (m *Model) Close() {
	m.isOpen = false
}

// IsOpen returns whether the dropdown is open
func (m *Model) IsOpen() bool {
	return m.isOpen
}

// GetSelectedOption returns the currently selected option
func (m *Model) GetSelectedOption() Option {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.options) {
		return m.options[m.selectedIndex]
	}
	return Option{}
}

// GetSelectedIndex returns the currently selected index
func (m *Model) GetSelectedIndex() int {
	return m.selectedIndex
}

// SetSelectedIndex sets the selected index
func (m *Model) SetSelectedIndex(index int) {
	if index >= 0 && index < len(m.options) {
		m.selectedIndex = index
	}
}

// SetSelectedValue sets the selected option by value
func (m *Model) SetSelectedValue(value interface{}) {
	for i, option := range m.options {
		if option.Value == value {
			m.selectedIndex = i
			return
		}
	}
}

// Init initializes the dropdown
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.isOpen {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keyMap.Up):
			if m.selectedIndex > 0 {
				m.selectedIndex--
			} else {
				m.selectedIndex = len(m.options) - 1
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Down):
			if m.selectedIndex < len(m.options)-1 {
				m.selectedIndex++
			} else {
				m.selectedIndex = 0
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Enter):
			m.isOpen = false
			return m, func() tea.Msg {
				return SelectedMsg{
					Option: m.GetSelectedOption(),
					Index:  m.selectedIndex,
				}
			}

		case key.Matches(msg, m.keyMap.Escape):
			m.isOpen = false
			return m, func() tea.Msg {
				return CancelledMsg{}
			}
		}
	}

	return m, nil
}

// View renders the dropdown
func (m Model) View() string {
	if !m.isOpen {
		return ""
	}

	var content strings.Builder

	// Title
	if m.title != "" {
		content.WriteString(m.titleStyle.Render(m.title))
		content.WriteString("\n")
	}

	// Options
	maxVisible := m.height - 2 // Account for borders
	if m.title != "" {
		maxVisible-- // Account for title
	}

	startIndex := 0
	endIndex := len(m.options)

	// Calculate visible range if we have too many options
	if len(m.options) > maxVisible {
		// Center the selected item in the visible range
		startIndex = m.selectedIndex - maxVisible/2
		if startIndex < 0 {
			startIndex = 0
		}
		endIndex = startIndex + maxVisible
		if endIndex > len(m.options) {
			endIndex = len(m.options)
			startIndex = endIndex - maxVisible
			if startIndex < 0 {
				startIndex = 0
			}
		}
	}

	// Render visible options
	for i := startIndex; i < endIndex; i++ {
		option := m.options[i]
		line := option.Label

		// Truncate if too long
		maxWidth := m.width - 4 // Account for borders and padding
		if len(line) > maxWidth {
			line = line[:maxWidth-3] + "..."
		}

		// Apply styling
		if i == m.selectedIndex {
			line = m.selectedStyle.Width(maxWidth).Render(line)
		} else {
			line = m.unselectedStyle.Width(maxWidth).Render(line)
		}

		content.WriteString(line)
		if i < endIndex-1 {
			content.WriteString("\n")
		}
	}

	// Add scroll indicators if needed
	if len(m.options) > maxVisible {
		scrollInfo := ""
		if startIndex > 0 {
			scrollInfo += "↑ "
		}
		if endIndex < len(m.options) {
			scrollInfo += "↓"
		}
		if scrollInfo != "" {
			content.WriteString("\n")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(scrollInfo))
		}
	}

	// Apply border
	return m.borderStyle.Width(m.width).Render(content.String())
}

// SelectedMsg is sent when an option is selected
type SelectedMsg struct {
	Option Option
	Index  int
}

// CancelledMsg is sent when the dropdown is cancelled
type CancelledMsg struct{}
