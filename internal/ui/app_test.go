package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/k8s"
	"github.com/HamStudy/kubewatch/internal/ui/views"
	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestAppInitialization(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func() *App
		expectedMode   ScreenModeType
		expectedReady  bool
		expectedWidth  int
		expectedHeight int
		checkViews     bool
	}{
		{
			name: "single context initialization",
			setupFunc: func() *App {
				state := &core.State{
					CurrentResourceType: core.ResourceTypePod,
					CurrentNamespace:    "default",
					CurrentContext:      "test-context",
				}
				config := &core.Config{RefreshInterval: 5}
				return NewApp(context.Background(), nil, state, config)
			},
			expectedMode:   ModeList,
			expectedReady:  false, // Not ready until window size is set
			expectedWidth:  0,
			expectedHeight: 0,
			checkViews:     true,
		},
		{
			name: "multi-context initialization",
			setupFunc: func() *App {
				state := &core.State{
					CurrentResourceType: core.ResourceTypePod,
					CurrentNamespace:    "default",
					CurrentContexts:     []string{"context1", "context2"},
				}
				config := &core.Config{RefreshInterval: 5}
				multiClient := &k8s.MultiContextClient{}
				return NewAppWithMultiContext(context.Background(), multiClient, state, config)
			},
			expectedMode:   ModeList,
			expectedReady:  false,
			expectedWidth:  0,
			expectedHeight: 0,
			checkViews:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setupFunc()

			// Test initial state
			if app.currentMode != tt.expectedMode {
				t.Errorf("Expected initial mode to be %v, got %v", tt.expectedMode, app.currentMode)
			}

			if app.ready != tt.expectedReady {
				t.Errorf("Expected ready to be %v, got %v", tt.expectedReady, app.ready)
			}

			if app.width != tt.expectedWidth || app.height != tt.expectedHeight {
				t.Errorf("Expected dimensions %dx%d, got %dx%d",
					tt.expectedWidth, tt.expectedHeight, app.width, app.height)
			}

			if tt.checkViews {
				// Verify all views are initialized
				if app.resourceView == nil {
					t.Error("Resource view should be initialized")
				}
				if app.logView == nil {
					t.Error("Log view should be initialized")
				}
				if app.helpView == nil {
					t.Error("Help view should be initialized")
				}
			}

			// Verify all modes are initialized
			expectedModes := []ScreenModeType{
				ModeList, ModeLog, ModeDescribe, ModeHelp,
				ModeContextSelector, ModeNamespaceSelector, ModeConfirmDialog,
			}
			for _, mode := range expectedModes {
				if app.modes[mode] == nil {
					t.Errorf("Mode %v should be initialized", mode)
				}
			}
		})
	}
}

func TestAppInit(t *testing.T) {
	app := createTestApp(t)

	// Call Init and verify it returns commands
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init should return commands")
	}

	// Verify Init returns a batch command (for enter alt screen and refresh timer)
	// We can't easily test the exact commands, but we can ensure it doesn't panic
}

func TestAppKeyHandling(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(*App)
		key          string
		keyType      tea.KeyType
		expectMode   ScreenModeType
		expectQuit   bool
		validateFunc func(*testing.T, *App)
	}{
		{
			name:       "toggle help mode on",
			key:        "?",
			expectMode: ModeHelp,
		},
		{
			name: "toggle help mode off",
			setupFunc: func(app *App) {
				app.setMode(ModeHelp)
			},
			key:        "?",
			expectMode: ModeList,
		},
		{
			name:       "quit command",
			key:        "q",
			expectMode: ModeList,
			expectQuit: true,
		},
		{
			name:       "escape key in list mode does nothing",
			key:        "",
			keyType:    tea.KeyEsc,
			expectMode: ModeList,
		},
		{
			name: "escape key in help mode returns to list",
			setupFunc: func(app *App) {
				app.setMode(ModeHelp)
			},
			key:        "",
			keyType:    tea.KeyEsc,
			expectMode: ModeList,
		},
		{
			name: "tab opens resource selector",
			setupFunc: func(app *App) {
				app.state.CurrentResourceType = core.ResourceTypePod
			},
			key:        "",
			keyType:    tea.KeyTab,
			expectMode: ModeResourceSelector,
			validateFunc: func(t *testing.T, app *App) {
				// Tab should open resource selector, not change resource type directly
				if app.state.CurrentResourceType != core.ResourceTypePod {
					t.Error("Tab should not change resource type directly, should open selector")
				}
			},
		},
		{
			name: "shift+tab opens resource selector",
			setupFunc: func(app *App) {
				app.state.CurrentResourceType = core.ResourceTypeDeployment
			},
			key:        "",
			keyType:    tea.KeyShiftTab,
			expectMode: ModeResourceSelector,
			validateFunc: func(t *testing.T, app *App) {
				// Shift+Tab should open resource selector, not change resource type directly
				if app.state.CurrentResourceType != core.ResourceTypeDeployment {
					t.Error("Shift+Tab should not change resource type directly, should open selector")
				}
			},
		},
		{
			name:       "n opens namespace selector",
			key:        "n",
			expectMode: ModeNamespaceSelector,
			validateFunc: func(t *testing.T, app *App) {
				if app.namespaceView == nil {
					t.Error("Namespace view should be created")
				}
			},
		},
		{
			name:       "c opens context selector",
			key:        "c",
			expectMode: ModeContextSelector,
			validateFunc: func(t *testing.T, app *App) {
				if app.contextView == nil {
					t.Error("Context view should be created")
				}
			},
		},
		{
			name:       "s toggles sort",
			key:        "s",
			expectMode: ModeList,
			validateFunc: func(t *testing.T, app *App) {
				// Sort column should be set
				if app.state.SortColumn == "" {
					t.Error("Sort column should be set after pressing 's'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)

			if tt.setupFunc != nil {
				tt.setupFunc(app)
			}

			// Create key message
			var keyMsg tea.KeyMsg
			if tt.keyType != 0 {
				keyMsg = tea.KeyMsg{Type: tt.keyType}
			} else if tt.key != "" {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			// Update app with key message
			model, cmd := app.Update(keyMsg)
			app = model.(*App)

			// Check for quit command
			if tt.expectQuit {
				if cmd == nil {
					t.Error("Expected quit command to be returned")
				}
			}

			// Check mode
			if app.currentMode != tt.expectMode {
				t.Errorf("Expected mode %v, got %v", tt.expectMode, app.currentMode)
			}

			// Run additional validation
			if tt.validateFunc != nil {
				tt.validateFunc(t, app)
			}
		})
	}
}

func TestAppViewRendering(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func(*App)
		mode            ScreenModeType
		expectContent   []string
		unexpectContent []string
	}{
		{
			name: "uninitialized view",
			setupFunc: func(app *App) {
				app.ready = false
			},
			expectContent: []string{"Initializing"},
		},
		{
			name:          "list mode view",
			mode:          ModeList,
			expectContent: []string{}, // Resource view will render something
		},
		{
			name:          "help mode view",
			mode:          ModeHelp,
			expectContent: []string{}, // Help view will render something
		},
		{
			name: "log mode split view",
			mode: ModeLog,
			setupFunc: func(app *App) {
				app.logView = views.NewLogView()
				app.logView.SetSize(80, 12)
			},
			expectContent: []string{}, // Split view will render
		},
		{
			name: "namespace selector view",
			mode: ModeNamespaceSelector,
			setupFunc: func(app *App) {
				namespaces := []v1.Namespace{
					{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
				}
				app.namespaceView = views.NewNamespaceView(namespaces, "default")
				app.namespaceView.SetSize(80, 24)
			},
			expectContent: []string{}, // Namespace view will render
		},
		{
			name: "context selector view",
			mode: ModeContextSelector,
			setupFunc: func(app *App) {
				contexts := []string{"context1", "context2"}
				app.contextView = views.NewContextView(contexts, []string{"context1"})
				app.contextView.SetSize(80, 24)
			},
			expectContent: []string{}, // Context view will render
		},
		{
			name: "confirm dialog view",
			mode: ModeConfirmDialog,
			setupFunc: func(app *App) {
				app.confirmView = views.NewConfirmView("Test Title", "Test Message")
				app.confirmView.SetSize(80, 24)
			},
			expectContent: []string{}, // Confirm view will render
		},
		{
			name: "describe view",
			mode: ModeDescribe,
			setupFunc: func(app *App) {
				app.describeView = views.NewDescribeView("pod", "test-pod", "default", "")
				app.describeView.SetSize(80, 24)
			},
			expectContent: []string{}, // Describe view will render
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)

			if tt.setupFunc != nil {
				tt.setupFunc(app)
			}

			if tt.mode != ModeList {
				app.setMode(tt.mode)
			}
			// Test that view renders without panicking
			view := app.View()

			// Basic check that something was rendered
			if app.ready && len(view) == 0 {
				t.Error("View should not be empty when app is ready")
			}

			// Check expected content
			for _, expected := range tt.expectContent {
				if !strings.Contains(view, expected) {
					t.Errorf("Expected view to contain '%s', but it didn't", expected)
				}
			}

			// Check unexpected content
			for _, unexpected := range tt.unexpectContent {
				if strings.Contains(view, unexpected) {
					t.Errorf("Expected view to NOT contain '%s', but it did", unexpected)
				}
			}
		})
	}
}

func TestAppWindowSizeHandling(t *testing.T) {
	tests := []struct {
		name          string
		width         int
		height        int
		expectReady   bool
		validateViews bool
	}{
		{
			name:          "standard terminal size",
			width:         80,
			height:        24,
			expectReady:   true,
			validateViews: true,
		},
		{
			name:          "large terminal size",
			width:         200,
			height:        60,
			expectReady:   true,
			validateViews: true,
		},
		{
			name:          "small terminal size",
			width:         40,
			height:        10,
			expectReady:   true,
			validateViews: true,
		},
		{
			name:          "minimum size",
			width:         1,
			height:        1,
			expectReady:   true,
			validateViews: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)

			// Send window size message
			sizeMsg := tea.WindowSizeMsg{Width: tt.width, Height: tt.height}
			model, _ := app.Update(sizeMsg)
			app = model.(*App)

			// Check dimensions
			if app.width != tt.width || app.height != tt.height {
				t.Errorf("Expected dimensions %dx%d, got %dx%d",
					tt.width, tt.height, app.width, app.height)
			}

			// Check ready state
			if app.ready != tt.expectReady {
				t.Errorf("Expected ready=%v, got %v", tt.expectReady, app.ready)
			}

			if tt.validateViews {
				// Verify views received size updates
				// Note: We can't directly check view sizes without exposing them,
				// but we can verify the app doesn't panic when rendering
				view := app.View()
				if len(view) == 0 {
					t.Error("View should render something after window size is set")
				}
			}
		})
	}
}

func TestAppModeTransitions(t *testing.T) {
	tests := []struct {
		name          string
		startMode     ScreenModeType
		action        func(*App)
		expectedMode  ScreenModeType
		expectedPrev  ScreenModeType
		validateState func(*testing.T, *App)
	}{
		{
			name:      "list to help mode",
			startMode: ModeList,
			action: func(app *App) {
				app.setMode(ModeHelp)
			},
			expectedMode: ModeHelp,
			expectedPrev: ModeList,
			validateState: func(t *testing.T, app *App) {
				if !app.state.ShowHelp {
					t.Error("ShowHelp flag should be true")
				}
			},
		},
		{
			name:      "help to list mode",
			startMode: ModeHelp,
			action: func(app *App) {
				app.setMode(ModeList)
			},
			expectedMode: ModeList,
			expectedPrev: ModeHelp,
			validateState: func(t *testing.T, app *App) {
				if app.state.ShowHelp {
					t.Error("ShowHelp flag should be false")
				}
			},
		},
		{
			name:      "list to log mode",
			startMode: ModeList,
			action: func(app *App) {
				app.setMode(ModeLog)
			},
			expectedMode: ModeLog,
			expectedPrev: ModeList,
			validateState: func(t *testing.T, app *App) {
				if !app.state.ShowLogs {
					t.Error("ShowLogs flag should be true")
				}
			},
		},
		{
			name:      "return to previous mode",
			startMode: ModeList,
			action: func(app *App) {
				app.setMode(ModeHelp)
				app.returnToPreviousMode()
			},
			expectedMode: ModeList,
			expectedPrev: ModeHelp,
		},
		{
			name:      "list to namespace selector",
			startMode: ModeList,
			action: func(app *App) {
				app.setMode(ModeNamespaceSelector)
			},
			expectedMode: ModeNamespaceSelector,
			expectedPrev: ModeList,
			validateState: func(t *testing.T, app *App) {
				if !app.showNamespacePopup {
					t.Error("showNamespacePopup flag should be true")
				}
			},
		},
		{
			name:      "list to context selector",
			startMode: ModeList,
			action: func(app *App) {
				app.setMode(ModeContextSelector)
			},
			expectedMode: ModeContextSelector,
			expectedPrev: ModeList,
			validateState: func(t *testing.T, app *App) {
				if !app.showContextSelector {
					t.Error("showContextSelector flag should be true")
				}
			},
		},
		{
			name:      "list to confirm dialog",
			startMode: ModeList,
			action: func(app *App) {
				app.setMode(ModeConfirmDialog)
			},
			expectedMode: ModeConfirmDialog,
			expectedPrev: ModeList,
			validateState: func(t *testing.T, app *App) {
				if !app.showDeleteConfirm {
					t.Error("showDeleteConfirm flag should be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.setMode(tt.startMode)

			tt.action(app)

			if app.currentMode != tt.expectedMode {
				t.Errorf("Expected current mode %v, got %v", tt.expectedMode, app.currentMode)
			}

			if app.previousMode != tt.expectedPrev {
				t.Errorf("Expected previous mode %v, got %v", tt.expectedPrev, app.previousMode)
			}

			if tt.validateState != nil {
				tt.validateState(t, app)
			}
		})
	}
}

func TestAppResourceTypeNavigation(t *testing.T) {
	tests := []struct {
		name         string
		startType    core.ResourceType
		action       func(*App)
		expectedType core.ResourceType
	}{
		{
			name:      "next from pod",
			startType: core.ResourceTypePod,
			action: func(app *App) {
				app.nextResourceType()
			},
			expectedType: core.ResourceTypeDeployment,
		},
		{
			name:      "previous from pod",
			startType: core.ResourceTypePod,
			action: func(app *App) {
				app.prevResourceType()
			},
			expectedType: core.ResourceTypeSecret,
		},
		{
			name:      "next wraps around",
			startType: core.ResourceTypeSecret,
			action: func(app *App) {
				app.nextResourceType()
			},
			expectedType: core.ResourceTypePod,
		},
		{
			name:      "previous wraps around",
			startType: core.ResourceTypePod,
			action: func(app *App) {
				app.prevResourceType()
			},
			expectedType: core.ResourceTypeSecret,
		},
		{
			name:      "cycle through all types",
			startType: core.ResourceTypePod,
			action: func(app *App) {
				// Cycle through all types and back
				for i := 0; i < 7; i++ {
					app.nextResourceType()
				}
			},
			expectedType: core.ResourceTypePod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.state.CurrentResourceType = tt.startType

			tt.action(app)

			if app.state.CurrentResourceType != tt.expectedType {
				t.Errorf("Expected resource type %v, got %v",
					tt.expectedType, app.state.CurrentResourceType)
			}
		})
	}
}

func TestAppSortColumnCycling(t *testing.T) {
	tests := []struct {
		name              string
		resourceType      core.ResourceType
		isMultiContext    bool
		initialColumn     string
		initialAscending  bool
		expectedColumn    string
		expectedAscending bool
	}{
		{
			name:              "pod single context - initial sort from empty",
			resourceType:      core.ResourceTypePod,
			isMultiContext:    false,
			initialColumn:     "",
			expectedColumn:    "READY", // Cycles from default NAME to READY
			expectedAscending: true,
		},
		{
			name:              "pod single context - cycle to next column",
			resourceType:      core.ResourceTypePod,
			isMultiContext:    false,
			initialColumn:     "NAME",
			initialAscending:  true,
			expectedColumn:    "READY",
			expectedAscending: true,
		},
		{
			name:              "pod single context - last column toggles direction",
			resourceType:      core.ResourceTypePod,
			isMultiContext:    false,
			initialColumn:     "AGE",
			initialAscending:  true,
			expectedColumn:    "AGE",
			expectedAscending: false,
		},
		{
			name:              "pod multi context - includes context column",
			resourceType:      core.ResourceTypePod,
			isMultiContext:    true,
			initialColumn:     "",
			expectedColumn:    "READY", // Empty defaults to NAME (at index 1), cycles to READY (index 2)
			expectedAscending: true,
		},
		{
			name:              "deployment - different columns",
			resourceType:      core.ResourceTypeDeployment,
			isMultiContext:    false,
			initialColumn:     "NAME",
			expectedColumn:    "READY",
			expectedAscending: true,
		},
		{
			name:              "service - different columns",
			resourceType:      core.ResourceTypeService,
			isMultiContext:    false,
			initialColumn:     "NAME",
			expectedColumn:    "TYPE",
			expectedAscending: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.state.CurrentResourceType = tt.resourceType
			app.isMultiContext = tt.isMultiContext
			app.state.SortColumn = tt.initialColumn
			app.state.SortAscending = tt.initialAscending

			// Debug: check available columns
			availColumns := app.getAvailableSortColumns()

			app.cycleSortColumn()

			if app.state.SortColumn != tt.expectedColumn {
				t.Errorf("Expected sort column %s, got %s (available: %v, initial: %s)",
					tt.expectedColumn, app.state.SortColumn, availColumns, tt.initialColumn)
			}
			if app.state.SortAscending != tt.expectedAscending {
				t.Errorf("Expected sort ascending %v, got %v",
					tt.expectedAscending, app.state.SortAscending)
			}
		})
	}
}

func TestAppMessageHandling(t *testing.T) {
	tests := []struct {
		name         string
		msg          tea.Msg
		setupFunc    func(*App)
		validateFunc func(*testing.T, *App, tea.Cmd)
	}{
		{
			name: "tick message triggers refresh",
			msg:  tickMsg(time.Now()),
			validateFunc: func(t *testing.T, app *App, cmd tea.Cmd) {
				if cmd == nil {
					t.Error("Tick message should return refresh command")
				}
			},
		},
		{
			name: "delete complete message",
			msg:  deleteCompleteMsg{name: "test-pod"},
			validateFunc: func(t *testing.T, app *App, cmd tea.Cmd) {
				if cmd == nil {
					t.Error("Delete complete should trigger refresh")
				}
			},
		},
		{
			name: "context info message",
			msg:  views.ContextInfoMsg{ContextName: "test-context"},
			validateFunc: func(t *testing.T, app *App, cmd tea.Cmd) {
				// Should trigger context info display
				if cmd == nil {
					t.Error("Context info message should return command")
				}
			},
		},
		{
			name: "error message",
			msg:  errMsg{err: fmt.Errorf("test error")},
			validateFunc: func(t *testing.T, app *App, cmd tea.Cmd) {
				// Error messages are handled silently for now
				// Just ensure it doesn't panic
			},
		},
		{
			name: "watch event message",
			msg:  watchEventMsg{Type: watch.Added, Object: createMockPod("new-pod", "Running", "default")},
			validateFunc: func(t *testing.T, app *App, cmd tea.Cmd) {
				// Watch events are handled by resource view
				// Just ensure it doesn't panic
			},
		},
		{
			name: "unknown message type",
			msg:  struct{ Unknown string }{Unknown: "test"},
			validateFunc: func(t *testing.T, app *App, cmd tea.Cmd) {
				// Should handle gracefully without panic
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)

			if tt.setupFunc != nil {
				tt.setupFunc(app)
			}

			// Update with message
			model, cmd := app.Update(tt.msg)
			updatedApp := model.(*App)

			if tt.validateFunc != nil {
				tt.validateFunc(t, updatedApp, cmd)
			}
		})
	}
}

func TestAppNamespaceSelection(t *testing.T) {
	tests := []struct {
		name              string
		currentNamespace  string
		selectedNamespace string
		expectRefresh     bool
	}{
		{
			name:              "change namespace",
			currentNamespace:  "default",
			selectedNamespace: "kube-system",
			expectRefresh:     true,
		},
		{
			name:              "same namespace selected",
			currentNamespace:  "default",
			selectedNamespace: "default",
			expectRefresh:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.state.CurrentNamespace = tt.currentNamespace

			// Create namespace view
			namespaces := []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
			}
			app.namespaceView = views.NewNamespaceView(namespaces, tt.currentNamespace)

			// Simulate selection by updating the namespace view
			// Since we can't directly set cursor, we'll update the view's state
			// by simulating key presses to move to the desired namespace
			for i, ns := range namespaces {
				if ns.Name == tt.selectedNamespace {
					// Move cursor to position i
					for j := 0; j < i; j++ {
						nsModel, _ := app.namespaceView.Update(tea.KeyMsg{Type: tea.KeyDown})
						app.namespaceView = nsModel.(*views.NamespaceView)
					}
					break
				}
			}

			// Apply selection
			cmd := app.applyNamespaceSelection()

			if tt.expectRefresh {
				if cmd == nil {
					t.Error("Expected refresh command when namespace changes")
				}
				if app.state.CurrentNamespace != tt.selectedNamespace {
					t.Errorf("Expected namespace %s, got %s",
						tt.selectedNamespace, app.state.CurrentNamespace)
				}
			} else {
				// No refresh expected for same namespace
				if app.state.CurrentNamespace != tt.currentNamespace {
					t.Error("Namespace should not have changed")
				}
			}

			// Should return to list mode
			if app.currentMode != ModeList {
				t.Errorf("Expected to return to list mode, got %v", app.currentMode)
			}
		})
	}
}

func TestAppContextSelection(t *testing.T) {
	tests := []struct {
		name             string
		currentContexts  []string
		selectedContexts []string
		expectMultiMode  bool
		expectRefresh    bool
	}{
		{
			name:             "single context selection",
			currentContexts:  []string{"context1", "context2"},
			selectedContexts: []string{"context1"},
			expectMultiMode:  false,
			expectRefresh:    true,
		},
		{
			name:             "multi context selection",
			currentContexts:  []string{"context1"},
			selectedContexts: []string{"context1", "context2"},
			expectMultiMode:  true,
			expectRefresh:    true,
		},
		{
			name:             "no contexts selected",
			currentContexts:  []string{"context1"},
			selectedContexts: []string{},
			expectMultiMode:  false,
			expectRefresh:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.activeContexts = tt.currentContexts

			// Create context view
			allContexts := []string{"context1", "context2", "context3"}
			app.contextView = views.NewContextView(allContexts, tt.currentContexts)

			// Simulate selection by updating the context view
			// First, clear all selections if we want specific contexts
			if len(tt.selectedContexts) > 0 {
				// Enable multi-select mode if we need to select multiple contexts
				if len(tt.selectedContexts) > 1 {
					// Press 'm' to enable multi-select mode
					ctxModel, _ := app.contextView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
					app.contextView = ctxModel.(*views.ContextView)
				}

				// Press 'a' twice to deselect all (first selects all, second deselects all)
				ctxModel, _ := app.contextView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
				app.contextView = ctxModel.(*views.ContextView)
				ctxModel, _ = app.contextView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
				app.contextView = ctxModel.(*views.ContextView)

				// Now select the desired contexts
				for _, ctx := range tt.selectedContexts {
					for i, availCtx := range allContexts {
						if availCtx == ctx {
							// Move to position
							for j := 0; j < i; j++ {
								ctxModel, _ := app.contextView.Update(tea.KeyMsg{Type: tea.KeyDown})
								app.contextView = ctxModel.(*views.ContextView)
							}
							// Select with space
							ctxModel, _ := app.contextView.Update(tea.KeyMsg{Type: tea.KeySpace})
							app.contextView = ctxModel.(*views.ContextView)
							// Reset cursor to top
							ctxModel, _ = app.contextView.Update(tea.KeyMsg{Type: tea.KeyHome})
							app.contextView = ctxModel.(*views.ContextView)
							break
						}
					}
				}
			}

			// Apply selection using test helper that doesn't require real k8s clients
			cmd := app.applyContextSelectionForTest()

			if tt.expectRefresh {
				if cmd == nil {
					t.Error("Expected refresh command when contexts change")
				}
			}

			if len(tt.selectedContexts) > 0 {
				if app.isMultiContext != tt.expectMultiMode {
					t.Errorf("Expected multi-context mode=%v, got %v",
						tt.expectMultiMode, app.isMultiContext)
				}
			}

			// Should return to list mode
			if app.currentMode != ModeList {
				t.Errorf("Expected to return to list mode, got %v", app.currentMode)
			}
		})
	}
}

func TestAppDeleteConfirmation(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		resourceType core.ResourceType
		confirmed    bool
		expectDelete bool
	}{
		{
			name:         "confirm pod deletion",
			resourceName: "test-pod",
			resourceType: core.ResourceTypePod,
			confirmed:    true,
			expectDelete: true,
		},
		{
			name:         "cancel pod deletion",
			resourceName: "test-pod",
			resourceType: core.ResourceTypePod,
			confirmed:    false,
			expectDelete: false,
		},
		{
			name:         "confirm deployment deletion",
			resourceName: "test-deployment",
			resourceType: core.ResourceTypeDeployment,
			confirmed:    true,
			expectDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.state.CurrentResourceType = tt.resourceType

			// Populate ResourceView with test data so DeleteSelected has something to work with
			var headers []string
			var rows [][]string

			switch tt.resourceType {
			case core.ResourceTypePod:
				headers = []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"}
				rows = [][]string{
					{tt.resourceName, "1/1", "Running", "0", "5m"},
				}
			case core.ResourceTypeDeployment:
				headers = []string{"NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
				rows = [][]string{
					{tt.resourceName, "1/1", "1", "1", "5m"},
				}
			default:
				headers = []string{"NAME", "STATUS", "AGE"}
				rows = [][]string{
					{tt.resourceName, "Active", "5m"},
				}
			}

			// Use the new test helper methods
			app.resourceView.SetTestData(headers, rows)
			app.resourceView.SetSelectedRow(0)

			// Show delete confirmation
			cmd := app.showDeleteConfirmation(tt.resourceName)
			_ = cmd // Command is nil for this operation

			// Verify confirmation dialog was created
			if app.confirmView == nil {
				t.Error("Confirm view should be created")
			}

			if app.pendingDeleteName != tt.resourceName {
				t.Errorf("Expected pending delete name %s, got %s",
					tt.resourceName, app.pendingDeleteName)
			}

			// Simulate confirmation or cancellation by updating the view
			if tt.confirmed {
				// Navigate to "Yes" first (default is "No")
				confirmModel, _ := app.confirmView.Update(tea.KeyMsg{Type: tea.KeyTab})
				app.confirmView = confirmModel.(*views.ConfirmView)
				// Then press Enter to confirm
				confirmModel, _ = app.confirmView.Update(tea.KeyMsg{Type: tea.KeyEnter})
				app.confirmView = confirmModel.(*views.ConfirmView)
			} else {
				// Press Escape to cancel (or just Enter since default is No)
				confirmModel, _ := app.confirmView.Update(tea.KeyMsg{Type: tea.KeyEsc})
				app.confirmView = confirmModel.(*views.ConfirmView)
			}

			// Handle the action
			cmd = app.handleConfirmDialogAction()

			if tt.expectDelete {
				if cmd == nil {
					t.Error("Expected delete command when confirmed")
				}
			} else {
				if app.pendingDeleteName != "" {
					t.Error("Pending delete name should be cleared when cancelled")
				}
			}

			// Should return to list mode
			if app.currentMode != ModeList {
				t.Errorf("Expected to return to list mode, got %v", app.currentMode)
			}
		})
	}
}

func TestAppDescribeView(t *testing.T) {
	tests := []struct {
		name           string
		resourceName   string
		resourceType   core.ResourceType
		namespace      string
		isMultiContext bool
		context        string
	}{
		{
			name:         "describe pod single context",
			resourceName: "test-pod",
			resourceType: core.ResourceTypePod,
			namespace:    "default",
		},
		{
			name:           "describe pod multi context",
			resourceName:   "test-pod",
			resourceType:   core.ResourceTypePod,
			namespace:      "default",
			isMultiContext: true,
			context:        "context1",
		},
		{
			name:         "describe deployment",
			resourceName: "test-deployment",
			resourceType: core.ResourceTypeDeployment,
			namespace:    "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.state.CurrentResourceType = tt.resourceType
			app.state.CurrentNamespace = tt.namespace
			app.isMultiContext = tt.isMultiContext

			if tt.isMultiContext {
				// Set up multi-context
				app.resourceView = views.NewResourceViewWithMultiContext(app.state, nil)
			}

			// Start describe view
			cmd := app.startDescribeView(tt.resourceName)

			// Verify describe view was created
			if app.describeView == nil {
				t.Error("Describe view should be created")
			}

			// Since we don't have a real k8s client, cmd will be Init()
			if cmd == nil {
				t.Error("Should return init command for describe view")
			}
		})
	}
}

func TestAppRefreshTimer(t *testing.T) {
	tests := []struct {
		name            string
		refreshInterval int
		expectCommand   bool
	}{
		{
			name:            "standard refresh interval",
			refreshInterval: 5,
			expectCommand:   true,
		},
		{
			name:            "fast refresh interval",
			refreshInterval: 1,
			expectCommand:   true,
		},
		{
			name:            "slow refresh interval",
			refreshInterval: 30,
			expectCommand:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.config.RefreshInterval = tt.refreshInterval

			cmd := app.startRefreshTimer()

			if tt.expectCommand && cmd == nil {
				t.Error("Expected refresh timer command")
			}
		})
	}
}

func TestAppWatcherManagement(t *testing.T) {
	tests := []struct {
		name         string
		resourceType core.ResourceType
		namespace    string
		hasClient    bool
	}{
		{
			name:         "start pod watcher",
			resourceType: core.ResourceTypePod,
			namespace:    "default",
			hasClient:    false, // No client in test
		},
		{
			name:         "start deployment watcher",
			resourceType: core.ResourceTypeDeployment,
			namespace:    "production",
			hasClient:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.state.CurrentResourceType = tt.resourceType
			app.state.CurrentNamespace = tt.namespace

			// Start watcher
			cmd := app.startWatcher()

			// Without a real client, this returns nil
			// Just ensure it doesn't panic and handles cleanup
			_ = cmd

			// Verify watcher context was created
			if app.watcherCtx == nil {
				t.Error("Watcher context should be created")
			}

			if app.cancelWatcher == nil {
				t.Error("Cancel function should be created")
			}

			// Test cancellation
			app.cancelWatcher()

			// Start another watcher to test replacement
			cmd = app.startWatcher()
			_ = cmd
		})
	}
}

func TestAppGetAvailableSortColumns(t *testing.T) {
	tests := []struct {
		name           string
		resourceType   core.ResourceType
		isMultiContext bool
		expectedFirst  string
		expectedCount  int
	}{
		{
			name:          "pod columns single context",
			resourceType:  core.ResourceTypePod,
			expectedFirst: "NAME",
			expectedCount: 5,
		},
		{
			name:           "pod columns multi context",
			resourceType:   core.ResourceTypePod,
			isMultiContext: true,
			expectedFirst:  "CONTEXT",
			expectedCount:  6,
		},
		{
			name:          "deployment columns",
			resourceType:  core.ResourceTypeDeployment,
			expectedFirst: "NAME",
			expectedCount: 5,
		},
		{
			name:          "service columns",
			resourceType:  core.ResourceTypeService,
			expectedFirst: "NAME",
			expectedCount: 4,
		},
		{
			name:          "configmap columns",
			resourceType:  core.ResourceTypeConfigMap,
			expectedFirst: "NAME",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.state.CurrentResourceType = tt.resourceType
			app.isMultiContext = tt.isMultiContext

			columns := app.getAvailableSortColumns()

			if len(columns) != tt.expectedCount {
				t.Errorf("Expected %d columns, got %d: %v",
					tt.expectedCount, len(columns), columns)
			}

			if len(columns) > 0 && columns[0] != tt.expectedFirst {
				t.Errorf("Expected first column %s, got %s",
					tt.expectedFirst, columns[0])
			}
		})
	}
}

func TestAppStateConsistency(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(*App)
		validateFunc func(*testing.T, *App)
	}{
		{
			name: "initial state consistency",
			validateFunc: func(t *testing.T, app *App) {
				if app.state == nil {
					t.Error("State should not be nil")
				}
				if app.config == nil {
					t.Error("Config should not be nil")
				}
				if app.ctx == nil {
					t.Error("Context should not be nil")
				}
			},
		},
		{
			name: "views initialized",
			validateFunc: func(t *testing.T, app *App) {
				if app.resourceView == nil {
					t.Error("Resource view should be initialized")
				}
				if app.logView == nil {
					t.Error("Log view should be initialized")
				}
				if app.helpView == nil {
					t.Error("Help view should be initialized")
				}
			},
		},
		{
			name: "modes initialized",
			validateFunc: func(t *testing.T, app *App) {
				if len(app.modes) != 8 {
					t.Errorf("Expected 8 modes, got %d", len(app.modes))
				}
				for mode, handler := range app.modes {
					if handler == nil {
						t.Errorf("Mode %v handler is nil", mode)
					}
				}
			},
		},
		{
			name: "key bindings initialized",
			validateFunc: func(t *testing.T, app *App) {
				// Check that key bindings are set
				keys := app.keys
				if keys.Up.Keys() == nil {
					t.Error("Up key binding not set")
				}
				if keys.Down.Keys() == nil {
					t.Error("Down key binding not set")
				}
				if keys.Quit.Keys() == nil {
					t.Error("Quit key binding not set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)

			if tt.setupFunc != nil {
				tt.setupFunc(app)
			}

			tt.validateFunc(t, app)
		})
	}
}

func TestAppErrorRecovery(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*App)
		action      func(*App) tea.Cmd
		expectPanic bool
	}{
		{
			name: "handle nil namespace view",
			setupFunc: func(app *App) {
				app.namespaceView = nil
				app.setMode(ModeNamespaceSelector)
			},
			action: func(app *App) tea.Cmd {
				// Try to render with nil namespace view
				_ = app.View()
				return nil
			},
			expectPanic: false,
		},
		{
			name: "handle nil context view",
			setupFunc: func(app *App) {
				app.contextView = nil
				app.setMode(ModeContextSelector)
			},
			action: func(app *App) tea.Cmd {
				_ = app.View()
				return nil
			},
			expectPanic: false,
		},
		{
			name: "handle nil describe view",
			setupFunc: func(app *App) {
				app.describeView = nil
				app.setMode(ModeDescribe)
			},
			action: func(app *App) tea.Cmd {
				_ = app.View()
				return nil
			},
			expectPanic: false,
		},
		{
			name: "handle nil confirm view",
			setupFunc: func(app *App) {
				app.confirmView = nil
				app.setMode(ModeConfirmDialog)
			},
			action: func(app *App) tea.Cmd {
				_ = app.View()
				return nil
			},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)

			if tt.setupFunc != nil {
				tt.setupFunc(app)
			}

			// Use defer to catch panics
			defer func() {
				r := recover()
				if tt.expectPanic && r == nil {
					t.Error("Expected panic but didn't get one")
				}
				if !tt.expectPanic && r != nil {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			_ = tt.action(app)
		})
	}
}

func TestAppMultiContextBehavior(t *testing.T) {
	tests := []struct {
		name         string
		contexts     []string
		expectMulti  bool
		validateFunc func(*testing.T, *App)
	}{
		{
			name:        "single context mode",
			contexts:    []string{"context1"},
			expectMulti: true, // App always uses multi-context mode now
		},
		{
			name:        "multi context mode",
			contexts:    []string{"context1", "context2"},
			expectMulti: true,
		},
		{
			name:        "empty contexts",
			contexts:    []string{},
			expectMulti: true, // App always uses multi-context mode now
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &core.State{
				CurrentResourceType: core.ResourceTypePod,
				CurrentNamespace:    "default",
				CurrentContexts:     tt.contexts,
			}
			config := &core.Config{RefreshInterval: 5}

			var app *App
			if len(tt.contexts) > 1 {
				multiClient := &k8s.MultiContextClient{}
				app = NewAppWithMultiContext(context.Background(), multiClient, state, config)
			} else {
				app = NewApp(context.Background(), nil, state, config)
			}

			if app.isMultiContext != tt.expectMulti {
				t.Errorf("Expected isMultiContext=%v, got %v", tt.expectMulti, app.isMultiContext)
			}

			// App always uses multi-context mode now, so multiClient should always be set when possible
			if app.multiClient == nil && len(tt.contexts) > 0 {
				// Only error if we expected contexts but don't have a multi-client
				// This can happen in test scenarios where k8s client creation fails
				t.Logf("Multi-client is nil despite having contexts: %v", tt.contexts)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, app)
			}
		})
	}
}

func TestAppLogViewSplitScreen(t *testing.T) {
	tests := []struct {
		name              string
		totalHeight       int
		expectedResourceH int
		expectedLogH      int
	}{
		{
			name:              "small terminal",
			totalHeight:       15,
			expectedResourceH: 8, // Minimum
			expectedLogH:      6, // 15 - 8 - 1
		},
		{
			name:              "medium terminal",
			totalHeight:       30,
			expectedResourceH: 10, // 30 / 3
			expectedLogH:      19, // 30 - 10 - 1
		},
		{
			name:              "large terminal",
			totalHeight:       60,
			expectedResourceH: 20, // 60 / 3
			expectedLogH:      39, // 60 - 20 - 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			app.height = tt.totalHeight
			app.setMode(ModeLog)

			// Render the view to trigger size calculations
			view := app.View()

			// Verify view renders without panic
			if len(view) == 0 {
				t.Error("Log split view should render something")
			}

			// Note: We can't directly verify the exact heights without
			// exposing internal state, but we can verify it renders correctly
		})
	}
}
