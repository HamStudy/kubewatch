package views

import (
	"strings"
	"testing"

	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/k8s"
	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestBug002ColumnMisalignmentSingleContext tests the column misalignment bug
// where headers remove CONTEXT column but body still shows context data
func TestBug002ColumnMisalignmentSingleContext(t *testing.T) {
	tests := []struct {
		name                 string
		isMultiContext       bool
		currentContexts      []string
		expectedHeaders      []string
		expectedRowStructure []string // What the first row should look like
	}{
		{
			name:                 "single context mode - no context column",
			isMultiContext:       false,
			currentContexts:      []string{"single-context"},
			expectedHeaders:      []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE", "CPU", "MEMORY", "IP", "NODE"},
			expectedRowStructure: []string{"test-pod-1", "1/1", "Running", "0", "5m", "-", "-", "-", "-"},
		},
		{
			name:                 "multi context mode with single context - no context column",
			isMultiContext:       true,
			currentContexts:      []string{"single-context"},
			expectedHeaders:      []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE", "CPU", "MEMORY", "IP", "NODE"},
			expectedRowStructure: []string{"test-pod-1", "1/1", "Running", "0", "5m", "-", "-", "-", "-"},
		},
		{
			name:                 "multi context mode with multiple contexts - has context column",
			isMultiContext:       true,
			currentContexts:      []string{"context-1", "context-2"},
			expectedHeaders:      []string{"CONTEXT", "NAME", "READY", "STATUS", "RESTARTS", "AGE", "CPU", "MEMORY", "IP", "NODE"},
			expectedRowStructure: []string{"context-1", "test-pod-1", "1/1", "Running", "0", "5m", "-", "-", "-", "-"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create state
			state := &core.State{
				CurrentResourceType: core.ResourceTypePod,
				CurrentNamespace:    "default",
				CurrentContexts:     tt.currentContexts,
				SortAscending:       true,
				SortColumn:          "NAME",
			}

			// Create resource view
			var rv *ResourceView
			if tt.isMultiContext {
				rv = NewResourceViewWithMultiContext(state, nil)
			} else {
				rv = NewResourceView(state, nil)
			}
			rv.SetSize(120, 24)

			// Create test pod
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-pod-1",
					Namespace:         "default",
					UID:               types.UID("uid-1"),
					CreationTimestamp: metav1.Now(),
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true, RestartCount: 0},
					},
				},
			}

			// Update table data
			if tt.isMultiContext && len(tt.currentContexts) > 1 {
				// Multi-context with multiple contexts
				podsWithContext := []k8s.PodWithContext{
					{Pod: pod, Context: "context-1"},
				}
				// Update state first (like RefreshResources does)
				allPods := []v1.Pod{pod}
				rv.state.UpdatePods(allPods)
				rv.updateTableWithPodsMultiContext(podsWithContext)
			} else {
				// Single context or multi-context with single context
				pods := []v1.Pod{pod}
				// Update state first (like RefreshResources does)
				rv.state.UpdatePods(pods)
				rv.updateTableWithPods(pods)
			}

			// Debug: Print current state
			t.Logf("isMultiContext: %v, showContextColumn: %v, currentContexts: %v",
				rv.isMultiContext, rv.showContextColumn, tt.currentContexts)
			t.Logf("Headers: %v", rv.headers)
			if len(rv.rows) > 0 {
				t.Logf("First row: %v", rv.rows[0])
			}
			t.Logf("Viewport: start=%d, height=%d, selectedRow=%d, totalRows=%d",
				rv.viewportStart, rv.viewportHeight, rv.selectedRow, len(rv.rows))
			t.Logf("Column widths: %v", rv.columnWidths)
			t.Logf("View width: %d, height: %d", rv.width, rv.height) // Check headers
			if len(rv.headers) != len(tt.expectedHeaders) {
				t.Errorf("Expected %d headers, got %d", len(tt.expectedHeaders), len(rv.headers))
				t.Errorf("Expected headers: %v", tt.expectedHeaders)
				t.Errorf("Actual headers: %v", rv.headers)
			}

			for i, expectedHeader := range tt.expectedHeaders {
				if i >= len(rv.headers) {
					t.Errorf("Missing header at index %d: expected %s", i, expectedHeader)
					continue
				}
				if rv.headers[i] != expectedHeader {
					t.Errorf("Header mismatch at index %d: expected %s, got %s", i, expectedHeader, rv.headers[i])
				}
			}

			// Check row structure
			if len(rv.rows) == 0 {
				t.Fatal("No rows found")
			}

			firstRow := rv.rows[0]
			if len(firstRow) != len(tt.expectedRowStructure) {
				t.Errorf("Expected row to have %d columns, got %d", len(tt.expectedRowStructure), len(firstRow))
				t.Errorf("Expected row structure: %v", tt.expectedRowStructure)
				t.Errorf("Actual row: %v", firstRow)
			}

			// Check that headers and rows have the same number of columns
			if len(rv.headers) != len(firstRow) {
				t.Errorf("Column count mismatch: headers have %d columns, row has %d columns", len(rv.headers), len(firstRow))
				t.Errorf("Headers: %v", rv.headers)
				t.Errorf("Row: %v", firstRow)
			}

			// Verify the rendered view doesn't have misaligned columns
			view := rv.View()
			lines := strings.Split(view, "\n")

			// Debug: Print the rendered view
			t.Logf("Rendered view:\n%s", view)

			// Find header line and first data line
			var headerLine, dataLine string
			for i, line := range lines {
				t.Logf("Line %d: %s", i, line)
				if strings.Contains(line, "NAME") && strings.Contains(line, "STATUS") {
					headerLine = line
					t.Logf("Found header line at %d: %s", i, line)
				} else if strings.Contains(line, "test-pod-1") {
					dataLine = line
					t.Logf("Found data line at %d: %s", i, line)
					break
				}
			}

			if headerLine == "" {
				t.Error("Could not find header line in rendered view")
			}
			if dataLine == "" {
				t.Error("Could not find data line in rendered view")
			}
			// Basic alignment check - the NAME column should align
			if headerLine != "" && dataLine != "" {
				nameHeaderPos := strings.Index(headerLine, "NAME")
				namePodPos := strings.Index(dataLine, "test-pod-1")

				// Allow some tolerance for styling and spacing
				if nameHeaderPos >= 0 && namePodPos >= 0 {
					diff := abs(nameHeaderPos - namePodPos)
					if diff > 5 { // Allow 5 character tolerance for styling
						t.Errorf("Column alignment issue: NAME header at position %d, pod name at position %d (diff: %d)",
							nameHeaderPos, namePodPos, diff)
						t.Errorf("Header line: %s", headerLine)
						t.Errorf("Data line:   %s", dataLine)
					}
				}
			}
		})
	}
}

// TestBug006WordWrapBroken tests the word wrap toggle functionality
func TestBug006WordWrapBroken(t *testing.T) {
	// Create resource view with test data
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
		SortAscending:       true,
		SortColumn:          "NAME",
	}

	rv := NewResourceView(state, nil)
	rv.SetSize(80, 24)

	// Create test data with long content that should be wrapped/truncated
	longPodName := "very-long-pod-name-that-should-be-wrapped-or-truncated-based-on-setting"
	longStatus := "ContainerCreatingWithVeryLongReasonThatShouldBeHandledProperly"

	rv.headers = []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"}
	rv.rows = [][]string{
		{longPodName, "1/1", longStatus, "0", "5m"},
		{"short-pod", "1/1", "Running", "0", "2m"},
	}

	// Set up resource map
	rv.resourceMap = make(map[int]*selection.ResourceIdentity)
	rv.resourceMap[0] = &selection.ResourceIdentity{
		Context:   "test-context",
		Namespace: "default",
		Name:      longPodName,
		UID:       "uid-1",
		Kind:      "Pod",
	}

	rv.selectedRow = 0
	rv.selectedIdentity = rv.resourceMap[0]

	tests := []struct {
		name              string
		wordWrap          bool
		expectedBehavior  string
		checkContent      string
		shouldBeTruncated bool
	}{
		{
			name:              "word wrap OFF - should show full content",
			wordWrap:          false,
			expectedBehavior:  "full content visible",
			checkContent:      longPodName,
			shouldBeTruncated: false,
		},
		{
			name:              "word wrap ON - should truncate content",
			wordWrap:          true,
			expectedBehavior:  "content truncated",
			checkContent:      longPodName,
			shouldBeTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set word wrap mode directly
			rv.wordWrap = tt.wordWrap

			// Verify word wrap state is set correctly
			if rv.wordWrap != tt.wordWrap {
				t.Errorf("Expected wordWrap to be %v, got %v", tt.wordWrap, rv.wordWrap)
			}
			// Render the view
			view := rv.View()

			// Check if content is handled according to word wrap setting
			if tt.shouldBeTruncated {
				// When word wrap is ON, long content should be truncated
				if strings.Contains(view, tt.checkContent) {
					// Check if it's actually truncated (contains ellipsis or is shortened)
					lines := strings.Split(view, "\n")
					found := false
					for _, line := range lines {
						if strings.Contains(line, tt.checkContent[:10]) { // Check first part
							if !strings.Contains(line, "...") && !strings.Contains(line, "…") {
								// Full content is shown, but it should be truncated
								if len(line) > 100 { // Arbitrary long line threshold
									t.Errorf("Word wrap ON but content not truncated. Line length: %d", len(line))
									t.Errorf("Line: %s", line)
								}
							}
							found = true
							break
						}
					}
					if !found {
						t.Error("Could not find the test content in the rendered view")
					}
				}
			} else {
				// When word wrap is OFF, full content should be visible
				if !strings.Contains(view, tt.checkContent) {
					t.Errorf("Word wrap OFF but full content not visible")
					t.Errorf("Expected to find: %s", tt.checkContent)
					t.Errorf("In view: %s", view)
				}
			}

			// Test the styleCellByColumn method directly
			testCases := []struct {
				columnName string
				value      string
				width      int
			}{
				{"NAME", longPodName, 20},
				{"STATUS", longStatus, 15},
			}

			for _, tc := range testCases {
				styledCell := rv.styleCellByColumn(tc.columnName, tc.value, tc.width, false)

				if tt.wordWrap {
					// When word wrap is ON, content should be truncated to fit width
					if len(styledCell) > tc.width+10 { // Allow some tolerance for styling
						t.Errorf("Word wrap ON but cell not truncated. Column: %s, Width: %d, Styled length: %d",
							tc.columnName, tc.width, len(styledCell))
					}
				} else {
					// When word wrap is OFF, content should not be truncated
					if !strings.Contains(styledCell, tc.value) && len(tc.value) <= tc.width {
						t.Errorf("Word wrap OFF but short content truncated. Column: %s", tc.columnName)
					}
				}
			}
		})
	}
}

// TestWordWrapToggleKeyBinding tests that the 'u' key properly toggles word wrap
func TestWordWrapToggleKeyBinding(t *testing.T) {
	rv := createTestResourceViewWithData(t)

	initialWrapState := rv.wordWrap

	// Press 'u' to toggle word wrap
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")}
	model, _ := rv.Update(keyMsg)
	rv = model.(*ResourceView)

	// Verify state changed
	if rv.wordWrap == initialWrapState {
		t.Errorf("Word wrap state should have changed from %v", initialWrapState)
	}

	// Press 'u' again to toggle back
	model, _ = rv.Update(keyMsg)
	rv = model.(*ResourceView)

	// Verify state changed back
	if rv.wordWrap != initialWrapState {
		t.Errorf("Word wrap state should have returned to %v, got %v", initialWrapState, rv.wordWrap)
	}
}

// Helper function to calculate absolute difference
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Helper function to check if text contains content (case insensitive)
func containsTextIgnoreCase(text, content string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(content))
}
