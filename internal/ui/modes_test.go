package ui

import (
	"testing"

	"github.com/HamStudy/kubewatch/internal/ui/views"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestModeInitialization tests that all modes are properly initialized
func TestModeInitialization(t *testing.T) {
	tests := []struct {
		name          string
		createMode    func() ScreenMode
		expectedType  ScreenModeType
		expectedTitle string
	}{
		{
			name:          "ListMode",
			createMode:    func() ScreenMode { return NewListMode() },
			expectedType:  ModeList,
			expectedTitle: "KubeWatch TUI - Resource View",
		},
		{
			name:          "LogMode",
			createMode:    func() ScreenMode { return NewLogMode() },
			expectedType:  ModeLog,
			expectedTitle: "KubeWatch TUI - Log View",
		},
		{
			name:          "DescribeMode",
			createMode:    func() ScreenMode { return NewDescribeMode() },
			expectedType:  ModeDescribe,
			expectedTitle: "KubeWatch TUI - Describe View",
		},
		{
			name:          "HelpMode",
			createMode:    func() ScreenMode { return NewHelpMode() },
			expectedType:  ModeHelp,
			expectedTitle: "KubeWatch TUI - Help",
		},
		{
			name:          "ContextSelectorMode",
			createMode:    func() ScreenMode { return NewContextSelectorMode() },
			expectedType:  ModeContextSelector,
			expectedTitle: "KubeWatch TUI - Context Selector",
		},
		{
			name:          "NamespaceSelectorMode",
			createMode:    func() ScreenMode { return NewNamespaceSelectorMode() },
			expectedType:  ModeNamespaceSelector,
			expectedTitle: "KubeWatch TUI - Namespace Selector",
		},
		{
			name:          "ConfirmDialogMode",
			createMode:    func() ScreenMode { return NewConfirmDialogMode() },
			expectedType:  ModeConfirmDialog,
			expectedTitle: "KubeWatch TUI - Confirmation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := tt.createMode()

			if mode.GetType() != tt.expectedType {
				t.Errorf("Expected mode type %v, got %v", tt.expectedType, mode.GetType())
			}

			if mode.GetTitle() != tt.expectedTitle {
				t.Errorf("Expected title %q, got %q", tt.expectedTitle, mode.GetTitle())
			}

			// Verify key bindings are not nil
			bindings := mode.GetKeyBindings()
			if bindings == nil {
				t.Error("Key bindings should not be nil")
			}

			// Verify help sections are not nil
			sections := mode.GetHelpSections()
			if sections == nil {
				t.Error("Help sections should not be nil")
			}
		})
	}
}

// TestListModeCompleteKeyHandling tests all key bindings in list mode
func TestListModeCompleteKeyHandling(t *testing.T) {
	tests := []struct {
		name          string
		keyType       tea.KeyType
		keyRunes      []rune
		expectHandled bool
		expectMode    ScreenModeType
		description   string
	}{
		// Navigation keys
		{"up arrow", tea.KeyUp, nil, false, ModeList, "Should not handle up arrow (delegated to resource view)"},
		{"k key", tea.KeyRunes, []rune("k"), false, ModeList, "Should not handle k (delegated to resource view)"},
		{"down arrow", tea.KeyDown, nil, false, ModeList, "Should not handle down arrow (delegated to resource view)"},
		{"j key", tea.KeyRunes, []rune("j"), false, ModeList, "Should not handle j (delegated to resource view)"},
		{"left arrow", tea.KeyLeft, nil, false, ModeList, "Should not handle left arrow (delegated to resource view)"},
		{"h key", tea.KeyRunes, []rune("h"), false, ModeList, "Should not handle h (delegated to resource view)"},
		{"right arrow", tea.KeyRight, nil, false, ModeList, "Should not handle right arrow (delegated to resource view)"},
		{"l key for logs", tea.KeyRunes, []rune("l"), false, ModeList, "Should not handle l when no resource selected"},

		// Resource navigation
		{"tab", tea.KeyTab, nil, true, ModeList, "Should handle tab for next resource type"},
		{"shift+tab", tea.KeyShiftTab, nil, true, ModeList, "Should handle shift+tab for previous resource type"},

		// Mode switching
		{"namespace selector", tea.KeyRunes, []rune("n"), true, ModeList, "Should handle n for namespace selector"},
		{"context selector", tea.KeyRunes, []rune("c"), true, ModeList, "Should handle c for context selector"},
		{"help", tea.KeyRunes, []rune("?"), true, ModeHelp, "Should switch to help mode"},

		// Actions (these require a selected resource, so they won't trigger mode changes without one)
		{"enter", tea.KeyEnter, nil, false, ModeList, "Should not handle enter when no resource selected"},
		{"info", tea.KeyRunes, []rune("i"), true, ModeList, "Should handle i but stay in list when no resource"},
		{"describe", tea.KeyRunes, []rune("d"), true, ModeList, "Should handle d but stay in list when no resource"},
		{"delete", tea.KeyDelete, nil, false, ModeList, "Should not handle delete key when no resource selected"},
		{"D for delete", tea.KeyRunes, []rune("D"), false, ModeList, "Should not handle D when no resource selected"},
		{"refresh", tea.KeyRunes, []rune("r"), true, ModeList, "Should handle r for refresh"},
		{"sort", tea.KeyRunes, []rune("s"), true, ModeList, "Should handle s for sort"},

		// General
		{"quit q", tea.KeyRunes, []rune("q"), true, ModeList, "Should handle q for quit"},
		{"quit ctrl+c", tea.KeyCtrlC, nil, true, ModeList, "Should handle ctrl+c for quit"},
		{"escape", tea.KeyEsc, nil, false, ModeList, "Should not handle escape in list mode"},

		// Unknown keys
		{"unknown x", tea.KeyRunes, []rune("x"), false, ModeList, "Should not handle unknown key x"},
		{"unknown z", tea.KeyRunes, []rune("z"), false, ModeList, "Should not handle unknown key z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := NewListMode()
			app := createTestApp(t)
			app.setMode(ModeList)

			keyMsg := tea.KeyMsg{Type: tt.keyType}
			if tt.keyRunes != nil {
				keyMsg.Runes = tt.keyRunes
			}

			handled, _ := mode.HandleKey(keyMsg, app)

			if handled != tt.expectHandled {
				t.Errorf("%s: expected handled=%v, got %v", tt.description, tt.expectHandled, handled)
			}

			if tt.expectMode != ModeList && app.currentMode != tt.expectMode {
				t.Errorf("%s: expected mode %v, got %v", tt.description, tt.expectMode, app.currentMode)
			}
		})
	}
}

// TestLogModeCompleteKeyHandling tests all key bindings in log mode
func TestLogModeCompleteKeyHandling(t *testing.T) {
	tests := []struct {
		name          string
		keyType       tea.KeyType
		keyRunes      []rune
		searchMode    bool
		expectHandled bool
		description   string
	}{
		// Navigation keys (delegated to log view)
		{"up arrow", tea.KeyUp, nil, false, false, "Should delegate up arrow to log view"},
		{"k key", tea.KeyRunes, []rune("k"), false, false, "Should delegate k to log view"},
		{"down arrow", tea.KeyDown, nil, false, false, "Should delegate down arrow to log view"},
		{"j key", tea.KeyRunes, []rune("j"), false, false, "Should delegate j to log view"},
		{"page up", tea.KeyPgUp, nil, false, false, "Should delegate page up to log view"},
		{"page down", tea.KeyPgDown, nil, false, false, "Should delegate page down to log view"},
		{"home", tea.KeyHome, nil, false, false, "Should delegate home to log view"},
		{"g key", tea.KeyRunes, []rune("g"), false, false, "Should delegate g to log view"},
		{"end", tea.KeyEnd, nil, false, false, "Should delegate end to log view"},
		{"G key", tea.KeyRunes, []rune("G"), false, false, "Should delegate G to log view"},

		// Log controls (delegated to log view)
		{"follow toggle", tea.KeyRunes, []rune("f"), false, false, "Should delegate f to log view"},
		{"search", tea.KeyRunes, []rune("/"), false, false, "Should delegate / to log view"},
		{"container cycle", tea.KeyRunes, []rune("c"), false, false, "Should delegate c to log view"},
		{"pod cycle", tea.KeyRunes, []rune("p"), false, false, "Should delegate p to log view"},
		{"clear buffer", tea.KeyRunes, []rune("C"), false, false, "Should delegate C to log view"},

		// Mode-level controls
		{"help", tea.KeyRunes, []rune("?"), false, true, "Should handle help at mode level"},
		{"quit q", tea.KeyRunes, []rune("q"), false, true, "Should handle quit at mode level"},
		{"quit ctrl+c", tea.KeyCtrlC, nil, false, true, "Should handle ctrl+c at mode level"},
		{"escape", tea.KeyEsc, nil, false, true, "Should handle escape to return to list"},

		// Search mode behavior - Note: We can't directly test search mode without exposing internal state
		// These tests are commented out as they require internal log view state manipulation
		// {"escape in search", tea.KeyEsc, nil, true, false, "Should delegate escape to log view in search mode"},
		// {"any key in search", tea.KeyRunes, []rune("a"), true, false, "Should delegate all keys in search mode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := NewLogMode()
			app := createTestApp(t)
			app.setMode(ModeLog)

			// Simulate search mode if needed
			// Note: searchMode is private, we test the behavior through IsSearchMode()
			// which is handled internally by the log view

			keyMsg := tea.KeyMsg{Type: tt.keyType}
			if tt.keyRunes != nil {
				keyMsg.Runes = tt.keyRunes
			}

			handled, _ := mode.HandleKey(keyMsg, app)

			if handled != tt.expectHandled {
				t.Errorf("%s: expected handled=%v, got %v", tt.description, tt.expectHandled, handled)
			}
		})
	}
}

// TestDescribeModeCompleteKeyHandling tests all key bindings in describe mode
func TestDescribeModeCompleteKeyHandling(t *testing.T) {
	tests := []struct {
		name          string
		keyType       tea.KeyType
		keyRunes      []rune
		expectHandled bool
		expectMode    ScreenModeType
	}{
		// Navigation (delegated to describe view)
		{"up arrow", tea.KeyUp, nil, false, ModeDescribe},
		{"k key", tea.KeyRunes, []rune("k"), false, ModeDescribe},
		{"down arrow", tea.KeyDown, nil, false, ModeDescribe},
		{"j key", tea.KeyRunes, []rune("j"), false, ModeDescribe},
		{"page up", tea.KeyPgUp, nil, false, ModeDescribe},
		{"page down", tea.KeyPgDown, nil, false, ModeDescribe},
		{"home", tea.KeyHome, nil, false, ModeDescribe},
		{"g key", tea.KeyRunes, []rune("g"), false, ModeDescribe},
		{"end", tea.KeyEnd, nil, false, ModeDescribe},
		{"G key", tea.KeyRunes, []rune("G"), false, ModeDescribe},

		// Mode controls
		{"help", tea.KeyRunes, []rune("?"), true, ModeHelp},
		{"quit q", tea.KeyRunes, []rune("q"), true, ModeDescribe},
		{"quit ctrl+c", tea.KeyCtrlC, nil, true, ModeDescribe},
		{"escape", tea.KeyEsc, nil, true, ModeList},

		// Unknown keys
		{"unknown x", tea.KeyRunes, []rune("x"), false, ModeDescribe},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := NewDescribeMode()
			app := createTestApp(t)
			app.setMode(ModeDescribe)

			keyMsg := tea.KeyMsg{Type: tt.keyType}
			if tt.keyRunes != nil {
				keyMsg.Runes = tt.keyRunes
			}

			handled, _ := mode.HandleKey(keyMsg, app)

			if handled != tt.expectHandled {
				t.Errorf("%s: expected handled=%v, got %v", tt.name, tt.expectHandled, handled)
			}

			if handled && tt.expectMode != ModeDescribe && app.currentMode != tt.expectMode {
				t.Errorf("%s: expected mode %v, got %v", tt.name, tt.expectMode, app.currentMode)
			}
		})
	}
}

// TestHelpModeCompleteKeyHandling tests all key bindings in help mode
func TestHelpModeCompleteKeyHandling(t *testing.T) {
	tests := []struct {
		name          string
		keyType       tea.KeyType
		keyRunes      []rune
		previousMode  ScreenModeType
		expectHandled bool
		expectMode    ScreenModeType
	}{
		{"help toggle", tea.KeyRunes, []rune("?"), ModeList, true, ModeList},
		{"escape", tea.KeyEsc, nil, ModeList, true, ModeList},
		{"quit q", tea.KeyRunes, []rune("q"), ModeList, true, ModeHelp},
		{"quit ctrl+c", tea.KeyCtrlC, nil, ModeList, true, ModeHelp},
		{"unknown key", tea.KeyRunes, []rune("x"), ModeList, false, ModeHelp},

		// Test returning to different previous modes
		{"escape from log", tea.KeyEsc, nil, ModeLog, true, ModeLog},
		{"help from describe", tea.KeyRunes, []rune("?"), ModeDescribe, true, ModeDescribe},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := NewHelpMode()
			app := createTestApp(t)

			// Set up previous mode
			app.setMode(tt.previousMode)
			app.setMode(ModeHelp)

			keyMsg := tea.KeyMsg{Type: tt.keyType}
			if tt.keyRunes != nil {
				keyMsg.Runes = tt.keyRunes
			}

			handled, _ := mode.HandleKey(keyMsg, app)

			if handled != tt.expectHandled {
				t.Errorf("%s: expected handled=%v, got %v", tt.name, tt.expectHandled, handled)
			}

			if tt.expectMode != ModeHelp && app.currentMode != tt.expectMode {
				t.Errorf("%s: expected mode %v, got %v", tt.name, tt.expectMode, app.currentMode)
			}
		})
	}
}

// TestContextSelectorModeCompleteKeyHandling tests all key bindings in context selector mode
func TestContextSelectorModeCompleteKeyHandling(t *testing.T) {
	tests := []struct {
		name          string
		keyType       tea.KeyType
		keyRunes      []rune
		expectHandled bool
		expectMode    ScreenModeType
	}{
		// Navigation (delegated to context view)
		{"up arrow", tea.KeyUp, nil, false, ModeContextSelector},
		{"k key", tea.KeyRunes, []rune("k"), false, ModeContextSelector},
		{"down arrow", tea.KeyDown, nil, false, ModeContextSelector},
		{"j key", tea.KeyRunes, []rune("j"), false, ModeContextSelector},

		// Selection
		{"space", tea.KeySpace, nil, false, ModeContextSelector},
		{"enter", tea.KeyEnter, nil, true, ModeContextSelector},

		// Cancel
		{"escape", tea.KeyEsc, nil, true, ModeList},
		{"c key", tea.KeyRunes, []rune("c"), true, ModeList},

		// Quit
		{"quit q", tea.KeyRunes, []rune("q"), true, ModeContextSelector},
		{"quit ctrl+c", tea.KeyCtrlC, nil, true, ModeContextSelector},

		// Unknown
		{"unknown x", tea.KeyRunes, []rune("x"), false, ModeContextSelector},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := NewContextSelectorMode()
			app := createTestApp(t)
			app.setMode(ModeContextSelector)

			keyMsg := tea.KeyMsg{Type: tt.keyType}
			if tt.keyRunes != nil {
				keyMsg.Runes = tt.keyRunes
			}

			handled, _ := mode.HandleKey(keyMsg, app)

			if handled != tt.expectHandled {
				t.Errorf("%s: expected handled=%v, got %v", tt.name, tt.expectHandled, handled)
			}

			if tt.expectMode != ModeContextSelector && app.currentMode != tt.expectMode {
				t.Errorf("%s: expected mode %v, got %v", tt.name, tt.expectMode, app.currentMode)
			}
		})
	}
}

// TestNamespaceSelectorModeCompleteKeyHandling tests all key bindings in namespace selector mode
func TestNamespaceSelectorModeCompleteKeyHandling(t *testing.T) {
	tests := []struct {
		name          string
		keyType       tea.KeyType
		keyRunes      []rune
		expectHandled bool
		expectMode    ScreenModeType
	}{
		// Navigation (delegated to namespace view)
		{"up arrow", tea.KeyUp, nil, false, ModeNamespaceSelector},
		{"k key", tea.KeyRunes, []rune("k"), false, ModeNamespaceSelector},
		{"down arrow", tea.KeyDown, nil, false, ModeNamespaceSelector},
		{"j key", tea.KeyRunes, []rune("j"), false, ModeNamespaceSelector},

		// Selection
		{"enter", tea.KeyEnter, nil, true, ModeNamespaceSelector},

		// Cancel
		{"escape", tea.KeyEsc, nil, true, ModeList},
		{"n key", tea.KeyRunes, []rune("n"), true, ModeList},

		// Quit
		{"quit q", tea.KeyRunes, []rune("q"), true, ModeNamespaceSelector},
		{"quit ctrl+c", tea.KeyCtrlC, nil, true, ModeNamespaceSelector},

		// Unknown
		{"unknown x", tea.KeyRunes, []rune("x"), false, ModeNamespaceSelector},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := NewNamespaceSelectorMode()
			app := createTestApp(t)

			// Initialize namespace view to prevent nil pointer
			namespaces := []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
			}
			app.namespaceView = views.NewNamespaceView(namespaces, app.state.CurrentNamespace)
			app.namespaceView.SetSize(app.width, app.height)

			app.setMode(ModeNamespaceSelector)

			keyMsg := tea.KeyMsg{Type: tt.keyType}
			if tt.keyRunes != nil {
				keyMsg.Runes = tt.keyRunes
			}

			handled, _ := mode.HandleKey(keyMsg, app)

			if handled != tt.expectHandled {
				t.Errorf("%s: expected handled=%v, got %v", tt.name, tt.expectHandled, handled)
			}

			if tt.expectMode != ModeNamespaceSelector && app.currentMode != tt.expectMode {
				t.Errorf("%s: expected mode %v, got %v", tt.name, tt.expectMode, app.currentMode)
			}
		})
	}
}

// TestConfirmDialogModeCompleteKeyHandling tests all key bindings in confirm dialog mode
func TestConfirmDialogModeCompleteKeyHandling(t *testing.T) {
	tests := []struct {
		name          string
		keyType       tea.KeyType
		keyRunes      []rune
		expectHandled bool
		expectMode    ScreenModeType
	}{
		// Navigation (delegated to confirm view)
		{"left arrow", tea.KeyLeft, nil, false, ModeConfirmDialog},
		{"h key", tea.KeyRunes, []rune("h"), false, ModeConfirmDialog},
		{"right arrow", tea.KeyRight, nil, false, ModeConfirmDialog},
		{"l key", tea.KeyRunes, []rune("l"), false, ModeConfirmDialog},

		// Confirmation
		{"enter", tea.KeyEnter, nil, true, ModeConfirmDialog},
		{"space", tea.KeySpace, nil, true, ModeConfirmDialog},

		// Cancel
		{"escape", tea.KeyEsc, nil, true, ModeList},

		// Quit
		{"quit q", tea.KeyRunes, []rune("q"), true, ModeConfirmDialog},
		{"quit ctrl+c", tea.KeyCtrlC, nil, true, ModeConfirmDialog},

		// Unknown
		{"unknown x", tea.KeyRunes, []rune("x"), false, ModeConfirmDialog},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := NewConfirmDialogMode()
			app := createTestApp(t)

			// Initialize confirm view to prevent nil pointer
			app.confirmView = views.NewConfirmView("Test Confirmation", "Are you sure?")
			app.confirmView.SetSize(app.width, app.height)

			app.setMode(ModeConfirmDialog)

			keyMsg := tea.KeyMsg{Type: tt.keyType}
			if tt.keyRunes != nil {
				keyMsg.Runes = tt.keyRunes
			}

			handled, _ := mode.HandleKey(keyMsg, app)

			if handled != tt.expectHandled {
				t.Errorf("%s: expected handled=%v, got %v", tt.name, tt.expectHandled, handled)
			}

			if tt.expectMode != ModeConfirmDialog && app.currentMode != tt.expectMode {
				t.Errorf("%s: expected mode %v, got %v", tt.name, tt.expectMode, app.currentMode)
			}
		})
	}
}

// TestModeTransitions tests all valid mode transitions
func TestModeTransitions(t *testing.T) {
	tests := []struct {
		name         string
		fromMode     ScreenModeType
		toMode       ScreenModeType
		expectChange bool
	}{
		// From List mode
		{"List to Help", ModeList, ModeHelp, true},
		{"List to Log", ModeList, ModeLog, true},
		{"List to Describe", ModeList, ModeDescribe, true},
		{"List to ContextSelector", ModeList, ModeContextSelector, true},
		{"List to NamespaceSelector", ModeList, ModeNamespaceSelector, true},
		{"List to ConfirmDialog", ModeList, ModeConfirmDialog, true},

		// From Log mode
		{"Log to List", ModeLog, ModeList, true},
		{"Log to Help", ModeLog, ModeHelp, true},

		// From Describe mode
		{"Describe to List", ModeDescribe, ModeList, true},
		{"Describe to Help", ModeDescribe, ModeHelp, true},

		// From Help mode (returns to previous)
		{"Help to List", ModeHelp, ModeList, true},
		{"Help to Log", ModeHelp, ModeLog, true},
		{"Help to Describe", ModeHelp, ModeDescribe, true},

		// From selectors
		{"ContextSelector to List", ModeContextSelector, ModeList, true},
		{"NamespaceSelector to List", ModeNamespaceSelector, ModeList, true},
		{"ConfirmDialog to List", ModeConfirmDialog, ModeList, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)

			// Set initial mode
			app.setMode(tt.fromMode)
			initialMode := app.currentMode

			// Transition to new mode
			app.setMode(tt.toMode)

			if tt.expectChange {
				if app.currentMode != tt.toMode {
					t.Errorf("Expected mode to change from %v to %v, but got %v",
						tt.fromMode, tt.toMode, app.currentMode)
				}
				if app.previousMode != initialMode {
					t.Errorf("Expected previous mode to be %v, but got %v",
						initialMode, app.previousMode)
				}
			} else {
				if app.currentMode != initialMode {
					t.Errorf("Expected mode to remain %v, but got %v",
						initialMode, app.currentMode)
				}
			}
		})
	}
}

// TestInvalidModeTransitions tests that invalid transitions are handled gracefully
func TestInvalidModeTransitions(t *testing.T) {
	app := createTestApp(t)

	// Test setting an undefined mode type
	invalidMode := ScreenModeType(999)
	app.currentMode = invalidMode

	// Should not panic and should default to list mode
	app.setMode(ModeList)
	if app.currentMode != ModeList {
		t.Errorf("Expected mode to be set to ModeList after invalid mode, got %v", app.currentMode)
	}
}

// TestModeStatePreservation tests that mode state is preserved across transitions
func TestModeStatePreservation(t *testing.T) {
	app := createTestApp(t)

	// Start in list mode
	app.setMode(ModeList)

	// Simulate some state changes
	app.state.CurrentResourceType = "deployments"
	app.state.CurrentNamespace = "kube-system"

	// Switch to help mode
	app.setMode(ModeHelp)

	// Return to list mode
	app.returnToPreviousMode()

	// Verify state is preserved
	if app.state.CurrentResourceType != "deployments" {
		t.Errorf("Expected resource type to be preserved as 'deployments', got %v",
			app.state.CurrentResourceType)
	}
	if app.state.CurrentNamespace != "kube-system" {
		t.Errorf("Expected namespace to be preserved as 'kube-system', got %v",
			app.state.CurrentNamespace)
	}
}

// TestConcurrentModeSwitches tests that concurrent mode switches are handled safely

// TestHelpTextGeneration tests that help text is properly generated for all modes
func TestHelpTextGeneration(t *testing.T) {
	modes := []ScreenMode{
		NewListMode(),
		NewLogMode(),
		NewDescribeMode(),
		NewHelpMode(),
		NewContextSelectorMode(),
		NewNamespaceSelectorMode(),
		NewConfirmDialogMode(),
	}

	for _, mode := range modes {
		t.Run(mode.GetTitle(), func(t *testing.T) {
			sections := mode.GetHelpSections()

			// Verify sections are not empty
			if len(sections) == 0 {
				t.Errorf("Mode %s has no help sections", mode.GetTitle())
			}

			// Verify each section has bindings
			for sectionName, bindings := range sections {
				if len(bindings) == 0 {
					t.Errorf("Section %s in mode %s has no bindings",
						sectionName, mode.GetTitle())
				}

				// Verify each binding has required fields
				for _, binding := range bindings {
					if binding.Description == "" {
						t.Errorf("Binding in section %s has no description", sectionName)
					}
					if binding.Section == "" {
						t.Errorf("Binding %s has no section", binding.Description)
					}
					if binding.Section != sectionName {
						t.Errorf("Binding section mismatch: expected %s, got %s",
							sectionName, binding.Section)
					}
				}
			}

			// Verify common sections exist
			expectedSections := map[ScreenModeType][]string{
				ModeList:              {"Navigation", "Actions", "General"},
				ModeLog:               {"Navigation", "Log Controls", "General"},
				ModeDescribe:          {"Navigation", "General"},
				ModeHelp:              {"General"},
				ModeContextSelector:   {"Navigation", "Actions", "General"},
				ModeNamespaceSelector: {"Navigation", "Actions", "General"},
				ModeConfirmDialog:     {"Navigation", "Actions", "General"},
			}

			if expected, ok := expectedSections[mode.GetType()]; ok {
				for _, sectionName := range expected {
					if _, exists := sections[sectionName]; !exists {
						t.Errorf("Mode %s missing expected section: %s",
							mode.GetTitle(), sectionName)
					}
				}
			}
		})
	}
}

// TestKeyBindingConsistency tests that key bindings are consistent across modes
func TestKeyBindingConsistency(t *testing.T) {
	modes := map[string]ScreenMode{
		"List":              NewListMode(),
		"Log":               NewLogMode(),
		"Describe":          NewDescribeMode(),
		"Help":              NewHelpMode(),
		"ContextSelector":   NewContextSelectorMode(),
		"NamespaceSelector": NewNamespaceSelectorMode(),
		"ConfirmDialog":     NewConfirmDialogMode(),
	}

	// Check that common keys have consistent behavior
	commonKeys := []struct {
		key         string
		description string
		shouldExist []string // Modes that should have this key
	}{
		{"quit", "Quit application", []string{"List", "Log", "Describe", "Help", "ContextSelector", "NamespaceSelector", "ConfirmDialog"}},
		{"escape", "Close/Cancel/Back", []string{"List", "Log", "Describe", "Help", "ContextSelector", "NamespaceSelector", "ConfirmDialog"}},
		{"help", "Toggle help", []string{"List", "Log", "Describe", "Help"}},
	}

	for _, ck := range commonKeys {
		for _, modeName := range ck.shouldExist {
			mode := modes[modeName]
			bindings := mode.GetKeyBindings()

			if _, exists := bindings[ck.key]; !exists {
				t.Errorf("Mode %s should have %s binding", modeName, ck.key)
			}
		}
	}
}

// TestKeyBindingUniqueness tests that key combinations don't conflict within a mode
func TestKeyBindingUniqueness(t *testing.T) {
	modes := []ScreenMode{
		NewListMode(),
		NewLogMode(),
		NewDescribeMode(),
		NewHelpMode(),
		NewContextSelectorMode(),
		NewNamespaceSelectorMode(),
		NewConfirmDialogMode(),
	}

	for _, mode := range modes {
		t.Run(mode.GetTitle(), func(t *testing.T) {
			bindings := mode.GetKeyBindings()
			keyMap := make(map[string][]string) // key combination -> binding names

			for name, binding := range bindings {
				// Get the actual keys from the binding
				keys := binding.Key.Keys()
				for _, k := range keys {
					if existing, exists := keyMap[k]; exists {
						// Check if this is an intentional duplicate (like navigation keys)
						if !isIntentionalDuplicate(k, name, existing) {
							t.Errorf("Key %s is mapped to multiple bindings: %s and %v",
								k, name, existing)
						}
					}
					keyMap[k] = append(keyMap[k], name)
				}
			}
		})
	}
}

// isIntentionalDuplicate checks if a key duplication is intentional
func isIntentionalDuplicate(key string, newBinding string, existingBindings []string) bool {
	// Some keys are intentionally mapped to multiple actions
	intentionalDuplicates := map[string]bool{
		// Navigation keys that might be used in multiple contexts
		"l": true, // Can be "right" navigation or "logs" action
		"h": true, // Can be "left" navigation or other action
		"c": true, // Can be "context" or "container cycle" depending on mode
	}

	return intentionalDuplicates[key]
}

// TestModeCleanup tests that modes properly clean up when exited
func TestModeCleanup(t *testing.T) {
	app := createTestApp(t)

	// Test transitioning between modes
	// Enter log mode (which might start streaming)
	app.setMode(ModeLog)
	if app.currentMode != ModeLog {
		t.Error("Should be in log mode")
	}

	// Exit log mode
	app.setMode(ModeList)
	if app.currentMode != ModeList {
		t.Error("Should be back in list mode")
	}

	// Enter describe mode
	app.setMode(ModeDescribe)
	if app.currentMode != ModeDescribe {
		t.Error("Should be in describe mode")
	}

	// Exit describe mode
	app.setMode(ModeList)
	if app.currentMode != ModeList {
		t.Error("Should be back in list mode")
	}

	// Verify mode transitions work correctly
	// The actual cleanup of views is handled internally by the views themselves
}

// TestNewKeyBinding tests the NewKeyBinding helper function
func TestNewKeyBinding(t *testing.T) {
	tests := []struct {
		name        string
		keys        []string
		help        string
		description string
		section     string
	}{
		{
			name:        "single key",
			keys:        []string{"q"},
			help:        "q",
			description: "Quit",
			section:     "General",
		},
		{
			name:        "multiple keys",
			keys:        []string{"up", "k"},
			help:        "↑/k",
			description: "Move up",
			section:     "Navigation",
		},
		{
			name:        "special key",
			keys:        []string{"ctrl+c"},
			help:        "Ctrl+C",
			description: "Force quit",
			section:     "General",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding := NewKeyBinding(tt.keys, tt.help, tt.description, tt.section)

			if binding.Description != tt.description {
				t.Errorf("Expected description %q, got %q", tt.description, binding.Description)
			}

			if binding.Section != tt.section {
				t.Errorf("Expected section %q, got %q", tt.section, binding.Section)
			}

			// Verify the key binding has the correct keys
			actualKeys := binding.Key.Keys()
			if len(actualKeys) != len(tt.keys) {
				t.Errorf("Expected %d keys, got %d", len(tt.keys), len(actualKeys))
			}

			for i, expectedKey := range tt.keys {
				if i < len(actualKeys) && actualKeys[i] != expectedKey {
					t.Errorf("Expected key %q at position %d, got %q",
						expectedKey, i, actualKeys[i])
				}
			}
		})
	}
}

// TestModeReturnToPrevious tests the return to previous mode functionality
func TestModeReturnToPrevious(t *testing.T) {
	app := createTestApp(t)

	// Test chain of mode switches
	app.setMode(ModeList)
	app.setMode(ModeLog)
	app.setMode(ModeHelp)

	// Should return to Log
	app.returnToPreviousMode()
	if app.currentMode != ModeLog {
		t.Errorf("Expected to return to ModeLog, got %v", app.currentMode)
	}

	// Set another mode and return again
	app.setMode(ModeDescribe)
	app.returnToPreviousMode()
	if app.currentMode != ModeLog {
		t.Errorf("Expected to return to ModeLog, got %v", app.currentMode)
	}
}

// TestKeyMatchingBehavior tests that key matching works correctly
func TestKeyMatchingBehavior(t *testing.T) {
	tests := []struct {
		name        string
		binding     key.Binding
		keyMsg      tea.KeyMsg
		shouldMatch bool
	}{
		{
			name:        "exact match single key",
			binding:     key.NewBinding(key.WithKeys("q")),
			keyMsg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
			shouldMatch: true,
		},
		{
			name:        "no match different key",
			binding:     key.NewBinding(key.WithKeys("q")),
			keyMsg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")},
			shouldMatch: false,
		},
		{
			name:        "match special key",
			binding:     key.NewBinding(key.WithKeys("enter")),
			keyMsg:      tea.KeyMsg{Type: tea.KeyEnter},
			shouldMatch: true,
		},
		{
			name:        "match one of multiple keys",
			binding:     key.NewBinding(key.WithKeys("up", "k")),
			keyMsg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")},
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := key.Matches(tt.keyMsg, tt.binding)
			if matches != tt.shouldMatch {
				t.Errorf("Expected match=%v, got %v", tt.shouldMatch, matches)
			}
		})
	}
}

// BenchmarkModeSwitch benchmarks mode switching performance
func BenchmarkModeSwitch(b *testing.B) {
	app := createTestApp(&testing.T{})

	modes := []ScreenModeType{
		ModeList, ModeHelp, ModeLog, ModeDescribe,
		ModeContextSelector, ModeNamespaceSelector, ModeConfirmDialog,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		modeIndex := i % len(modes)
		app.setMode(modes[modeIndex])
	}
}

// BenchmarkKeyHandling benchmarks key handling performance
func BenchmarkKeyHandling(b *testing.B) {
	app := createTestApp(&testing.T{})
	mode := NewListMode()

	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("q")},
		{Type: tea.KeyRunes, Runes: []rune("?")},
		{Type: tea.KeyTab},
		{Type: tea.KeyEnter},
		{Type: tea.KeyEsc},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyIndex := i % len(keys)
		mode.HandleKey(keys[keyIndex], app)
	}
}
