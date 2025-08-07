package ui

import (
	"context"
	"testing"

	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/ui/views"
	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ModeTestSetup provides utilities for setting up mode tests
type ModeTestSetup struct {
	App  *App
	Mode ScreenMode
}

// NewModeTestSetup creates a test setup for a specific mode
func NewModeTestSetup(t *testing.T, modeType ScreenModeType) *ModeTestSetup {
	app := createTestAppForModes(t)

	// Set up mode-specific dependencies
	switch modeType {
	case ModeList:
		// List mode needs resource view to be properly initialized
		if app.resourceView == nil {
			state := &core.State{
				CurrentResourceType: core.ResourceTypePod,
				CurrentNamespace:    "default",
			}
			app.resourceView = views.NewResourceView(state, nil)
			app.resourceView.SetSize(80, 24)
		}

	case ModeNamespaceSelector:
		// Namespace selector needs namespace view
		namespaces := []v1.Namespace{
			{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}},
		}
		app.namespaceView = views.NewNamespaceView(namespaces, app.state.CurrentNamespace)
		app.namespaceView.SetSize(80, 24)

	case ModeContextSelector:
		// Context selector needs context view
		contexts := []string{"test-context", "prod-context", "dev-context"}
		selected := []string{"test-context"}
		app.contextView = views.NewContextView(contexts, selected)
		app.contextView.SetSize(80, 24)

	case ModeConfirmDialog:
		// Confirm dialog needs confirm view
		app.confirmView = views.NewConfirmView("Delete Resource", "Are you sure?")
		app.confirmView.SetSize(80, 24)

	case ModeDescribe:
		// Describe mode needs describe view
		app.describeView = views.NewDescribeView("pod", "test-pod", "default", "test-context")
		app.describeView.SetSize(80, 24)

	case ModeLog:
		// Log mode needs log view
		if app.logView == nil {
			app.logView = views.NewLogView()
			app.logView.SetSize(80, 24)
		}

	case ModeHelp:
		// Help mode needs help view
		if app.helpView == nil {
			app.helpView = views.NewHelpView()
		}
	}

	// Set the mode
	app.setMode(modeType)

	return &ModeTestSetup{
		App:  app,
		Mode: app.modes[modeType],
	}
}

// createTestAppForModes creates a minimal app for mode testing
func createTestAppForModes(t *testing.T) *App {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
	}

	config := &core.Config{
		RefreshInterval: 5,
	}

	app := NewApp(context.Background(), nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true

	// Ensure modes are initialized
	if app.modes == nil {
		app.modes = map[ScreenModeType]ScreenMode{
			ModeList:              NewListMode(),
			ModeLog:               NewLogMode(),
			ModeDescribe:          NewDescribeMode(),
			ModeHelp:              NewHelpMode(),
			ModeContextSelector:   NewContextSelectorMode(),
			ModeNamespaceSelector: NewNamespaceSelectorMode(),
			ModeConfirmDialog:     NewConfirmDialogMode(),
		}
	}

	return app
}

// SimulateSelectedResource mocks having a selected resource in list mode
func SimulateSelectedResource(app *App, resourceName string) {
	// We need to work with the actual ResourceView API
	// Since we can't directly set selected resources, we'll mock the behavior
	// by ensuring GetSelectedResourceName returns the expected value

	// This is a workaround for testing - in real usage, the resource view
	// would have resources and selection state
	if app.resourceView != nil {
		// The resource view needs to have actual resources to select from
		// For testing, we'll ensure the view is in a state where it would
		// return a resource name when asked

		// Since we can't directly manipulate the view's internal state,
		// we need to work with what's available or mock at a different level
	}
}

// TestKeyHandling tests a key press and returns whether it was handled
func TestKeyHandling(t *testing.T, setup *ModeTestSetup, keyType tea.KeyType, runes []rune) (bool, tea.Cmd) {
	keyMsg := tea.KeyMsg{Type: keyType}
	if runes != nil {
		keyMsg.Runes = runes
	}

	return setup.Mode.HandleKey(keyMsg, setup.App)
}

// AssertKeyHandled checks if a key was handled as expected
func AssertKeyHandled(t *testing.T, setup *ModeTestSetup, keyType tea.KeyType, runes []rune, expectHandled bool, description string) {
	t.Helper()

	handled, _ := TestKeyHandling(t, setup, keyType, runes)
	if handled != expectHandled {
		t.Errorf("%s: expected handled=%v, got %v", description, expectHandled, handled)
	}
}

// AssertModeChanged checks if the mode changed as expected
func AssertModeChanged(t *testing.T, app *App, expectedMode ScreenModeType, description string) {
	t.Helper()

	if app.currentMode != expectedMode {
		t.Errorf("%s: expected mode %v, got %v", description, expectedMode, app.currentMode)
	}
}

// MockLogViewSearchMode is a helper to simulate search mode in log view
// Since we can't directly access private fields, this is a best-effort approach
type MockLogViewSearchMode struct {
	OriginalLogView *views.LogView
	InSearchMode    bool
}

// SetupLogViewForSearchTest prepares log view for search mode testing
func SetupLogViewForSearchTest(app *App) *MockLogViewSearchMode {
	// This is a limitation - we can't truly mock search mode without
	// exposing it or using reflection. The test should be adjusted
	// to work with the public API
	return &MockLogViewSearchMode{
		OriginalLogView: app.logView,
		InSearchMode:    false,
	}
}

// CreateKeyBinding is a helper to create key messages for testing
func CreateKeyBinding(keyType tea.KeyType, runes string) tea.KeyMsg {
	msg := tea.KeyMsg{Type: keyType}
	if runes != "" {
		msg.Runes = []rune(runes)
	}
	return msg
}
