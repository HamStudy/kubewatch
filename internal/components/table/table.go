package table

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

// Column represents a table column configuration
type Column struct {
	Title      string
	Width      int
	MinWidth   int
	MaxWidth   int
	Flex       bool // If true, column can expand to fill available space
	Align      lipgloss.Position
	TruncateAt string // Where to truncate: "end", "middle", "start"
}

// Row represents a single row of data
type Row struct {
	ID     string   // Unique identifier for the row
	Values []string // Cell values
	Style  lipgloss.Style
}

// Model represents the table component state
type Model struct {
	// Configuration
	columns           []Column
	rows              []Row
	width             int
	height            int
	showHeader        bool
	wordWrap          bool
	borderStyle       lipgloss.Style
	headerStyle       lipgloss.Style
	selectedStyle     lipgloss.Style
	rowStyle          lipgloss.Style
	alternateRowStyle lipgloss.Style

	// State
	selectedIndex int
	viewportStart int
	viewportSize  int
	columnWidths  []int // Calculated column widths

	// Behavior
	selectable bool
	focusable  bool
	focused    bool
}

// New creates a new table model
func New(columns []Column) *Model {
	return &Model{
		columns:           columns,
		rows:              []Row{},
		showHeader:        true,
		wordWrap:          false,
		selectable:        true,
		focusable:         true,
		selectedIndex:     0,
		viewportStart:     0,
		headerStyle:       lipgloss.NewStyle().Bold(true),
		selectedStyle:     lipgloss.NewStyle().Background(lipgloss.Color("240")),
		rowStyle:          lipgloss.NewStyle(),
		alternateRowStyle: lipgloss.NewStyle(),
		borderStyle:       lipgloss.NewStyle(),
	}
}

// SetRows sets the table rows
func (m *Model) SetRows(rows []Row) {
	m.rows = rows
	if m.selectedIndex >= len(rows) && len(rows) > 0 {
		m.selectedIndex = len(rows) - 1
	}
	m.updateViewport()
}

// AddRow adds a single row to the table
func (m *Model) AddRow(row Row) {
	m.rows = append(m.rows, row)
	m.updateViewport()
}

// ClearRows removes all rows from the table
func (m *Model) ClearRows() {
	m.rows = []Row{}
	m.selectedIndex = 0
	m.viewportStart = 0
}

// SetSize sets the table dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.calculateColumnWidths()
	m.calculateViewportSize()
	m.updateViewport()
}

// SetWordWrap enables or disables word wrapping
func (m *Model) SetWordWrap(enabled bool) {
	m.wordWrap = enabled
}

// SetShowHeader shows or hides the header row
func (m *Model) SetShowHeader(show bool) {
	m.showHeader = show
	m.calculateViewportSize()
	m.updateViewport()
}

// SetStyles sets the table styles
func (m *Model) SetStyles(header, selected, row, alternateRow, border lipgloss.Style) {
	m.headerStyle = header
	m.selectedStyle = selected
	m.rowStyle = row
	m.alternateRowStyle = alternateRow
	m.borderStyle = border
}

// Focus sets the focus state
func (m *Model) Focus() {
	m.focused = true
}

// Blur removes focus
func (m *Model) Blur() {
	m.focused = false
}

// IsFocused returns the focus state
func (m *Model) IsFocused() bool {
	return m.focused
}

// MoveUp moves selection up
func (m *Model) MoveUp() {
	if !m.selectable || len(m.rows) == 0 {
		return
	}

	if m.selectedIndex > 0 {
		m.selectedIndex--
		m.ensureSelectedVisible()
	}
}

// MoveDown moves selection down
func (m *Model) MoveDown() {
	if !m.selectable || len(m.rows) == 0 {
		return
	}

	if m.selectedIndex < len(m.rows)-1 {
		m.selectedIndex++
		m.ensureSelectedVisible()
	}
}

// MoveToTop moves selection to the first row
func (m *Model) MoveToTop() {
	if !m.selectable || len(m.rows) == 0 {
		return
	}

	m.selectedIndex = 0
	m.viewportStart = 0
}

// MoveToBottom moves selection to the last row
func (m *Model) MoveToBottom() {
	if !m.selectable || len(m.rows) == 0 {
		return
	}

	m.selectedIndex = len(m.rows) - 1
	m.ensureSelectedVisible()
}

// PageUp moves selection up by viewport size
func (m *Model) PageUp() {
	if !m.selectable || len(m.rows) == 0 {
		return
	}

	m.selectedIndex -= m.viewportSize
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
	m.viewportStart -= m.viewportSize
	if m.viewportStart < 0 {
		m.viewportStart = 0
	}
}

// PageDown moves selection down by viewport size
func (m *Model) PageDown() {
	if !m.selectable || len(m.rows) == 0 {
		return
	}

	m.selectedIndex += m.viewportSize
	if m.selectedIndex >= len(m.rows) {
		m.selectedIndex = len(m.rows) - 1
	}
	m.ensureSelectedVisible()
}

// GetSelectedRow returns the currently selected row
func (m *Model) GetSelectedRow() *Row {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.rows) {
		return &m.rows[m.selectedIndex]
	}
	return nil
}

// GetSelectedIndex returns the current selection index
func (m *Model) GetSelectedIndex() int {
	return m.selectedIndex
}

// SetSelectedIndex sets the selection index
func (m *Model) SetSelectedIndex(index int) {
	if index >= 0 && index < len(m.rows) {
		m.selectedIndex = index
		m.ensureSelectedVisible()
	}
}

// GetRowCount returns the total number of rows
func (m *Model) GetRowCount() int {
	return len(m.rows)
}

// GetVisibleRowCount returns the number of visible rows
func (m *Model) GetVisibleRowCount() int {
	if len(m.rows) < m.viewportSize {
		return len(m.rows)
	}
	return m.viewportSize
}

// View renders the table
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	m.calculateColumnWidths()

	var lines []string

	// Render header
	if m.showHeader {
		header := m.renderHeader()
		lines = append(lines, header)
	}

	// Render visible rows
	visibleRows := m.getVisibleRows()
	for i, row := range visibleRows {
		actualIndex := m.viewportStart + i
		isSelected := m.selectable && actualIndex == m.selectedIndex
		rowStr := m.renderRow(row, actualIndex, isSelected)
		lines = append(lines, rowStr)
	}

	// Fill remaining space with empty lines if needed
	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	// Truncate to height
	if len(lines) > m.height {
		lines = lines[:m.height]
	}

	return strings.Join(lines, "\n")
}

// Private methods

func (m *Model) calculateColumnWidths() {
	if len(m.columns) == 0 || m.width == 0 {
		return
	}

	m.columnWidths = make([]int, len(m.columns))
	totalFixed := 0
	flexCount := 0

	// Calculate fixed widths and count flex columns
	for i, col := range m.columns {
		if col.Width > 0 {
			// Fixed width column
			width := col.Width
			if col.MaxWidth > 0 && width > col.MaxWidth {
				width = col.MaxWidth
			}
			if col.MinWidth > 0 && width < col.MinWidth {
				width = col.MinWidth
			}
			m.columnWidths[i] = width
			totalFixed += width
		} else if col.Flex {
			flexCount++
		} else {
			// Default minimum width
			width := col.MinWidth
			if width == 0 {
				width = 10 // Default minimum
			}
			m.columnWidths[i] = width
			totalFixed += width
		}
	}

	// Distribute remaining space among flex columns
	if flexCount > 0 {
		// Account for column separators (1 space between columns)
		separatorWidth := len(m.columns) - 1
		if separatorWidth < 0 {
			separatorWidth = 0
		}

		availableWidth := m.width - totalFixed - separatorWidth

		// Initialize flex columns with their minimum widths
		for i, col := range m.columns {
			if col.Flex && col.Width == 0 {
				minWidth := col.MinWidth
				if minWidth == 0 {
					minWidth = 1 // Default minimum
				}
				m.columnWidths[i] = minWidth
			}
		}

		// Calculate total minimum width needed for flex columns
		totalMinWidth := 0
		for i, col := range m.columns {
			if col.Flex && col.Width == 0 {
				totalMinWidth += m.columnWidths[i]
			}
		}

		// If we have extra space beyond minimums, distribute it
		if availableWidth > totalMinWidth {
			extraSpace := availableWidth - totalMinWidth

			// For max width constraints, we need to handle them specially
			// First pass: distribute space equally, respecting max constraints
			remainingFlexCount := flexCount
			remainingSpace := extraSpace

			for pass := 0; pass < flexCount && remainingSpace > 0 && remainingFlexCount > 0; pass++ {
				spacePerColumn := remainingSpace / remainingFlexCount
				extraSpaceRemainder := remainingSpace % remainingFlexCount

				for i, col := range m.columns {
					if col.Flex && col.Width == 0 && remainingSpace > 0 {
						currentWidth := m.columnWidths[i]
						additionalSpace := spacePerColumn
						if extraSpaceRemainder > 0 {
							additionalSpace++
							extraSpaceRemainder--
						}

						newWidth := currentWidth + additionalSpace

						// Check max width constraint
						if col.MaxWidth > 0 && newWidth > col.MaxWidth {
							// This column is constrained
							actualAdditional := col.MaxWidth - currentWidth
							if actualAdditional < 0 {
								actualAdditional = 0
							}
							m.columnWidths[i] = col.MaxWidth
							remainingSpace -= actualAdditional
							remainingFlexCount--
						} else {
							// This column can take the full additional space
							m.columnWidths[i] = newWidth
							remainingSpace -= additionalSpace
						}
					}
				}
			}
		} else if availableWidth < totalMinWidth {
			// Not enough space for minimum widths, distribute proportionally
			if availableWidth > 0 {
				for i, col := range m.columns {
					if col.Flex && col.Width == 0 {
						proportion := float64(m.columnWidths[i]) / float64(totalMinWidth)
						m.columnWidths[i] = int(float64(availableWidth) * proportion)
						if m.columnWidths[i] < 1 {
							m.columnWidths[i] = 1
						}
					}
				}
			} else {
				// No space at all, set to minimum
				for i, col := range m.columns {
					if col.Flex && col.Width == 0 {
						m.columnWidths[i] = 1
					}
				}
			}
		}
	}
}

func (m *Model) calculateViewportSize() {
	m.viewportSize = m.height
	if m.showHeader {
		m.viewportSize--
	}
	if m.viewportSize < 0 {
		m.viewportSize = 0
	}
}

func (m *Model) updateViewport() {
	m.calculateViewportSize()

	if len(m.rows) == 0 {
		m.viewportStart = 0
		return
	}

	// Ensure viewport shows as many rows as possible
	maxStart := len(m.rows) - m.viewportSize
	if maxStart < 0 {
		maxStart = 0
	}

	if m.viewportStart > maxStart {
		m.viewportStart = maxStart
	}

	if m.viewportStart < 0 {
		m.viewportStart = 0
	}
}

func (m *Model) ensureSelectedVisible() {
	if m.selectedIndex < m.viewportStart {
		m.viewportStart = m.selectedIndex
	} else if m.selectedIndex >= m.viewportStart+m.viewportSize {
		m.viewportStart = m.selectedIndex - m.viewportSize + 1
	}

	if m.viewportStart < 0 {
		m.viewportStart = 0
	}
}

func (m *Model) getVisibleRows() []Row {
	if len(m.rows) == 0 {
		return []Row{}
	}

	end := m.viewportStart + m.viewportSize
	if end > len(m.rows) {
		end = len(m.rows)
	}

	return m.rows[m.viewportStart:end]
}

func (m *Model) renderHeader() string {
	if len(m.columns) == 0 || len(m.columnWidths) == 0 {
		return ""
	}

	cells := make([]string, len(m.columns))
	for i, col := range m.columns {
		width := m.columnWidths[i]
		if width <= 0 {
			continue
		}

		text := col.Title
		if len(text) > width {
			text = truncate.StringWithTail(text, uint(width), "…")
		}

		// Apply alignment
		switch col.Align {
		case lipgloss.Right:
			text = lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(text)
		case lipgloss.Center:
			text = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(text)
		default:
			text = lipgloss.NewStyle().Width(width).Align(lipgloss.Left).Render(text)
		}

		cells[i] = text
	}

	row := strings.Join(cells, " ")
	return m.headerStyle.Render(row)
}

func (m *Model) renderRow(row Row, index int, isSelected bool) string {
	if len(m.columns) == 0 || len(m.columnWidths) == 0 {
		return ""
	}

	cells := make([]string, len(m.columns))
	for i, col := range m.columns {
		width := m.columnWidths[i]
		if width <= 0 {
			continue
		}

		text := ""
		if i < len(row.Values) {
			text = row.Values[i]
		}

		// Handle word wrap or truncation
		if m.wordWrap {
			// When word wrap is ON, allow content to expand beyond column width
			// Don't truncate - let the content flow naturally
			// For now, we don't implement multi-line wrapping in table cells
			// but we don't truncate either
		} else if len(text) > width {
			// When word wrap is OFF, truncate content to fit within column width
			switch col.TruncateAt {
			case "middle":
				if width > 3 {
					keep := (width - 1) / 2
					text = text[:keep] + "…" + text[len(text)-keep:]
				} else {
					text = truncate.StringWithTail(text, uint(width), "…")
				}
			case "start":
				if width > 1 {
					text = "…" + text[len(text)-width+1:]
				} else {
					text = "…"
				}
			default: // "end"
				text = truncate.StringWithTail(text, uint(width), "…")
			}
		}

		// Apply alignment
		switch col.Align {
		case lipgloss.Right:
			text = lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(text)
		case lipgloss.Center:
			text = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(text)
		default:
			text = lipgloss.NewStyle().Width(width).Align(lipgloss.Left).Render(text)
		}

		cells[i] = text
	}

	rowStr := strings.Join(cells, " ")

	// Apply row style
	style := m.rowStyle
	if index%2 == 1 {
		style = m.alternateRowStyle
	}
	// Row-specific style overrides default (check if it has any styling)
	if row.Style.String() != "" {
		style = row.Style
	}
	if isSelected && m.focused {
		style = m.selectedStyle
	}

	return style.Render(rowStr)
}
