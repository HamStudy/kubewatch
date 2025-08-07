package table

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestEnhancedTableCreation(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20, Flex: false},
		{Title: "Status", Width: 10, Flex: false},
		{Title: "Age", Width: 0, Flex: true},
	}

	enhanced := NewEnhanced(columns)

	if enhanced == nil {
		t.Fatal("Expected enhanced table to be created")
	}

	if len(enhanced.columnOrder) != len(columns) {
		t.Errorf("Expected column order length %d, got %d", len(columns), len(enhanced.columnOrder))
	}

	if !enhanced.resizableColumns {
		t.Error("Expected resizable columns to be enabled by default")
	}

	if enhanced.theme == nil {
		t.Error("Expected default theme to be set")
	}
}

func TestColumnVisibility(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
		{Title: "Age", Width: 15},
	}

	enhanced := NewEnhanced(columns)

	// Test initial visibility
	for i := range columns {
		if !enhanced.IsColumnVisible(i) {
			t.Errorf("Expected column %d to be visible initially", i)
		}
	}

	// Test hiding a column
	enhanced.SetColumnVisibility(1, false)
	if enhanced.IsColumnVisible(1) {
		t.Error("Expected column 1 to be hidden")
	}

	// Test showing a column
	enhanced.SetColumnVisibility(1, true)
	if !enhanced.IsColumnVisible(1) {
		t.Error("Expected column 1 to be visible")
	}

	// Test toggle
	enhanced.ToggleColumnVisibility(2)
	if enhanced.IsColumnVisible(2) {
		t.Error("Expected column 2 to be hidden after toggle")
	}

	enhanced.ToggleColumnVisibility(2)
	if !enhanced.IsColumnVisible(2) {
		t.Error("Expected column 2 to be visible after second toggle")
	}
}

func TestColumnReordering(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
		{Title: "Age", Width: 15},
	}

	enhanced := NewEnhanced(columns)

	// Test initial order
	order := enhanced.GetColumnOrder()
	expected := []int{0, 1, 2}
	for i, col := range order {
		if col != expected[i] {
			t.Errorf("Expected initial order %v, got %v", expected, order)
			break
		}
	}

	// Test reordering
	enhanced.ReorderColumns(0, 2) // Move first column to position 2
	order = enhanced.GetColumnOrder()
	expected = []int{1, 0, 2}
	for i, col := range order {
		if col != expected[i] {
			t.Errorf("Expected reordered order %v, got %v", expected, order)
			break
		}
	}

	// Test invalid reordering
	enhanced.ReorderColumns(-1, 1) // Should be ignored
	enhanced.ReorderColumns(0, 5)  // Should be ignored
	order = enhanced.GetColumnOrder()
	for i, col := range order {
		if col != expected[i] {
			t.Errorf("Expected order to remain %v after invalid reorder, got %v", expected, order)
			break
		}
	}
}

func TestColumnResizing(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
	}

	enhanced := NewEnhanced(columns)

	// Test resizing
	enhanced.ResizeColumn(0, 30)
	if enhanced.columns[0].Width != 30 {
		t.Errorf("Expected column 0 width to be 30, got %d", enhanced.columns[0].Width)
	}

	// Test min/max constraints
	enhanced.ResizeColumn(0, 1) // Below minimum
	if enhanced.columns[0].Width != enhanced.minColumnWidth {
		t.Errorf("Expected column 0 width to be constrained to minimum %d, got %d",
			enhanced.minColumnWidth, enhanced.columns[0].Width)
	}

	enhanced.ResizeColumn(0, 200) // Above maximum
	if enhanced.columns[0].Width != enhanced.maxColumnWidth {
		t.Errorf("Expected column 0 width to be constrained to maximum %d, got %d",
			enhanced.maxColumnWidth, enhanced.columns[0].Width)
	}
}

func TestSelectionModes(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}

	enhanced := NewEnhanced(columns)

	// Add some test rows
	rows := []Row{
		{ID: "1", Values: []string{"Row 1"}},
		{ID: "2", Values: []string{"Row 2"}},
		{ID: "3", Values: []string{"Row 3"}},
	}
	enhanced.SetRows(rows)

	// Test single selection (default)
	enhanced.SelectRow(0, true)
	enhanced.SelectRow(1, true)

	selected := enhanced.GetSelectedRows()
	if len(selected) != 1 || selected[0] != 1 {
		t.Errorf("Expected single selection [1], got %v", selected)
	}

	// Test multi-selection
	enhanced.ClearSelection()
	enhanced.SetSelectionMode(MultiSelection)
	enhanced.SelectRow(0, true)
	enhanced.SelectRow(2, true)

	selected = enhanced.GetSelectedRows()
	if len(selected) != 2 {
		t.Errorf("Expected 2 selected rows, got %d", len(selected))
	}

	// Test range selection
	enhanced.ClearSelection()
	enhanced.SetSelectionMode(RangeSelection)
	enhanced.SelectRange(0, 2)

	selected = enhanced.GetSelectedRows()
	if len(selected) != 3 {
		t.Errorf("Expected 3 selected rows in range, got %d", len(selected))
	}

	// Test no selection
	enhanced.SetSelectionMode(NoSelection)
	enhanced.SelectRow(0, true) // Should be ignored

	selected = enhanced.GetSelectedRows()
	if len(selected) != 0 {
		t.Errorf("Expected no selected rows with NoSelection mode, got %d", len(selected))
	}
}

func TestThemes(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}

	enhanced := NewEnhanced(columns)

	// Test default theme
	if enhanced.theme.Name != "default" {
		t.Errorf("Expected default theme name 'default', got '%s'", enhanced.theme.Name)
	}

	// Test dark theme
	darkTheme := DarkTheme()
	enhanced.SetTheme(darkTheme)
	if enhanced.theme.Name != "dark" {
		t.Errorf("Expected theme name 'dark', got '%s'", enhanced.theme.Name)
	}

	// Test light theme
	lightTheme := LightTheme()
	enhanced.SetTheme(lightTheme)
	if enhanced.theme.Name != "light" {
		t.Errorf("Expected theme name 'light', got '%s'", enhanced.theme.Name)
	}
}

func TestAnimations(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}

	enhanced := NewEnhanced(columns)

	// Test animation settings
	if !enhanced.animations {
		t.Error("Expected animations to be enabled by default")
	}

	enhanced.EnableAnimations(false)
	if enhanced.animations {
		t.Error("Expected animations to be disabled")
	}

	// Test transition speed
	newSpeed := 500 * time.Millisecond
	enhanced.SetTransitionSpeed(newSpeed)
	if enhanced.transitionSpeed != newSpeed {
		t.Errorf("Expected transition speed %v, got %v", newSpeed, enhanced.transitionSpeed)
	}
}

func TestVirtualization(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}

	enhanced := NewEnhanced(columns)

	// Test virtualization settings
	if !enhanced.virtualization {
		t.Error("Expected virtualization to be enabled by default")
	}

	enhanced.EnableVirtualization(false)
	if enhanced.virtualization {
		t.Error("Expected virtualization to be disabled")
	}
}

func TestRenderDebouncing(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}

	enhanced := NewEnhanced(columns)

	// Test initial debounce setting
	if enhanced.renderDebounce != 16*time.Millisecond {
		t.Errorf("Expected default render debounce 16ms, got %v", enhanced.renderDebounce)
	}

	// Test should render initially
	if !enhanced.ShouldRender() {
		t.Error("Expected initial render to be allowed")
	}

	// Mark as rendered
	enhanced.MarkRendered()

	// Should not render immediately after
	if enhanced.ShouldRender() {
		t.Error("Expected render to be debounced immediately after marking rendered")
	}

	// Wait for debounce period
	time.Sleep(enhanced.renderDebounce + time.Millisecond)

	// Should render after debounce period
	if !enhanced.ShouldRender() {
		t.Error("Expected render to be allowed after debounce period")
	}
}

func TestEnhancedCustomStyles(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}

	enhanced := NewEnhanced(columns)

	// Test setting custom style
	customStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
	enhanced.SetCustomStyle("test", customStyle)

	// Test getting custom style
	retrievedStyle, exists := enhanced.GetCustomStyle("test")
	if !exists {
		t.Error("Expected custom style to exist")
	}

	if retrievedStyle.String() != customStyle.String() {
		t.Error("Expected retrieved style to match set style")
	}

	// Test non-existent style
	_, exists = enhanced.GetCustomStyle("nonexistent")
	if exists {
		t.Error("Expected non-existent style to not exist")
	}
}

func TestPerformanceStats(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}

	enhanced := NewEnhanced(columns)

	// Add some test data
	rows := []Row{
		{ID: "1", Values: []string{"Row 1"}},
		{ID: "2", Values: []string{"Row 2"}},
	}
	enhanced.SetRows(rows)

	// Hide a column
	enhanced.SetColumnVisibility(0, false)

	// Select a row
	enhanced.SelectRow(0, true)

	stats := enhanced.GetPerformanceStats()

	if stats["total_rows"] != 2 {
		t.Errorf("Expected total_rows to be 2, got %v", stats["total_rows"])
	}

	if stats["selected_rows"] != 1 {
		t.Errorf("Expected selected_rows to be 1, got %v", stats["selected_rows"])
	}

	if stats["hidden_columns"] != 1 {
		t.Errorf("Expected hidden_columns to be 1, got %v", stats["hidden_columns"])
	}

	if stats["virtualization"] != true {
		t.Errorf("Expected virtualization to be true, got %v", stats["virtualization"])
	}

	if stats["animations"] != true {
		t.Errorf("Expected animations to be true, got %v", stats["animations"])
	}
}

func TestRowAnimations(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
	}

	enhanced := NewEnhanced(columns)

	// Test adding row with animation
	row := Row{ID: "1", Values: []string{"Test Row"}}
	enhanced.AddRowWithAnimation(row)

	if len(enhanced.rows) != 1 {
		t.Errorf("Expected 1 row after adding, got %d", len(enhanced.rows))
	}

	// Check if fade-in animation was registered
	if len(enhanced.fadeInRows) != 1 {
		t.Errorf("Expected 1 fade-in animation, got %d", len(enhanced.fadeInRows))
	}

	// Test removing row with animation (without actually waiting for the animation)
	enhanced.EnableAnimations(false) // Disable animations for immediate removal
	enhanced.RemoveRowWithAnimation(0)

	if len(enhanced.rows) != 0 {
		t.Errorf("Expected 0 rows after removal, got %d", len(enhanced.rows))
	}
}

func TestGetVisibleColumns(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
		{Title: "Age", Width: 15},
	}

	enhanced := NewEnhanced(columns)

	// Test all visible initially
	visible := enhanced.GetVisibleColumns()
	expected := []int{0, 1, 2}
	for i, col := range visible {
		if col != expected[i] {
			t.Errorf("Expected visible columns %v, got %v", expected, visible)
			break
		}
	}

	// Hide middle column
	enhanced.SetColumnVisibility(1, false)
	visible = enhanced.GetVisibleColumns()
	expected = []int{0, 2}
	for i, col := range visible {
		if col != expected[i] {
			t.Errorf("Expected visible columns %v after hiding column 1, got %v", expected, visible)
			break
		}
	}

	// Reorder columns and test visibility
	enhanced.ReorderColumns(0, 1) // Move first to second position
	visible = enhanced.GetVisibleColumns()
	expected = []int{0, 2} // Column 1 is still hidden, order should be [1, 0, 2] but 1 is hidden
	for i, col := range visible {
		if col != expected[i] {
			t.Errorf("Expected visible columns %v after reordering, got %v", expected, visible)
			break
		}
	}
}
