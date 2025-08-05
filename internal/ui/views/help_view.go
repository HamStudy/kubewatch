package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpView displays help information
type HelpView struct {
	width  int
	height int
}

// NewHelpView creates a new help view
func NewHelpView() *HelpView {
	return &HelpView{}
}

// Init initializes the view
func (v *HelpView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *HelpView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}
	return v, nil
}

// View renders the help screen
func (v *HelpView) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(2)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		MarginTop(1).
		MarginBottom(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	var help strings.Builder

	help.WriteString(titleStyle.Render("KubeWatch TUI - Help"))
	help.WriteString("\n\n")

	help.WriteString(sectionStyle.Render("Navigation"))
	help.WriteString("\n")
	help.WriteString(keyStyle.Render("↑/k") + descStyle.Render("    Move up") + "\n")
	help.WriteString(keyStyle.Render("↓/j") + descStyle.Render("    Move down") + "\n")
	help.WriteString(keyStyle.Render("←/h") + descStyle.Render("    Move left") + "\n")
	help.WriteString(keyStyle.Render("→/l") + descStyle.Render("    Move right") + "\n")
	help.WriteString(keyStyle.Render("Tab") + descStyle.Render("    Next resource type") + "\n")
	help.WriteString(keyStyle.Render("S-Tab") + descStyle.Render("  Previous resource type") + "\n")

	help.WriteString(sectionStyle.Render("Actions"))
	help.WriteString("\n")
	help.WriteString(keyStyle.Render("Enter") + descStyle.Render("  Select item") + "\n")
	help.WriteString(keyStyle.Render("Space") + descStyle.Render("  Multi-select") + "\n")
	help.WriteString(keyStyle.Render("d") + descStyle.Render("      Delete selected") + "\n")
	help.WriteString(keyStyle.Render("l") + descStyle.Render("      View logs") + "\n")
	help.WriteString(keyStyle.Render("r") + descStyle.Render("      Refresh") + "\n")

	help.WriteString(sectionStyle.Render("General"))
	help.WriteString("\n")
	help.WriteString(keyStyle.Render("?") + descStyle.Render("      Toggle help") + "\n")
	help.WriteString(keyStyle.Render("q") + descStyle.Render("      Quit") + "\n")
	help.WriteString(keyStyle.Render("Esc") + descStyle.Render("    Close dialog") + "\n")

	help.WriteString(sectionStyle.Render("Resource Types"))
	help.WriteString("\n")
	help.WriteString(descStyle.Render("• Pods\n"))
	help.WriteString(descStyle.Render("• Deployments\n"))
	help.WriteString(descStyle.Render("• StatefulSets\n"))
	help.WriteString(descStyle.Render("• Services\n"))
	help.WriteString(descStyle.Render("• ConfigMaps\n"))
	help.WriteString(descStyle.Render("• Secrets\n"))

	help.WriteString("\n\n")
	help.WriteString(descStyle.Render("Press ? to close help"))

	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		help.String(),
	)
}
