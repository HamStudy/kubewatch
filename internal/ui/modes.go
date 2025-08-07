package ui

import (
	"github.com/HamStudy/kubewatch/internal/k8s"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ScreenModeType represents different screen modes
type ScreenModeType int

const (
	ModeList ScreenModeType = iota
	ModeLog
	ModeDescribe
	ModeHelp
	ModeContextSelector
	ModeNamespaceSelector
	ModeConfirmDialog
	ModeResourceSelector
)

// KeyBinding represents a key binding with help text
type KeyBinding struct {
	Key         key.Binding
	Description string
	Section     string // For grouping in help
}

// ScreenMode defines the interface for different screen modes
type ScreenMode interface {
	// GetType returns the mode type
	GetType() ScreenModeType

	// GetKeyBindings returns the key bindings for this mode
	GetKeyBindings() map[string]KeyBinding

	// HandleKey processes a key message and returns whether it was handled
	HandleKey(msg tea.KeyMsg, app *App) (handled bool, cmd tea.Cmd)

	// GetHelpSections returns organized help sections for this mode
	GetHelpSections() map[string][]KeyBinding

	// GetTitle returns the title for this mode (used in help)
	GetTitle() string
}

// BaseMode provides common functionality for screen modes
type BaseMode struct {
	modeType ScreenModeType
	title    string
}

func (m *BaseMode) GetType() ScreenModeType {
	return m.modeType
}

func (m *BaseMode) GetTitle() string {
	return m.title
}

// Helper function to create key bindings
func NewKeyBinding(keys []string, help string, description string, section string) KeyBinding {
	return KeyBinding{
		Key: key.NewBinding(
			key.WithKeys(keys...),
			key.WithHelp(help, description),
		),
		Description: description,
		Section:     section,
	}
}

// ListMode handles the main resource list view
type ListMode struct {
	BaseMode
}

func NewListMode() *ListMode {
	return &ListMode{
		BaseMode: BaseMode{
			modeType: ModeList,
			title:    "KubeWatch TUI - Resource View",
		},
	}
}

func (m *ListMode) GetKeyBindings() map[string]KeyBinding {
	return map[string]KeyBinding{
		"up":        NewKeyBinding([]string{"up", "k"}, "↑/k", "Move up", "Navigation"),
		"down":      NewKeyBinding([]string{"down", "j"}, "↓/j", "Move down", "Navigation"),
		"left":      NewKeyBinding([]string{"left", "h"}, "←/h", "Move left", "Navigation"),
		"right":     NewKeyBinding([]string{"right", "l"}, "→/l", "Move right", "Navigation"),
		"tab":       NewKeyBinding([]string{"tab"}, "Tab", "Next resource type", "Navigation"),
		"shift+tab": NewKeyBinding([]string{"shift+tab"}, "S-Tab", "Previous resource type", "Navigation"),
		"namespace": NewKeyBinding([]string{"n"}, "n", "Change namespace", "Navigation"),
		"context":   NewKeyBinding([]string{"c"}, "c", "Switch contexts", "Navigation"),
		"enter":     NewKeyBinding([]string{"enter"}, "Enter", "Select/View logs", "Actions"),
		"logs":      NewKeyBinding([]string{"l"}, "l", "View logs", "Actions"),
		"info":      NewKeyBinding([]string{"i"}, "i", "Show resource info", "Actions"),
		"describe":  NewKeyBinding([]string{"d"}, "d", "Describe resource", "Actions"),
		"delete":    NewKeyBinding([]string{"delete", "D"}, "Del/D", "Delete resource", "Actions"),
		"refresh":   NewKeyBinding([]string{"r", "ctrl+r"}, "r", "Refresh", "Actions"),
		"sort":      NewKeyBinding([]string{"s"}, "s", "Cycle sort column/direction", "Actions"),
		"help":      NewKeyBinding([]string{"?"}, "?", "Toggle help", "General"),
		"quit":      NewKeyBinding([]string{"q", "ctrl+c"}, "q", "Quit", "General"),
		"escape":    NewKeyBinding([]string{"esc"}, "Esc", "Close dialog/Back", "General"),
	}
}

func (m *ListMode) GetHelpSections() map[string][]KeyBinding {
	bindings := m.GetKeyBindings()
	sections := make(map[string][]KeyBinding)

	for _, binding := range bindings {
		sections[binding.Section] = append(sections[binding.Section], binding)
	}

	return sections
}

func (m *ListMode) HandleKey(msg tea.KeyMsg, app *App) (bool, tea.Cmd) {
	bindings := m.GetKeyBindings()

	switch {
	case key.Matches(msg, bindings["quit"].Key):
		return true, tea.Quit

	case key.Matches(msg, bindings["help"].Key):
		app.setMode(ModeHelp)
		return true, nil

	case key.Matches(msg, bindings["namespace"].Key):
		return true, app.openNamespaceSelector()

	case key.Matches(msg, bindings["context"].Key):
		return true, app.openContextSelector()

	case key.Matches(msg, bindings["tab"].Key):
		return true, app.openResourceSelector()

	case key.Matches(msg, bindings["shift+tab"].Key):
		return true, app.openResourceSelector()

	case key.Matches(msg, bindings["logs"].Key), key.Matches(msg, bindings["enter"].Key):
		selectedName := app.resourceView.GetSelectedResourceName()
		if selectedName != "" {
			// Get the appropriate client for logs
			var client *k8s.Client
			if app.isMultiContext {
				contextName := app.getSelectedResourceContext()
				if contextName != "" {
					client, _ = app.multiClient.GetClient(contextName)
				}
			} else {
				client = app.k8sClient
			}

			// Only proceed if we have a valid client
			if client != nil {
				app.setMode(ModeLog)
				app.resourceView.SetCompactMode(true)
				return true, app.logView.StartStreaming(app.ctx, client, app.state, selectedName)
			}
		}
	case key.Matches(msg, bindings["info"].Key):
		selectedName := app.resourceView.GetSelectedResourceName()
		if selectedName != "" {
			// Check if we have a valid client before proceeding
			var hasClient bool
			if app.isMultiContext {
				contextName := app.getSelectedResourceContext()
				if contextName != "" {
					_, err := app.multiClient.GetClient(contextName)
					hasClient = err == nil
				}
			} else {
				hasClient = app.k8sClient != nil
			}

			if hasClient {
				app.setMode(ModeDescribe)
				return true, app.startDescribeView(selectedName)
			}
		}
		return true, nil // Always handle the key, even if we can't process it

	case key.Matches(msg, bindings["describe"].Key):
		selectedName := app.resourceView.GetSelectedResourceName()
		if selectedName != "" {
			// Check if we have a valid client before proceeding
			var hasClient bool
			if app.isMultiContext {
				contextName := app.getSelectedResourceContext()
				if contextName != "" {
					_, err := app.multiClient.GetClient(contextName)
					hasClient = err == nil
				}
			} else {
				hasClient = app.k8sClient != nil
			}

			if hasClient {
				app.setMode(ModeDescribe)
				return true, app.startDescribeView(selectedName)
			}
		}
		return true, nil // Always handle the key, even if we can't process it

	case key.Matches(msg, bindings["delete"].Key):
		selectedName := app.resourceView.GetSelectedResourceName()
		if selectedName != "" {
			app.setMode(ModeConfirmDialog)
			return true, app.showDeleteConfirmation(selectedName)
		}

	case key.Matches(msg, bindings["refresh"].Key):
		return true, app.resourceView.RefreshResources()

	case key.Matches(msg, bindings["sort"].Key):
		app.cycleSortColumn()
		return true, app.resourceView.RefreshResources()
	}

	return false, nil
}

// LogMode handles the log view
type LogMode struct {
	BaseMode
}

func NewLogMode() *LogMode {
	return &LogMode{
		BaseMode: BaseMode{
			modeType: ModeLog,
			title:    "KubeWatch TUI - Log View",
		},
	}
}

func (m *LogMode) GetKeyBindings() map[string]KeyBinding {
	return map[string]KeyBinding{
		"up":        NewKeyBinding([]string{"up", "k"}, "↑/k", "Scroll up", "Navigation"),
		"down":      NewKeyBinding([]string{"down", "j"}, "↓/j", "Scroll down", "Navigation"),
		"pageup":    NewKeyBinding([]string{"pgup"}, "PgUp", "Page up", "Navigation"),
		"pagedown":  NewKeyBinding([]string{"pgdown"}, "PgDn", "Page down", "Navigation"),
		"home":      NewKeyBinding([]string{"home", "g"}, "Home/g", "Jump to top", "Navigation"),
		"end":       NewKeyBinding([]string{"end", "G"}, "End/G", "Jump to bottom (follow)", "Navigation"),
		"follow":    NewKeyBinding([]string{"f"}, "f", "Toggle follow mode", "Log Controls"),
		"search":    NewKeyBinding([]string{"/"}, "/", "Search in logs", "Log Controls"),
		"container": NewKeyBinding([]string{"c"}, "c", "Cycle containers", "Log Controls"),
		"pod":       NewKeyBinding([]string{"p"}, "p", "Cycle pods", "Log Controls"),
		"clear":     NewKeyBinding([]string{"C"}, "C", "Clear log buffer", "Log Controls"),
		"help":      NewKeyBinding([]string{"?"}, "?", "Toggle help", "General"),
		"quit":      NewKeyBinding([]string{"q", "ctrl+c"}, "q", "Quit application", "General"),
		"escape":    NewKeyBinding([]string{"esc"}, "Esc", "Close logs", "General"),
	}
}

func (m *LogMode) GetHelpSections() map[string][]KeyBinding {
	bindings := m.GetKeyBindings()
	sections := make(map[string][]KeyBinding)

	for _, binding := range bindings {
		sections[binding.Section] = append(sections[binding.Section], binding)
	}

	return sections
}

func (m *LogMode) HandleKey(msg tea.KeyMsg, app *App) (bool, tea.Cmd) {
	bindings := m.GetKeyBindings()

	// When in search mode, only handle ESC and let log view handle everything else
	if app.logView.IsSearchMode() {
		if key.Matches(msg, bindings["escape"].Key) {
			// Let log view handle search cancellation
			return false, nil
		}
		// Skip all other app-level key processing when in search mode
		return false, nil
	}

	switch {
	case key.Matches(msg, bindings["quit"].Key):
		return true, tea.Quit

	case key.Matches(msg, bindings["help"].Key):
		app.setMode(ModeHelp)
		return true, nil

	case key.Matches(msg, bindings["escape"].Key):
		app.setMode(ModeList)
		app.resourceView.SetCompactMode(false)
		app.resourceView.SetSize(app.width, app.height)
		return true, app.logView.StopStreaming()
	}

	// Let log view handle all other keys
	return false, nil
}

// DescribeMode handles the kubectl describe view
type DescribeMode struct {
	BaseMode
}

func NewDescribeMode() *DescribeMode {
	return &DescribeMode{
		BaseMode: BaseMode{
			modeType: ModeDescribe,
			title:    "KubeWatch TUI - Describe View",
		},
	}
}

func (m *DescribeMode) GetKeyBindings() map[string]KeyBinding {
	return map[string]KeyBinding{
		"up":          NewKeyBinding([]string{"up", "k"}, "↑/k", "Scroll up", "Navigation"),
		"down":        NewKeyBinding([]string{"down", "j"}, "↓/j", "Scroll down", "Navigation"),
		"pageup":      NewKeyBinding([]string{"pgup"}, "PgUp", "Page up", "Navigation"),
		"pagedown":    NewKeyBinding([]string{"pgdown"}, "PgDn", "Page down", "Navigation"),
		"home":        NewKeyBinding([]string{"home", "g"}, "Home/g", "Jump to top", "Navigation"),
		"end":         NewKeyBinding([]string{"end", "G"}, "End/G", "Jump to bottom", "Navigation"),
		"wordwrap":    NewKeyBinding([]string{"u"}, "u", "Toggle word wrap", "Display"),
		"refresh":     NewKeyBinding([]string{"r", "ctrl+r"}, "r", "Manual refresh", "Actions"),
		"autorefresh": NewKeyBinding([]string{"a"}, "a", "Toggle auto-refresh", "Actions"),
		"help":        NewKeyBinding([]string{"?"}, "?", "Toggle help", "General"),
		"quit":        NewKeyBinding([]string{"q", "ctrl+c"}, "q", "Quit application", "General"),
		"escape":      NewKeyBinding([]string{"esc"}, "Esc", "Back to list", "General"),
	}
}

func (m *DescribeMode) GetHelpSections() map[string][]KeyBinding {
	bindings := m.GetKeyBindings()
	sections := make(map[string][]KeyBinding)

	for _, binding := range bindings {
		sections[binding.Section] = append(sections[binding.Section], binding)
	}

	return sections
}

func (m *DescribeMode) HandleKey(msg tea.KeyMsg, app *App) (bool, tea.Cmd) {
	bindings := m.GetKeyBindings()

	switch {
	case key.Matches(msg, bindings["quit"].Key):
		return true, tea.Quit

	case key.Matches(msg, bindings["help"].Key):
		app.setMode(ModeHelp)
		return true, nil

	case key.Matches(msg, bindings["escape"].Key):
		app.setMode(ModeList)
		return true, nil
	}

	// Let describe view handle all other keys
	return false, nil
}

// HelpMode handles the help view
type HelpMode struct {
	BaseMode
}

func NewHelpMode() *HelpMode {
	return &HelpMode{
		BaseMode: BaseMode{
			modeType: ModeHelp,
			title:    "KubeWatch TUI - Help",
		},
	}
}

func (m *HelpMode) GetKeyBindings() map[string]KeyBinding {
	return map[string]KeyBinding{
		"help":   NewKeyBinding([]string{"?"}, "?", "Close help", "General"),
		"quit":   NewKeyBinding([]string{"q", "ctrl+c"}, "q", "Quit application", "General"),
		"escape": NewKeyBinding([]string{"esc"}, "Esc", "Close help", "General"),
	}
}

func (m *HelpMode) GetHelpSections() map[string][]KeyBinding {
	bindings := m.GetKeyBindings()
	sections := make(map[string][]KeyBinding)

	for _, binding := range bindings {
		sections[binding.Section] = append(sections[binding.Section], binding)
	}

	return sections
}

func (m *HelpMode) HandleKey(msg tea.KeyMsg, app *App) (bool, tea.Cmd) {
	bindings := m.GetKeyBindings()

	switch {
	case key.Matches(msg, bindings["quit"].Key):
		return true, tea.Quit

	case key.Matches(msg, bindings["help"].Key), key.Matches(msg, bindings["escape"].Key):
		app.returnToPreviousMode()
		return true, nil
	}

	return false, nil
}

// ContextSelectorMode handles context selection
type ContextSelectorMode struct {
	BaseMode
}

func NewContextSelectorMode() *ContextSelectorMode {
	return &ContextSelectorMode{
		BaseMode: BaseMode{
			modeType: ModeContextSelector,
			title:    "KubeWatch TUI - Context Selector",
		},
	}
}

func (m *ContextSelectorMode) GetKeyBindings() map[string]KeyBinding {
	return map[string]KeyBinding{
		"up":     NewKeyBinding([]string{"up", "k"}, "↑/k", "Move up", "Navigation"),
		"down":   NewKeyBinding([]string{"down", "j"}, "↓/j", "Move down", "Navigation"),
		"space":  NewKeyBinding([]string{" "}, "Space", "Toggle selection", "Actions"),
		"enter":  NewKeyBinding([]string{"enter"}, "Enter", "Apply selection", "Actions"),
		"info":   NewKeyBinding([]string{"i"}, "i", "Show context info", "Actions"),
		"search": NewKeyBinding([]string{"/"}, "/", "Search contexts", "Actions"),
		"quit":   NewKeyBinding([]string{"q", "ctrl+c"}, "q", "Quit application", "General"),
		"escape": NewKeyBinding([]string{"esc", "c"}, "Esc/c", "Cancel", "General"),
	}
}

func (m *ContextSelectorMode) GetHelpSections() map[string][]KeyBinding {
	bindings := m.GetKeyBindings()
	sections := make(map[string][]KeyBinding)

	for _, binding := range bindings {
		sections[binding.Section] = append(sections[binding.Section], binding)
	}

	return sections
}

func (m *ContextSelectorMode) HandleKey(msg tea.KeyMsg, app *App) (bool, tea.Cmd) {
	bindings := m.GetKeyBindings()

	// If context view is in search mode, don't handle any keys here
	if app.contextView != nil && app.contextView.SearchMode {
		return false, nil
	}

	switch {
	case key.Matches(msg, bindings["quit"].Key):
		return true, tea.Quit

	case key.Matches(msg, bindings["enter"].Key):
		return true, app.applyContextSelection()

	case key.Matches(msg, bindings["escape"].Key):
		app.setMode(ModeList)
		return true, nil

	case key.Matches(msg, bindings["info"].Key):
		// Let context view handle info key
		return false, nil

	case key.Matches(msg, bindings["search"].Key):
		// Let context view handle search key
		return false, nil
	}

	// Let context view handle navigation keys
	return false, nil
}

// NamespaceSelectorMode handles namespace selection
type NamespaceSelectorMode struct {
	BaseMode
}

func NewNamespaceSelectorMode() *NamespaceSelectorMode {
	return &NamespaceSelectorMode{
		BaseMode: BaseMode{
			modeType: ModeNamespaceSelector,
			title:    "KubeWatch TUI - Namespace Selector",
		},
	}
}

func (m *NamespaceSelectorMode) GetKeyBindings() map[string]KeyBinding {
	return map[string]KeyBinding{
		"up":     NewKeyBinding([]string{"up", "k"}, "↑/k", "Move up", "Navigation"),
		"down":   NewKeyBinding([]string{"down", "j"}, "↓/j", "Move down", "Navigation"),
		"enter":  NewKeyBinding([]string{"enter"}, "Enter", "Select namespace", "Actions"),
		"quit":   NewKeyBinding([]string{"q", "ctrl+c"}, "q", "Quit application", "General"),
		"escape": NewKeyBinding([]string{"esc", "n"}, "Esc/n", "Cancel", "General"),
	}
}

func (m *NamespaceSelectorMode) GetHelpSections() map[string][]KeyBinding {
	bindings := m.GetKeyBindings()
	sections := make(map[string][]KeyBinding)

	for _, binding := range bindings {
		sections[binding.Section] = append(sections[binding.Section], binding)
	}

	return sections
}

func (m *NamespaceSelectorMode) HandleKey(msg tea.KeyMsg, app *App) (bool, tea.Cmd) {
	bindings := m.GetKeyBindings()

	switch {
	case key.Matches(msg, bindings["quit"].Key):
		return true, tea.Quit

	case key.Matches(msg, bindings["enter"].Key):
		return true, app.applyNamespaceSelection()

	case key.Matches(msg, bindings["escape"].Key):
		app.setMode(ModeList)
		return true, nil
	}

	// Let namespace view handle navigation keys
	return false, nil
}

// ConfirmDialogMode handles confirmation dialogs
type ConfirmDialogMode struct {
	BaseMode
}

func NewConfirmDialogMode() *ConfirmDialogMode {
	return &ConfirmDialogMode{
		BaseMode: BaseMode{
			modeType: ModeConfirmDialog,
			title:    "KubeWatch TUI - Confirmation",
		},
	}
}

func (m *ConfirmDialogMode) GetKeyBindings() map[string]KeyBinding {
	return map[string]KeyBinding{
		"left":   NewKeyBinding([]string{"left", "h"}, "←/h", "Move left", "Navigation"),
		"right":  NewKeyBinding([]string{"right", "l"}, "→/l", "Move right", "Navigation"),
		"enter":  NewKeyBinding([]string{"enter", " "}, "Enter/Space", "Confirm selection", "Actions"),
		"quit":   NewKeyBinding([]string{"q", "ctrl+c"}, "q", "Quit application", "General"),
		"escape": NewKeyBinding([]string{"esc"}, "Esc", "Cancel", "General"),
	}
}

func (m *ConfirmDialogMode) GetHelpSections() map[string][]KeyBinding {
	bindings := m.GetKeyBindings()
	sections := make(map[string][]KeyBinding)

	for _, binding := range bindings {
		sections[binding.Section] = append(sections[binding.Section], binding)
	}

	return sections
}

func (m *ConfirmDialogMode) HandleKey(msg tea.KeyMsg, app *App) (bool, tea.Cmd) {
	bindings := m.GetKeyBindings()

	switch {
	case key.Matches(msg, bindings["quit"].Key):
		return true, tea.Quit

	case key.Matches(msg, bindings["enter"].Key):
		return true, app.handleConfirmDialogAction()

	case key.Matches(msg, bindings["escape"].Key):
		app.setMode(ModeList)
		return true, nil
	}

	// Let confirm view handle navigation keys
	return false, nil
}

// ResourceSelectorMode handles resource type selection
type ResourceSelectorMode struct {
	BaseMode
}

func NewResourceSelectorMode() *ResourceSelectorMode {
	return &ResourceSelectorMode{
		BaseMode: BaseMode{
			modeType: ModeResourceSelector,
			title:    "KubeWatch TUI - Resource Selector",
		},
	}
}

func (m *ResourceSelectorMode) GetKeyBindings() map[string]KeyBinding {
	return map[string]KeyBinding{
		"up":     NewKeyBinding([]string{"up", "k"}, "↑/k", "Move up", "Navigation"),
		"down":   NewKeyBinding([]string{"down", "j"}, "↓/j", "Move down", "Navigation"),
		"enter":  NewKeyBinding([]string{"enter"}, "Enter", "Select resource type", "Actions"),
		"quit":   NewKeyBinding([]string{"q", "ctrl+c"}, "q", "Quit application", "General"),
		"escape": NewKeyBinding([]string{"esc", "tab"}, "Esc/Tab", "Cancel", "General"),
	}
}

func (m *ResourceSelectorMode) GetHelpSections() map[string][]KeyBinding {
	bindings := m.GetKeyBindings()
	sections := make(map[string][]KeyBinding)

	for _, binding := range bindings {
		sections[binding.Section] = append(sections[binding.Section], binding)
	}

	return sections
}

func (m *ResourceSelectorMode) HandleKey(msg tea.KeyMsg, app *App) (bool, tea.Cmd) {
	bindings := m.GetKeyBindings()

	switch {
	case key.Matches(msg, bindings["quit"].Key):
		return true, tea.Quit

	case key.Matches(msg, bindings["enter"].Key):
		// Resource selection not implemented yet
		app.setMode(ModeList)
		return true, nil

	case key.Matches(msg, bindings["escape"].Key):
		app.setMode(ModeList)
		return true, nil
	}

	// Let resource selector view handle navigation keys
	return false, nil
}
