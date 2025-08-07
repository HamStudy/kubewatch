package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func createTestLogView(t *testing.T) *LogView {
	lv := NewLogView()
	lv.SetSize(80, 24)
	return lv
}

func TestLogViewInitialization(t *testing.T) {
	lv := createTestLogView(t)

	// Test initial state
	if lv.content == nil {
		t.Error("Content slice should be initialized")
	}

	// containers and pods are initialized as nil and populated during streaming
	if lv.containers != nil && len(lv.containers) != 0 {
		t.Error("Containers slice should be empty initially")
	}

	if lv.pods != nil && len(lv.pods) != 0 {
		t.Error("Pods slice should be empty initially")
	}
	if lv.selectedContainer != -1 {
		t.Errorf("Expected selectedContainer to be -1 (all), got %d", lv.selectedContainer)
	}

	if lv.selectedPod != -1 {
		t.Errorf("Expected selectedPod to be -1 (all), got %d", lv.selectedPod)
	}

	// Test Init command
	cmd := lv.Init()
	if cmd != nil {
		t.Error("LogView Init should return nil command")
	}
}

func TestLogViewKeyHandling(t *testing.T) {
	lv := createTestLogView(t)

	// Add some test logs
	lv.content = []string{
		"Log line 1",
		"Log line 2",
		"Log line 3",
		"Log line 4",
		"Log line 5",
	}
	tests := []struct {
		name        string
		key         string
		description string
	}{
		{"scroll down", "j", "Should scroll down"},
		{"scroll up", "k", "Should scroll up"},
		{"page down", "pgdown", "Should page down"},
		{"page up", "pgup", "Should page up"},
		{"home", "g", "Should go to top"},
		{"end", "G", "Should go to bottom"},
		{"toggle follow", "f", "Should toggle follow mode"},
		{"search", "/", "Should enter search mode"},
		{"clear", "C", "Should clear logs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var keyMsg tea.KeyMsg
			switch tt.key {
			case "j":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
			case "k":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
			case "pgdown":
				keyMsg = tea.KeyMsg{Type: tea.KeyPgDown}
			case "pgup":
				keyMsg = tea.KeyMsg{Type: tea.KeyPgUp}
			case "g":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")}
			case "G":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")}
			case "f":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}
			case "/":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
			case "C":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")}
			}

			// Test that key handling doesn't panic
			model, cmd := lv.Update(keyMsg)
			lv = model.(*LogView)

			// Commands may or may not be returned depending on the key
			_ = cmd
		})
	}
}

func TestLogViewFollowMode(t *testing.T) {
	lv := createTestLogView(t)

	// Test initial follow state (should be false by default)
	if lv.following {
		t.Error("Follow mode should be disabled by default")
	}

	// Toggle follow mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}
	model, _ := lv.Update(keyMsg)
	lv = model.(*LogView)

	if !lv.following {
		t.Error("Follow mode should be enabled after toggle")
	}

	// Toggle again
	model, _ = lv.Update(keyMsg)
	lv = model.(*LogView)

	if lv.following {
		t.Error("Follow mode should be disabled after second toggle")
	}
}

func TestLogViewSearchMode(t *testing.T) {
	lv := createTestLogView(t)

	// Test initial search state
	if lv.searchMode {
		t.Error("Search mode should be disabled by default")
	}

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	model, _ := lv.Update(keyMsg)
	lv = model.(*LogView)

	if !lv.searchMode {
		t.Error("Search mode should be enabled after pressing '/'")
	}

	// Test IsSearchMode method
	if !lv.IsSearchMode() {
		t.Error("IsSearchMode should return true when in search mode")
	}

	// Exit search mode with escape
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = lv.Update(keyMsg)
	lv = model.(*LogView)

	if lv.searchMode {
		t.Error("Search mode should be disabled after escape")
	}
}

func TestLogViewContainerCycling(t *testing.T) {
	lv := createTestLogView(t)

	// Add test containers (need more than 1 for cycling to work)
	lv.containers = []string{"container1", "container2", "container3"}
	lv.selectedContainer = -1 // Start with all containers

	// Test container cycling
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
	model, _ := lv.Update(keyMsg)
	lv = model.(*LogView)

	if lv.selectedContainer != 0 {
		t.Errorf("Expected selectedContainer 0, got %d", lv.selectedContainer)
	}

	// Cycle again
	model, _ = lv.Update(keyMsg)
	lv = model.(*LogView)

	if lv.selectedContainer != 1 {
		t.Errorf("Expected selectedContainer 1, got %d", lv.selectedContainer)
	}

	// Cycle again
	model, _ = lv.Update(keyMsg)
	lv = model.(*LogView)

	if lv.selectedContainer != 2 {
		t.Errorf("Expected selectedContainer 2, got %d", lv.selectedContainer)
	}

	// Cycle to wrap around to all (-1)
	model, _ = lv.Update(keyMsg)
	lv = model.(*LogView)

	if lv.selectedContainer != -1 {
		t.Errorf("Expected selectedContainer -1 (all), got %d", lv.selectedContainer)
	}
}
func TestLogViewPodCycling(t *testing.T) {
	lv := createTestLogView(t)

	// Add test pods (need more than 1 for cycling to work)
	lv.pods = []string{"pod1", "pod2", "pod3"}
	lv.selectedPod = -1 // Start with all pods

	// Test pod cycling
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")}
	model, _ := lv.Update(keyMsg)
	lv = model.(*LogView)

	if lv.selectedPod != 0 {
		t.Errorf("Expected selectedPod 0, got %d", lv.selectedPod)
	}

	// Cycle again
	model, _ = lv.Update(keyMsg)
	lv = model.(*LogView)

	if lv.selectedPod != 1 {
		t.Errorf("Expected selectedPod 1, got %d", lv.selectedPod)
	}

	// Cycle again
	model, _ = lv.Update(keyMsg)
	lv = model.(*LogView)

	if lv.selectedPod != 2 {
		t.Errorf("Expected selectedPod 2, got %d", lv.selectedPod)
	}

	// Cycle to wrap around to all (-1)
	model, _ = lv.Update(keyMsg)
	lv = model.(*LogView)

	if lv.selectedPod != -1 {
		t.Errorf("Expected selectedPod -1 (all), got %d", lv.selectedPod)
	}
}
func TestLogViewClearLogs(t *testing.T) {
	lv := createTestLogView(t)

	// Add test logs
	lv.content = []string{"Log 1", "Log 2", "Log 3"}

	if len(lv.content) != 3 {
		t.Errorf("Expected 3 logs, got %d", len(lv.content))
	}

	// Clear logs
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")}
	model, _ := lv.Update(keyMsg)
	lv = model.(*LogView)

	if len(lv.content) != 0 {
		t.Errorf("Expected 0 logs after clear, got %d", len(lv.content))
	}
}
func TestLogViewScrolling(t *testing.T) {
	lv := createTestLogView(t)

	// Add many test logs
	for i := 0; i < 100; i++ {
		lv.content = append(lv.content, "Log line "+string(rune('0'+i%10)))
	}

	// Test scrolling operations don't panic
	keys := []string{"j", "k", "g", "G"}

	for _, key := range keys {
		var keyMsg tea.KeyMsg
		switch key {
		case "j":
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		case "k":
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
		case "g":
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")}
		case "G":
			keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")}
		}

		model, _ := lv.Update(keyMsg)
		lv = model.(*LogView)

		// Just verify the operation doesn't panic
	}

	// Test end (go to bottom) enables follow mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")}
	model, _ := lv.Update(keyMsg)
	lv = model.(*LogView)

	// Should be at bottom and follow mode should be enabled
	if !lv.following {
		t.Error("Follow mode should be enabled after going to bottom")
	}
}
func TestLogViewViewRendering(t *testing.T) {
	lv := createTestLogView(t)

	// Test empty view
	view := lv.View()
	if len(view) == 0 {
		t.Error("View should not be empty")
	}

	// Add some logs and test again
	lv.content = []string{"Test log 1", "Test log 2", "Test log 3"}
	// Need to set content in viewport for it to appear in view
	lv.viewport.SetContent("Test log 1\nTest log 2\nTest log 3")
	view = lv.View()

	if len(view) == 0 {
		t.Error("View with logs should not be empty")
	}

	// View should contain log content
	if !containsText(view, "Test log") {
		t.Error("View should contain log content")
	}
}
func TestLogViewSetSize(t *testing.T) {
	lv := createTestLogView(t)

	// Test size setting
	lv.SetSize(100, 50)

	if lv.width != 100 {
		t.Errorf("Expected width 100, got %d", lv.width)
	}

	if lv.height != 50 {
		t.Errorf("Expected height 50, got %d", lv.height)
	}
}

func TestLogViewStopStreaming(t *testing.T) {
	lv := createTestLogView(t)

	// Test stop streaming (should not panic)
	cmd := lv.StopStreaming()

	// Should return a command or nil
	_ = cmd

	// Test that tailing is stopped
	if lv.tailing {
		t.Error("Tailing should be stopped")
	}
}
