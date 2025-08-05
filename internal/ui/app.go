package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/kubewatch-tui/internal/core"
	"github.com/user/kubewatch-tui/internal/k8s"
	"github.com/user/kubewatch-tui/internal/ui/views"
	"k8s.io/apimachinery/pkg/watch"
)

// KeyMap defines the key bindings
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Space    key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Delete   key.Binding
	Logs     key.Binding
	Help     key.Binding
	Quit     key.Binding
	Refresh  key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "multi-select"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next resource"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev resource"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d", "delete"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "view logs"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "ctrl+r"),
			key.WithHelp("r", "refresh"),
		),
	}
}

// watchEventMsg represents a Kubernetes watch event
type watchEventMsg struct {
	Type   watch.EventType
	Object interface{}
}

// tickMsg represents a periodic refresh tick
type tickMsg time.Time

// App represents the main application model
type App struct {
	ctx       context.Context
	k8sClient *k8s.Client
	state     *core.State
	config    *core.Config
	keys      KeyMap

	// Views
	resourceView  *views.ResourceView
	logView       *views.LogView
	helpView      *views.HelpView
	namespaceView *views.NamespaceView
	confirmView   *views.ConfirmView

	// UI state
	width              int
	height             int
	ready              bool
	showNamespacePopup bool
	showDeleteConfirm  bool
	pendingDeleteName  string
	loadingNamespaces  bool

	// Watchers
	cancelWatcher context.CancelFunc
	watcherCtx    context.Context
}

// NewApp creates a new application instance
func NewApp(ctx context.Context, k8sClient *k8s.Client, state *core.State, config *core.Config) *App {
	return &App{
		ctx:          ctx,
		k8sClient:    k8sClient,
		state:        state,
		config:       config,
		keys:         DefaultKeyMap(),
		resourceView: views.NewResourceView(state, k8sClient),
		logView:      views.NewLogView(),
		helpView:     views.NewHelpView(),
	}
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.resourceView.Init(),
		tea.EnterAltScreen,
		a.startRefreshTimer(), // Start the refresh timer
	)
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		// Auto-refresh on tick
		return a, tea.Batch(
			a.resourceView.RefreshResources(),
			a.startRefreshTimer(), // Schedule next tick
		)

	case tea.KeyMsg:
		// Handle delete confirmation first
		if a.showDeleteConfirm {
			switch msg.String() {
			case "enter", " ":
				// Check if confirmed
				if a.confirmView.IsConfirmed() {
					// Proceed with deletion
					a.showDeleteConfirm = false
					return a, a.resourceView.DeleteSelected()
				}
				// Cancelled
				a.showDeleteConfirm = false
				a.pendingDeleteName = ""
				return a, nil
			case "esc", "q":
				// Cancel deletion
				a.showDeleteConfirm = false
				a.pendingDeleteName = ""
				return a, nil
			default:
				// Pass to confirm view
				confirmModel, cmd := a.confirmView.Update(msg)
				a.confirmView = confirmModel.(*views.ConfirmView)
				return a, cmd
			}
		}

		// Handle namespace popup
		if a.showNamespacePopup {
			switch msg.String() {
			case "enter":
				// Apply the selected namespace
				newNamespace := a.namespaceView.GetSelectedNamespace()
				if newNamespace != a.state.CurrentNamespace {
					a.state.CurrentNamespace = newNamespace
					a.config.CurrentNamespace = newNamespace
					// Refresh resources with new namespace
					a.showNamespacePopup = false
					return a, a.resourceView.RefreshResources()
				}
				a.showNamespacePopup = false
				return a, nil
			case "esc", "q", "n":
				// Cancel namespace selection
				a.showNamespacePopup = false
				return a, nil
			default:
				// Pass to namespace view
				nsModel, cmd := a.namespaceView.Update(msg)
				a.namespaceView = nsModel.(*views.NamespaceView)
				return a, cmd
			}
		}

		switch {
		case key.Matches(msg, a.keys.Quit):
			return a, tea.Quit

		case key.Matches(msg, a.keys.Help):
			a.state.ShowHelp = !a.state.ShowHelp
			return a, nil

		case msg.String() == "n":
			// Open namespace selector
			return a, a.openNamespaceSelector()

		case key.Matches(msg, a.keys.Tab):
			a.nextResourceType()
			return a, a.resourceView.RefreshResources()

		case key.Matches(msg, a.keys.ShiftTab):
			a.prevResourceType()
			return a, a.resourceView.RefreshResources()

		case key.Matches(msg, a.keys.Logs):
			if !a.state.ShowLogs {
				selectedName := a.resourceView.GetSelectedResourceName()
				if selectedName != "" {
					a.state.ShowLogs = true
					return a, a.logView.StartStreaming(a.ctx, a.k8sClient, a.state, selectedName)
				}
			} else {
				a.state.ShowLogs = false
				return a, a.logView.StopStreaming()
			}

		case key.Matches(msg, a.keys.Delete):
			// Show delete confirmation
			selectedName := a.resourceView.GetSelectedResourceName()
			if selectedName != "" {
				a.pendingDeleteName = selectedName
				resourceType := string(a.state.CurrentResourceType)
				// Remove the 's' at the end for singular form
				if strings.HasSuffix(resourceType, "s") {
					resourceType = resourceType[:len(resourceType)-1]
				}
				message := fmt.Sprintf("Are you sure you want to delete %s '%s'?",
					strings.ToLower(resourceType), selectedName)
				a.confirmView = views.NewConfirmView("⚠️  Confirm Deletion", message)
				a.confirmView.SetSize(a.width, a.height)
				a.confirmView.SetConfirmText("Delete")
				a.confirmView.SetCancelText("Cancel")
				a.showDeleteConfirm = true
				return a, nil
			}

		case key.Matches(msg, a.keys.Refresh):
			return a, a.resourceView.RefreshResources()
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true

		// Update child views
		a.resourceView.SetSize(msg.Width, msg.Height)
		a.logView.SetSize(msg.Width, msg.Height/2)
		if a.namespaceView != nil {
			a.namespaceView.SetSize(msg.Width, msg.Height)
		}
		if a.confirmView != nil {
			a.confirmView.SetSize(msg.Width, msg.Height)
		}
		return a, nil

	case deleteCompleteMsg:
		// Resource deleted successfully, refresh the list
		return a, a.resourceView.RefreshResources()
	}

	// Update child views
	if a.state.ShowHelp {
		helpModel, cmd := a.helpView.Update(msg)
		a.helpView = helpModel.(*views.HelpView)
		cmds = append(cmds, cmd)
	} else if a.state.ShowLogs {
		// Split view: only pass keyboard events to the log view
		// Resource view only gets non-keyboard messages (like refresh ticks)
		switch msg.(type) {
		case tea.KeyMsg:
			// Keyboard events go only to log view
			logModel, cmd := a.logView.Update(msg)
			a.logView = logModel.(*views.LogView)
			cmds = append(cmds, cmd)
		default:
			// Non-keyboard events go to both views
			resourceModel, cmd := a.resourceView.Update(msg)
			a.resourceView = resourceModel.(*views.ResourceView)
			cmds = append(cmds, cmd)

			logModel, cmd := a.logView.Update(msg)
			a.logView = logModel.(*views.LogView)
			cmds = append(cmds, cmd)
		}
	} else {
		// Full resource view
		resourceModel, cmd := a.resourceView.Update(msg)
		a.resourceView = resourceModel.(*views.ResourceView)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// View renders the application
func (a *App) View() string {
	if !a.ready {
		return "Initializing..."
	}

	// Show delete confirmation over everything if active
	if a.showDeleteConfirm && a.confirmView != nil {
		return a.confirmView.View()
	}

	// Show namespace popup over everything if active
	if a.showNamespacePopup && a.namespaceView != nil {
		return a.namespaceView.View()
	}

	if a.state.ShowHelp {
		return a.helpView.View()
	}

	if a.state.ShowLogs {
		// Split view
		resourceHeight := a.height / 2
		logHeight := a.height - resourceHeight - 1

		topView := lipgloss.NewStyle().
			Height(resourceHeight).
			Render(a.resourceView.View())

		bottomView := lipgloss.NewStyle().
			Height(logHeight).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			Render(a.logView.View())

		return lipgloss.JoinVertical(lipgloss.Left, topView, bottomView)
	}

	return a.resourceView.View()
}

// nextResourceType cycles to the next resource type
func (a *App) nextResourceType() {
	types := []core.ResourceType{
		core.ResourceTypePod,
		core.ResourceTypeDeployment,
		core.ResourceTypeStatefulSet,
		core.ResourceTypeService,
		core.ResourceTypeIngress,
		core.ResourceTypeConfigMap,
		core.ResourceTypeSecret,
	}

	current := a.state.CurrentResourceType
	for i, t := range types {
		if t == current {
			next := types[(i+1)%len(types)]
			a.state.SetResourceType(next)
			return
		}
	}
}

// prevResourceType cycles to the previous resource type
func (a *App) prevResourceType() {
	types := []core.ResourceType{
		core.ResourceTypePod,
		core.ResourceTypeDeployment,
		core.ResourceTypeStatefulSet,
		core.ResourceTypeService,
		core.ResourceTypeIngress,
		core.ResourceTypeConfigMap,
		core.ResourceTypeSecret,
	}

	current := a.state.CurrentResourceType
	for i, t := range types {
		if t == current {
			prev := types[(i-1+len(types))%len(types)]
			a.state.SetResourceType(prev)
			return
		}
	}
}

// startWatcher starts watching for resource changes
func (a *App) startWatcher() tea.Cmd {
	// Cancel any existing watcher
	if a.cancelWatcher != nil {
		a.cancelWatcher()
	}

	// Create new context for watcher
	ctx, cancel := context.WithCancel(a.ctx)
	a.watcherCtx = ctx
	a.cancelWatcher = cancel

	return func() tea.Msg {
		// Start watcher based on current resource type
		var watcher watch.Interface
		var err error

		switch a.state.CurrentResourceType {
		case core.ResourceTypePod:
			watcher, err = a.k8sClient.WatchPods(ctx, a.state.CurrentNamespace)
		case core.ResourceTypeDeployment:
			watcher, err = a.k8sClient.WatchDeployments(ctx, a.state.CurrentNamespace)
		case core.ResourceTypeStatefulSet:
			watcher, err = a.k8sClient.WatchStatefulSets(ctx, a.state.CurrentNamespace)
		case core.ResourceTypeService:
			watcher, err = a.k8sClient.WatchServices(ctx, a.state.CurrentNamespace)
		case core.ResourceTypeIngress:
			watcher, err = a.k8sClient.WatchIngresses(ctx, a.state.CurrentNamespace)
		case core.ResourceTypeConfigMap:
			watcher, err = a.k8sClient.WatchConfigMaps(ctx, a.state.CurrentNamespace)
		case core.ResourceTypeSecret:
			watcher, err = a.k8sClient.WatchSecrets(ctx, a.state.CurrentNamespace)
		default:
			return nil
		}

		if err != nil {
			return nil // Silently fail for now
		}

		// Watch for events
		go func() {
			defer watcher.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case _, ok := <-watcher.ResultChan():
					if !ok {
						return
					}
					// For now, just trigger a refresh
					// In a full implementation, we'd send the event as a message
				}
			}
		}()

		return nil
	}
}

// startRefreshTimer returns a command that sends a tick message after the configured interval
func (a *App) startRefreshTimer() tea.Cmd {
	interval := time.Duration(a.config.RefreshInterval) * time.Second
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// openNamespaceSelector opens the namespace selection popup
func (a *App) openNamespaceSelector() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		namespaces, err := a.k8sClient.ListNamespaces(ctx)
		if err != nil {
			// Return error message
			return errMsg{err}
		}

		// Create namespace view
		a.namespaceView = views.NewNamespaceView(namespaces, a.state.CurrentNamespace)
		a.namespaceView.SetSize(a.width, a.height)
		a.showNamespacePopup = true

		return nil
	}
}

// Message types
type errMsg struct{ err error }
type deleteCompleteMsg struct{ name string }
