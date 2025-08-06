package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmView displays a confirmation dialog
type ConfirmView struct {
	title       string
	message     string
	confirmText string
	cancelText  string
	confirmed   bool
	width       int
	height      int
}

// NewConfirmView creates a new confirmation dialog
func NewConfirmView(title, message string) *ConfirmView {
	return &ConfirmView{
		title:       title,
		message:     message,
		confirmText: "Yes",
		cancelText:  "No",
		confirmed:   false, // Default to No for safety
	}
}

// Init initializes the view
func (v *ConfirmView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *ConfirmView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "tab":
			v.confirmed = !v.confirmed
		case "right", "l":
			v.confirmed = !v.confirmed
		case "y", "Y":
			v.confirmed = true
			return v, nil
		case "n", "N":
			v.confirmed = false
			return v, nil
		case "enter", " ":
			// Action confirmed or cancelled - will be handled by parent
			return v, nil
		case "q":
			// Cancel with q
			v.confirmed = false
			return v, nil
			// Don't handle ESC here - let parent handle it
		}
	}
	return v, nil
}

// View renders the confirmation dialog
func (v *ConfirmView) View() string {
	// Create styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("1")). // Red for delete confirmation
		MarginBottom(1)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")).
		MarginBottom(2)

	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("1")). // Red border for danger
		Padding(1, 2).
		Width(60).
		Height(10)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("7")).
		Bold(true).
		Padding(0, 2)

	unselectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")).
		Padding(0, 2)

	// Build content
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render(v.title))
	content.WriteString("\n\n")

	// Message
	content.WriteString(messageStyle.Render(v.message))
	content.WriteString("\n\n")

	// Buttons
	var yesButton, noButton string
	if v.confirmed {
		yesButton = selectedStyle.Render(v.confirmText)
		noButton = unselectedStyle.Render(v.cancelText)
	} else {
		yesButton = unselectedStyle.Render(v.confirmText)
		noButton = selectedStyle.Render(v.cancelText)
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Center,
		noButton,
		"    ",
		yesButton,
	)

	// Center the buttons
	buttonsWidth := lipgloss.Width(buttons)
	contentWidth := 56 // borderStyle width minus padding
	padding := (contentWidth - buttonsWidth) / 2
	if padding > 0 {
		buttons = strings.Repeat(" ", padding) + buttons
	}

	content.WriteString(buttons)

	// Help text
	helpText := "\n\n[←→/Tab] Switch  [Y/N] Select  [Enter] Confirm  [Esc] Cancel"
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(helpText))

	// Center the dialog
	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		borderStyle.Render(content.String()),
	)
}

// SetSize updates the view size
func (v *ConfirmView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// IsConfirmed returns whether the user confirmed the action
func (v *ConfirmView) IsConfirmed() bool {
	return v.confirmed
}

// SetConfirmText sets custom confirm button text
func (v *ConfirmView) SetConfirmText(text string) {
	v.confirmText = text
}

// SetCancelText sets custom cancel button text
func (v *ConfirmView) SetCancelText(text string) {
	v.cancelText = text
}
