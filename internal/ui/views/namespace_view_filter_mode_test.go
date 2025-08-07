package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
)

// TestNamespaceViewFilterMode tests the proper filter mode behavior
func TestNamespaceViewFilterMode(t *testing.T) {
	namespaces := []v1.Namespace{
		createNamespace("production"),
		createNamespace("development"),
		createNamespace("testing"),
	}

	t.Run("navigation keys work in normal mode", func(t *testing.T) {
		view := NewNamespaceView(namespaces, "")
		initialIndex := view.selectedIndex

		// Press 'j' - should move down, not add to filter
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		model, _ := view.Update(msg)
		view = model.(*NamespaceView)

		if view.filter != "" {
			t.Errorf("'j' key should not add to filter in normal mode, got filter: %q", view.filter)
		}

		if view.selectedIndex != initialIndex+1 {
			t.Errorf("'j' key should move selection down, got index %d, want %d", view.selectedIndex, initialIndex+1)
		}
	})

	t.Run("k key works for navigation in normal mode", func(t *testing.T) {
		view := NewNamespaceView(namespaces, "")
		view.selectedIndex = 2 // Start at a position where we can move up

		// Press 'k' - should move up, not add to filter
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
		model, _ := view.Update(msg)
		view = model.(*NamespaceView)

		if view.filter != "" {
			t.Errorf("'k' key should not add to filter in normal mode, got filter: %q", view.filter)
		}

		if view.selectedIndex != 1 {
			t.Errorf("'k' key should move selection up, got index %d, want %d", view.selectedIndex, 1)
		}
	})

	t.Run("slash enters filter mode", func(t *testing.T) {
		view := NewNamespaceView(namespaces, "")

		// Press '/' - should enter filter mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
		model, _ := view.Update(msg)
		view = model.(*NamespaceView)

		// Filter should be empty but we should be in filter mode
		if view.filter != "" {
			t.Errorf("filter should be empty after pressing '/', got: %q", view.filter)
		}

		// Now typing should go to filter
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")}
		model, _ = view.Update(msg)
		view = model.(*NamespaceView)

		if view.filter != "p" {
			t.Errorf("typing after '/' should add to filter, got: %q", view.filter)
		}
	})

	t.Run("navigation keys add to filter in filter mode", func(t *testing.T) {
		view := NewNamespaceView(namespaces, "")
		initialIndex := view.selectedIndex

		// Enter filter mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
		model, _ := view.Update(msg)
		view = model.(*NamespaceView)

		// Press 'j' - should add to filter, not navigate
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		model, _ = view.Update(msg)
		view = model.(*NamespaceView)

		if view.filter != "j" {
			t.Errorf("'j' key should add to filter in filter mode, got filter: %q", view.filter)
		}

		if view.selectedIndex != initialIndex {
			t.Errorf("'j' key should not navigate in filter mode, selection changed from %d to %d", initialIndex, view.selectedIndex)
		}
	})

	t.Run("enter exits filter mode and applies filter", func(t *testing.T) {
		view := NewNamespaceView(namespaces, "")

		// Enter filter mode and type a filter that matches multiple items
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
		model, _ := view.Update(msg)
		view = model.(*NamespaceView)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}
		model, _ = view.Update(msg)
		view = model.(*NamespaceView)

		if view.filter != "e" {
			t.Errorf("filter should be 'e', got: %q", view.filter)
		}

		// Should match "development" and "testing" (plus "all" doesn't match)
		if len(view.filteredItems) < 2 {
			t.Errorf("filter 'e' should match multiple items, got %d items", len(view.filteredItems))
		}

		// Press enter - should exit filter mode
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		model, _ = view.Update(msg)
		view = model.(*NamespaceView)

		// Filter should still be applied
		if view.filter != "e" {
			t.Errorf("filter should remain 'e' after enter, got: %q", view.filter)
		}

		// Should not be in filter mode anymore
		if view.filterMode {
			t.Error("should not be in filter mode after pressing enter")
		}

		// Now 'j' should navigate again, not add to filter
		initialIndex := view.selectedIndex
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		model, _ = view.Update(msg)
		view = model.(*NamespaceView)

		if view.filter != "e" {
			t.Errorf("filter should remain 'e' after navigation, got: %q", view.filter)
		}

		// Should navigate if there are multiple items
		if len(view.filteredItems) > 1 && view.selectedIndex == initialIndex && initialIndex < len(view.filteredItems)-1 {
			t.Error("'j' key should navigate after exiting filter mode")
		}
	})

	t.Run("slash again re-enters filter mode for editing", func(t *testing.T) {
		view := NewNamespaceView(namespaces, "")

		// Set up existing filter
		view.filter = "prod"
		view.applyFilter()

		// Press '/' - should enter filter mode for editing
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
		model, _ := view.Update(msg)
		view = model.(*NamespaceView)

		// Should still have the existing filter
		if view.filter != "prod" {
			t.Errorf("existing filter should be preserved when re-entering filter mode, got: %q", view.filter)
		}

		// Typing should modify the filter
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")}
		model, _ = view.Update(msg)
		view = model.(*NamespaceView)

		if view.filter != "produ" {
			t.Errorf("typing should modify existing filter, got: %q", view.filter)
		}
	})

	t.Run("delete key clears filter when editing", func(t *testing.T) {
		view := NewNamespaceView(namespaces, "")

		// Set up existing filter and enter filter mode
		view.filter = "prod"
		view.applyFilter()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
		model, _ := view.Update(msg)
		view = model.(*NamespaceView)

		// Press delete - should clear the filter
		msg = tea.KeyMsg{Type: tea.KeyDelete}
		model, _ = view.Update(msg)
		view = model.(*NamespaceView)

		if view.filter != "" {
			t.Errorf("delete key should clear filter, got: %q", view.filter)
		}

		// Should show all namespaces again
		if len(view.filteredItems) != len(view.namespaces) {
			t.Errorf("clearing filter should show all namespaces, got %d items, want %d", len(view.filteredItems), len(view.namespaces))
		}
	})

	t.Run("escape exits filter mode without applying changes", func(t *testing.T) {
		view := NewNamespaceView(namespaces, "")

		// Set up existing filter
		originalFilter := "prod"
		view.filter = originalFilter
		view.applyFilter()
		originalFilteredCount := len(view.filteredItems)

		// Enter filter mode and modify
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
		model, _ := view.Update(msg)
		view = model.(*NamespaceView)

		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
		model, _ = view.Update(msg)
		view = model.(*NamespaceView)

		if view.filter != "prodx" {
			t.Errorf("filter should be modified to 'prodx', got: %q", view.filter)
		}

		// Press escape - should revert to original filter
		msg = tea.KeyMsg{Type: tea.KeyEsc}
		model, _ = view.Update(msg)
		view = model.(*NamespaceView)

		if view.filter != originalFilter {
			t.Errorf("escape should revert filter to original '%s', got: %q", originalFilter, view.filter)
		}

		if len(view.filteredItems) != originalFilteredCount {
			t.Errorf("escape should revert filtered items count to %d, got %d", originalFilteredCount, len(view.filteredItems))
		}
	})
}
