package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
)

func createNamespace(name string) v1.Namespace {
	ns := v1.Namespace{}
	ns.Name = name
	return ns
}

func getNamespaceNames(namespaces []v1.Namespace) []string {
	names := make([]string, len(namespaces))
	for i, ns := range namespaces {
		names[i] = ns.Name
	}
	return names
}

func TestNamespaceViewInitialization(t *testing.T) {
	tests := []struct {
		name              string
		namespaces        []v1.Namespace
		currentNamespace  string
		wantSelectedIndex int
		wantFirstItem     string
	}{
		{
			name: "creates view with namespaces",
			namespaces: []v1.Namespace{
				createNamespace("default"),
				createNamespace("kube-system"),
			},
			currentNamespace:  "default",
			wantSelectedIndex: 1, // "all" is 0, "default" is 1
			wantFirstItem:     "all",
		},
		{
			name: "selects current namespace",
			namespaces: []v1.Namespace{
				createNamespace("ns1"),
				createNamespace("ns2"),
				createNamespace("ns3"),
			},
			currentNamespace:  "ns2",
			wantSelectedIndex: 2, // "all" is 0, "ns1" is 1, "ns2" is 2
			wantFirstItem:     "all",
		},
		{
			name:              "handles empty namespace list",
			namespaces:        []v1.Namespace{},
			currentNamespace:  "",
			wantSelectedIndex: 0,
			wantFirstItem:     "all",
		},
		{
			name: "handles 'all' namespace",
			namespaces: []v1.Namespace{
				createNamespace("ns1"),
			},
			currentNamespace:  "all",
			wantSelectedIndex: 0,
			wantFirstItem:     "all",
		},
		{
			name: "handles empty current namespace",
			namespaces: []v1.Namespace{
				createNamespace("ns1"),
			},
			currentNamespace:  "",
			wantSelectedIndex: 0,
			wantFirstItem:     "all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewNamespaceView(tt.namespaces, tt.currentNamespace)

			if view == nil {
				t.Fatal("NewNamespaceView returned nil")
			}

			if view.selectedIndex != tt.wantSelectedIndex {
				t.Errorf("selectedIndex = %d, want %d", view.selectedIndex, tt.wantSelectedIndex)
			}

			if len(view.namespaces) > 0 && view.namespaces[0].Name != tt.wantFirstItem {
				t.Errorf("first item = %q, want %q", view.namespaces[0].Name, tt.wantFirstItem)
			}

			if view.currentNamespace != tt.currentNamespace {
				t.Errorf("currentNamespace = %q, want %q", view.currentNamespace, tt.currentNamespace)
			}
		})
	}
}

func TestNamespaceViewNavigation(t *testing.T) {
	namespaces := []v1.Namespace{
		createNamespace("ns1"),
		createNamespace("ns2"),
		createNamespace("ns3"),
	}

	tests := []struct {
		name      string
		keys      []string
		wantIndex int
	}{
		{
			name:      "move down",
			keys:      []string{"down"},
			wantIndex: 2, // Starting at index 1 (ns1), moving down goes to index 2 (ns2)
		},
		{
			name:      "move down with j",
			keys:      []string{"j"},
			wantIndex: 2, // Starting at index 1 (ns1), moving down goes to index 2 (ns2)
		},
		{
			name:      "move up",
			keys:      []string{"down", "down", "up"},
			wantIndex: 2, // Start at 1, down to 2, down to 3, up to 2
		},
		{
			name:      "move up with k",
			keys:      []string{"j", "j", "k"},
			wantIndex: 2, // Start at 1, down to 2, down to 3, up to 2
		},
		{
			name:      "home key",
			keys:      []string{"down", "down", "home"},
			wantIndex: 0,
		},
		{
			name:      "end key",
			keys:      []string{"end"},
			wantIndex: 3, // "all" + 3 namespaces
		},
		{
			name:      "page down",
			keys:      []string{"pgdown"},
			wantIndex: 3, // Limited by list size
		},
		{
			name:      "page up",
			keys:      []string{"end", "pgup"},
			wantIndex: 0, // Goes back to top
		},
		{
			name:      "stay at top",
			keys:      []string{"up"},
			wantIndex: 0,
		},
		{
			name:      "stay at bottom",
			keys:      []string{"end", "down"},
			wantIndex: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewNamespaceView(namespaces, "ns1")
			t.Logf("Initial selectedIndex: %d", view.selectedIndex)

			for _, key := range tt.keys {
				var msg tea.KeyMsg
				switch key {
				case "up":
					msg = tea.KeyMsg{Type: tea.KeyUp}
				case "down":
					msg = tea.KeyMsg{Type: tea.KeyDown}
				case "home":
					msg = tea.KeyMsg{Type: tea.KeyHome}
				case "end":
					msg = tea.KeyMsg{Type: tea.KeyEnd}
				case "pgup":
					msg = tea.KeyMsg{Type: tea.KeyPgUp}
				case "pgdown":
					msg = tea.KeyMsg{Type: tea.KeyPgDown}
				default:
					msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
				}

				model, _ := view.Update(msg)
				view = model.(*NamespaceView)
			}

			if view.selectedIndex != tt.wantIndex {
				t.Errorf("selectedIndex = %d, want %d", view.selectedIndex, tt.wantIndex)
			}
		})
	}
}

func TestNamespaceViewFiltering(t *testing.T) {
	namespaces := []v1.Namespace{
		createNamespace("production"),
		createNamespace("development"),
		createNamespace("testing"),
		createNamespace("staging"),
	}

	tests := []struct {
		name            string
		filter          string
		wantFiltered    []string
		wantNotFiltered []string
	}{
		{
			name:            "filter 'prod'",
			filter:          "prod",
			wantFiltered:    []string{"production"},
			wantNotFiltered: []string{"development", "testing", "staging"},
		},
		{
			name:            "filter 'ing'",
			filter:          "ing",
			wantFiltered:    []string{"testing", "staging"},
			wantNotFiltered: []string{"production", "development"},
		},
		{
			name:            "case insensitive filter",
			filter:          "PROD",
			wantFiltered:    []string{"production"},
			wantNotFiltered: []string{"development"},
		},
		{
			name:            "empty filter shows all",
			filter:          "",
			wantFiltered:    []string{"all", "production", "development", "testing", "staging"},
			wantNotFiltered: []string{},
		},
		{
			name:            "no matches",
			filter:          "xyz",
			wantFiltered:    []string{},
			wantNotFiltered: []string{"production", "development"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewNamespaceView(namespaces, "")

			// Start filtering
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
			model, _ := view.Update(msg)
			view = model.(*NamespaceView)

			// Type filter
			for _, ch := range tt.filter {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
				model, _ = view.Update(msg)
				view = model.(*NamespaceView)
			}

			// Check filtered items
			t.Logf("Filter: %q, FilteredItems: %v", view.filter, getNamespaceNames(view.filteredItems))
			for _, want := range tt.wantFiltered {
				found := false
				for _, ns := range view.filteredItems {
					if ns.Name == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("namespace %q should be in filtered list", want)
				}
			}

			// Check not filtered items
			for _, notWant := range tt.wantNotFiltered {
				for _, ns := range view.filteredItems {
					if ns.Name == notWant {
						t.Errorf("namespace %q should not be in filtered list", notWant)
					}
				}
			}
		})
	}
}

func TestNamespaceViewFilterInput(t *testing.T) {
	view := NewNamespaceView([]v1.Namespace{createNamespace("test")}, "")

	// Start filtering with /
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	model, _ := view.Update(msg)
	view = model.(*NamespaceView)

	if view.filter != "" {
		t.Error("filter should start empty")
	}

	// Type some characters
	for _, ch := range "test" {
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		model, _ = view.Update(msg)
		view = model.(*NamespaceView)
	}

	if view.filter != "test" {
		t.Errorf("filter = %q, want %q", view.filter, "test")
	}

	// Backspace
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	model, _ = view.Update(msg)
	view = model.(*NamespaceView)

	if view.filter != "tes" {
		t.Errorf("filter after backspace = %q, want %q", view.filter, "tes")
	}

	// Clear with ESC
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = view.Update(msg)
	view = model.(*NamespaceView)

	if view.filter != "" {
		t.Errorf("filter should be empty after ESC, got %q", view.filter)
	}
}

func TestNamespaceViewSelection(t *testing.T) {
	namespaces := []v1.Namespace{
		createNamespace("ns1"),
		createNamespace("ns2"),
		createNamespace("ns3"),
	}

	tests := []struct {
		name         string
		selectedIdx  int
		wantSelected string
	}{
		{
			name:         "select 'all'",
			selectedIdx:  0,
			wantSelected: "",
		},
		{
			name:         "select first namespace",
			selectedIdx:  1,
			wantSelected: "ns1",
		},
		{
			name:         "select last namespace",
			selectedIdx:  3,
			wantSelected: "ns3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewNamespaceView(namespaces, "")
			view.selectedIndex = tt.selectedIdx

			selected := view.GetSelectedNamespace()
			if selected != tt.wantSelected {
				t.Errorf("GetSelectedNamespace() = %q, want %q", selected, tt.wantSelected)
			}
		})
	}
}

func TestNamespaceViewRendering(t *testing.T) {
	namespaces := []v1.Namespace{
		createNamespace("default"),
		createNamespace("kube-system"),
	}

	tests := []struct {
		name             string
		currentNamespace string
		filter           string
		wantContains     []string
	}{
		{
			name:             "renders title",
			currentNamespace: "default",
			wantContains:     []string{"Select Namespace"},
		},
		{
			name:             "shows all namespaces option",
			currentNamespace: "default",
			wantContains:     []string{"All Namespaces"},
		},
		{
			name:             "shows current namespace marker",
			currentNamespace: "default",
			wantContains:     []string{"•", "default"},
		},
		{
			name:             "shows filter",
			currentNamespace: "default",
			filter:           "kube",
			wantContains:     []string{"Filter:", "kube"},
		},
		{
			name:             "shows help text",
			currentNamespace: "default",
			wantContains:     []string{"Navigate", "Filter", "Select", "Cancel"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewNamespaceView(namespaces, tt.currentNamespace)
			view.filter = tt.filter
			view.SetSize(80, 24)

			if tt.filter != "" {
				view.applyFilter()
			}

			output := view.View()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output does not contain %q", want)
				}
			}
		})
	}
}

func TestNamespaceViewExitKeys(t *testing.T) {
	view := NewNamespaceView([]v1.Namespace{createNamespace("test")}, "")

	exitKeys := []string{"q", "n", "enter"}

	for _, key := range exitKeys {
		t.Run("exit with "+key, func(t *testing.T) {
			var msg tea.KeyMsg
			if key == "enter" {
				msg = tea.KeyMsg{Type: tea.KeyEnter}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
			}

			// Update should return the view unchanged for parent to handle
			model, _ := view.Update(msg)
			if model != view {
				t.Error("view should return itself for exit keys")
			}
		})
	}
}

func TestNamespaceViewSetSize(t *testing.T) {
	view := NewNamespaceView([]v1.Namespace{createNamespace("test")}, "")

	view.SetSize(100, 50)

	if view.width != 100 || view.height != 50 {
		t.Errorf("size = (%d, %d), want (100, 50)", view.width, view.height)
	}
}

func TestNamespaceViewInit(t *testing.T) {
	view := NewNamespaceView([]v1.Namespace{createNamespace("test")}, "")
	cmd := view.Init()

	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

func TestNamespaceViewScrollIndicator(t *testing.T) {
	// Create many namespaces to trigger scrolling
	var namespaces []v1.Namespace
	for i := 0; i < 20; i++ {
		namespaces = append(namespaces, createNamespace(strings.Repeat("ns", i+1)))
	}

	view := NewNamespaceView(namespaces, "")
	view.SetSize(80, 24)

	output := view.View()

	// Should show scroll indicator
	if !strings.Contains(output, "of") {
		t.Error("should show scroll indicator with many namespaces")
	}
}

func TestNamespaceViewEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *NamespaceView
		action   func(*NamespaceView)
		validate func(*testing.T, *NamespaceView)
	}{
		{
			name: "handles printable characters in filter",
			setup: func() *NamespaceView {
				return NewNamespaceView([]v1.Namespace{createNamespace("test")}, "")
			},
			action: func(v *NamespaceView) {
				// Add various printable characters
				chars := "abc123-_."
				for _, ch := range chars {
					msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
					model, _ := v.Update(msg)
					*v = *model.(*NamespaceView)
				}
			},
			validate: func(t *testing.T, v *NamespaceView) {
				if v.filter != "abc123-_." {
					t.Errorf("filter = %q, want %q", v.filter, "abc123-_.")
				}
			},
		},
		{
			name: "ignores non-printable characters",
			setup: func() *NamespaceView {
				return NewNamespaceView([]v1.Namespace{createNamespace("test")}, "")
			},
			action: func(v *NamespaceView) {
				// Try to add non-printable character
				msg := tea.KeyMsg{Type: tea.KeyCtrlA}
				model, _ := v.Update(msg)
				*v = *model.(*NamespaceView)
			},
			validate: func(t *testing.T, v *NamespaceView) {
				if v.filter != "" {
					t.Error("should ignore non-printable characters")
				}
			},
		},
		{
			name: "resets selection on filter change",
			setup: func() *NamespaceView {
				return NewNamespaceView([]v1.Namespace{
					createNamespace("ns1"),
					createNamespace("ns2"),
				}, "")
			},
			action: func(v *NamespaceView) {
				v.selectedIndex = 2 // Select ns2
				v.filter = "xyz"    // Filter that matches nothing
				v.applyFilter()
			},
			validate: func(t *testing.T, v *NamespaceView) {
				if v.selectedIndex != 0 {
					t.Error("should reset selection when filter results are empty")
				}
			},
		},
		{
			name: "handles unknown messages",
			setup: func() *NamespaceView {
				return NewNamespaceView([]v1.Namespace{createNamespace("test")}, "")
			},
			action: func(v *NamespaceView) {
				// Send a non-key message
				msg := tea.WindowSizeMsg{Width: 80, Height: 24}
				model, _ := v.Update(msg)
				*v = *model.(*NamespaceView)
			},
			validate: func(t *testing.T, v *NamespaceView) {
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
