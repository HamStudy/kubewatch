package views

import (
	"testing"
	"time"

	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/components/table"
	"github.com/HamStudy/kubewatch/internal/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// createTestState creates a State with consistent defaults for testing
func createTestState(resourceType core.ResourceType, namespace, context string) *core.State {
	return &core.State{
		CurrentResourceType: resourceType,
		CurrentNamespace:    namespace,
		CurrentContext:      context,
		SortAscending:       true, // Default to A-Z sorting for predictable tests
		SortColumn:          "NAME",
	}
}

func createTestResourceView(t *testing.T) *ResourceView {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
		SortAscending:       true, // Default to A-Z sorting
	}

	// Create with nil client for pure UI testing
	rv := NewResourceView(state, nil)
	rv.SetSize(80, 24)

	return rv
}

func createTestResourceViewWithData(t *testing.T) *ResourceView {
	rv := createTestResourceView(t)

	// Add some test data
	rv.headers = []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"}
	rv.rows = [][]string{
		{"test-pod-1", "1/1", "Running", "0", "5m"},
		{"test-pod-2", "0/1", "Pending", "0", "2m"},
		{"test-pod-3", "1/1", "Running", "1", "10m"},
	}

	// Set up resource map for selection tracking
	rv.resourceMap = make(map[int]*selection.ResourceIdentity)
	rv.resourceMap[0] = &selection.ResourceIdentity{
		Context:   "test-context",
		Namespace: "default",
		Name:      "test-pod-1",
		UID:       "uid-1",
		Kind:      "Pod",
	}
	rv.resourceMap[1] = &selection.ResourceIdentity{
		Context:   "test-context",
		Namespace: "default",
		Name:      "test-pod-2",
		UID:       "uid-2",
		Kind:      "Pod",
	}
	rv.resourceMap[2] = &selection.ResourceIdentity{
		Context:   "test-context",
		Namespace: "default",
		Name:      "test-pod-3",
		UID:       "uid-3",
		Kind:      "Pod",
	}

	rv.selectedRow = 0
	rv.selectedIdentity = rv.resourceMap[0]

	// If new components are enabled, populate the table component with test data
	if rv.useNewComponents && rv.tableComponent != nil {
		tableRows := []table.Row{
			{ID: "resource-0", Values: []string{"test-pod-1", "1/1", "Running", "0", "5m"}, Style: lipgloss.NewStyle()},
			{ID: "resource-1", Values: []string{"test-pod-2", "0/1", "Pending", "0", "2m"}, Style: lipgloss.NewStyle()},
			{ID: "resource-2", Values: []string{"test-pod-3", "1/1", "Running", "1", "10m"}, Style: lipgloss.NewStyle()},
		}
		rv.tableComponent.SetRows(tableRows)
	}

	return rv
}

func TestResourceViewInitialization(t *testing.T) {
	rv := createTestResourceView(t)

	// Test initial state
	if rv.state == nil {
		t.Error("State should not be nil")
	}

	if rv.selectedRow != 0 {
		t.Errorf("Expected selectedRow to be 0, got %d", rv.selectedRow)
	}

	if rv.resourceMap == nil {
		t.Error("Resource map should be initialized")
	}

	// Test Init command
	cmd := rv.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestResourceViewNavigation(t *testing.T) {
	rv := createTestResourceViewWithData(t)

	tests := []struct {
		name         string
		key          string
		expectedRow  int
		expectedName string
	}{
		{"move down", "j", 1, "test-pod-2"},
		{"move down again", "j", 2, "test-pod-3"},
		{"move up", "k", 1, "test-pod-2"},
		{"move to top", "k", 0, "test-pod-1"},
		{"home key", "home", 0, "test-pod-1"},
		{"end key", "end", 2, "test-pod-3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var keyMsg tea.KeyMsg
			switch tt.key {
			case "j":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
			case "k":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
			case "home":
				keyMsg = tea.KeyMsg{Type: tea.KeyHome}
			case "end":
				keyMsg = tea.KeyMsg{Type: tea.KeyEnd}
			}

			model, _ := rv.Update(keyMsg)
			rv = model.(*ResourceView)

			if rv.selectedRow != tt.expectedRow {
				t.Errorf("Expected selectedRow %d, got %d", tt.expectedRow, rv.selectedRow)
			}

			selectedName := rv.GetSelectedResourceName()
			if selectedName != tt.expectedName {
				t.Errorf("Expected selected name %s, got %s", tt.expectedName, selectedName)
			}
		})
	}
}

func TestResourceViewSelectionTracking(t *testing.T) {
	rv := createTestResourceViewWithData(t)

	// Test initial selection
	if rv.selectedIdentity == nil {
		t.Error("Selected identity should not be nil")
	}

	if rv.selectedIdentity.Name != "test-pod-1" {
		t.Errorf("Expected selected name test-pod-1, got %s", rv.selectedIdentity.Name)
	}

	// Move selection
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	model, _ := rv.Update(keyMsg)
	rv = model.(*ResourceView)

	if rv.selectedIdentity.Name != "test-pod-2" {
		t.Errorf("Expected selected name test-pod-2, got %s", rv.selectedIdentity.Name)
	}

	// Test selection restoration after data change
	originalIdentity := rv.selectedIdentity

	// Simulate data refresh that changes order
	rv.rows = [][]string{
		{"test-pod-3", "1/1", "Running", "1", "10m"},
		{"test-pod-2", "0/1", "Pending", "0", "2m"}, // This was selected
		{"test-pod-1", "1/1", "Running", "0", "5m"},
	}

	// Update resource map to reflect new order
	rv.resourceMap[0] = &selection.ResourceIdentity{
		Context:   "test-context",
		Namespace: "default",
		Name:      "test-pod-3",
		UID:       "uid-3",
		Kind:      "Pod",
	}
	rv.resourceMap[1] = &selection.ResourceIdentity{
		Context:   "test-context",
		Namespace: "default",
		Name:      "test-pod-2",
		UID:       "uid-2",
		Kind:      "Pod",
	}
	rv.resourceMap[2] = &selection.ResourceIdentity{
		Context:   "test-context",
		Namespace: "default",
		Name:      "test-pod-1",
		UID:       "uid-1",
		Kind:      "Pod",
	}

	// Restore selection should find the same resource at new index
	rv.restoreSelectionByIdentity()

	if rv.selectedRow != 1 {
		t.Errorf("Expected selectedRow 1 after restore, got %d", rv.selectedRow)
	}

	if rv.selectedIdentity.UID != originalIdentity.UID {
		t.Errorf("Expected same UID after restore, got %s", rv.selectedIdentity.UID)
	}
}

func TestResourceViewViewRendering(t *testing.T) {
	rv := createTestResourceViewWithData(t)

	// Test that view renders without panicking
	view := rv.View()
	if len(view) == 0 {
		t.Error("View should not be empty")
	}

	// Test that view contains expected content
	expectedContent := []string{
		"test-pod-1",
		"test-pod-2",
		"test-pod-3",
		"Running",
		"Pending",
	}

	for _, content := range expectedContent {
		if !containsText(view, content) {
			t.Errorf("View should contain '%s'", content)
		}
	}
}

func TestResourceViewCompactMode(t *testing.T) {
	rv := createTestResourceViewWithData(t)

	// Test that compact mode can be set
	rv.SetCompactMode(true)
	if !rv.compactMode {
		t.Error("Compact mode should be enabled")
	}

	// Test that compact mode can be disabled
	rv.SetCompactMode(false)
	if rv.compactMode {
		t.Error("Compact mode should be disabled")
	}

	// Test view still renders in both modes
	rv.SetCompactMode(true)
	compactView := rv.View()
	if len(compactView) == 0 {
		t.Error("Compact view should not be empty")
	}

	rv.SetCompactMode(false)
	normalView := rv.View()
	if len(normalView) == 0 {
		t.Error("Normal view should not be empty")
	}
}
func TestResourceViewMultiContext(t *testing.T) {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContexts:     []string{"context-1", "context-2"},
	}

	// Create multi-context resource view
	rv := NewResourceViewWithMultiContext(state, nil)
	rv.SetSize(80, 24)

	if !rv.isMultiContext {
		t.Error("Should be in multi-context mode")
	}

	if !rv.showContextColumn {
		t.Error("Should show context column in multi-context mode")
	}

	// Test view rendering with context column
	view := rv.View()
	if len(view) == 0 {
		t.Error("Multi-context view should not be empty")
	}
}

func TestResourceViewSorting(t *testing.T) {
	rv := createTestResourceViewWithData(t)

	// Test initial sort state
	if rv.state.SortColumn == "" {
		rv.state.SortColumn = "NAME"
		rv.state.SortAscending = true
	}

	// Test that data can be sorted (this would normally be done by refresh)
	// We'll just verify the sort state is maintained
	originalColumn := rv.state.SortColumn
	originalAscending := rv.state.SortAscending

	// Simulate sort column change
	rv.state.SortColumn = "STATUS"
	rv.state.SortAscending = false

	if rv.state.SortColumn != "STATUS" {
		t.Error("Sort column should be updated")
	}

	if rv.state.SortAscending != false {
		t.Error("Sort direction should be updated")
	}

	// Restore original state
	rv.state.SortColumn = originalColumn
	rv.state.SortAscending = originalAscending
}

func TestResourceViewWordWrap(t *testing.T) {
	rv := createTestResourceViewWithData(t)

	// Test word wrap toggle
	originalWrap := rv.wordWrap

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")}
	model, _ := rv.Update(keyMsg)
	rv = model.(*ResourceView)

	if rv.wordWrap == originalWrap {
		t.Error("Word wrap should be toggled")
	}

	// Toggle again
	model, _ = rv.Update(keyMsg)
	rv = model.(*ResourceView)

	if rv.wordWrap != originalWrap {
		t.Error("Word wrap should be restored to original state")
	}
}

func TestResourceViewPagination(t *testing.T) {
	rv := createTestResourceViewWithData(t)

	// Add more rows to test pagination
	for i := 4; i <= 20; i++ {
		rv.rows = append(rv.rows, []string{
			"test-pod-" + string(rune('0'+i)),
			"1/1",
			"Running",
			"0",
			"1m",
		})
		rv.resourceMap[i-1] = &selection.ResourceIdentity{
			Context:   "test-context",
			Namespace: "default",
			Name:      "test-pod-" + string(rune('0'+i)),
			UID:       "uid-" + string(rune('0'+i)),
			Kind:      "Pod",
		}
	}

	// Set small viewport for testing
	rv.viewportHeight = 5

	// Test page down
	keyMsg := tea.KeyMsg{Type: tea.KeyPgDown}
	model, _ := rv.Update(keyMsg)
	rv = model.(*ResourceView)

	if rv.selectedRow < 5 {
		t.Error("Page down should move selection significantly")
	}

	// Test page up
	keyMsg = tea.KeyMsg{Type: tea.KeyPgUp}
	model, _ = rv.Update(keyMsg)
	rv = model.(*ResourceView)

	if rv.selectedRow >= 5 {
		t.Error("Page up should move selection back")
	}
}

func TestResourceViewGetters(t *testing.T) {
	rv := createTestResourceViewWithData(t)

	// Test GetSelectedResourceName
	name := rv.GetSelectedResourceName()
	if name != "test-pod-1" {
		t.Errorf("Expected selected name test-pod-1, got %s", name)
	}

	// Test GetSelectedResourceContext (returns empty for single-context mode)
	context := rv.GetSelectedResourceContext()
	if context != "" {
		t.Errorf("Expected empty context for single-context mode, got %s", context)
	}

	// Move selection and test again
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	model, _ := rv.Update(keyMsg)
	rv = model.(*ResourceView)

	name = rv.GetSelectedResourceName()
	if name != "test-pod-2" {
		t.Errorf("Expected selected name test-pod-2, got %s", name)
	}
}
func TestResourceViewEdgeCases(t *testing.T) {
	rv := createTestResourceView(t)

	// Test with empty data
	view := rv.View()
	if len(view) == 0 {
		t.Error("View should render even with no data")
	}

	// Test navigation with no data
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	model, _ := rv.Update(keyMsg)
	rv = model.(*ResourceView)

	if rv.selectedRow != 0 {
		t.Error("Selection should stay at 0 with no data")
	}

	// Test GetSelectedResourceName with no data
	name := rv.GetSelectedResourceName()
	if name != "" {
		t.Errorf("Expected empty name with no data, got %s", name)
	}
}

// TestResourceViewSelectionJumpingBugDuringRefresh reproduces the bug where
// selection jumps to the top after a refresh, instead of staying on the
// currently selected resource.
//
// This test simulates the actual refresh flow that happens in production,
// where updateTableWithPods is called with new data.
//
// BUG SUMMARY:
// The selection persistence mechanism in ResourceView has issues when:
// 1. Pods are reordered (e.g., due to sorting) - selection doesn't follow the pod
// 2. New pods are added - selection jumps to wrong pod
// 3. The restoreSelectionByIdentity() method is not working correctly in all cases
//
// EXPECTED BEHAVIOR:
// - Selection should follow the same pod (by UID) even if its position changes
// - If selected pod is deleted, selection should move to the next logical pod
// - Adding/removing other pods should not affect which pod is selected
//
// ACTUAL BEHAVIOR (BUG):
// - Selection often jumps to top or to a different pod after refresh
// - The selectedIdentity tracking is not properly maintained across updates
func TestResourceViewSelectionJumpingBugDuringRefresh(t *testing.T) {
	tests := []struct {
		name                string
		initialSelection    int
		expectedSelection   int
		refreshDataModifier func(rv *ResourceView)
		shouldFail          bool
		failureMessage      string
	}{
		{
			name:              "selection stays on same pod after refresh with same data",
			initialSelection:  2,
			expectedSelection: 2,
			refreshDataModifier: func(rv *ResourceView) {
				// Simulate refresh with same pods but updated status
				rv.rows = [][]string{
					{"test-pod-1", "1/1", "Running", "0", "6m"},  // Age changed
					{"test-pod-2", "1/1", "Running", "0", "3m"},  // Status changed to Running
					{"test-pod-3", "1/1", "Running", "2", "11m"}, // Restarts increased
				}
				// Resource map stays the same (same UIDs)
			},
			shouldFail:     true,
			failureMessage: "BUG: Selection should stay on test-pod-3 but jumps to top",
		},
		{
			name:              "selection stays when pods are reordered",
			initialSelection:  1,
			expectedSelection: 2, // test-pod-2 should now be at index 2
			refreshDataModifier: func(rv *ResourceView) {
				// Simulate refresh where pods are reordered (e.g., by age)
				rv.rows = [][]string{
					{"test-pod-3", "1/1", "Running", "1", "10m"},
					{"test-pod-1", "1/1", "Running", "0", "5m"},
					{"test-pod-2", "0/1", "Pending", "0", "2m"}, // Selected pod moved to index 2
				}
				// Update resource map to reflect new order
				rv.resourceMap[0] = &selection.ResourceIdentity{
					Context:   "test-context",
					Namespace: "default",
					Name:      "test-pod-3",
					UID:       "uid-3",
					Kind:      "Pod",
				}
				rv.resourceMap[1] = &selection.ResourceIdentity{
					Context:   "test-context",
					Namespace: "default",
					Name:      "test-pod-1",
					UID:       "uid-1",
					Kind:      "Pod",
				}
				rv.resourceMap[2] = &selection.ResourceIdentity{
					Context:   "test-context",
					Namespace: "default",
					Name:      "test-pod-2",
					UID:       "uid-2",
					Kind:      "Pod",
				}
			},
			shouldFail:     true,
			failureMessage: "BUG: Selection should follow test-pod-2 to new position but resets",
		},
		{
			name:              "selection moves to next pod when selected pod is deleted",
			initialSelection:  1,
			expectedSelection: 1, // Should select next available pod at same index
			refreshDataModifier: func(rv *ResourceView) {
				// Simulate refresh where test-pod-2 is deleted
				rv.rows = [][]string{
					{"test-pod-1", "1/1", "Running", "0", "5m"},
					{"test-pod-3", "1/1", "Running", "1", "10m"},
				}
				// Update resource map
				rv.resourceMap = make(map[int]*selection.ResourceIdentity)
				rv.resourceMap[0] = &selection.ResourceIdentity{
					Context:   "test-context",
					Namespace: "default",
					Name:      "test-pod-1",
					UID:       "uid-1",
					Kind:      "Pod",
				}
				rv.resourceMap[1] = &selection.ResourceIdentity{
					Context:   "test-context",
					Namespace: "default",
					Name:      "test-pod-3",
					UID:       "uid-3",
					Kind:      "Pod",
				}
			},
			shouldFail:     false, // This should work correctly now
			failureMessage: "",
		},
		{
			name:              "selection stays at bottom when last pod selected",
			initialSelection:  2,
			expectedSelection: 2, // Should stay on test-pod-3 which remains at index 2
			refreshDataModifier: func(rv *ResourceView) {
				// Add a new pod
				rv.rows = [][]string{
					{"test-pod-1", "1/1", "Running", "0", "5m"},
					{"test-pod-2", "0/1", "Pending", "0", "2m"},
					{"test-pod-3", "1/1", "Running", "1", "10m"},
					{"test-pod-4", "1/1", "Running", "0", "1m"}, // New pod
				}
				// Update resource map - keep existing ones
				rv.resourceMap[3] = &selection.ResourceIdentity{
					Context:   "test-context",
					Namespace: "default",
					Name:      "test-pod-4",
					UID:       "uid-4",
					Kind:      "Pod",
				}
			},
			shouldFail:     false, // This should work correctly now
			failureMessage: "",
		},
		{
			name:              "selection handles all pods being replaced",
			initialSelection:  1,
			expectedSelection: 0, // Should reset to top when all pods are new
			refreshDataModifier: func(rv *ResourceView) {
				// Replace all pods with new ones
				rv.rows = [][]string{
					{"new-pod-1", "1/1", "Running", "0", "1m"},
					{"new-pod-2", "0/1", "Pending", "0", "30s"},
					{"new-pod-3", "1/1", "Running", "0", "15s"},
				}
				// Update resource map with all new UIDs
				rv.resourceMap = make(map[int]*selection.ResourceIdentity)
				rv.resourceMap[0] = &selection.ResourceIdentity{
					Context:   "test-context",
					Namespace: "default",
					Name:      "new-pod-1",
					UID:       "new-uid-1",
					Kind:      "Pod",
				}
				rv.resourceMap[1] = &selection.ResourceIdentity{
					Context:   "test-context",
					Namespace: "default",
					Name:      "new-pod-2",
					UID:       "new-uid-2",
					Kind:      "Pod",
				}
				rv.resourceMap[2] = &selection.ResourceIdentity{
					Context:   "test-context",
					Namespace: "default",
					Name:      "new-pod-3",
					UID:       "new-uid-3",
					Kind:      "Pod",
				}
			},
			shouldFail:     false, // This case should work correctly (reset to top)
			failureMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create ResourceView with initial data
			rv := createTestResourceViewWithData(t)

			// Move selection to desired position
			for i := 0; i < tt.initialSelection; i++ {
				keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
				model, _ := rv.Update(keyMsg)
				rv = model.(*ResourceView)
			}

			// Verify we're at the right position
			if rv.selectedRow != tt.initialSelection {
				t.Fatalf("Failed to set initial selection to %d, got %d", tt.initialSelection, rv.selectedRow)
			}

			// Store the selected pod's identity before refresh
			selectedPodBefore := rv.selectedIdentity
			if selectedPodBefore == nil {
				t.Fatal("Selected identity should not be nil before refresh")
			}
			selectedNameBefore := selectedPodBefore.Name
			selectedUIDBefore := selectedPodBefore.UID

			// Simulate a refresh by updating the data
			tt.refreshDataModifier(rv)

			// This is where the bug manifests - the view doesn't properly
			// restore selection after data refresh
			// In a real scenario, this would be triggered by UpdateResourceData
			// but we're simulating it directly here

			// Try to restore selection (this is what should happen automatically)
			// but currently doesn't work correctly
			rv.restoreSelectionByIdentity()

			// Check if selection is where we expect it
			if tt.shouldFail {
				// These tests demonstrate the bug - they will fail
				if rv.selectedRow != tt.expectedSelection {
					t.Logf("%s - Selection jumped from %d to %d (expected %d)",
						tt.failureMessage, tt.initialSelection, rv.selectedRow, tt.expectedSelection)
					t.Logf("Selected pod before: %s (UID: %s)", selectedNameBefore, selectedUIDBefore)
					if rv.selectedIdentity != nil {
						t.Logf("Selected pod after: %s (UID: %s)", rv.selectedIdentity.Name, rv.selectedIdentity.UID)
					} else {
						t.Logf("Selected pod after: nil")
					}
					t.Fail() // Use Fail() instead of Error() to show this is expected to fail
				}
			} else {
				// This test should pass even with the bug
				if rv.selectedRow != tt.expectedSelection {
					t.Errorf("Expected selection at %d, got %d", tt.expectedSelection, rv.selectedRow)
				}
			}
		})
	}
}

// TestResourceViewRefreshWithRealTimeUpdates simulates real-time updates
// that would come from Kubernetes watch events
func TestResourceViewRefreshWithRealTimeUpdates(t *testing.T) {
	rv := createTestResourceViewWithData(t)

	// User navigates to the middle pod
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	model, _ := rv.Update(keyMsg)
	rv = model.(*ResourceView)

	if rv.selectedRow != 1 {
		t.Fatalf("Expected selection at row 1, got %d", rv.selectedRow)
	}

	selectedPodName := rv.GetSelectedResourceName()
	if selectedPodName != "test-pod-2" {
		t.Fatalf("Expected selected pod to be test-pod-2, got %s", selectedPodName)
	}

	// Simulate multiple rapid updates (like what happens with watch events)
	updates := []struct {
		description string
		updateFunc  func()
	}{
		{
			description: "Pod status update",
			updateFunc: func() {
				rv.rows[1][2] = "Running" // test-pod-2 becomes Running
			},
		},
		{
			description: "New pod added at top",
			updateFunc: func() {
				// Insert new pod at beginning
				newRow := []string{"test-pod-0", "1/1", "Running", "0", "1s"}
				rv.rows = append([][]string{newRow}, rv.rows...)

				// Shift resource map indices
				newMap := make(map[int]*selection.ResourceIdentity)
				newMap[0] = &selection.ResourceIdentity{
					Context:   "test-context",
					Namespace: "default",
					Name:      "test-pod-0",
					UID:       "uid-0",
					Kind:      "Pod",
				}
				for k, v := range rv.resourceMap {
					newMap[k+1] = v
				}
				rv.resourceMap = newMap
			},
		},
		{
			description: "Pod restart count increases",
			updateFunc: func() {
				// Find test-pod-2 and update its restart count
				for i, row := range rv.rows {
					if row[0] == "test-pod-2" {
						rv.rows[i][3] = "1" // Increment restart count
						break
					}
				}
			},
		},
	}

	for _, update := range updates {
		t.Run(update.description, func(t *testing.T) {
			// Apply the update
			update.updateFunc()

			// Try to restore selection
			rv.restoreSelectionByIdentity()

			// Check if we're still on test-pod-2
			currentSelectedName := rv.GetSelectedResourceName()

			// This will fail due to the bug - selection jumps around
			if currentSelectedName != selectedPodName {
				t.Logf("BUG: After %s, selection jumped from %s to %s",
					update.description, selectedPodName, currentSelectedName)
				t.Fail() // Expected to fail, demonstrating the bug
			}
		})
	}
}

// TestResourceViewSelectionPersistenceAcrossContextSwitch tests selection
// persistence when switching between single and multi-context modes
// TestResourceViewUpdateTableWithPodsSelectionBug tests the actual bug scenario
// where selection jumps during updateTableWithPods calls (the real refresh flow)
func TestResourceViewUpdateTableWithPodsSelectionBug(t *testing.T) {
	// We need to import the k8s types for this test
	// Create test pods using the actual Pod type
	createTestPods := func() []v1.Pod {
		return []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-1",
					Namespace: "default",
					UID:       "uid-1",
					CreationTimestamp: metav1.Time{
						Time: time.Now().Add(-5 * time.Minute),
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true, RestartCount: 0},
					},
					PodIP: "10.0.0.1",
				},
				Spec: v1.PodSpec{
					NodeName: "node-1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-2",
					Namespace: "default",
					UID:       "uid-2",
					CreationTimestamp: metav1.Time{
						Time: time.Now().Add(-2 * time.Minute),
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: false, RestartCount: 0},
					},
					PodIP: "",
				},
				Spec: v1.PodSpec{
					NodeName: "",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-3",
					Namespace: "default",
					UID:       "uid-3",
					CreationTimestamp: metav1.Time{
						Time: time.Now().Add(-10 * time.Minute),
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true, RestartCount: 1},
					},
					PodIP: "10.0.0.3",
				},
				Spec: v1.PodSpec{
					NodeName: "node-2",
				},
			},
		}
	}

	t.Run("selection jumps to top after refresh with same pods", func(t *testing.T) {
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			CurrentContext:      "test-context",
		}

		rv := NewResourceView(state, nil)
		rv.SetSize(80, 24)

		// Initial load of pods
		pods := createTestPods()
		rv.updateTableWithPods(pods)

		// Navigate to middle pod
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		model, _ := rv.Update(keyMsg)
		rv = model.(*ResourceView)

		// Verify we're on test-pod-2
		if rv.selectedRow != 1 {
			t.Fatalf("Expected to be on row 1, got %d", rv.selectedRow)
		}
		if rv.GetSelectedResourceName() != "test-pod-2" {
			t.Fatalf("Expected to select test-pod-2, got %s", rv.GetSelectedResourceName())
		}

		// Simulate a refresh with updated pod data (status changes, etc)
		updatedPods := createTestPods()
		// Change test-pod-2 status to Running
		updatedPods[1].Status.Phase = v1.PodRunning
		updatedPods[1].Status.ContainerStatuses[0].Ready = true
		updatedPods[1].Status.PodIP = "10.0.0.2"

		// This is where the bug should manifest
		rv.updateTableWithPods(updatedPods)

		// Check if selection stayed on test-pod-2
		if rv.GetSelectedResourceName() != "test-pod-2" {
			t.Errorf("BUG: Selection jumped from test-pod-2 to %s after refresh",
				rv.GetSelectedResourceName())
			t.Logf("Selected row is now: %d", rv.selectedRow)
		}
		if rv.selectedRow != 1 {
			t.Errorf("BUG: Selected row jumped from 1 to %d", rv.selectedRow)
		}
	})

	t.Run("selection follows pod when list is reordered", func(t *testing.T) {
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			CurrentContext:      "test-context",
		}

		rv := NewResourceView(state, nil)
		rv.SetSize(80, 24)

		// Initial load of pods
		pods := createTestPods()
		rv.updateTableWithPods(pods)

		// Navigate to test-pod-2
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		model, _ := rv.Update(keyMsg)
		rv = model.(*ResourceView)

		if rv.GetSelectedResourceName() != "test-pod-2" {
			t.Fatalf("Expected to select test-pod-2, got %s", rv.GetSelectedResourceName())
		}

		// Reorder pods (simulating a sort by age or status change)
		// Note: updateTableWithPods will sort by NAME by default, so the order we pass doesn't matter
		// The pods will always be sorted as test-pod-1, test-pod-2, test-pod-3
		reorderedPods := []v1.Pod{pods[2], pods[0], pods[1]} // test-pod-3, test-pod-1, test-pod-2

		rv.updateTableWithPods(reorderedPods)

		// test-pod-2 should still be at index 1 (because of alphabetical sorting)
		if rv.GetSelectedResourceName() != "test-pod-2" {
			t.Errorf("BUG: Lost selection of test-pod-2, now on %s",
				rv.GetSelectedResourceName())
		}
		if rv.selectedRow != 1 {
			t.Errorf("BUG: test-pod-2 should be at row 1 after refresh (alphabetical sort), but selected row is %d",
				rv.selectedRow)
		}
	})

	t.Run("selection handles deleted pod gracefully", func(t *testing.T) {
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			CurrentContext:      "test-context",
		}

		rv := NewResourceView(state, nil)
		rv.SetSize(80, 24)

		// Initial load of pods
		pods := createTestPods()
		rv.updateTableWithPods(pods)

		// Navigate to test-pod-2
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		model, _ := rv.Update(keyMsg)
		rv = model.(*ResourceView)

		if rv.GetSelectedResourceName() != "test-pod-2" {
			t.Fatalf("Expected to select test-pod-2, got %s", rv.GetSelectedResourceName())
		}

		// Remove test-pod-2 (it was deleted)
		remainingPods := []v1.Pod{pods[0], pods[2]} // Only test-pod-1 and test-pod-3

		rv.updateTableWithPods(remainingPods)

		// Selection should stay at row 1 (which is now test-pod-3) or move to a valid row
		if rv.selectedRow >= len(rv.rows) {
			t.Errorf("BUG: Selected row %d is out of bounds (only %d rows)",
				rv.selectedRow, len(rv.rows))
		}

		// Should select something valid
		selectedName := rv.GetSelectedResourceName()
		if selectedName != "test-pod-1" && selectedName != "test-pod-3" {
			t.Errorf("BUG: After deletion, should select a remaining pod, got %s", selectedName)
		}
	})
}

// TestResourceViewSelectionBugComprehensive is the main test that demonstrates
// the selection jumping bug in various scenarios. This test should FAIL until
// the bug is fixed, then serve as a regression test.
func TestResourceViewSelectionBugComprehensive(t *testing.T) {
	// Helper to create a pod with specific attributes
	createPod := func(name, namespace, uid string, phase v1.PodPhase, ready bool, restarts int32, ageMinutes int) v1.Pod {
		return v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				UID:       types.UID(uid),
				CreationTimestamp: metav1.Time{
					Time: time.Now().Add(-time.Duration(ageMinutes) * time.Minute),
				},
			},
			Status: v1.PodStatus{
				Phase: phase,
				ContainerStatuses: []v1.ContainerStatus{
					{Ready: ready, RestartCount: restarts},
				},
				PodIP: "10.0.0." + uid[len(uid)-1:], // Use last char of UID for IP
			},
			Spec: v1.PodSpec{
				NodeName: "node-" + uid[len(uid)-1:],
			},
		}
	}

	t.Run("CRITICAL: Selection must persist through refresh cycles", func(t *testing.T) {
		// Create initial pod list that will be used as base for all scenarios
		basePods := []v1.Pod{
			createPod("pod-alpha", "default", "uid-1", v1.PodRunning, true, 0, 60),
			createPod("pod-bravo", "default", "uid-2", v1.PodRunning, true, 0, 45),
			createPod("pod-charlie", "default", "uid-3", v1.PodPending, false, 0, 30),
			createPod("pod-delta", "default", "uid-4", v1.PodRunning, true, 2, 120),
			createPod("pod-echo", "default", "uid-5", v1.PodRunning, true, 0, 90),
		}

		// Test various refresh scenarios
		refreshScenarios := []struct {
			name       string
			modifyPods func([]v1.Pod) []v1.Pod
			expectName string
			expectRow  int
		}{
			{
				name: "Status update only",
				modifyPods: func(pods []v1.Pod) []v1.Pod {
					// Make a copy to avoid modifying the original
					modifiedPods := make([]v1.Pod, len(pods))
					copy(modifiedPods, pods)
					// pod-charlie becomes Running
					for i := range modifiedPods {
						if modifiedPods[i].Name == "pod-charlie" {
							modifiedPods[i].Status.Phase = v1.PodRunning
							modifiedPods[i].Status.ContainerStatuses[0].Ready = true
							break
						}
					}
					return modifiedPods
				},
				expectName: "pod-charlie",
				expectRow:  2, // Should stay at same position after status update
			},
			{
				name: "New pod added that sorts before selected",
				modifyPods: func(pods []v1.Pod) []v1.Pod {
					newPod := createPod("pod-aaa", "default", "uid-0", v1.PodRunning, true, 0, 1)
					return append([]v1.Pod{newPod}, pods...)
				},
				expectName: "pod-charlie",
				expectRow:  3, // Should shift down by 1 due to alphabetical sorting
			},
			{
				name: "New pod added that sorts after selected",
				modifyPods: func(pods []v1.Pod) []v1.Pod {
					newPod := createPod("pod-foxtrot", "default", "uid-6", v1.PodRunning, true, 0, 1)
					return append(pods, newPod)
				},
				expectName: "pod-charlie",
				expectRow:  2, // Should stay at same position (pod-charlie is still 3rd alphabetically)
			},
		}

		for _, scenario := range refreshScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				// Create a fresh ResourceView for each scenario
				state := &core.State{
					CurrentResourceType: core.ResourceTypePod,
					CurrentNamespace:    "default",
					CurrentContext:      "test-context",
					SortAscending:       true, // Sort A-Z for predictable test results
				}

				rv := NewResourceView(state, nil)
				rv.SetSize(80, 24)

				// Load initial pods
				rv.updateTableWithPods(basePods)

				// Verify initial state
				if len(rv.rows) != 5 {
					t.Fatalf("Expected 5 rows, got %d", len(rv.rows))
				}

				// Navigate to pod-charlie (which will be at row 2 after alphabetical sorting)
				for i := 0; i < 2; i++ {
					keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
					model, _ := rv.Update(keyMsg)
					rv = model.(*ResourceView)
				}

				if rv.selectedRow != 2 {
					t.Fatalf("Expected to be at row 2, but at row %d", rv.selectedRow)
				}

				if rv.GetSelectedResourceName() != "pod-charlie" {
					t.Fatalf("Expected to select pod-charlie, got %s", rv.GetSelectedResourceName())
				}

				// Apply the scenario's modifications
				modifiedPods := scenario.modifyPods(basePods)
				rv.updateTableWithPods(modifiedPods)

				// Check if selection persisted correctly
				actualName := rv.GetSelectedResourceName()
				actualRow := rv.selectedRow

				if actualName != scenario.expectName {
					t.Errorf("After %s, selection jumped from %s to %s",
						scenario.name, scenario.expectName, actualName)
				}

				if actualRow != scenario.expectRow {
					t.Errorf("After %s, selected row should be %d but is %d",
						scenario.name, scenario.expectRow, actualRow)
				}
			})
		}
	})

	t.Run("Edge case: Rapid successive refreshes", func(t *testing.T) {
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			CurrentContext:      "test-context",
		}

		rv := NewResourceView(state, nil)
		rv.SetSize(80, 24)

		pods := []v1.Pod{
			createPod("pod-a", "default", "uid-a", v1.PodRunning, true, 0, 10),
			createPod("pod-b", "default", "uid-b", v1.PodRunning, true, 0, 10),
			createPod("pod-c", "default", "uid-c", v1.PodRunning, true, 0, 10),
		}

		rv.updateTableWithPods(pods)

		// Select pod-b
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		model, _ := rv.Update(keyMsg)
		rv = model.(*ResourceView)

		selectedBefore := rv.GetSelectedResourceName()
		if selectedBefore != "pod-b" {
			t.Fatalf("Expected to select pod-b, got %s", selectedBefore)
		}

		// Simulate rapid refreshes (like from watch events)
		for i := 0; i < 10; i++ {
			// Each refresh slightly modifies the pods
			pods[i%3].Status.ContainerStatuses[0].RestartCount++
			rv.updateTableWithPods(pods)

			// Selection should stay on pod-b
			if rv.GetSelectedResourceName() != "pod-b" {
				t.Errorf("BUG: Selection jumped away from pod-b on refresh %d to %s",
					i+1, rv.GetSelectedResourceName())
				break
			}
		}
	})
}

func TestResourceViewSelectionPersistenceAcrossContextSwitch(t *testing.T) {
	// Start with single context
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
	}

	rv := NewResourceView(state, nil)
	rv.SetSize(80, 24)

	// Add test data
	rv.headers = []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"}
	rv.rows = [][]string{
		{"test-pod-1", "1/1", "Running", "0", "5m"},
		{"test-pod-2", "0/1", "Pending", "0", "2m"},
		{"test-pod-3", "1/1", "Running", "1", "10m"},
	}

	rv.resourceMap = make(map[int]*selection.ResourceIdentity)
	for i := 0; i < 3; i++ {
		rv.resourceMap[i] = &selection.ResourceIdentity{
			Context:   "test-context",
			Namespace: "default",
			Name:      rv.rows[i][0],
			UID:       "uid-" + string(rune('1'+i)),
			Kind:      "Pod",
		}
	}

	// Select middle pod
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	model, _ := rv.Update(keyMsg)
	rv = model.(*ResourceView)

	selectedBefore := rv.GetSelectedResourceName()

	// Simulate switching to multi-context mode
	rv.isMultiContext = true
	rv.showContextColumn = true

	// Add context column to data
	for i := range rv.rows {
		rv.rows[i] = append([]string{"test-context"}, rv.rows[i]...)
	}
	rv.headers = append([]string{"CONTEXT"}, rv.headers...)

	// Try to maintain selection
	rv.restoreSelectionByIdentity()

	selectedAfter := rv.GetSelectedResourceName()

	// This will likely fail due to the bug
	if selectedBefore != selectedAfter {
		t.Logf("BUG: Selection changed from %s to %s after context mode switch",
			selectedBefore, selectedAfter)
		t.Fail() // Expected to fail
	}
}

// TestMultiContextSelectionJumpingBug reproduces the selection jumping bug
// specifically in multi-context mode where pods from different contexts are interleaved.
// This test demonstrates the bug where selection jumps to the wrong pod or to the top
// after a refresh when multiple contexts are active.
func TestMultiContextSelectionJumpingBug(t *testing.T) {
	// Helper to create a pod with context information
	createMultiContextPod := func(name, context, namespace, uid string, phase v1.PodPhase, ready bool, restarts int32, ageMinutes int) v1.Pod {
		return v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				UID:       types.UID(uid),
				CreationTimestamp: metav1.Time{
					Time: time.Now().Add(-time.Duration(ageMinutes) * time.Minute),
				},
				// Store context in annotations for multi-context scenarios
				Annotations: map[string]string{
					"context": context,
				},
			},
			Status: v1.PodStatus{
				Phase: phase,
				ContainerStatuses: []v1.ContainerStatus{
					{Ready: ready, RestartCount: restarts},
				},
				PodIP: "10.0.0." + uid[len(uid)-1:],
			},
			Spec: v1.PodSpec{
				NodeName: "node-" + context + "-" + uid[len(uid)-1:],
			},
		}
	}

	t.Run("Multi-context: Selection jumps when pods from different contexts are interleaved", func(t *testing.T) {
		// Create a multi-context state
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			CurrentContexts:     []string{"prod-cluster", "staging-cluster", "dev-cluster"},
		}

		// Create ResourceView with multi-context enabled
		rv := NewResourceViewWithMultiContext(state, nil)
		rv.SetSize(120, 30) // Wider for context column

		// Verify multi-context mode is enabled
		if !rv.isMultiContext {
			t.Fatal("ResourceView should be in multi-context mode")
		}
		if !rv.showContextColumn {
			t.Fatal("ResourceView should show context column")
		}

		// Create initial pods from different contexts
		// These will be interleaved when displayed
		// Note: We're manually setting up the table data below to simulate the exact
		// display order, as updateTableWithPods would sort and format these pods
		_ = []v1.Pod{
			createMultiContextPod("api-server-1", "prod-cluster", "default", "prod-uid-1", v1.PodRunning, true, 0, 120),
			createMultiContextPod("api-server-2", "staging-cluster", "default", "stage-uid-1", v1.PodRunning, true, 1, 90),
			createMultiContextPod("api-server-3", "dev-cluster", "default", "dev-uid-1", v1.PodPending, false, 0, 60),
			createMultiContextPod("database-1", "prod-cluster", "default", "prod-uid-2", v1.PodRunning, true, 0, 150),
			createMultiContextPod("database-2", "staging-cluster", "default", "stage-uid-2", v1.PodRunning, true, 2, 100),
			createMultiContextPod("database-3", "dev-cluster", "default", "dev-uid-2", v1.PodRunning, true, 0, 45),
			createMultiContextPod("worker-1", "prod-cluster", "default", "prod-uid-3", v1.PodRunning, true, 0, 200),
			createMultiContextPod("worker-2", "staging-cluster", "default", "stage-uid-3", v1.PodFailed, false, 5, 80),
			createMultiContextPod("worker-3", "dev-cluster", "default", "dev-uid-3", v1.PodRunning, true, 0, 30),
		}

		// Manually set up the table data to simulate what updateTableWithPods would do
		// In multi-context mode, pods are typically sorted by name then context
		rv.headers = []string{"CONTEXT", "NAME", "READY", "STATUS", "RESTARTS", "AGE", "NODE"}
		rv.rows = [][]string{
			{"prod-cluster", "api-server-1", "1/1", "Running", "0", "2h", "node-prod-cluster-1"},
			{"staging-cluster", "api-server-2", "1/1", "Running", "1", "1h30m", "node-staging-cluster-1"},
			{"dev-cluster", "api-server-3", "0/1", "Pending", "0", "1h", "node-dev-cluster-1"},
			{"prod-cluster", "database-1", "1/1", "Running", "0", "2h30m", "node-prod-cluster-2"},
			{"staging-cluster", "database-2", "1/1", "Running", "2", "1h40m", "node-staging-cluster-2"},
			{"dev-cluster", "database-3", "1/1", "Running", "0", "45m", "node-dev-cluster-2"},
			{"prod-cluster", "worker-1", "1/1", "Running", "0", "3h20m", "node-prod-cluster-3"},
			{"staging-cluster", "worker-2", "0/1", "Failed", "5", "1h20m", "node-staging-cluster-3"},
			{"dev-cluster", "worker-3", "1/1", "Running", "0", "30m", "node-dev-cluster-3"},
		}

		// Set up resource map with context information
		rv.resourceMap = make(map[int]*selection.ResourceIdentity)
		contexts := []string{"prod-cluster", "staging-cluster", "dev-cluster"}
		names := []string{"api-server-1", "api-server-2", "api-server-3", "database-1", "database-2", "database-3", "worker-1", "worker-2", "worker-3"}
		uids := []string{"prod-uid-1", "stage-uid-1", "dev-uid-1", "prod-uid-2", "stage-uid-2", "dev-uid-2", "prod-uid-3", "stage-uid-3", "dev-uid-3"}

		for i := 0; i < 9; i++ {
			rv.resourceMap[i] = &selection.ResourceIdentity{
				Context:   contexts[i%3],
				Namespace: "default",
				Name:      names[i],
				UID:       uids[i],
				Kind:      "Pod",
			}
		}

		// Navigate to a pod in the middle (database-2 from staging-cluster at row 4)
		for i := 0; i < 4; i++ {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			model, _ := rv.Update(keyMsg)
			rv = model.(*ResourceView)
		}

		// Verify we're on database-2 from staging-cluster
		if rv.selectedRow != 4 {
			t.Fatalf("Expected to be at row 4, got %d", rv.selectedRow)
		}
		if rv.GetSelectedResourceName() != "database-2" {
			t.Fatalf("Expected to select database-2, got %s", rv.GetSelectedResourceName())
		}
		if rv.selectedIdentity.Context != "staging-cluster" {
			t.Fatalf("Expected selected context to be staging-cluster, got %s", rv.selectedIdentity.Context)
		}

		// Store selection info before refresh
		selectedNameBefore := rv.GetSelectedResourceName()
		selectedContextBefore := rv.selectedIdentity.Context
		selectedUIDBefore := rv.selectedIdentity.UID

		t.Logf("Before refresh: Selected %s from %s (UID: %s) at row %d",
			selectedNameBefore, selectedContextBefore, selectedUIDBefore, rv.selectedRow)

		// Simulate a refresh that updates pod data
		// This simulates what happens when watch events come in from multiple clusters
		refreshedRows := [][]string{
			{"prod-cluster", "api-server-1", "1/1", "Running", "0", "2h1m", "node-prod-cluster-1"},
			{"staging-cluster", "api-server-2", "1/1", "Running", "1", "1h31m", "node-staging-cluster-1"},
			{"dev-cluster", "api-server-3", "1/1", "Running", "0", "1h1m", "node-dev-cluster-1"}, // Now Running
			{"prod-cluster", "database-1", "1/1", "Running", "0", "2h31m", "node-prod-cluster-2"},
			{"staging-cluster", "database-2", "1/1", "Running", "3", "1h41m", "node-staging-cluster-2"}, // Restarts increased
			{"dev-cluster", "database-3", "1/1", "Running", "0", "46m", "node-dev-cluster-2"},
			{"prod-cluster", "worker-1", "1/1", "Running", "0", "3h21m", "node-prod-cluster-3"},
			{"staging-cluster", "worker-2", "1/1", "Running", "0", "1h21m", "node-staging-cluster-3"}, // Now Running, restarts reset
			{"dev-cluster", "worker-3", "1/1", "Running", "0", "31m", "node-dev-cluster-3"},
		}

		rv.rows = refreshedRows

		// Attempt to restore selection
		rv.restoreSelectionByIdentity()

		// Check if selection stayed on database-2 from staging-cluster
		selectedNameAfter := rv.GetSelectedResourceName()
		selectedRowAfter := rv.selectedRow
		var selectedContextAfter string
		if rv.selectedIdentity != nil {
			selectedContextAfter = rv.selectedIdentity.Context
		}

		t.Logf("After refresh: Selected %s from %s at row %d",
			selectedNameAfter, selectedContextAfter, selectedRowAfter)

		// This should fail, demonstrating the bug
		if selectedNameAfter != selectedNameBefore {
			t.Errorf("BUG: Selection jumped from %s to %s after refresh",
				selectedNameBefore, selectedNameAfter)
		}
		if selectedContextAfter != selectedContextBefore {
			t.Errorf("BUG: Selected context changed from %s to %s",
				selectedContextBefore, selectedContextAfter)
		}
		if selectedRowAfter != 4 {
			t.Errorf("BUG: Selected row jumped from 4 to %d", selectedRowAfter)
		}
	})

	t.Run("Multi-context: Selection jumps when new pods are added from different contexts", func(t *testing.T) {
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			CurrentContexts:     []string{"cluster-a", "cluster-b"},
		}

		rv := NewResourceViewWithMultiContext(state, nil)
		rv.SetSize(100, 25)

		// Initial setup with pods from two clusters
		rv.headers = []string{"CONTEXT", "NAME", "READY", "STATUS", "AGE"}
		rv.rows = [][]string{
			{"cluster-a", "pod-1", "1/1", "Running", "10m"},
			{"cluster-b", "pod-2", "1/1", "Running", "8m"},
			{"cluster-a", "pod-3", "1/1", "Running", "6m"},
			{"cluster-b", "pod-4", "1/1", "Running", "4m"},
		}

		// Store original resource map for later reference
		originalMap := make(map[int]*selection.ResourceIdentity)
		originalMap[0] = &selection.ResourceIdentity{Context: "cluster-a", Namespace: "default", Name: "pod-1", UID: "a-1", Kind: "Pod"}
		originalMap[1] = &selection.ResourceIdentity{Context: "cluster-b", Namespace: "default", Name: "pod-2", UID: "b-1", Kind: "Pod"}
		originalMap[2] = &selection.ResourceIdentity{Context: "cluster-a", Namespace: "default", Name: "pod-3", UID: "a-2", Kind: "Pod"}
		originalMap[3] = &selection.ResourceIdentity{Context: "cluster-b", Namespace: "default", Name: "pod-4", UID: "b-2", Kind: "Pod"}

		rv.resourceMap = make(map[int]*selection.ResourceIdentity)
		for k, v := range originalMap {
			rv.resourceMap[k] = v
		}

		// Select pod-3 from cluster-a (row 2)
		for i := 0; i < 2; i++ {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			model, _ := rv.Update(keyMsg)
			rv = model.(*ResourceView)
		}

		if rv.GetSelectedResourceName() != "pod-3" {
			t.Fatalf("Expected to select pod-3, got %s", rv.GetSelectedResourceName())
		}

		selectedBefore := rv.selectedIdentity
		t.Logf("Before adding pods: Selected %s from %s at row %d",
			selectedBefore.Name, selectedBefore.Context, rv.selectedRow)

		// Add new pods from both clusters (simulating new deployments)
		rv.rows = [][]string{
			{"cluster-a", "pod-0", "1/1", "Running", "1m"}, // New pod that sorts first
			{"cluster-a", "pod-1", "1/1", "Running", "11m"},
			{"cluster-b", "pod-2", "1/1", "Running", "9m"},
			{"cluster-a", "pod-3", "1/1", "Running", "7m"}, // Our selected pod
			{"cluster-b", "pod-4", "1/1", "Running", "5m"},
			{"cluster-b", "pod-5", "1/1", "Running", "1m"}, // New pod from cluster-b
		}

		// Update resource map
		newMap := make(map[int]*selection.ResourceIdentity)
		newMap[0] = &selection.ResourceIdentity{Context: "cluster-a", Namespace: "default", Name: "pod-0", UID: "a-0", Kind: "Pod"}
		newMap[1] = originalMap[0] // pod-1
		newMap[2] = originalMap[1] // pod-2
		newMap[3] = originalMap[2] // pod-3 (our selected pod, now at index 3)
		newMap[4] = originalMap[3] // pod-4
		newMap[5] = &selection.ResourceIdentity{Context: "cluster-b", Namespace: "default", Name: "pod-5", UID: "b-3", Kind: "Pod"}
		rv.resourceMap = newMap

		// Try to restore selection
		rv.restoreSelectionByIdentity()

		// Check if selection stayed on pod-3
		if rv.GetSelectedResourceName() != "pod-3" {
			t.Errorf("BUG: Selection jumped from pod-3 to %s when new pods were added",
				rv.GetSelectedResourceName())
		}
		if rv.selectedRow != 3 {
			t.Errorf("BUG: pod-3 should now be at row 3, but selected row is %d",
				rv.selectedRow)
		}
	})

	t.Run("Multi-context: Selection behavior when switching between contexts rapidly", func(t *testing.T) {
		state := &core.State{
			CurrentResourceType: core.ResourceTypePod,
			CurrentNamespace:    "default",
			CurrentContexts:     []string{"context-1", "context-2", "context-3"},
		}

		rv := NewResourceViewWithMultiContext(state, nil)
		rv.SetSize(100, 25)

		// Set up initial data
		rv.headers = []string{"CONTEXT", "NAME", "STATUS"}
		initialRows := [][]string{
			{"context-1", "app-1", "Running"},
			{"context-2", "app-2", "Running"},
			{"context-3", "app-3", "Running"},
			{"context-1", "db-1", "Running"},
			{"context-2", "db-2", "Running"},
			{"context-3", "db-3", "Running"},
		}
		rv.rows = initialRows

		rv.resourceMap = make(map[int]*selection.ResourceIdentity)
		for i := 0; i < 6; i++ {
			ctx := []string{"context-1", "context-2", "context-3"}[i%3]
			name := rv.rows[i][1]
			rv.resourceMap[i] = &selection.ResourceIdentity{
				Context:   ctx,
				Namespace: "default",
				Name:      name,
				UID:       "uid-" + name,
				Kind:      "Pod",
			}
		}

		// Store original resource map for reference
		originalResourceMap := make(map[int]*selection.ResourceIdentity)
		for k, v := range rv.resourceMap {
			originalResourceMap[k] = &selection.ResourceIdentity{
				Context:   v.Context,
				Namespace: v.Namespace,
				Name:      v.Name,
				UID:       v.UID,
				Kind:      v.Kind,
			}
		}

		// Select db-2 from context-2 (row 4)
		for i := 0; i < 4; i++ {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			model, _ := rv.Update(keyMsg)
			rv = model.(*ResourceView)
		}

		originalSelection := rv.GetSelectedResourceName()
		originalContext := rv.selectedIdentity.Context

		t.Logf("Initial selection: %s from %s", originalSelection, originalContext)

		// Simulate rapid context filtering changes
		// (In real usage, this might happen when user toggles context filters)
		scenarios := []struct {
			name            string
			visibleContexts []string
			expectedRow     int
		}{
			{
				name:            "Filter to only context-2",
				visibleContexts: []string{"context-2"},
				expectedRow:     1, // db-2 should be at row 1 when only context-2 is visible
			},
			{
				name:            "Show all contexts again",
				visibleContexts: []string{"context-1", "context-2", "context-3"},
				expectedRow:     4, // db-2 should be back at row 4
			},
			{
				name:            "Filter to context-1 and context-2",
				visibleContexts: []string{"context-1", "context-2"},
				expectedRow:     3, // db-2 should be at row 3
			},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				// Simulate filtering by updating rows to only show pods from visible contexts
				var filteredRows [][]string
				var filteredMap = make(map[int]*selection.ResourceIdentity)
				idx := 0

				for i, row := range initialRows {
					ctx := row[0]
					for _, visibleCtx := range scenario.visibleContexts {
						if ctx == visibleCtx {
							filteredRows = append(filteredRows, row)
							// Copy the original resource identity
							filteredMap[idx] = &selection.ResourceIdentity{
								Context:   originalResourceMap[i].Context,
								Namespace: originalResourceMap[i].Namespace,
								Name:      originalResourceMap[i].Name,
								UID:       originalResourceMap[i].UID,
								Kind:      originalResourceMap[i].Kind,
							}
							idx++
							break
						}
					}
				}

				rv.rows = filteredRows
				rv.resourceMap = filteredMap

				// Try to restore selection
				rv.restoreSelectionByIdentity()

				// Check if selection is maintained correctly
				currentSelection := rv.GetSelectedResourceName()

				if currentSelection != originalSelection {
					t.Errorf("BUG: Selection jumped from %s to %s after filtering to %v",
						originalSelection, currentSelection, scenario.visibleContexts)
				}

				if rv.selectedRow != scenario.expectedRow {
					t.Errorf("BUG: Expected row %d after filtering, got row %d",
						scenario.expectedRow, rv.selectedRow)
				}
			})
		}
	})
}
