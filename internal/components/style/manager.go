package style

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// Manager handles styling and theming for the application
type Manager struct {
	theme *Theme
	cache map[string]lipgloss.Style
	mu    sync.RWMutex
}

// Theme defines color schemes and styling
type Theme struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Colors      *ColorScheme      `yaml:"colors"`
	Styles      *StyleDefinitions `yaml:"styles"`
}

// ColorScheme defines the color palette
type ColorScheme struct {
	// Base colors
	Background lipgloss.Color `yaml:"background"`
	Foreground lipgloss.Color `yaml:"foreground"`

	// Selection colors
	Selection *SelectionColors `yaml:"selection"`

	// Status colors
	Status *StatusColors `yaml:"status"`

	// Metric colors
	Metrics *MetricColors `yaml:"metrics"`

	// UI element colors
	UI *UIColors `yaml:"ui"`
}

// SelectionColors for selected items
type SelectionColors struct {
	Background lipgloss.Color `yaml:"background"`
	Foreground lipgloss.Color `yaml:"foreground"`
}

// StatusColors for different resource states
type StatusColors struct {
	Running     lipgloss.Color `yaml:"running"`
	Pending     lipgloss.Color `yaml:"pending"`
	Failed      lipgloss.Color `yaml:"failed"`
	Completed   lipgloss.Color `yaml:"completed"`
	Terminating lipgloss.Color `yaml:"terminating"`
	Unknown     lipgloss.Color `yaml:"unknown"`
}

// MetricColors for resource usage indicators
type MetricColors struct {
	CPU    *GradientColors `yaml:"cpu"`
	Memory *GradientColors `yaml:"memory"`
}

// GradientColors for metric thresholds
type GradientColors struct {
	Low      lipgloss.Color `yaml:"low"`
	Medium   lipgloss.Color `yaml:"medium"`
	High     lipgloss.Color `yaml:"high"`
	Critical lipgloss.Color `yaml:"critical"`
}

// UIColors for interface elements
type UIColors struct {
	Border  lipgloss.Color `yaml:"border"`
	Header  lipgloss.Color `yaml:"header"`
	Info    lipgloss.Color `yaml:"info"`
	Warning lipgloss.Color `yaml:"warning"`
	Error   lipgloss.Color `yaml:"error"`
	Success lipgloss.Color `yaml:"success"`
}

// StyleDefinitions for common UI elements
type StyleDefinitions struct {
	Header   lipgloss.Style `yaml:"header"`
	Selected lipgloss.Style `yaml:"selected"`
	Cell     lipgloss.Style `yaml:"cell"`
	Border   lipgloss.Style `yaml:"border"`
	Title    lipgloss.Style `yaml:"title"`
	Subtitle lipgloss.Style `yaml:"subtitle"`
}

// NewManager creates a new style manager with the default theme
func NewManager() *Manager {
	return &Manager{
		theme: getDefaultTheme(),
		cache: make(map[string]lipgloss.Style),
	}
}

// SetTheme sets the current theme
func (m *Manager) SetTheme(theme *Theme) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.theme = theme
	m.cache = make(map[string]lipgloss.Style) // Clear cache
}

// GetTheme returns the current theme
func (m *Manager) GetTheme() *Theme {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.theme
}

// StatusCell returns a styled status cell
func (m *Manager) StatusCell(status string, selected bool) lipgloss.Style {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cacheKey := fmt.Sprintf("status_%s_%t", status, selected)
	if style, ok := m.cache[cacheKey]; ok {
		return style
	}

	var style lipgloss.Style

	if selected {
		style = lipgloss.NewStyle().
			Background(m.theme.Colors.Selection.Background).
			Foreground(m.theme.Colors.Selection.Foreground)
	} else {
		color := m.getStatusColor(status)
		style = lipgloss.NewStyle().Foreground(color)
	}

	m.cache[cacheKey] = style
	return style
}

// MetricCell returns a styled metric cell with color coding
func (m *Manager) MetricCell(value string, metricType MetricType, selected bool) lipgloss.Style {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cacheKey := fmt.Sprintf("metric_%s_%d_%t", value, metricType, selected)
	if style, ok := m.cache[cacheKey]; ok {
		return style
	}

	var style lipgloss.Style

	if selected {
		style = lipgloss.NewStyle().
			Background(m.theme.Colors.Selection.Background).
			Foreground(m.theme.Colors.Selection.Foreground).
			Align(lipgloss.Right)
	} else {
		color := m.getMetricColor(value, metricType)
		style = lipgloss.NewStyle().
			Foreground(color).
			Align(lipgloss.Right)
	}

	m.cache[cacheKey] = style
	return style
}

// HeaderCell returns a styled header cell
func (m *Manager) HeaderCell(text string, width int) lipgloss.Style {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cacheKey := fmt.Sprintf("header_%d", width)
	if style, ok := m.cache[cacheKey]; ok {
		return style
	}

	style := lipgloss.NewStyle().
		Width(width).
		Bold(true).
		Foreground(m.theme.Colors.UI.Header).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(m.theme.Colors.UI.Border)

	m.cache[cacheKey] = style
	return style
}

// NumericCell returns a styled numeric cell (right-aligned)
func (m *Manager) NumericCell(value string, width int, selected bool) lipgloss.Style {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cacheKey := fmt.Sprintf("numeric_%d_%t", width, selected)
	if style, ok := m.cache[cacheKey]; ok {
		return style
	}

	var style lipgloss.Style

	if selected {
		style = lipgloss.NewStyle().
			Width(width).
			Align(lipgloss.Right).
			Background(m.theme.Colors.Selection.Background).
			Foreground(m.theme.Colors.Selection.Foreground)
	} else {
		style = lipgloss.NewStyle().
			Width(width).
			Align(lipgloss.Right).
			Foreground(m.theme.Colors.Foreground)
	}

	m.cache[cacheKey] = style
	return style
}

// TextCell returns a styled text cell
func (m *Manager) TextCell(text string, width int, selected bool) lipgloss.Style {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cacheKey := fmt.Sprintf("text_%d_%t", width, selected)
	if style, ok := m.cache[cacheKey]; ok {
		return style
	}

	var style lipgloss.Style

	if selected {
		style = lipgloss.NewStyle().
			Width(width).
			Background(m.theme.Colors.Selection.Background).
			Foreground(m.theme.Colors.Selection.Foreground)
	} else {
		style = lipgloss.NewStyle().
			Width(width).
			Foreground(m.theme.Colors.Foreground)
	}

	m.cache[cacheKey] = style
	return style
}

// ColorText applies a color to text
func (m *Manager) ColorText(text, color string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return style.Render(text)
}

// GradientText applies gradient coloring based on value and thresholds
func (m *Manager) GradientText(text string, value, min, max float64, colors []string) string {
	if len(colors) == 0 {
		return text
	}

	// Calculate position in gradient
	position := (value - min) / (max - min)
	if position < 0 {
		position = 0
	} else if position > 1 {
		position = 1
	}

	// Select color based on position
	colorIndex := int(position * float64(len(colors)-1))
	if colorIndex >= len(colors) {
		colorIndex = len(colors) - 1
	}

	return m.ColorText(text, colors[colorIndex])
}

// getStatusColor returns the appropriate color for a status
func (m *Manager) getStatusColor(status string) lipgloss.Color {
	status = strings.ToLower(status)

	switch {
	case strings.Contains(status, "running"):
		return m.theme.Colors.Status.Running
	case strings.Contains(status, "pending"), strings.Contains(status, "creating"):
		return m.theme.Colors.Status.Pending
	case strings.Contains(status, "failed"), strings.Contains(status, "error"),
		strings.Contains(status, "crash"), strings.Contains(status, "backoff"):
		return m.theme.Colors.Status.Failed
	case strings.Contains(status, "completed"), strings.Contains(status, "succeeded"):
		return m.theme.Colors.Status.Completed
	case strings.Contains(status, "terminating"):
		return m.theme.Colors.Status.Terminating
	default:
		return m.theme.Colors.Status.Unknown
	}
}

// getMetricColor returns the appropriate color for a metric value
func (m *Manager) getMetricColor(value string, metricType MetricType) lipgloss.Color {
	if value == "-" || value == "" {
		return m.theme.Colors.UI.Info
	}

	var colors *GradientColors
	var numValue float64

	switch metricType {
	case MetricTypeCPU:
		colors = m.theme.Colors.Metrics.CPU
		numValue = m.parseCPUValue(value)
	case MetricTypeMemory:
		colors = m.theme.Colors.Metrics.Memory
		numValue = m.parseMemoryValue(value)
	default:
		return m.theme.Colors.Foreground
	}

	// Apply thresholds
	if numValue < 0.3 {
		return colors.Low
	} else if numValue < 0.7 {
		return colors.Medium
	} else if numValue < 0.9 {
		return colors.High
	}
	return colors.Critical
}

// parseCPUValue parses CPU value and returns normalized value (0-1)
func (m *Manager) parseCPUValue(value string) float64 {
	if strings.HasSuffix(value, "m") {
		// Millicores
		numStr := strings.TrimSuffix(value, "m")
		if val, err := strconv.ParseFloat(numStr, 64); err == nil {
			return val / 1000.0 // Normalize to cores, assume 1 core = 100%
		}
	} else {
		// Cores
		if val, err := strconv.ParseFloat(value, 64); err == nil {
			return val // Assume 1 core = 100%
		}
	}
	return 0
}

// parseMemoryValue parses memory value and returns normalized value (0-1)
func (m *Manager) parseMemoryValue(value string) float64 {
	var multiplier float64 = 1
	cleanValue := value

	if strings.HasSuffix(value, "Gi") {
		multiplier = 1024
		cleanValue = strings.TrimSuffix(value, "Gi")
	} else if strings.HasSuffix(value, "Mi") {
		multiplier = 1
		cleanValue = strings.TrimSuffix(value, "Mi")
	} else if strings.HasSuffix(value, "Ki") {
		multiplier = 1.0 / 1024.0
		cleanValue = strings.TrimSuffix(value, "Ki")
	}

	if val, err := strconv.ParseFloat(cleanValue, 64); err == nil {
		mb := val * multiplier
		// Normalize to 0-1 scale, assume 1Gi = 100%
		return mb / 1024.0
	}
	return 0
}

// MetricType represents different types of metrics
type MetricType int

const (
	MetricTypeCPU MetricType = iota
	MetricTypeMemory
	MetricTypeNetwork
	MetricTypeStorage
)

// ClearCache clears the style cache
func (m *Manager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = make(map[string]lipgloss.Style)
}

// getDefaultTheme returns the default dark theme
func getDefaultTheme() *Theme {
	return &Theme{
		Name:        "default",
		Description: "Default dark theme",
		Colors: &ColorScheme{
			Background: lipgloss.Color("#1e1e1e"),
			Foreground: lipgloss.Color("#d4d4d4"),
			Selection: &SelectionColors{
				Background: lipgloss.Color("#264f78"),
				Foreground: lipgloss.Color("#ffffff"),
			},
			Status: &StatusColors{
				Running:     lipgloss.Color("#4ec9b0"),
				Pending:     lipgloss.Color("#dcdcaa"),
				Failed:      lipgloss.Color("#f44747"),
				Completed:   lipgloss.Color("#569cd6"),
				Terminating: lipgloss.Color("#c586c0"),
				Unknown:     lipgloss.Color("#808080"),
			},
			Metrics: &MetricColors{
				CPU: &GradientColors{
					Low:      lipgloss.Color("#4ec9b0"),
					Medium:   lipgloss.Color("#dcdcaa"),
					High:     lipgloss.Color("#ce9178"),
					Critical: lipgloss.Color("#f44747"),
				},
				Memory: &GradientColors{
					Low:      lipgloss.Color("#4ec9b0"),
					Medium:   lipgloss.Color("#dcdcaa"),
					High:     lipgloss.Color("#ce9178"),
					Critical: lipgloss.Color("#f44747"),
				},
			},
			UI: &UIColors{
				Border:  lipgloss.Color("#3c3c3c"),
				Header:  lipgloss.Color("#cccccc"),
				Info:    lipgloss.Color("#569cd6"),
				Warning: lipgloss.Color("#dcdcaa"),
				Error:   lipgloss.Color("#f44747"),
				Success: lipgloss.Color("#4ec9b0"),
			},
		},
	}
}

// GetLightTheme returns a light theme
func GetLightTheme() *Theme {
	return &Theme{
		Name:        "light",
		Description: "Light theme",
		Colors: &ColorScheme{
			Background: lipgloss.Color("#ffffff"),
			Foreground: lipgloss.Color("#000000"),
			Selection: &SelectionColors{
				Background: lipgloss.Color("#0078d4"),
				Foreground: lipgloss.Color("#ffffff"),
			},
			Status: &StatusColors{
				Running:     lipgloss.Color("#107c10"),
				Pending:     lipgloss.Color("#ffb900"),
				Failed:      lipgloss.Color("#d13438"),
				Completed:   lipgloss.Color("#0078d4"),
				Terminating: lipgloss.Color("#881798"),
				Unknown:     lipgloss.Color("#605e5c"),
			},
			Metrics: &MetricColors{
				CPU: &GradientColors{
					Low:      lipgloss.Color("#107c10"),
					Medium:   lipgloss.Color("#ffb900"),
					High:     lipgloss.Color("#ff8c00"),
					Critical: lipgloss.Color("#d13438"),
				},
				Memory: &GradientColors{
					Low:      lipgloss.Color("#107c10"),
					Medium:   lipgloss.Color("#ffb900"),
					High:     lipgloss.Color("#ff8c00"),
					Critical: lipgloss.Color("#d13438"),
				},
			},
			UI: &UIColors{
				Border:  lipgloss.Color("#d1d1d1"),
				Header:  lipgloss.Color("#323130"),
				Info:    lipgloss.Color("#0078d4"),
				Warning: lipgloss.Color("#ffb900"),
				Error:   lipgloss.Color("#d13438"),
				Success: lipgloss.Color("#107c10"),
			},
		},
	}
}

// GetHighContrastTheme returns a high contrast theme for accessibility
func GetHighContrastTheme() *Theme {
	return &Theme{
		Name:        "high-contrast",
		Description: "High contrast theme for accessibility",
		Colors: &ColorScheme{
			Background: lipgloss.Color("#000000"),
			Foreground: lipgloss.Color("#ffffff"),
			Selection: &SelectionColors{
				Background: lipgloss.Color("#ffffff"),
				Foreground: lipgloss.Color("#000000"),
			},
			Status: &StatusColors{
				Running:     lipgloss.Color("#00ff00"),
				Pending:     lipgloss.Color("#ffff00"),
				Failed:      lipgloss.Color("#ff0000"),
				Completed:   lipgloss.Color("#00ffff"),
				Terminating: lipgloss.Color("#ff00ff"),
				Unknown:     lipgloss.Color("#808080"),
			},
			Metrics: &MetricColors{
				CPU: &GradientColors{
					Low:      lipgloss.Color("#00ff00"),
					Medium:   lipgloss.Color("#ffff00"),
					High:     lipgloss.Color("#ff8000"),
					Critical: lipgloss.Color("#ff0000"),
				},
				Memory: &GradientColors{
					Low:      lipgloss.Color("#00ff00"),
					Medium:   lipgloss.Color("#ffff00"),
					High:     lipgloss.Color("#ff8000"),
					Critical: lipgloss.Color("#ff0000"),
				},
			},
			UI: &UIColors{
				Border:  lipgloss.Color("#ffffff"),
				Header:  lipgloss.Color("#ffffff"),
				Info:    lipgloss.Color("#00ffff"),
				Warning: lipgloss.Color("#ffff00"),
				Error:   lipgloss.Color("#ff0000"),
				Success: lipgloss.Color("#00ff00"),
			},
		},
	}
}
