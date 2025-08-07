package table

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// EnhancedModel extends the basic table with advanced features
type EnhancedModel struct {
	*Model

	// Column management
	columnOrder      []int        // Custom column order
	hiddenColumns    map[int]bool // Hidden columns
	resizableColumns bool         // Allow column resizing
	minColumnWidth   int          // Minimum column width
	maxColumnWidth   int          // Maximum column width

	// Themes and styling
	theme           *TableTheme
	customStyles    map[string]lipgloss.Style
	rowHighlighting bool
	alternateRows   bool

	// Performance optimizations
	virtualization bool
	renderBuffer   []string
	lastRenderTime time.Time
	renderDebounce time.Duration

	// Interactive features
	columnResizing bool
	resizeColumn   int
	resizeStartX   int
	dragStartX     int
	dragColumn     int

	// Selection enhancements
	multiSelect   bool
	selectedRows  map[int]bool
	selectionMode SelectionMode

	// Animation and transitions
	animations      bool
	transitionSpeed time.Duration
	fadeInRows      map[int]time.Time
	fadeOutRows     map[int]time.Time
}

// TableTheme defines styling for different table elements
type TableTheme struct {
	Name           string
	HeaderStyle    lipgloss.Style
	RowStyle       lipgloss.Style
	AlternateStyle lipgloss.Style
	SelectedStyle  lipgloss.Style
	BorderStyle    lipgloss.Style
	FocusedStyle   lipgloss.Style
	HighlightStyle lipgloss.Style

	// Color scheme
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Background lipgloss.Color
	Foreground lipgloss.Color

	// Status colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color
}

// SelectionMode defines how selection works
type SelectionMode int

const (
	SingleSelection SelectionMode = iota
	MultiSelection
	RangeSelection
	NoSelection
)

// ColumnResizeEvent represents a column resize event
type ColumnResizeEvent struct {
	Column   int
	OldWidth int
	NewWidth int
}

// ColumnReorderEvent represents a column reorder event
type ColumnReorderEvent struct {
	FromIndex int
	ToIndex   int
}

// NewEnhanced creates a new enhanced table model
func NewEnhanced(columns []Column) *EnhancedModel {
	base := New(columns)

	enhanced := &EnhancedModel{
		Model:            base,
		columnOrder:      make([]int, len(columns)),
		hiddenColumns:    make(map[int]bool),
		resizableColumns: true,
		minColumnWidth:   5,
		maxColumnWidth:   100,
		theme:            DefaultTheme(),
		customStyles:     make(map[string]lipgloss.Style),
		rowHighlighting:  true,
		alternateRows:    true,
		virtualization:   true,
		renderDebounce:   16 * time.Millisecond, // ~60fps
		multiSelect:      false,
		selectedRows:     make(map[int]bool),
		selectionMode:    SingleSelection,
		animations:       true,
		transitionSpeed:  200 * time.Millisecond,
		fadeInRows:       make(map[int]time.Time),
		fadeOutRows:      make(map[int]time.Time),
	}

	// Initialize column order
	for i := range columns {
		enhanced.columnOrder[i] = i
	}

	return enhanced
}

// DefaultTheme returns the default table theme
func DefaultTheme() *TableTheme {
	return &TableTheme{
		Name:           "default",
		HeaderStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
		RowStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		AlternateStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("235")),
		SelectedStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("33")),
		BorderStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		FocusedStyle:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("33")),
		HighlightStyle: lipgloss.NewStyle().Background(lipgloss.Color("220")).Foreground(lipgloss.Color("0")),

		Primary:    lipgloss.Color("33"),
		Secondary:  lipgloss.Color("240"),
		Accent:     lipgloss.Color("220"),
		Background: lipgloss.Color("0"),
		Foreground: lipgloss.Color("252"),

		Success: lipgloss.Color("46"),
		Warning: lipgloss.Color("226"),
		Error:   lipgloss.Color("196"),
		Info:    lipgloss.Color("39"),
	}
}

// DarkTheme returns a dark theme
func DarkTheme() *TableTheme {
	theme := DefaultTheme()
	theme.Name = "dark"
	theme.RowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	theme.AlternateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("234"))
	theme.Background = lipgloss.Color("0")
	return theme
}

// LightTheme returns a light theme
func LightTheme() *TableTheme {
	theme := DefaultTheme()
	theme.Name = "light"
	theme.HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0"))
	theme.RowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0"))
	theme.AlternateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("255"))
	theme.SelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("33"))
	theme.Background = lipgloss.Color("255")
	theme.Foreground = lipgloss.Color("0")
	return theme
}

// SetTheme applies a theme to the table
func (m *EnhancedModel) SetTheme(theme *TableTheme) {
	m.theme = theme
	m.applyTheme()
}

// applyTheme applies the current theme to the base model
func (m *EnhancedModel) applyTheme() {
	if m.theme == nil {
		return
	}

	m.Model.SetStyles(
		m.theme.HeaderStyle,
		m.theme.SelectedStyle,
		m.theme.RowStyle,
		m.theme.AlternateStyle,
		m.theme.BorderStyle,
	)
}

// SetColumnVisibility shows or hides a column
func (m *EnhancedModel) SetColumnVisibility(columnIndex int, visible bool) {
	if columnIndex < 0 || columnIndex >= len(m.columns) {
		return
	}

	if visible {
		delete(m.hiddenColumns, columnIndex)
	} else {
		m.hiddenColumns[columnIndex] = true
	}

	m.calculateColumnWidths()
}

// IsColumnVisible returns whether a column is visible
func (m *EnhancedModel) IsColumnVisible(columnIndex int) bool {
	return !m.hiddenColumns[columnIndex]
}

// ToggleColumnVisibility toggles column visibility
func (m *EnhancedModel) ToggleColumnVisibility(columnIndex int) {
	visible := m.IsColumnVisible(columnIndex)
	m.SetColumnVisibility(columnIndex, !visible)
}

// GetVisibleColumns returns indices of visible columns in display order
func (m *EnhancedModel) GetVisibleColumns() []int {
	visible := make([]int, 0, len(m.columnOrder))
	for _, colIndex := range m.columnOrder {
		if m.IsColumnVisible(colIndex) {
			visible = append(visible, colIndex)
		}
	}
	return visible
}

// ReorderColumns changes the column display order
func (m *EnhancedModel) ReorderColumns(fromIndex, toIndex int) {
	if fromIndex < 0 || fromIndex >= len(m.columnOrder) ||
		toIndex < 0 || toIndex >= len(m.columnOrder) ||
		fromIndex == toIndex {
		return
	}

	// Remove from old position
	column := m.columnOrder[fromIndex]
	m.columnOrder = append(m.columnOrder[:fromIndex], m.columnOrder[fromIndex+1:]...)

	// Insert at new position
	if toIndex > fromIndex {
		toIndex-- // Adjust for removal
	}

	newOrder := make([]int, 0, len(m.columnOrder)+1)
	newOrder = append(newOrder, m.columnOrder[:toIndex]...)
	newOrder = append(newOrder, column)
	newOrder = append(newOrder, m.columnOrder[toIndex:]...)

	m.columnOrder = newOrder
}

// ResizeColumn changes the width of a column
func (m *EnhancedModel) ResizeColumn(columnIndex, newWidth int) {
	if columnIndex < 0 || columnIndex >= len(m.columns) {
		return
	}

	// Enforce min/max constraints
	if newWidth < m.minColumnWidth {
		newWidth = m.minColumnWidth
	}
	if newWidth > m.maxColumnWidth {
		newWidth = m.maxColumnWidth
	}

	// Update column width
	m.columns[columnIndex].Width = newWidth
	m.calculateColumnWidths()
}

// SetSelectionMode changes the selection behavior
func (m *EnhancedModel) SetSelectionMode(mode SelectionMode) {
	m.selectionMode = mode
	m.multiSelect = (mode == MultiSelection || mode == RangeSelection)

	// Clear selection if switching to no selection
	if mode == NoSelection {
		m.selectedRows = make(map[int]bool)
	}
}

// SelectRow selects or deselects a row
func (m *EnhancedModel) SelectRow(index int, selected bool) {
	if m.selectionMode == NoSelection {
		return
	}

	if index < 0 || index >= len(m.rows) {
		return
	}

	if m.selectionMode == SingleSelection {
		// Clear other selections
		m.selectedRows = make(map[int]bool)
	}

	if selected {
		m.selectedRows[index] = true
	} else {
		delete(m.selectedRows, index)
	}
}

// ToggleRowSelection toggles selection for a row
func (m *EnhancedModel) ToggleRowSelection(index int) {
	selected := m.IsRowSelected(index)
	m.SelectRow(index, !selected)
}

// IsRowSelected returns whether a row is selected
func (m *EnhancedModel) IsRowSelected(index int) bool {
	return m.selectedRows[index]
}

// GetSelectedRows returns all selected row indices
func (m *EnhancedModel) GetSelectedRows() []int {
	selected := make([]int, 0, len(m.selectedRows))
	for index := range m.selectedRows {
		selected = append(selected, index)
	}
	sort.Ints(selected)
	return selected
}

// ClearSelection clears all row selections
func (m *EnhancedModel) ClearSelection() {
	m.selectedRows = make(map[int]bool)
}

// SelectRange selects a range of rows
func (m *EnhancedModel) SelectRange(start, end int) {
	if m.selectionMode != RangeSelection && m.selectionMode != MultiSelection {
		return
	}

	if start > end {
		start, end = end, start
	}

	for i := start; i <= end && i < len(m.rows); i++ {
		m.selectedRows[i] = true
	}
}

// EnableVirtualization enables viewport-based rendering for large datasets
func (m *EnhancedModel) EnableVirtualization(enabled bool) {
	m.virtualization = enabled
}

// SetRenderDebounce sets the minimum time between renders
func (m *EnhancedModel) SetRenderDebounce(duration time.Duration) {
	m.renderDebounce = duration
}

// ShouldRender returns whether the table should render based on debouncing
func (m *EnhancedModel) ShouldRender() bool {
	if m.renderDebounce == 0 {
		return true
	}

	return time.Since(m.lastRenderTime) >= m.renderDebounce
}

// MarkRendered marks the table as having been rendered
func (m *EnhancedModel) MarkRendered() {
	m.lastRenderTime = time.Now()
}

// AddRowWithAnimation adds a row with fade-in animation
func (m *EnhancedModel) AddRowWithAnimation(row Row) {
	m.AddRow(row)

	if m.animations {
		rowIndex := len(m.rows) - 1
		m.fadeInRows[rowIndex] = time.Now()
	}
}

// RemoveRowWithAnimation removes a row with fade-out animation
func (m *EnhancedModel) RemoveRowWithAnimation(index int) {
	if index < 0 || index >= len(m.rows) {
		return
	}

	if m.animations {
		m.fadeOutRows[index] = time.Now()

		// Remove after animation completes
		go func() {
			time.Sleep(m.transitionSpeed)
			m.removeRowAtIndex(index)
		}()
	} else {
		m.removeRowAtIndex(index)
	}
}

// removeRowAtIndex removes a row at the specified index
func (m *EnhancedModel) removeRowAtIndex(index int) {
	if index < 0 || index >= len(m.rows) {
		return
	}

	// Remove from rows
	m.rows = append(m.rows[:index], m.rows[index+1:]...)

	// Update selection
	newSelected := make(map[int]bool)
	for i, selected := range m.selectedRows {
		if i < index {
			newSelected[i] = selected
		} else if i > index {
			newSelected[i-1] = selected
		}
		// Skip the removed row
	}
	m.selectedRows = newSelected

	// Update selected index
	if m.selectedIndex >= len(m.rows) && len(m.rows) > 0 {
		m.selectedIndex = len(m.rows) - 1
	}

	// Clean up animation tracking
	delete(m.fadeInRows, index)
	delete(m.fadeOutRows, index)

	m.updateViewport()
}

// SetRowHighlighting enables or disables row highlighting
func (m *EnhancedModel) SetRowHighlighting(enabled bool) {
	m.rowHighlighting = enabled
}

// SetAlternateRows enables or disables alternating row colors
func (m *EnhancedModel) SetAlternateRows(enabled bool) {
	m.alternateRows = enabled
}

// EnableAnimations enables or disables animations
func (m *EnhancedModel) EnableAnimations(enabled bool) {
	m.animations = enabled
	if !enabled {
		m.fadeInRows = make(map[int]time.Time)
		m.fadeOutRows = make(map[int]time.Time)
	}
}

// SetTransitionSpeed sets the animation transition speed
func (m *EnhancedModel) SetTransitionSpeed(duration time.Duration) {
	m.transitionSpeed = duration
}

// GetColumnOrder returns the current column display order
func (m *EnhancedModel) GetColumnOrder() []int {
	order := make([]int, len(m.columnOrder))
	copy(order, m.columnOrder)
	return order
}

// SetColumnOrder sets the column display order
func (m *EnhancedModel) SetColumnOrder(order []int) {
	if len(order) != len(m.columns) {
		return
	}

	// Validate that all column indices are present
	seen := make(map[int]bool)
	for _, index := range order {
		if index < 0 || index >= len(m.columns) || seen[index] {
			return // Invalid order
		}
		seen[index] = true
	}

	m.columnOrder = make([]int, len(order))
	copy(m.columnOrder, order)
}

// GetTheme returns the current theme
func (m *EnhancedModel) GetTheme() *TableTheme {
	return m.theme
}

// SetCustomStyle sets a custom style for a specific element
func (m *EnhancedModel) SetCustomStyle(element string, style lipgloss.Style) {
	m.customStyles[element] = style
}

// GetCustomStyle returns a custom style for an element
func (m *EnhancedModel) GetCustomStyle(element string) (lipgloss.Style, bool) {
	style, exists := m.customStyles[element]
	return style, exists
}

// View renders the enhanced table with all features
func (m *EnhancedModel) View() string {
	if !m.ShouldRender() {
		// Return cached render if debouncing
		if len(m.renderBuffer) > 0 {
			return strings.Join(m.renderBuffer, "\n")
		}
	}

	// Apply theme before rendering
	m.applyTheme()

	// Use base model's View method but with enhancements
	baseView := m.Model.View()

	// Apply any post-processing effects
	result := m.applyEnhancements(baseView)

	// Cache the result
	m.renderBuffer = strings.Split(result, "\n")
	m.MarkRendered()

	return result
}

// applyEnhancements applies visual enhancements to the rendered table
func (m *EnhancedModel) applyEnhancements(baseView string) string {
	lines := strings.Split(baseView, "\n")

	// Apply animations if enabled
	if m.animations {
		lines = m.applyAnimations(lines)
	}

	// Apply custom highlighting
	if m.rowHighlighting {
		lines = m.applyHighlighting(lines)
	}

	return strings.Join(lines, "\n")
}

// applyAnimations applies fade-in/fade-out animations
func (m *EnhancedModel) applyAnimations(lines []string) []string {
	now := time.Now()

	// Clean up expired animations
	for index, startTime := range m.fadeInRows {
		if now.Sub(startTime) > m.transitionSpeed {
			delete(m.fadeInRows, index)
		}
	}

	for index, startTime := range m.fadeOutRows {
		if now.Sub(startTime) > m.transitionSpeed {
			delete(m.fadeOutRows, index)
		}
	}

	// Apply fade effects (simplified - would need more complex implementation)
	// This is a placeholder for animation logic

	return lines
}

// applyHighlighting applies custom row highlighting
func (m *EnhancedModel) applyHighlighting(lines []string) []string {
	// Apply highlighting based on row content or status
	// This is a placeholder for highlighting logic

	return lines
}

// GetPerformanceStats returns performance statistics
func (m *EnhancedModel) GetPerformanceStats() map[string]interface{} {
	return map[string]interface{}{
		"total_rows":       len(m.rows),
		"visible_rows":     m.GetVisibleRowCount(),
		"selected_rows":    len(m.selectedRows),
		"hidden_columns":   len(m.hiddenColumns),
		"last_render_time": m.lastRenderTime,
		"render_debounce":  m.renderDebounce,
		"virtualization":   m.virtualization,
		"animations":       m.animations,
	}
}
