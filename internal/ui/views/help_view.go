package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpView displays help information
type HelpView struct {
	width       int
	height      int
	contextMode string // "resource" or "logs"
}

// NewHelpView creates a new help view
func NewHelpView() *HelpView {
	return &HelpView{
		contextMode: "resource",
	}
}

// SetContext sets the help context (resource or logs)
func (v *HelpView) SetContext(context string) {
	v.contextMode = context
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
	if v.contextMode == "logs" {
		return v.renderLogHelp()
	}
	return v.renderResourceHelp()
}

// renderResourceHelp renders help for resource view
func (v *HelpView) renderResourceHelp() string {
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

	help.WriteString(titleStyle.Render("KubeWatch TUI - Resource View Help"))
	help.WriteString("\n\n")

	help.WriteString(sectionStyle.Render("Navigation"))
	help.WriteString("\n")
	help.WriteString(keyStyle.Render("↑/k") + descStyle.Render("    Move up") + "\n")
	help.WriteString(keyStyle.Render("↓/j") + descStyle.Render("    Move down") + "\n")
	help.WriteString(keyStyle.Render("Tab") + descStyle.Render("    Next resource type") + "\n")
	help.WriteString(keyStyle.Render("S-Tab") + descStyle.Render("  Previous resource type") + "\n")
	help.WriteString(keyStyle.Render("n") + descStyle.Render("      Change namespace") + "\n")

	help.WriteString(sectionStyle.Render("Actions"))
	help.WriteString("\n")
	help.WriteString(keyStyle.Render("Enter/l") + descStyle.Render(" View logs") + "\n")
	help.WriteString(keyStyle.Render("d") + descStyle.Render("       Delete selected") + "\n")
	help.WriteString(keyStyle.Render("r") + descStyle.Render("       Manual refresh") + "\n")
	help.WriteString(keyStyle.Render("u") + descStyle.Render("       Toggle word wrap") + "\n")

	help.WriteString(sectionStyle.Render("General"))
	help.WriteString("\n")
	help.WriteString(keyStyle.Render("?") + descStyle.Render("      Toggle help") + "\n")
	help.WriteString(keyStyle.Render("q") + descStyle.Render("      Quit") + "\n")
	help.WriteString(keyStyle.Render("Esc") + descStyle.Render("    Close dialog") + "\n")

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

// renderLogHelp renders help for log view
func (v *HelpView) renderLogHelp() string {
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

	help.WriteString(titleStyle.Render("KubeWatch TUI - Log View Help"))
	help.WriteString("\n\n")

	help.WriteString(sectionStyle.Render("Navigation"))
	help.WriteString("\n")
	help.WriteString(keyStyle.Render("↑/k") + descStyle.Render("    Scroll up") + "\n")
	help.WriteString(keyStyle.Render("↓/j") + descStyle.Render("    Scroll down") + "\n")
	help.WriteString(keyStyle.Render("PgUp") + descStyle.Render("   Page up") + "\n")
	help.WriteString(keyStyle.Render("PgDn") + descStyle.Render("   Page down") + "\n")
	help.WriteString(keyStyle.Render("Home/g") + descStyle.Render(" Jump to top") + "\n")
	help.WriteString(keyStyle.Render("End/G") + descStyle.Render("  Jump to bottom (follow)") + "\n")

	help.WriteString(sectionStyle.Render("Log Controls"))
	help.WriteString("\n")
	help.WriteString(keyStyle.Render("f") + descStyle.Render("      Toggle follow mode") + "\n")
	help.WriteString(keyStyle.Render("/") + descStyle.Render("      Search in logs") + "\n")
	help.WriteString(keyStyle.Render("c") + descStyle.Render("      Cycle containers (all/individual)") + "\n")
	help.WriteString(keyStyle.Render("p") + descStyle.Render("      Cycle pods (for deployments)") + "\n")
	help.WriteString(keyStyle.Render("C") + descStyle.Render("      Clear log buffer") + "\n")

	help.WriteString(sectionStyle.Render("General"))
	help.WriteString("\n")
	help.WriteString(keyStyle.Render("Esc/q") + descStyle.Render("  Close logs") + "\n")
	help.WriteString(keyStyle.Render("?") + descStyle.Render("      Toggle help") + "\n")

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
