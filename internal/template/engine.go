package template

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Engine provides template execution with custom K8s formatting functions
type Engine struct {
	funcMap   template.FuncMap
	templates map[string]*template.Template
	cache     *Cache
	mu        sync.RWMutex
}

// FormattedValue represents a formatted output with styling hints
type FormattedValue struct {
	Text      string
	Color     string
	Bold      bool
	Italic    bool
	Underline bool
}

// NewEngine creates a new template engine with K8s-specific functions
func NewEngine() *Engine {
	e := &Engine{
		funcMap:   make(template.FuncMap),
		templates: make(map[string]*template.Template),
		cache:     NewCache(1000), // Cache last 1000 results
	}
	e.registerBuiltinFuncs()
	return e
}

// registerBuiltinFuncs adds all custom functions to the engine
func (e *Engine) registerBuiltinFuncs() {
	// Add basic template functions (we'll add sprig later when dependencies are set up)
	// For now, just add our custom functions

	// Color and styling functions
	e.funcMap["color"] = e.colorFunc
	e.funcMap["gradient"] = e.gradientFunc
	e.funcMap["style"] = e.styleFunc
	e.funcMap["bg"] = e.bgFunc
	e.funcMap["bold"] = e.boldFunc
	e.funcMap["italic"] = e.italicFunc
	e.funcMap["underline"] = e.underlineFunc

	// K8s-specific formatting
	e.funcMap["humanizeBytes"] = e.humanizeBytesFunc
	e.funcMap["humanizeDuration"] = e.humanizeDurationFunc
	e.funcMap["millicores"] = e.millicoresFunc
	e.funcMap["cores"] = e.coresFunc
	e.funcMap["toMB"] = e.toMBFunc
	e.funcMap["toGB"] = e.toGBFunc
	e.funcMap["toMillicores"] = e.toMillicoresFunc

	// Comparison and logic
	e.funcMap["colorIf"] = e.colorIfFunc
	e.funcMap["choose"] = e.chooseFunc
	e.funcMap["hasPrefix"] = strings.HasPrefix
	e.funcMap["hasSuffix"] = strings.HasSuffix
	e.funcMap["contains"] = strings.Contains
	e.funcMap["matches"] = e.matchesFunc

	// Icons
	e.funcMap["icon"] = e.iconFunc
	e.funcMap["iconIf"] = e.iconIfFunc

	// Math operations
	e.funcMap["percent"] = e.percentFunc
	e.funcMap["div"] = e.divFunc
	e.funcMap["mul"] = e.mulFunc
	e.funcMap["sub"] = e.subFunc
	e.funcMap["add"] = e.addFunc
	e.funcMap["min"] = e.minFunc
	e.funcMap["max"] = e.maxFunc

	// String operations
	e.funcMap["join"] = e.joinFunc
	e.funcMap["split"] = strings.Split
	e.funcMap["trim"] = strings.TrimSpace
	e.funcMap["upper"] = strings.ToUpper
	e.funcMap["lower"] = strings.ToLower
	e.funcMap["len"] = e.lenFunc
	e.funcMap["toString"] = e.toStringFunc

	// Time functions
	e.funcMap["ago"] = e.agoFunc
	e.funcMap["ageInSeconds"] = e.ageInSecondsFunc
	e.funcMap["timestamp"] = e.timestampFunc

	// List/collection operations
	e.funcMap["list"] = e.listFunc
	e.funcMap["append"] = e.appendFunc
	e.funcMap["slice"] = e.sliceFunc
	e.funcMap["default"] = e.defaultFunc
}

// Execute runs a template with the given data
func (e *Engine) Execute(tmplStr string, data interface{}) (string, error) {
	// Check cache first
	if cached, ok := e.cache.Get(tmplStr, data); ok {
		return cached, nil
	}

	// Parse and execute template
	tmpl, err := e.getOrParseTemplate(tmplStr)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	result := buf.String()
	e.cache.Set(tmplStr, data, result)
	return result, nil
}

// getOrParseTemplate retrieves or creates a template
func (e *Engine) getOrParseTemplate(tmplStr string) (*template.Template, error) {
	e.mu.RLock()
	if tmpl, ok := e.templates[tmplStr]; ok {
		e.mu.RUnlock()
		return tmpl, nil
	}
	e.mu.RUnlock()

	// Parse new template
	tmpl, err := template.New("").Funcs(e.funcMap).Parse(tmplStr)
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.templates[tmplStr] = tmpl
	e.mu.Unlock()

	return tmpl, nil
}

// Color functions
func (e *Engine) colorFunc(color, text string) string {
	if text == "" {
		return ""
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return style.Render(text)
}

func (e *Engine) gradientFunc(value, min, max float64, colors ...string) string {
	if len(colors) == 0 {
		return fmt.Sprintf("%.0f", value)
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

	return e.colorFunc(colors[colorIndex], fmt.Sprintf("%.0f", value))
}

func (e *Engine) styleFunc(styles ...string) func(string) string {
	return func(text string) string {
		style := lipgloss.NewStyle()
		for _, s := range styles {
			switch s {
			case "bold":
				style = style.Bold(true)
			case "italic":
				style = style.Italic(true)
			case "underline":
				style = style.Underline(true)
			default:
				// Assume it's a color
				style = style.Foreground(lipgloss.Color(s))
			}
		}
		return style.Render(text)
	}
}

func (e *Engine) bgFunc(color, text string) string {
	style := lipgloss.NewStyle().Background(lipgloss.Color(color))
	return style.Render(text)
}

func (e *Engine) boldFunc(text string) string {
	return lipgloss.NewStyle().Bold(true).Render(text)
}

func (e *Engine) italicFunc(text string) string {
	return lipgloss.NewStyle().Italic(true).Render(text)
}

func (e *Engine) underlineFunc(text string) string {
	return lipgloss.NewStyle().Underline(true).Render(text)
}

// K8s formatting functions
func (e *Engine) humanizeBytesFunc(bytes interface{}) string {
	var b int64
	switch v := bytes.(type) {
	case int:
		b = int64(v)
	case int64:
		b = v
	case float64:
		b = int64(v)
	case string:
		// Parse K8s quantity string
		return v // For now, return as-is
	default:
		return "0"
	}

	units := []string{"", "Ki", "Mi", "Gi", "Ti", "Pi"}
	value := float64(b)
	unit := 0

	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}

	if unit == 0 {
		return fmt.Sprintf("%d%s", int(value), units[unit])
	}
	return fmt.Sprintf("%.1f%s", value, units[unit])
}

func (e *Engine) humanizeDurationFunc(d interface{}) string {
	var duration time.Duration
	switch v := d.(type) {
	case time.Duration:
		duration = v
	case int64:
		duration = time.Duration(v) * time.Second
	case float64:
		duration = time.Duration(v) * time.Second
	default:
		return "0s"
	}

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else if duration < 30*24*time.Hour {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	} else if duration < 365*24*time.Hour {
		return fmt.Sprintf("%dmo", int(duration.Hours()/24/30))
	}
	return fmt.Sprintf("%dy", int(duration.Hours()/24/365))
}

func (e *Engine) millicoresFunc(cores interface{}) string {
	var c float64
	switch v := cores.(type) {
	case float64:
		c = v
	case int:
		c = float64(v)
	case string:
		// Parse K8s CPU string
		if strings.HasSuffix(v, "m") {
			return v
		}
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			c = val
		}
	default:
		return "0m"
	}
	return fmt.Sprintf("%dm", int(c*1000))
}

func (e *Engine) coresFunc(millicores interface{}) string {
	var m float64
	switch v := millicores.(type) {
	case float64:
		m = v
	case int:
		m = float64(v)
	case string:
		if strings.HasSuffix(v, "m") {
			v = strings.TrimSuffix(v, "m")
			if val, err := strconv.ParseFloat(v, 64); err == nil {
				m = val
			}
		}
	default:
		return "0"
	}
	return fmt.Sprintf("%.2f", m/1000)
}

func (e *Engine) toMBFunc(bytes interface{}) float64 {
	var b int64
	switch v := bytes.(type) {
	case int:
		b = int64(v)
	case int64:
		b = v
	case float64:
		b = int64(v)
	case string:
		// Parse K8s quantity
		return 0 // TODO: implement quantity parsing
	default:
		return 0
	}
	return float64(b) / 1024 / 1024
}

func (e *Engine) toGBFunc(bytes interface{}) float64 {
	return e.toMBFunc(bytes) / 1024
}

func (e *Engine) toMillicoresFunc(cpu interface{}) float64 {
	switch v := cpu.(type) {
	case string:
		if strings.HasSuffix(v, "m") {
			val, _ := strconv.ParseFloat(strings.TrimSuffix(v, "m"), 64)
			return val
		}
		val, _ := strconv.ParseFloat(v, 64)
		return val * 1000
	case float64:
		return v * 1000
	case int:
		return float64(v) * 1000
	default:
		return 0
	}
}

// Conditional functions
func (e *Engine) colorIfFunc(condition bool, trueColor, falseColor, text string) string {
	if condition {
		return e.colorFunc(trueColor, text)
	}
	return e.colorFunc(falseColor, text)
}

func (e *Engine) chooseFunc(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}

func (e *Engine) matchesFunc(pattern, text string) bool {
	matched, _ := regexp.MatchString(pattern, text)
	return matched
}

// Icon functions
func (e *Engine) iconFunc(name string) string {
	icons := map[string]string{
		"success":    "✓",
		"error":      "✗",
		"warning":    "⚠",
		"info":       "ℹ",
		"running":    "●",
		"pending":    "◐",
		"stopped":    "■",
		"pod":        "⬢",
		"deployment": "⬡",
		"service":    "⬨",
		"configmap":  "☰",
		"secret":     "🔒",
	}
	if icon, ok := icons[name]; ok {
		return icon
	}
	return ""
}

func (e *Engine) iconIfFunc(condition bool, trueIcon, falseIcon string) string {
	if condition {
		return e.iconFunc(trueIcon)
	}
	return e.iconFunc(falseIcon)
}

// Math functions
func (e *Engine) percentFunc(value, total interface{}) string {
	v := toFloat64(value)
	t := toFloat64(total)
	if t == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", (v/t)*100)
}

func (e *Engine) divFunc(a, b interface{}) float64 {
	divisor := toFloat64(b)
	if divisor == 0 {
		return 0
	}
	return toFloat64(a) / divisor
}

func (e *Engine) mulFunc(a, b interface{}) float64 {
	return toFloat64(a) * toFloat64(b)
}

func (e *Engine) subFunc(a, b interface{}) float64 {
	return toFloat64(a) - toFloat64(b)
}

func (e *Engine) minFunc(values ...interface{}) float64 {
	if len(values) == 0 {
		return 0
	}
	min := toFloat64(values[0])
	for _, v := range values[1:] {
		val := toFloat64(v)
		if val < min {
			min = val
		}
	}
	return min
}

func (e *Engine) maxFunc(values ...interface{}) float64 {
	if len(values) == 0 {
		return 0
	}
	max := toFloat64(values[0])
	for _, v := range values[1:] {
		val := toFloat64(v)
		if val > max {
			max = val
		}
	}
	return max
}

// Time functions
func (e *Engine) agoFunc(t interface{}) string {
	var ts time.Time
	switch v := t.(type) {
	case time.Time:
		ts = v
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return "unknown"
		}
		ts = parsed
	default:
		return "unknown"
	}

	duration := time.Since(ts)
	return e.humanizeDurationFunc(duration)
}

func (e *Engine) ageInSecondsFunc(t interface{}) float64 {
	var ts time.Time
	switch v := t.(type) {
	case time.Time:
		ts = v
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return 0
		}
		ts = parsed
	default:
		return 0
	}
	return time.Since(ts).Seconds()
}

func (e *Engine) timestampFunc(t interface{}) string {
	var ts time.Time
	switch v := t.(type) {
	case time.Time:
		ts = v
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return "unknown"
		}
		ts = parsed
	default:
		return "unknown"
	}
	return ts.Format("2006-01-02 15:04:05")
}

// String functions - handles both (slice, sep) and (sep, slice) signatures
func (e *Engine) joinFunc(arg1 interface{}, arg2 interface{}) string {
	// Try to determine which argument is the slice and which is the separator
	var slice interface{}
	var separator string

	// Check if arg1 is a string (separator)
	if sep, ok := arg1.(string); ok {
		separator = sep
		slice = arg2
	} else {
		// arg1 is the slice, arg2 should be the separator
		slice = arg1
		if sep, ok := arg2.(string); ok {
			separator = sep
		} else {
			separator = fmt.Sprintf("%v", arg2)
		}
	}

	switch v := slice.(type) {
	case []string:
		return strings.Join(v, separator)
	case []interface{}:
		var strs []string
		for _, item := range v {
			strs = append(strs, fmt.Sprintf("%v", item))
		}
		return strings.Join(strs, separator)
	default:
		return fmt.Sprintf("%v", slice)
	}
}

// addFunc adds multiple numbers together
func (e *Engine) addFunc(values ...interface{}) float64 {
	sum := 0.0
	for _, v := range values {
		sum += toFloat64(v)
	}
	return sum
}

// lenFunc returns the length of a string, slice, or map
func (e *Engine) lenFunc(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case string:
		return len(val)
	case []interface{}:
		return len(val)
	case []string:
		return len(val)
	case map[string]interface{}:
		return len(val)
	case map[string]string:
		return len(val)
	default:
		// Try reflection for other slice types
		return 0
	}
}

// toStringFunc converts any value to a string
func (e *Engine) toStringFunc(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		// Format float without unnecessary decimals
		if val == float64(int(val)) {
			return strconv.Itoa(int(val))
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// listFunc creates a new list from the given values
func (e *Engine) listFunc(values ...interface{}) []interface{} {
	return values
}

// appendFunc appends values to a list
func (e *Engine) appendFunc(list interface{}, values ...interface{}) []interface{} {
	var result []interface{}

	// Convert list to []interface{}
	switch l := list.(type) {
	case []interface{}:
		result = l
	case []string:
		for _, v := range l {
			result = append(result, v)
		}
	default:
		// If not a list, start with empty
		result = []interface{}{}
	}

	// Append new values
	result = append(result, values...)
	return result
}

// sliceFunc returns a slice of a list from start to end (exclusive)
func (e *Engine) sliceFunc(list interface{}, indices ...int) []interface{} {
	var items []interface{}

	// Convert list to []interface{}
	switch l := list.(type) {
	case []interface{}:
		items = l
	case []string:
		for _, v := range l {
			items = append(items, v)
		}
	default:
		return []interface{}{}
	}

	// Parse indices
	start := 0
	end := len(items)

	if len(indices) > 0 {
		start = indices[0]
		if start < 0 {
			start = 0
		}
		if start > len(items) {
			start = len(items)
		}
	}

	if len(indices) > 1 {
		end = indices[1]
		if end < start {
			end = start
		}
		if end > len(items) {
			end = len(items)
		}
	}

	return items[start:end]
}

// defaultFunc returns the default value if the given value is empty
func (e *Engine) defaultFunc(defaultVal, val interface{}) interface{} {
	// Check if val is empty/nil/zero
	if val == nil {
		return defaultVal
	}

	switch v := val.(type) {
	case string:
		if v == "" {
			return defaultVal
		}
	case int:
		if v == 0 {
			return defaultVal
		}
	case int64:
		if v == 0 {
			return defaultVal
		}
	case float64:
		if v == 0 {
			return defaultVal
		}
	case bool:
		if !v {
			return defaultVal
		}
	case []interface{}:
		if len(v) == 0 {
			return defaultVal
		}
	case []string:
		if len(v) == 0 {
			return defaultVal
		}
	case map[string]interface{}:
		if len(v) == 0 {
			return defaultVal
		}
	}

	return val
}

// Helper function to convert interface to float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

// Validate checks if a template is valid
func (e *Engine) Validate(tmplStr string) error {
	_, err := template.New("validate").Funcs(e.funcMap).Parse(tmplStr)
	return err
}

// LoadTemplate loads a named template
func (e *Engine) LoadTemplate(name, tmplStr string) error {
	tmpl, err := template.New(name).Funcs(e.funcMap).Parse(tmplStr)
	if err != nil {
		return err
	}

	e.mu.Lock()
	e.templates[name] = tmpl
	e.mu.Unlock()

	return nil
}

// ExecuteNamed executes a named template
func (e *Engine) ExecuteNamed(name string, data interface{}) (string, error) {
	e.mu.RLock()
	tmpl, ok := e.templates[name]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
