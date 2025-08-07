package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestContextViewInitialization(t *testing.T) {
	tests := []struct {
		name              string
		contexts          []string
		currentContexts   []string
		wantMultiSelect   bool
		wantSelectedCount int
	}{
		{
			name:              "single context mode",
			contexts:          []string{"context1", "context2", "context3"},
			currentContexts:   []string{"context1"},
			wantMultiSelect:   false,
			wantSelectedCount: 1,
		},
		{
			name:              "multi context mode",
			contexts:          []string{"context1", "context2", "context3"},
			currentContexts:   []string{"context1", "context2"},
			wantMultiSelect:   true,
			wantSelectedCount: 2,
		},
		{
			name:              "empty current contexts",
			contexts:          []string{"context1", "context2"},
			currentContexts:   []string{},
			wantMultiSelect:   false,
			wantSelectedCount: 0,
		},
		{
			name:              "all contexts selected",
			contexts:          []string{"context1", "context2"},
			currentContexts:   []string{"context1", "context2"},
			wantMultiSelect:   true,
			wantSelectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewContextView(tt.contexts, tt.currentContexts)

			if view == nil {
				t.Fatal("NewContextView returned nil")
			}

			if view.multiSelect != tt.wantMultiSelect {
				t.Errorf("multiSelect = %v, want %v", view.multiSelect, tt.wantMultiSelect)
			}

			selectedCount := len(view.selectedContexts)
			if selectedCount != tt.wantSelectedCount {
				t.Errorf("selected count = %d, want %d", selectedCount, tt.wantSelectedCount)
			}

			// Verify selected contexts match input
			for _, ctx := range tt.currentContexts {
				if !view.selectedContexts[ctx] {
					t.Errorf("context %q should be selected", ctx)
				}
			}
		})
	}
}

func TestContextViewNavigation(t *testing.T) {
	contexts := []string{"context1", "context2", "context3", "context4"}

	tests := []struct {
		name      string
		keys      []string
		wantIndex int
	}{
		{
			name:      "move down",
			keys:      []string{"down"},
			wantIndex: 1,
		},
		{
			name:      "move down with j",
			keys:      []string{"j"},
			wantIndex: 1,
		},
		{
			name:      "move up",
			keys:      []string{"down", "down", "up"},
			wantIndex: 1,
		},
		{
			name:      "move up with k",
			keys:      []string{"j", "j", "k"},
			wantIndex: 1,
		},
		{
			name:      "stay at top when moving up",
			keys:      []string{"up"},
			wantIndex: 0,
		},
		{
			name:      "stay at bottom when moving down",
			keys:      []string{"down", "down", "down", "down", "down"},
			wantIndex: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewContextView(contexts, []string{})

			for _, key := range tt.keys {
				var msg tea.KeyMsg
				switch key {
				case "up":
					msg = tea.KeyMsg{Type: tea.KeyUp}
				case "down":
					msg = tea.KeyMsg{Type: tea.KeyDown}
				default:
					msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
				}

				model, _ := view.Update(msg)
				view = model.(*ContextView)
			}

			if view.currentIndex != tt.wantIndex {
				t.Errorf("currentIndex = %d, want %d", view.currentIndex, tt.wantIndex)
			}
		})
	}
}

func TestContextViewSelection(t *testing.T) {
	contexts := []string{"context1", "context2", "context3"}

	tests := []struct {
		name         string
		multiSelect  bool
		keys         []string
		wantSelected []string
	}{
		{
			name:         "single select with space",
			multiSelect:  false,
			keys:         []string{" "},
			wantSelected: []string{"context1"},
		},
		{
			name:         "multi select with space",
			multiSelect:  true,
			keys:         []string{" ", "down", " "},
			wantSelected: []string{"context1", "context2"},
		},
		{
			name:         "toggle selection",
			multiSelect:  true,
			keys:         []string{" ", " "},
			wantSelected: []string{},
		},
		{
			name:         "select all with a",
			multiSelect:  true,
			keys:         []string{"a"},
			wantSelected: []string{"context1", "context2", "context3"},
		},
		{
			name:         "deselect all with a",
			multiSelect:  true,
			keys:         []string{"a", "a"},
			wantSelected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewContextView(contexts, []string{})
			view.multiSelect = tt.multiSelect

			for _, key := range tt.keys {
				var msg tea.KeyMsg
				switch key {
				case " ":
					msg = tea.KeyMsg{Type: tea.KeySpace}
				case "down":
					msg = tea.KeyMsg{Type: tea.KeyDown}
				default:
					msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
				}

				model, _ := view.Update(msg)
				view = model.(*ContextView)
				t.Logf("After key %q: selectedContexts = %v, multiSelect = %v", key, view.selectedContexts, view.multiSelect)
			}

			selected := view.GetSelectedContexts()
			if len(selected) != len(tt.wantSelected) {
				t.Errorf("selected count = %d, want %d", len(selected), len(tt.wantSelected))
			}

			// Check each expected selection
			for _, want := range tt.wantSelected {
				found := false
				for _, got := range selected {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("context %q should be selected", want)
				}
			}
		})
	}
}

func TestContextViewModeToggle(t *testing.T) {
	contexts := []string{"context1", "context2", "context3"}
	view := NewContextView(contexts, []string{"context1", "context2"})

	// Should start in multi-select mode
	if !view.multiSelect {
		t.Error("should start in multi-select mode with multiple contexts")
	}

	// Toggle to single mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")}
	model, _ := view.Update(msg)
	view = model.(*ContextView)

	if view.multiSelect {
		t.Error("should switch to single-select mode")
	}

	// Should keep only current selection
	selected := view.GetSelectedContexts()
	if len(selected) > 1 {
		t.Errorf("single mode should have at most 1 selection, got %d", len(selected))
	}

	// Toggle back to multi mode
	model, _ = view.Update(msg)
	view = model.(*ContextView)

	if !view.multiSelect {
		t.Error("should switch back to multi-select mode")
	}
}

func TestContextViewSearch(t *testing.T) {
	contexts := []string{"prod-cluster", "dev-cluster", "test-cluster", "staging"}

	tests := []struct {
		name           string
		searchQuery    string
		wantVisible    []string
		wantNotVisible []string
	}{
		{
			name:           "search for 'dev'",
			searchQuery:    "dev",
			wantVisible:    []string{"dev-cluster"},
			wantNotVisible: []string{"prod-cluster", "test-cluster", "staging"},
		},
		{
			name:           "search for 'cluster'",
			searchQuery:    "cluster",
			wantVisible:    []string{"prod-cluster", "dev-cluster", "test-cluster"},
			wantNotVisible: []string{"staging"},
		},
		{
			name:        "case insensitive search",
			searchQuery: "PROD",
			wantVisible: []string{"prod-cluster"},
		},
		{
			name:        "empty search shows all",
			searchQuery: "",
			wantVisible: contexts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewContextView(contexts, []string{})

			// Enter search mode
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
			model, _ := view.Update(msg)
			view = model.(*ContextView)

			// Type search query
			for _, ch := range tt.searchQuery {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
				model, _ = view.Update(msg)
				view = model.(*ContextView)
			}

			// Get visible contexts
			visible := view.getVisibleContexts()

			// Check expected visible
			for _, want := range tt.wantVisible {
				found := false
				for _, v := range visible {
					if v == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("context %q should be visible", want)
				}
			}

			// Check expected not visible
			for _, notWant := range tt.wantNotVisible {
				for _, v := range visible {
					if v == notWant {
						t.Errorf("context %q should not be visible", v)
					}
				}
			}
		})
	}
}

func TestContextViewSearchMode(t *testing.T) {
	view := NewContextView([]string{"context1", "context2"}, []string{})

	// Enter search mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	model, _ := view.Update(msg)
	view = model.(*ContextView)

	if !view.SearchMode {
		t.Error("should be in search mode after pressing /")
	}

	// Type some text
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
	model, _ = view.Update(msg)
	view = model.(*ContextView)

	if view.searchQuery != "c" {
		t.Errorf("searchQuery = %q, want %q", view.searchQuery, "c")
	}

	// Backspace
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	model, _ = view.Update(msg)
	view = model.(*ContextView)

	if view.searchQuery != "" {
		t.Errorf("searchQuery should be empty after backspace, got %q", view.searchQuery)
	}

	// Exit search with ESC
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = view.Update(msg)
	view = model.(*ContextView)

	if view.SearchMode {
		t.Error("should exit search mode after pressing ESC")
	}

	// Enter search mode again and exit with Enter
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	model, _ = view.Update(msg)
	view = model.(*ContextView)

	msg = tea.KeyMsg{Type: tea.KeyEnter}
	model, _ = view.Update(msg)
	view = model.(*ContextView)

	if view.SearchMode {
		t.Error("should exit search mode after pressing Enter")
	}
}

func TestContextViewRendering(t *testing.T) {
	tests := []struct {
		name            string
		contexts        []string
		selected        []string
		multiSelect     bool
		searchMode      bool
		searchQuery     string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:         "renders title",
			contexts:     []string{"context1"},
			selected:     []string{},
			multiSelect:  false,
			wantContains: []string{"Select Kubernetes Context"},
		},
		{
			name:            "shows multi-select with clear terminology",
			contexts:        []string{"context1"},
			selected:        []string{},
			multiSelect:     true,
			wantContains:    []string{"Multi-Select"},
			wantNotContains: []string{"Multi-Select Mode"}, // Should not use confusing "Mode" terminology
		},
		{
			name:         "shows search query",
			contexts:     []string{"context1"},
			selected:     []string{},
			searchMode:   true,
			searchQuery:  "test",
			wantContains: []string{"Search:", "test"},
		},
		{
			name:         "shows selected contexts with left-aligned checkboxes",
			contexts:     []string{"context1", "context2"},
			selected:     []string{"context1"},
			wantContains: []string{"[✓]"},
		},
		{
			name:            "shows help text with clear selection terminology in single-select",
			contexts:        []string{"context1"},
			selected:        []string{},
			multiSelect:     false,
			wantContains:    []string{"Navigate", "Toggle", "Confirm", "Cancel", "Multi-select"},
			wantNotContains: []string{"Multi mode"}, // Should not use confusing "mode" terminology
		},
		{
			name:            "shows help text with clear selection terminology in multi-select",
			contexts:        []string{"context1"},
			selected:        []string{},
			multiSelect:     true,
			wantContains:    []string{"Navigate", "Toggle", "Confirm", "Cancel", "Single-select"},
			wantNotContains: []string{"Single mode"}, // Should not use confusing "mode" terminology
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewContextView(tt.contexts, tt.selected)
			view.multiSelect = tt.multiSelect
			view.SearchMode = tt.searchMode
			view.searchQuery = tt.searchQuery
			view.SetSize(80, 24)

			output := view.View()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output does not contain %q", want)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if strings.Contains(output, notWant) {
					t.Errorf("output should not contain %q", notWant)
				}
			}
		})
	}
}

func TestContextViewWindowResize(t *testing.T) {
	view := NewContextView([]string{"context1"}, []string{})

	// Send window size message
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	model, _ := view.Update(msg)
	view = model.(*ContextView)

	if view.width != 100 || view.height != 50 {
		t.Errorf("size = (%d, %d), want (100, 50)", view.width, view.height)
	}
}

func TestContextViewSetSize(t *testing.T) {
	view := NewContextView([]string{"context1"}, []string{})

	view.SetSize(120, 40)

	if view.width != 120 || view.height != 40 {
		t.Errorf("size = (%d, %d), want (120, 40)", view.width, view.height)
	}
}

func TestContextViewInit(t *testing.T) {
	view := NewContextView([]string{"context1"}, []string{})
	cmd := view.Init()

	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

func TestContextViewInfoCommand(t *testing.T) {
	view := NewContextView([]string{"context1", "context2"}, []string{})

	// Press 'i' to show context info
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")}
	model, _ := view.Update(msg)
	view = model.(*ContextView)

	// Should now be showing info
	if !view.showingInfo {
		t.Error("should be showing info after pressing 'i'")
	}

	if view.infoContext != "context1" {
		t.Errorf("infoContext = %q, want %q", view.infoContext, "context1")
	}

	// View should render info view
	output := view.View()
	if !strings.Contains(output, "Context Information") {
		t.Error("info view should contain 'Context Information'")
	}

	if !strings.Contains(output, "context1") {
		t.Error("info view should contain the context name")
	}

	// Press 'i' again to toggle off
	model, _ = view.Update(msg)
	view = model.(*ContextView)

	if view.showingInfo {
		t.Error("should not be showing info after pressing 'i' again")
	}

	// Press 'i' to show info, then 'esc' to exit
	model, _ = view.Update(msg)
	view = model.(*ContextView)

	if !view.showingInfo {
		t.Error("should be showing info after pressing 'i'")
	}

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = view.Update(escMsg)
	view = model.(*ContextView)

	if view.showingInfo {
		t.Error("should not be showing info after pressing 'esc'")
	}
}

func TestContextViewInfoFunctionality(t *testing.T) {
	// Test that info functionality is now working
	view := NewContextView([]string{"context1"}, []string{})
	view.SetSize(80, 24)

	// Should show info hint since info is now functional
	output := view.View()
	if !strings.Contains(output, "i: Info") {
		t.Error("output should contain 'i: Info' since info functionality is implemented")
	}

	// Test that pressing 'i' actually shows info
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")}
	model, _ := view.Update(msg)
	view = model.(*ContextView)

	if !view.showingInfo {
		t.Error("should be showing info after pressing 'i'")
	}

	infoOutput := view.View()
	if !strings.Contains(infoOutput, "Context Information") {
		t.Error("info view should contain 'Context Information'")
	}
}

func TestContextViewCheckboxAlignment(t *testing.T) {
	view := NewContextView([]string{"context1", "context2"}, []string{"context1"})
	view.SetSize(80, 24)

	output := view.View()

	// Check that checkboxes are present and properly formatted
	if !strings.Contains(output, "[✓]") {
		t.Error("output should contain selected checkbox [✓]")
	}

	if !strings.Contains(output, "[ ]") {
		t.Error("output should contain unselected checkbox [ ]")
	}

	// The output should not be completely centered (which would make checkboxes look awkward)
	// Instead, content should be left-aligned for better checkbox presentation
	lines := strings.Split(output, "\n")
	var contextLines []string
	for _, line := range lines {
		if strings.Contains(line, "[") && (strings.Contains(line, "✓") || strings.Contains(line, " ]")) {
			contextLines = append(contextLines, line)
		}
	}

	if len(contextLines) == 0 {
		t.Error("should find context lines with checkboxes")
	}

	// Verify that checkbox lines start consistently (left-aligned within content area)
	for i, line := range contextLines {
		if len(line) == 0 {
			continue
		}
		// Lines should have consistent indentation for left-aligned appearance
		if i > 0 && len(contextLines[i-1]) > 0 {
			// Both lines should start with similar whitespace pattern for alignment
			prevTrimmed := strings.TrimLeft(contextLines[i-1], " ")
			currTrimmed := strings.TrimLeft(line, " ")
			if len(prevTrimmed) > 0 && len(currTrimmed) > 0 {
				// Both should start with checkbox pattern
				if !strings.HasPrefix(prevTrimmed, "[") || !strings.HasPrefix(currTrimmed, "[") {
					continue // Skip non-checkbox lines
				}
				// Check that indentation is consistent
				prevIndent := len(contextLines[i-1]) - len(prevTrimmed)
				currIndent := len(line) - len(currTrimmed)
				if prevIndent != currIndent {
					t.Errorf("checkbox alignment inconsistent: line %d has %d spaces, line %d has %d spaces",
						i-1, prevIndent, i, currIndent)
				}
			}
		}
	}
}

func TestContextViewEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *ContextView
		action   func(*ContextView)
		validate func(*testing.T, *ContextView)
	}{
		{
			name: "handles empty context list",
			setup: func() *ContextView {
				return NewContextView([]string{}, []string{})
			},
			action: func(v *ContextView) {
				// Try to move down
				msg := tea.KeyMsg{Type: tea.KeyDown}
				model, _ := v.Update(msg)
				*v = *model.(*ContextView)
			},
			validate: func(t *testing.T, v *ContextView) {
				if v.currentIndex != 0 {
					t.Error("index should stay at 0 with empty list")
				}
			},
		},
		{
			name: "handles very long context names",
			setup: func() *ContextView {
				longName := strings.Repeat("very-long-context-name-", 10)
				return NewContextView([]string{longName}, []string{})
			},
			action: func(v *ContextView) {
				v.SetSize(80, 24)
			},
			validate: func(t *testing.T, v *ContextView) {
				output := v.View()
				if output == "" {
					t.Error("should render with long names")
				}
			},
		},
		{
			name: "handles index bounds after filtering",
			setup: func() *ContextView {
				return NewContextView([]string{"context1", "context2", "test"}, []string{})
			},
			action: func(v *ContextView) {
				// Move to last item
				v.currentIndex = 2
				// Set filter that excludes current selection
				v.searchQuery = "context"
				v.filterContexts()
			},
			validate: func(t *testing.T, v *ContextView) {
				visible := v.getVisibleContexts()
				if v.currentIndex >= len(visible) && len(visible) > 0 {
					t.Error("index should be adjusted after filtering")
				}
			},
		},
		{
			name: "handles unknown keys gracefully",
			setup: func() *ContextView {
				return NewContextView([]string{"context1"}, []string{})
			},
			action: func(v *ContextView) {
				msg := tea.KeyMsg{Type: tea.KeyF1}
				model, _ := v.Update(msg)
				*v = *model.(*ContextView)
			},
			validate: func(t *testing.T, v *ContextView) {
				// Should not crash
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.setup()
			tt.action(view)
			tt.validate(t, view)
		})
	}
}

func TestContextViewEnsureValidIndex(t *testing.T) {
	tests := []struct {
		name        string
		contexts    []string
		searchQuery string
		startIndex  int
		wantIndex   int
	}{
		{
			name:        "adjusts negative index",
			contexts:    []string{"c1", "c2"},
			searchQuery: "",
			startIndex:  -1,
			wantIndex:   0,
		},
		{
			name:        "adjusts out of bounds index",
			contexts:    []string{"c1", "c2"},
			searchQuery: "",
			startIndex:  5,
			wantIndex:   1,
		},
		{
			name:        "handles empty filtered list",
			contexts:    []string{"c1", "c2"},
			searchQuery: "xyz",
			startIndex:  0,
			wantIndex:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewContextView(tt.contexts, []string{})
			view.searchQuery = tt.searchQuery
			view.currentIndex = tt.startIndex
			view.ensureValidIndex()

			if view.currentIndex != tt.wantIndex {
				t.Errorf("currentIndex = %d, want %d", view.currentIndex, tt.wantIndex)
			}
		})
	}
}
