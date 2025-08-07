package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/HamStudy/kubewatch/internal/components/dropdown"
	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/k8s"
	"github.com/HamStudy/kubewatch/internal/ui/views"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// KeyMap defines the key bindings
type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Left          key.Binding
	Right         key.Binding
	Enter         key.Binding
	Space         key.Binding
	Tab           key.Binding
	ShiftTab      key.Binding
	Delete        key.Binding
	Logs          key.Binding
	Help          key.Binding
	Quit          key.Binding
	Refresh       key.Binding
	ContextSwitch key.Binding
	SortToggle    key.Binding
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
			key.WithKeys("delete", "D"),
			key.WithHelp("Del/D", "delete"),
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
		ContextSwitch: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "switch context"),
		),
		SortToggle: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort toggle"),
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

	// Multi-context support
	multiClient         *k8s.MultiContextClient
	isMultiContext      bool
	contextView         *views.ContextView
	showContextSelector bool
	activeContexts      []string

	// Views
	resourceView         *views.ResourceView
	logView              *views.LogView
	helpView             *views.HelpView
	namespaceView        *views.NamespaceView
	confirmView          *views.ConfirmView
	describeView         *views.DescribeView
	resourceSelectorView *views.ResourceSelectorView

	// Screen mode system
	currentMode  ScreenModeType
	previousMode ScreenModeType
	modes        map[ScreenModeType]ScreenMode

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
	// Always use multi-context mode - get current context and create multi-client
	var activeContexts []string
	if _, currentCtx, err := k8s.GetAvailableContexts(); err == nil && currentCtx != "" {
		activeContexts = []string{currentCtx}
	}

	// Create multi-context client even for single context
	var multiClient *k8s.MultiContextClient
	if len(activeContexts) > 0 {
		if mc, err := k8s.NewMultiContextClient(activeContexts); err == nil {
			multiClient = mc
		}
	}

	app := &App{
		ctx:            ctx,
		multiClient:    multiClient,
		state:          state,
		config:         config,
		keys:           DefaultKeyMap(),
		resourceView:   views.NewResourceViewWithMultiContext(state, multiClient),
		logView:        views.NewLogView(),
		helpView:       views.NewHelpView(),
		isMultiContext: true, // Always use multi-context mode
		activeContexts: activeContexts,
		currentMode:    ModeList,
		previousMode:   ModeList,
	}

	// Initialize screen modes
	app.modes = map[ScreenModeType]ScreenMode{
		ModeList:              NewListMode(),
		ModeLog:               NewLogMode(),
		ModeDescribe:          NewDescribeMode(),
		ModeHelp:              NewHelpMode(),
		ModeContextSelector:   NewContextSelectorMode(),
		ModeNamespaceSelector: NewNamespaceSelectorMode(),
		ModeConfirmDialog:     NewConfirmDialogMode(),
		ModeResourceSelector:  NewResourceSelectorMode(),
	}

	return app
}

// NewAppWithMultiContext creates a new application instance with multi-context support
func NewAppWithMultiContext(ctx context.Context, multiClient *k8s.MultiContextClient, state *core.State, config *core.Config) *App {
	app := &App{
		ctx:                  ctx,
		multiClient:          multiClient,
		state:                state,
		config:               config,
		keys:                 DefaultKeyMap(),
		resourceView:         views.NewResourceViewWithMultiContext(state, multiClient),
		logView:              views.NewLogView(),
		helpView:             views.NewHelpView(),
		resourceSelectorView: views.NewResourceSelectorView(),
		isMultiContext:       true, activeContexts: state.CurrentContexts,
		currentMode:  ModeList,
		previousMode: ModeList,
	}

	// Initialize screen modes
	app.modes = map[ScreenModeType]ScreenMode{
		ModeList:              NewListMode(),
		ModeLog:               NewLogMode(),
		ModeDescribe:          NewDescribeMode(),
		ModeHelp:              NewHelpMode(),
		ModeContextSelector:   NewContextSelectorMode(),
		ModeNamespaceSelector: NewNamespaceSelectorMode(),
		ModeConfirmDialog:     NewConfirmDialogMode(),
		ModeResourceSelector:  NewResourceSelectorMode(),
	}

	return app
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
		// Use the new mode system for key handling
		currentMode := a.getCurrentMode()
		handled, cmd := currentMode.HandleKey(msg, a)

		if handled {
			return a, cmd
		}

		// If not handled by mode, let the appropriate view handle it
		// This maintains compatibility with existing view-specific key handling
		switch a.currentMode {
		case ModeList:
			// Pass unhandled keys to resource view
			resourceModel, viewCmd := a.resourceView.Update(msg)
			a.resourceView = resourceModel.(*views.ResourceView)
			return a, viewCmd
		case ModeContextSelector:
			if a.contextView != nil {
				ctxModel, viewCmd := a.contextView.Update(msg)
				a.contextView = ctxModel.(*views.ContextView)
				return a, viewCmd
			}
		case ModeNamespaceSelector:
			if a.namespaceView != nil {
				nsModel, viewCmd := a.namespaceView.Update(msg)
				a.namespaceView = nsModel.(*views.NamespaceView)
				return a, viewCmd
			}
		case ModeConfirmDialog:
			if a.confirmView != nil {
				confirmModel, viewCmd := a.confirmView.Update(msg)
				a.confirmView = confirmModel.(*views.ConfirmView)

				// Check if the dialog was completed with 'y' or 'n'
				if a.confirmView.IsCompleted() {
					return a, a.handleConfirmDialogAction()
				}

				return a, viewCmd
			}
		case ModeLog:
			// Log view handles its own keys
			logModel, viewCmd := a.logView.Update(msg)
			a.logView = logModel.(*views.LogView)
			return a, viewCmd
		case ModeDescribe:
			if a.describeView != nil {
				describeModel, viewCmd := a.describeView.Update(msg)
				a.describeView = describeModel.(*views.DescribeView)
				return a, viewCmd
			}
		case ModeResourceSelector:
			if a.resourceSelectorView != nil {
				selectorModel, viewCmd := a.resourceSelectorView.Update(msg)
				a.resourceSelectorView = selectorModel.(*views.ResourceSelectorView)
				return a, viewCmd
			}
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
		if a.contextView != nil {
			a.contextView.SetSize(msg.Width, msg.Height)
		}
		return a, nil

	case deleteCompleteMsg:
		// Resource deleted successfully, refresh the list
		return a, a.resourceView.RefreshResources()

	case views.ContextInfoMsg:
		// Show context information
		return a, a.showContextInfo(msg.ContextName)

	case contextInfoDisplayMsg:
		// For now, just log the context info (in a real app, you might show a popup)
		// The context selector will remain open
		return a, nil

	case namespacesLoadedMsg:
		// Update namespace view with loaded namespaces
		if a.namespaceView != nil {
			if msg.err != nil {
				// Handle error - for now, just show empty list
				a.namespaceView.SetNamespaces([]v1.Namespace{})
			} else {
				a.namespaceView.SetNamespaces(msg.namespaces)
			}
		}
		return a, nil

	case dropdown.SelectedMsg:
		// Handle dropdown selection
		if a.currentMode == ModeResourceSelector {
			if resourceType, ok := msg.Option.Value.(core.ResourceType); ok {
				a.state.SetResourceType(resourceType)
				a.setMode(ModeList)
				return a, a.resourceView.RefreshResources()
			}
		}
		return a, nil

	case dropdown.CancelledMsg:
		// Handle dropdown cancellation
		if a.currentMode == ModeResourceSelector {
			a.setMode(ModeList)
		}
		return a, nil
	}

	// Update child views based on current mode
	switch a.currentMode {
	case ModeHelp:
		helpModel, cmd := a.helpView.Update(msg)
		a.helpView = helpModel.(*views.HelpView)
		cmds = append(cmds, cmd)

	case ModeLog:
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

	case ModeDescribe:
		if a.describeView != nil {
			describeModel, cmd := a.describeView.Update(msg)
			a.describeView = describeModel.(*views.DescribeView)
			cmds = append(cmds, cmd)
		}

	case ModeContextSelector:
		if a.contextView != nil {
			ctxModel, cmd := a.contextView.Update(msg)
			a.contextView = ctxModel.(*views.ContextView)
			cmds = append(cmds, cmd)
		}

	case ModeNamespaceSelector:
		if a.namespaceView != nil {
			nsModel, cmd := a.namespaceView.Update(msg)
			a.namespaceView = nsModel.(*views.NamespaceView)
			cmds = append(cmds, cmd)
		}

	case ModeConfirmDialog:
		if a.confirmView != nil {
			confirmModel, cmd := a.confirmView.Update(msg)
			a.confirmView = confirmModel.(*views.ConfirmView)
			cmds = append(cmds, cmd)
		}

	default:
		// Default to resource view (list mode)
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

	// Render based on current mode
	switch a.currentMode {
	case ModeConfirmDialog:
		if a.confirmView != nil {
			return a.confirmView.View()
		}

	case ModeNamespaceSelector:
		if a.namespaceView != nil {
			return a.namespaceView.View()
		}

	case ModeContextSelector:
		if a.contextView != nil {
			return a.contextView.View()
		}

	case ModeHelp:
		return a.helpView.View()

	case ModeResourceSelector:
		if a.resourceSelectorView != nil {
			return a.resourceSelectorView.View()
		}

	case ModeDescribe:
		if a.describeView != nil {
			return a.describeView.View()
		}

	case ModeLog:
		// Split view - give more space to logs, keep resource view compact
		minResourceHeight := 8 // Minimum height for resource view (header + 5-6 rows)
		resourceHeight := minResourceHeight

		// If we have more space, show a bit more context
		if a.height > 20 {
			resourceHeight = a.height / 3 // Give 1/3 to resources, 2/3 to logs
			if resourceHeight < minResourceHeight {
				resourceHeight = minResourceHeight
			}
		}

		logHeight := a.height - resourceHeight - 1

		// Update sizes for both views
		a.resourceView.SetSize(a.width, resourceHeight)
		a.logView.SetSize(a.width, logHeight)

		topView := lipgloss.NewStyle().
			Height(resourceHeight).
			MaxHeight(resourceHeight).
			Render(a.resourceView.View())

		bottomView := lipgloss.NewStyle().
			Height(logHeight).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			Render(a.logView.View())

		return lipgloss.JoinVertical(lipgloss.Left, topView, bottomView)
	}

	// Default to list mode (resource view)
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
	// For testing or when no clients are available, use mock namespaces
	if a.k8sClient == nil && a.multiClient == nil {
		// Create namespace view with test namespaces
		testNamespaces := []v1.Namespace{
			{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}},
		}
		a.namespaceView = views.NewNamespaceView(testNamespaces, a.state.CurrentNamespace)
		a.namespaceView.SetSize(a.width, a.height)
		a.showNamespacePopup = true
		a.setMode(ModeNamespaceSelector)
		return nil
	}

	// Create loading namespace view
	loadingMessage := "Loading namespaces..."
	if a.isMultiContext {
		loadingMessage = fmt.Sprintf("Loading namespaces from %d contexts...", len(a.activeContexts))
	}

	a.namespaceView = views.NewNamespaceViewWithLoading(a.state.CurrentNamespace, loadingMessage)
	a.namespaceView.SetSize(a.width, a.height)
	a.showNamespacePopup = true
	a.setMode(ModeNamespaceSelector)

	// Start async namespace loading
	return func() tea.Msg {
		ctx := context.Background()

		if a.isMultiContext && a.multiClient != nil {
			// Multi-context mode: get unique namespaces from all contexts
			namespaces, err := a.multiClient.GetUniqueNamespaces(ctx)
			return namespacesLoadedMsg{namespaces: namespaces, err: err}
		} else if a.k8sClient != nil {
			// Single-context mode
			namespaces, err := a.k8sClient.ListNamespaces(ctx)
			return namespacesLoadedMsg{namespaces: namespaces, err: err}
		}

		return namespacesLoadedMsg{namespaces: []v1.Namespace{}, err: fmt.Errorf("no kubernetes client available")}
	}
}

// openContextSelector opens the context selection popup
func (a *App) openContextSelector() tea.Cmd {
	// For testing or when k8s client is not available, use mock contexts
	if a.k8sClient == nil && a.multiClient == nil {
		// Create context view with test contexts
		testContexts := []string{"test-context", "context-1", "context-2"}
		a.contextView = views.NewContextView(testContexts, a.activeContexts)
		a.contextView.SetSize(a.width, a.height)
		a.showContextSelector = true
		a.setMode(ModeContextSelector)
		return nil
	}

	return func() tea.Msg {
		// Get available contexts
		contexts, _, err := k8s.GetAvailableContexts()
		if err != nil {
			return errMsg{err}
		}

		// Create context view with current selections
		a.contextView = views.NewContextView(contexts, a.activeContexts)
		a.contextView.SetSize(a.width, a.height)
		a.showContextSelector = true
		a.setMode(ModeContextSelector)

		return nil
	}
}

// getSelectedResourceContext returns the context of the currently selected resource in multi-context mode
func (a *App) getSelectedResourceContext() string {
	if !a.isMultiContext {
		return ""
	}

	return a.resourceView.GetSelectedResourceContext()
}

// cycleSortColumn cycles through available sort columns or toggles sort direction
func (a *App) cycleSortColumn() {
	// Get available columns for current resource type
	availableColumns := a.getAvailableSortColumns()

	currentColumn := a.state.SortColumn
	if currentColumn == "" {
		currentColumn = "NAME"
	}

	// Find current column index
	currentIndex := -1
	for i, col := range availableColumns {
		if col == currentColumn {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		// Column not found, start with first column
		a.state.SortColumn = availableColumns[0]
		a.state.SortAscending = true
	} else if currentIndex == len(availableColumns)-1 {
		// Last column, toggle direction
		a.state.SortAscending = !a.state.SortAscending
	} else {
		// Move to next column
		a.state.SortColumn = availableColumns[currentIndex+1]
		a.state.SortAscending = true
	}
}

// getAvailableSortColumns returns the sortable columns for the current resource type
func (a *App) getAvailableSortColumns() []string {
	switch a.state.CurrentResourceType {
	case core.ResourceTypePod:
		if a.isMultiContext {
			return []string{"CONTEXT", "NAME", "READY", "STATUS", "RESTARTS", "AGE"}
		}
		return []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"}
	case core.ResourceTypeDeployment:
		if a.isMultiContext {
			return []string{"CONTEXT", "NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
		}
		return []string{"NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
	case core.ResourceTypeService:
		if a.isMultiContext {
			return []string{"CONTEXT", "NAME", "TYPE", "CLUSTER-IP", "AGE"}
		}
		return []string{"NAME", "TYPE", "CLUSTER-IP", "AGE"}
	default:
		if a.isMultiContext {
			return []string{"CONTEXT", "NAME", "AGE"}
		}
		return []string{"NAME", "AGE"}
	}
}

// Mode management methods

// setMode changes the current screen mode
func (a *App) setMode(mode ScreenModeType) {
	a.previousMode = a.currentMode
	a.currentMode = mode

	// Update legacy state flags for compatibility
	switch mode {
	case ModeHelp:
		a.state.ShowHelp = true
	case ModeLog:
		a.state.ShowLogs = true
	case ModeContextSelector:
		a.showContextSelector = true
	case ModeNamespaceSelector:
		a.showNamespacePopup = true
	case ModeConfirmDialog:
		a.showDeleteConfirm = true
	default:
		// Clear all legacy flags when returning to list mode
		a.state.ShowHelp = false
		a.state.ShowLogs = false
		a.showContextSelector = false
		a.showNamespacePopup = false
		a.showDeleteConfirm = false
	}
}

// returnToPreviousMode returns to the previous screen mode
func (a *App) returnToPreviousMode() {
	a.setMode(a.previousMode)
}

// getCurrentMode returns the current screen mode handler
func (a *App) getCurrentMode() ScreenMode {
	if mode, exists := a.modes[a.currentMode]; exists {
		return mode
	}
	return a.modes[ModeList] // fallback
}

// Mode-specific action methods

// startDescribeView starts the describe view for a resource
func (a *App) startDescribeView(resourceName string) tea.Cmd {
	resourceType := string(a.state.CurrentResourceType)
	namespace := a.state.CurrentNamespace
	context := ""

	if a.isMultiContext {
		context = a.getSelectedResourceContext()
	}

	a.describeView = views.NewDescribeView(resourceType, resourceName, namespace, context)
	a.describeView.SetSize(a.width, a.height)

	// Use the appropriate client
	if a.isMultiContext && context != "" {
		if client, err := a.multiClient.GetClient(context); err == nil {
			return a.describeView.LoadDescribeWithClient(a.ctx, client)
		}
	} else if a.k8sClient != nil {
		return a.describeView.LoadDescribeWithClient(a.ctx, a.k8sClient)
	}

	// Fallback to placeholder content
	return a.describeView.Init()
}

// showDeleteConfirmation shows the delete confirmation dialog
func (a *App) showDeleteConfirmation(resourceName string) tea.Cmd {
	a.pendingDeleteName = resourceName
	resourceType := string(a.state.CurrentResourceType)

	// Remove the 's' at the end for singular form
	if strings.HasSuffix(resourceType, "s") {
		resourceType = resourceType[:len(resourceType)-1]
	}

	message := fmt.Sprintf("Are you sure you want to delete %s '%s'?",
		strings.ToLower(resourceType), resourceName)
	a.confirmView = views.NewConfirmView("⚠️  Confirm Deletion", message)
	a.confirmView.SetSize(a.width, a.height)
	a.confirmView.SetConfirmText("Delete")
	a.confirmView.SetCancelText("Cancel")

	return nil
}

// applyContextSelection applies the selected contexts
func (a *App) applyContextSelection() tea.Cmd {
	if a.contextView == nil {
		return nil
	}
	newContexts := a.contextView.GetSelectedContexts()
	if len(newContexts) > 0 {
		// Show loading indicators for selected contexts
		for _, ctx := range newContexts {
			a.contextView.SetContextLoading(ctx, true)
		}

		a.activeContexts = newContexts
		a.state.SetCurrentContexts(newContexts)

		// Always use multi-context mode regardless of number of contexts
		multiClient, err := k8s.NewMultiContextClient(newContexts)
		if err == nil {
			a.multiClient = multiClient
			a.k8sClient = nil
			a.isMultiContext = true
			// Update resource view with multi-client
			a.resourceView = views.NewResourceViewWithMultiContext(a.state, multiClient)
			a.resourceView.SetSize(a.width, a.height)
		}

		// Clear loading indicators
		for _, ctx := range newContexts {
			a.contextView.SetContextLoading(ctx, false)
		}

		// Refresh resources with new contexts
		a.setMode(ModeList)
		return a.resourceView.RefreshResources()
	}
	a.setMode(ModeList)
	return nil
}

// applyNamespaceSelection applies the selected namespace
func (a *App) applyNamespaceSelection() tea.Cmd {
	newNamespace := a.namespaceView.GetSelectedNamespace()
	if newNamespace != a.state.CurrentNamespace {
		a.state.CurrentNamespace = newNamespace
		a.config.CurrentNamespace = newNamespace
		// Refresh resources with new namespace
		a.setMode(ModeList)
		return a.resourceView.RefreshResources()
	}
	a.setMode(ModeList)
	return nil
}

// handleConfirmDialogAction handles the confirm dialog action
func (a *App) handleConfirmDialogAction() tea.Cmd {
	if a.confirmView.IsConfirmed() {
		// Proceed with deletion
		a.setMode(ModeList)
		return a.resourceView.DeleteSelected()
	}
	// Cancelled
	a.setMode(ModeList)
	a.pendingDeleteName = ""
	return nil
}

// showContextInfo displays detailed information about a context
func (a *App) showContextInfo(contextName string) tea.Cmd {
	return func() tea.Msg {
		// Get context information
		contexts, currentCtx, err := k8s.GetAvailableContexts()
		if err != nil {
			return errMsg{err}
		}

		// Find the context details
		info := fmt.Sprintf("Context: %s\n", contextName)
		if contextName == currentCtx {
			info += "Status: Current context\n"
		} else {
			info += "Status: Available\n"
		}

		// Add more context details here if available
		info += fmt.Sprintf("Total contexts available: %d\n", len(contexts))

		// For now, just show a simple info message
		// In a real implementation, you might want to create a dedicated info view
		return contextInfoDisplayMsg{
			contextName: contextName,
			info:        info,
		}
	}
}

// openResourceSelector opens the resource type selection dropdown
func (a *App) openResourceSelector() tea.Cmd {
	if a.resourceSelectorView == nil {
		a.resourceSelectorView = views.NewResourceSelectorView()
	}

	// Set current resource type in the dropdown
	a.resourceSelectorView.SetCurrentResourceType(a.state.CurrentResourceType)
	a.resourceSelectorView.SetSize(a.width, a.height)
	a.resourceSelectorView.Open()
	a.setMode(ModeResourceSelector)

	return nil
}

// applyResourceSelection applies the selected resource type
func (a *App) applyResourceSelection() tea.Cmd {
	if a.resourceSelectorView == nil {
		a.setMode(ModeList)
		return nil
	}

	// Get the selected resource type from the dropdown
	selectedOption := a.resourceSelectorView.GetSelectedOption()
	if resourceType, ok := selectedOption.Value.(core.ResourceType); ok {
		a.state.SetResourceType(resourceType)
		a.setMode(ModeList)
		return a.resourceView.RefreshResources()
	}

	a.setMode(ModeList)
	return nil
}

// Message types
type errMsg struct{ err error }
type deleteCompleteMsg struct{ name string }
type contextSelectionMsg struct{ contexts []string }
type contextInfoDisplayMsg struct {
	contextName string
	info        string
}
type namespacesLoadedMsg struct {
	namespaces []v1.Namespace
	err        error
}
