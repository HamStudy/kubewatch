package table

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNew(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
	}

	table := New(columns)

	if table == nil {
		t.Fatal("New() returned nil")
	}

	if len(table.columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(table.columns))
	}

	if table.showHeader != true {
		t.Error("Expected showHeader to be true by default")
	}

	if table.selectable != true {
		t.Error("Expected selectable to be true by default")
	}

	if table.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0, got %d", table.selectedIndex)
	}
}

func TestSetRows(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
	}
	table := New(columns)

	rows := []Row{
		{ID: "1", Values: []string{"pod-1", "Running"}},
		{ID: "2", Values: []string{"pod-2", "Pending"}},
		{ID: "3", Values: []string{"pod-3", "Failed"}},
	}

	table.SetRows(rows)

	if len(table.rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(table.rows))
	}

	if table.GetRowCount() != 3 {
		t.Errorf("Expected GetRowCount() to return 3, got %d", table.GetRowCount())
	}

	// Test that selected index is within bounds
	if table.selectedIndex >= len(table.rows) {
		t.Errorf("Selected index %d is out of bounds for %d rows", table.selectedIndex, len(table.rows))
	}
}

func TestSetRowsWithLargeSelection(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	// Set initial rows and select last one
	initialRows := []Row{
		{ID: "1", Values: []string{"pod-1"}},
		{ID: "2", Values: []string{"pod-2"}},
		{ID: "3", Values: []string{"pod-3"}},
		{ID: "4", Values: []string{"pod-4"}},
		{ID: "5", Values: []string{"pod-5"}},
	}
	table.SetRows(initialRows)
	table.SetSelectedIndex(4) // Select last row

	// Now set fewer rows
	newRows := []Row{
		{ID: "1", Values: []string{"pod-1"}},
		{ID: "2", Values: []string{"pod-2"}},
	}
	table.SetRows(newRows)

	// Selection should be adjusted to last available row
	if table.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be adjusted to 1, got %d", table.selectedIndex)
	}
}

func TestEmptyTable(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	// Test with no rows
	table.SetSize(80, 10)
	view := table.View()

	if view == "" {
		t.Error("Expected non-empty view even with no rows")
	}

	// Should still show header
	if !strings.Contains(view, "Name") {
		t.Error("Expected header to be visible even with no rows")
	}
}

func TestSingleRow(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
	}
	table := New(columns)

	rows := []Row{
		{ID: "1", Values: []string{"single-pod", "Running"}},
	}
	table.SetRows(rows)
	table.SetSize(40, 5)

	view := table.View()

	if !strings.Contains(view, "single-pod") {
		t.Error("Expected to see row data in view")
	}

	if !strings.Contains(view, "Running") {
		t.Error("Expected to see status in view")
	}
}

func TestLargeDataset(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
	}
	table := New(columns)

	// Create 1000 rows
	rows := make([]Row, 1000)
	for i := 0; i < 1000; i++ {
		rows[i] = Row{
			ID:     string(rune(i)),
			Values: []string{fmt.Sprintf("pod-%d", i), "Running"},
		}
	}

	table.SetRows(rows)
	table.SetSize(40, 10)

	if table.GetRowCount() != 1000 {
		t.Errorf("Expected 1000 rows, got %d", table.GetRowCount())
	}

	// Test that view renders without error
	view := table.View()
	if view == "" {
		t.Error("Expected non-empty view for large dataset")
	}

	// Test navigation through large dataset
	table.MoveToBottom()
	if table.selectedIndex != 999 {
		t.Errorf("Expected selectedIndex to be 999, got %d", table.selectedIndex)
	}

	table.MoveToTop()
	if table.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0, got %d", table.selectedIndex)
	}
}

func TestViewportCalculations(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	// Create 20 rows
	rows := make([]Row, 20)
	for i := 0; i < 20; i++ {
		rows[i] = Row{
			ID:     string(rune(i)),
			Values: []string{fmt.Sprintf("pod-%d", i)},
		}
	}
	table.SetRows(rows)

	// Set small viewport
	table.SetSize(30, 5) // 5 lines total, 1 for header, 4 for data

	// Test initial viewport
	if table.viewportSize != 4 {
		t.Errorf("Expected viewportSize to be 4, got %d", table.viewportSize)
	}

	if table.viewportStart != 0 {
		t.Errorf("Expected viewportStart to be 0, got %d", table.viewportStart)
	}

	// Move to middle
	table.SetSelectedIndex(10)

	// Viewport should adjust to show selected item
	if table.selectedIndex < table.viewportStart ||
		table.selectedIndex >= table.viewportStart+table.viewportSize {
		t.Errorf("Selected item %d not visible in viewport [%d, %d)",
			table.selectedIndex, table.viewportStart, table.viewportStart+table.viewportSize)
	}
}

func TestSelectionTracking(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	rows := []Row{
		{ID: "1", Values: []string{"pod-1"}},
		{ID: "2", Values: []string{"pod-2"}},
		{ID: "3", Values: []string{"pod-3"}},
	}
	table.SetRows(rows)

	// Test initial selection
	if table.GetSelectedIndex() != 0 {
		t.Errorf("Expected initial selection to be 0, got %d", table.GetSelectedIndex())
	}

	selectedRow := table.GetSelectedRow()
	if selectedRow == nil {
		t.Fatal("Expected GetSelectedRow() to return a row")
	}
	if selectedRow.ID != "1" {
		t.Errorf("Expected selected row ID to be '1', got '%s'", selectedRow.ID)
	}

	// Test moving selection
	table.MoveDown()
	if table.GetSelectedIndex() != 1 {
		t.Errorf("Expected selection to be 1 after MoveDown(), got %d", table.GetSelectedIndex())
	}

	table.MoveUp()
	if table.GetSelectedIndex() != 0 {
		t.Errorf("Expected selection to be 0 after MoveUp(), got %d", table.GetSelectedIndex())
	}

	// Test boundary conditions
	table.MoveUp() // Should stay at 0
	if table.GetSelectedIndex() != 0 {
		t.Errorf("Expected selection to stay at 0 when moving up from first row, got %d", table.GetSelectedIndex())
	}

	table.MoveToBottom()
	table.MoveDown() // Should stay at last row
	if table.GetSelectedIndex() != 2 {
		t.Errorf("Expected selection to stay at 2 when moving down from last row, got %d", table.GetSelectedIndex())
	}
}

func TestColumnWidthCalculations(t *testing.T) {
	tests := []struct {
		name     string
		columns  []Column
		width    int
		expected []int
	}{
		{
			name: "fixed widths",
			columns: []Column{
				{Title: "Name", Width: 20},
				{Title: "Status", Width: 10},
			},
			width:    40,
			expected: []int{20, 10},
		},
		{
			name: "flex columns",
			columns: []Column{
				{Title: "Name", Width: 20},
				{Title: "Status", Flex: true},
			},
			width:    40,
			expected: []int{20, 19}, // 40 - 20 - 1 (separator) = 19
		},
		{
			name: "min width constraints",
			columns: []Column{
				{Title: "Name", Flex: true, MinWidth: 15},
				{Title: "Status", Flex: true, MinWidth: 10},
			},
			width:    30,
			expected: []int{17, 12}, // Distributes extra space while respecting minimum widths
		},
		{
			name: "max width constraints",
			columns: []Column{
				{Title: "Name", Flex: true, MaxWidth: 10},
				{Title: "Status", Flex: true},
			},
			width:    40,
			expected: []int{10, 29}, // Name capped at 10, Status gets remainder
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := New(tt.columns)
			table.SetSize(tt.width, 10)

			if len(table.columnWidths) != len(tt.expected) {
				t.Fatalf("Expected %d column widths, got %d", len(tt.expected), len(table.columnWidths))
			}

			for i, expected := range tt.expected {
				if table.columnWidths[i] != expected {
					t.Errorf("Column %d: expected width %d, got %d", i, expected, table.columnWidths[i])
				}
			}
		})
	}
}

func TestWordWrap(t *testing.T) {
	columns := []Column{
		{Title: "Description", Width: 10},
	}
	table := New(columns)

	rows := []Row{
		{ID: "1", Values: []string{"This is a very long description that should be wrapped"}},
	}
	table.SetRows(rows)
	table.SetSize(20, 5)

	// Test without word wrap
	table.SetWordWrap(false)
	view := table.View()

	// Should be truncated
	if strings.Contains(view, "This is a very long description that should be wrapped") {
		t.Error("Expected long text to be truncated when word wrap is disabled")
	}

	// Test with word wrap
	table.SetWordWrap(true)
	view = table.View()

	// Should show wrapped content (at least first part)
	if !strings.Contains(view, "This is a") {
		t.Error("Expected to see beginning of wrapped text")
	}
}

func TestFocusAndSelection(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	rows := []Row{
		{ID: "1", Values: []string{"pod-1"}},
		{ID: "2", Values: []string{"pod-2"}},
	}
	table.SetRows(rows)
	table.SetSize(30, 5)

	// Test focus state
	if table.IsFocused() {
		t.Error("Expected table to not be focused initially")
	}

	table.Focus()
	if !table.IsFocused() {
		t.Error("Expected table to be focused after Focus()")
	}

	table.Blur()
	if table.IsFocused() {
		t.Error("Expected table to not be focused after Blur()")
	}
}

func TestPageNavigation(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	// Create 20 rows
	rows := make([]Row, 20)
	for i := 0; i < 20; i++ {
		rows[i] = Row{
			ID:     string(rune(i)),
			Values: []string{fmt.Sprintf("pod-%d", i)},
		}
	}
	table.SetRows(rows)
	table.SetSize(30, 6) // 5 visible rows

	// Test page down
	initialIndex := table.GetSelectedIndex()
	table.PageDown()

	if table.GetSelectedIndex() <= initialIndex {
		t.Error("Expected selection to move down after PageDown()")
	}

	// Test page up
	currentIndex := table.GetSelectedIndex()
	table.PageUp()

	if table.GetSelectedIndex() >= currentIndex {
		t.Error("Expected selection to move up after PageUp()")
	}
}

func TestAddRow(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	// Start with empty table
	if table.GetRowCount() != 0 {
		t.Errorf("Expected empty table, got %d rows", table.GetRowCount())
	}

	// Add a row
	row := Row{ID: "1", Values: []string{"pod-1"}}
	table.AddRow(row)

	if table.GetRowCount() != 1 {
		t.Errorf("Expected 1 row after AddRow(), got %d", table.GetRowCount())
	}

	// Add another row
	row2 := Row{ID: "2", Values: []string{"pod-2"}}
	table.AddRow(row2)

	if table.GetRowCount() != 2 {
		t.Errorf("Expected 2 rows after second AddRow(), got %d", table.GetRowCount())
	}
}

func TestClearRows(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	// Add some rows
	rows := []Row{
		{ID: "1", Values: []string{"pod-1"}},
		{ID: "2", Values: []string{"pod-2"}},
	}
	table.SetRows(rows)

	if table.GetRowCount() != 2 {
		t.Errorf("Expected 2 rows before clear, got %d", table.GetRowCount())
	}

	// Clear rows
	table.ClearRows()

	if table.GetRowCount() != 0 {
		t.Errorf("Expected 0 rows after ClearRows(), got %d", table.GetRowCount())
	}

	if table.GetSelectedIndex() != 0 {
		t.Errorf("Expected selectedIndex to be reset to 0, got %d", table.GetSelectedIndex())
	}
}

func TestBasicCustomStyles(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	// Set custom styles
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("2"))
	rowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	alternateStyle := lipgloss.NewStyle().Background(lipgloss.Color("4"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	table.SetStyles(headerStyle, selectedStyle, rowStyle, alternateStyle, borderStyle)

	// Add some rows and render
	rows := []Row{
		{ID: "1", Values: []string{"pod-1"}},
		{ID: "2", Values: []string{"pod-2"}},
	}
	table.SetRows(rows)
	table.SetSize(30, 5)
	table.Focus()

	view := table.View()

	// Should render without error
	if view == "" {
		t.Error("Expected non-empty view with custom styles")
	}
}

func TestRowWithCustomStyle(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	// Create row with custom style
	customStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	rows := []Row{
		{ID: "1", Values: []string{"pod-1"}},
		{ID: "2", Values: []string{"pod-2"}, Style: customStyle},
	}
	table.SetRows(rows)
	table.SetSize(30, 5)

	view := table.View()

	// Should render without error
	if view == "" {
		t.Error("Expected non-empty view with custom row style")
	}
}

func TestAlignment(t *testing.T) {
	columns := []Column{
		{Title: "Left", Width: 10, Align: lipgloss.Left},
		{Title: "Center", Width: 10, Align: lipgloss.Center},
		{Title: "Right", Width: 10, Align: lipgloss.Right},
	}
	table := New(columns)

	rows := []Row{
		{ID: "1", Values: []string{"L", "C", "R"}},
	}
	table.SetRows(rows)
	table.SetSize(40, 5)

	view := table.View()

	// Should render without error
	if view == "" {
		t.Error("Expected non-empty view with different alignments")
	}

	// Should contain all values
	if !strings.Contains(view, "L") || !strings.Contains(view, "C") || !strings.Contains(view, "R") {
		t.Error("Expected to see all alignment test values")
	}
}

func TestTruncation(t *testing.T) {
	columns := []Column{
		{Title: "End", Width: 5, TruncateAt: "end"},
		{Title: "Middle", Width: 5, TruncateAt: "middle"},
		{Title: "Start", Width: 5, TruncateAt: "start"},
	}
	table := New(columns)

	rows := []Row{
		{ID: "1", Values: []string{"VeryLongText", "VeryLongText", "VeryLongText"}},
	}
	table.SetRows(rows)
	table.SetSize(20, 5)

	view := table.View()

	// Should render without error
	if view == "" {
		t.Error("Expected non-empty view with truncation")
	}

	// Should contain ellipsis character
	if !strings.Contains(view, "…") {
		t.Error("Expected to see truncation ellipsis")
	}
}

func BenchmarkTableRender(b *testing.B) {
	columns := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
		{Title: "Age", Width: 8},
	}
	table := New(columns)

	// Create 100 rows
	rows := make([]Row, 100)
	for i := 0; i < 100; i++ {
		rows[i] = Row{
			ID:     string(rune(i)),
			Values: []string{fmt.Sprintf("pod-%d", i), "Running", "5m"},
		}
	}
	table.SetRows(rows)
	table.SetSize(50, 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.View()
	}
}

func BenchmarkLargeTableRender(b *testing.B) {
	columns := []Column{
		{Title: "Name", Width: 30},
		{Title: "Status", Width: 15},
		{Title: "Age", Width: 10},
		{Title: "CPU", Width: 10},
		{Title: "Memory", Width: 10},
	}
	table := New(columns)

	// Create 1000 rows
	rows := make([]Row, 1000)
	for i := 0; i < 1000; i++ {
		rows[i] = Row{
			ID: string(rune(i)),
			Values: []string{
				fmt.Sprintf("very-long-pod-name-%d", i),
				"Running",
				"5m30s",
				"250m",
				"512Mi",
			},
		}
	}
	table.SetRows(rows)
	table.SetSize(100, 25)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.View()
	}
}

func BenchmarkTableNavigation(b *testing.B) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}
	table := New(columns)

	// Create 1000 rows
	rows := make([]Row, 1000)
	for i := 0; i < 1000; i++ {
		rows[i] = Row{
			ID:     string(rune(i)),
			Values: []string{fmt.Sprintf("pod-%d", i)},
		}
	}
	table.SetRows(rows)
	table.SetSize(30, 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table.MoveDown()
		if table.GetSelectedIndex() >= 999 {
			table.MoveToTop()
		}
	}
}
