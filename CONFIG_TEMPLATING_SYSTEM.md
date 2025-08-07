# Kubewatch Template Formatting System

## Overview
A domain-specific template language for formatting Kubernetes resource data, inspired by Go templates but optimized for terminal UI formatting with colors, icons, and conditional logic.

## Template Syntax

### Basic Syntax
```template
# Simple field reference
{{ .Status }}

# With formatting function
{{ .CPU | bytes }}

# With color
{{ .Status | color "green" }}

# Conditional coloring
{{ .Status | colorIf (eq . "Running") "green" "red" }}

# Complex expression
{{ if gt .Restarts 5 }}⚠️ {{ .Restarts }}{{ else }}{{ .Restarts }}{{ end }}
```

### Template Language Features

#### 1. Field Access
```template
# Direct field
{{ .Name }}

# Nested field (using JSONPath-like syntax)
{{ .metadata.labels.app }}

# Array access
{{ .containers[0].name }}

# Map access with default
{{ .labels["version"] | default "unknown" }}
```

#### 2. Conditionals
```template
# Simple if
{{ if .Ready }}✓{{ else }}✗{{ end }}

# If-else-if chains
{{ if eq .Status "Running" }}
  {{ color "green" "●" }}
{{ else if eq .Status "Pending" }}
  {{ color "yellow" "◐" }}
{{ else if eq .Status "Failed" }}
  {{ color "red" "✗" }}
{{ else }}
  {{ color "gray" "○" }}
{{ end }}

# Ternary-like using choose
{{ choose (gt .CPU 80) "🔥" "✓" }}
```

#### 3. Loops
```template
# Iterate over containers
{{ range .containers }}
  {{ .name }}:{{ .image | truncate 20 }}
{{ end | join ", " }}

# With index
{{ range $i, $container := .containers }}
  {{ if $i }}, {{ end }}{{ $container.name }}
{{ end }}
```

#### 4. Variables
```template
# Define variables
{{ $memoryMB := .Memory | toMB }}
{{ $threshold := 512 }}

{{ if gt $memoryMB $threshold }}
  {{ color "red" $memoryMB }}MB
{{ else }}
  {{ color "green" $memoryMB }}MB
{{ end }}
```

## Built-in Functions

### Formatting Functions

#### Memory/Storage
```template
{{ .Memory | bytes }}           # 1.5Gi
{{ .Memory | kilobytes }}        # 1572864Ki  
{{ .Memory | megabytes }}        # 1536Mi
{{ .Memory | gigabytes }}        # 1.5Gi
{{ .Memory | humanize }}         # 1.5Gi (auto-scales)
{{ .Memory | toMB }}             # 1536 (numeric value)
```

#### CPU
```template
{{ .CPU | millicores }}          # 250m
{{ .CPU | cores }}               # 0.25
{{ .CPU | percentage }}          # 25%
```

#### Time/Duration
```template
{{ .Age | duration }}            # 2h30m
{{ .Age | humanize }}            # 2 hours ago
{{ .Age | relative }}            # 2 hours ago
{{ .Age | timestamp }}           # 2024-01-15 10:30:45
{{ .Age | date "15:04" }}        # 10:30
```

#### Numbers
```template
{{ .Count | int }}               # 42
{{ .Ratio | float }}             # 0.75
{{ .Ratio | percent }}           # 75%
{{ .Ratio | fixed 2 }}           # 0.75
{{ .Value | round }}             # 43
{{ .Value | ceil }}              # 43
{{ .Value | floor }}             # 42
```

#### Strings
```template
{{ .Name | upper }}              # NGINX-POD
{{ .Name | lower }}              # nginx-pod
{{ .Name | title }}              # Nginx-Pod
{{ .Name | truncate 10 }}        # nginx-p...
{{ .Name | truncateMiddle 15 }}  # nginx...pod
{{ .Name | padLeft 10 }}         #    nginx-pod
{{ .Name | padRight 10 }}        # nginx-pod   
{{ .Name | trim }}               # nginx-pod
{{ .Name | replace "-" "_" }}    # nginx_pod
{{ .Name | regexp "^nginx-" "" }} # pod
```

### Comparison Functions
```template
{{ eq .Status "Running" }}       # Equality
{{ ne .Count 0 }}                # Not equal
{{ lt .CPU 100 }}                # Less than
{{ le .CPU 100 }}                # Less than or equal
{{ gt .Memory 512 }}             # Greater than
{{ ge .Memory 512 }}             # Greater than or equal
{{ contains .Name "nginx" }}     # String contains
{{ hasPrefix .Name "nginx-" }}   # String prefix
{{ hasSuffix .Name "-pod" }}     # String suffix
{{ matches .Name "^nginx-.*" }}  # Regex match
{{ in .Status "Running" "Ready" }} # Value in list
```

### Logical Functions
```template
{{ and .Ready .Available }}      # Logical AND
{{ or .Error .Warning }}         # Logical OR
{{ not .Disabled }}              # Logical NOT
```

### Color Functions
```template
# Basic colors
{{ color "red" "text" }}
{{ color "green" "✓" }}
{{ color "yellow" "⚠" }}
{{ color "blue" "ℹ" }}
{{ color "magenta" "◆" }}
{{ color "cyan" "◉" }}
{{ color "white" "text" }}
{{ color "gray" "text" }}

# Bright colors
{{ color "brightRed" "!" }}
{{ color "brightGreen" "✓" }}

# RGB colors
{{ color "#FF5733" "text" }}
{{ rgb 255 87 51 "text" }}

# Background colors
{{ bg "red" "text" }}
{{ bg "#FF5733" "text" }}

# Combined styling
{{ style "red" "bold" "text" }}
{{ style "#FF5733" "bold,underline" "text" }}

# Conditional coloring
{{ colorIf (gt .CPU 80) "red" "green" .CPU }}

# Gradient coloring (based on value ranges)
{{ gradient .CPU 0 100 "green" "yellow" "red" }}
```

### Icon Functions
```template
# Status icons
{{ icon "success" }}             # ✓
{{ icon "error" }}               # ✗
{{ icon "warning" }}             # ⚠
{{ icon "info" }}                # ℹ
{{ icon "running" }}             # ●
{{ icon "pending" }}             # ◐
{{ icon "stopped" }}             # ■

# Resource icons (Nerd Fonts)
{{ icon "pod" }}                 # 
{{ icon "deployment" }}          # 
{{ icon "service" }}             # 
{{ icon "configmap" }}           # 
{{ icon "secret" }}              # 

# Conditional icons
{{ iconIf .Ready "success" "error" }}
```

### Utility Functions
```template
# Default values
{{ .MissingField | default "N/A" }}

# First non-empty value
{{ coalesce .PreferredName .Name .ID }}

# JSON path query
{{ jsonPath . ".spec.containers[?(@.name=='nginx')].image" }}

# Join arrays
{{ .Labels | join ", " }}

# Length
{{ len .Containers }}

# Math
{{ add .Requested .Buffer }}
{{ sub .Limit .Used }}
{{ mul .Cores 1000 }}            # Convert to millicores
{{ div .Total .Count }}
{{ mod .Value 10 }}
{{ max .Requested .Used }}
{{ min .Limit 1000 }}

# Type conversion
{{ toString .Number }}
{{ toInt .String }}
{{ toFloat .String }}
{{ toBool .String }}
```

## Column Definition Examples

### Pod Columns Configuration
```yaml
# ~/.config/kubewatch/columns/pods.yaml
apiVersion: kubewatch.io/v1
kind: ColumnConfig
metadata:
  resourceType: Pod
spec:
  columns:
    - name: STATUS
      width: 12
      template: |
        {{ if eq .Status.Phase "Running" }}
          {{- color "green" "●" }} Running
        {{ else if eq .Status.Phase "Pending" }}
          {{- color "yellow" "◐" }} Pending
        {{ else if eq .Status.Phase "Failed" }}
          {{- color "red" "✗" }} Failed
        {{ else }}
          {{- color "gray" "○" }} {{ .Status.Phase }}
        {{ end }}
        
    - name: READY
      width: 8
      align: center
      template: |
        {{ $ready := 0 }}
        {{ $total := len .Status.ContainerStatuses }}
        {{ range .Status.ContainerStatuses }}
          {{ if .Ready }}{{ $ready = add $ready 1 }}{{ end }}
        {{ end }}
        {{ if eq $ready $total }}
          {{- color "green" (printf "%d/%d" $ready $total) }}
        {{ else }}
          {{- color "yellow" (printf "%d/%d" $ready $total) }}
        {{ end }}
        
    - name: RESTARTS
      width: 10
      align: right
      template: |
        {{ $restarts := 0 }}
        {{ range .Status.ContainerStatuses }}
          {{ $restarts = add $restarts .RestartCount }}
        {{ end }}
        {{ if eq $restarts 0 }}
          {{- color "gray" "0" }}
        {{ else if lt $restarts 5 }}
          {{- color "yellow" $restarts }}
        {{ else }}
          {{- color "red" (printf "⚠ %d" $restarts) }}
        {{ end }}
        
    - name: CPU
      width: 10
      align: right
      template: |
        {{ $cpu := .Metrics.CPU | toMillicores }}
        {{ $requested := .Spec.Containers[0].Resources.Requests.cpu | toMillicores }}
        {{ $percent := 0 }}
        {{ if gt $requested 0 }}
          {{ $percent = div (mul $cpu 100) $requested }}
        {{ end }}
        {{ gradient $percent 0 100 "green" "yellow" "red" (printf "%dm" $cpu) }}
        
    - name: MEMORY  
      width: 10
      align: right
      template: |
        {{ $mem := .Metrics.Memory | toMB }}
        {{ if lt $mem 128 }}
          {{- color "green" (.Metrics.Memory | humanize) }}
        {{ else if lt $mem 512 }}
          {{- color "yellow" (.Metrics.Memory | humanize) }}
        {{ else }}
          {{- color "red" (.Metrics.Memory | humanize) }}
        {{ end }}
        
    - name: EFFICIENCY
      width: 12
      computed: true
      template: |
        {{ $cpuUsed := .Metrics.CPU | toMillicores }}
        {{ $cpuRequested := .Spec.Containers[0].Resources.Requests.cpu | toMillicores }}
        {{ if and (gt $cpuUsed 0) (gt $cpuRequested 0) }}
          {{ $efficiency := div (mul $cpuUsed 100) $cpuRequested }}
          {{ if lt $efficiency 20 }}
            {{- color "blue" (printf "↓ %d%%" $efficiency) }}
          {{ else if gt $efficiency 80 }}
            {{- color "yellow" (printf "↑ %d%%" $efficiency) }}
          {{ else }}
            {{- color "green" (printf "● %d%%" $efficiency) }}
          {{ end }}
        {{ else }}
          {{- color "gray" "N/A" }}
        {{ end }}
        
    - name: LABELS
      width: 30
      template: |
        {{ $important := list "app" "version" "env" }}
        {{ $labels := list }}
        {{ range $key, $value := .Metadata.Labels }}
          {{ if in $key $important }}
            {{ $labels = append $labels (printf "%s=%s" $key $value) }}
          {{ end }}
        {{ end }}
        {{ if gt (len $labels) 3 }}
          {{- join (slice $labels 0 3) ", " }}...
        {{ else }}
          {{- join $labels ", " }}
        {{ end }}
```

### Service Columns Configuration
```yaml
columns:
  - name: TYPE
    width: 12
    template: |
      {{ if eq .Spec.Type "LoadBalancer" }}
        {{- color "blue" "🌐" }} {{ .Spec.Type }}
      {{ else if eq .Spec.Type "NodePort" }}
        {{- color "cyan" "📡" }} {{ .Spec.Type }}
      {{ else if eq .Spec.Type "ClusterIP" }}
        {{- color "green" "🔒" }} {{ .Spec.Type }}
      {{ else }}
        {{ .Spec.Type }}
      {{ end }}
      
  - name: ENDPOINTS
    width: 20
    template: |
      {{ $count := len .Endpoints.Subsets }}
      {{ if eq $count 0 }}
        {{- color "red" "No endpoints" }}
      {{ else }}
        {{ $ready := 0 }}
        {{ range .Endpoints.Subsets }}
          {{ $ready = add $ready (len .Addresses) }}
        {{ end }}
        {{- color "green" (printf "%d endpoints" $ready) }}
      {{ end }}
```

## Advanced Templates

### Progress Bar Template
```template
{{ define "progressBar" }}
  {{ $width := 10 }}
  {{ $filled := div (mul .Value $width) .Max }}
  {{ $empty := sub $width $filled }}
  {{ repeat "█" $filled }}{{ repeat "░" $empty }}
{{ end }}

# Usage
{{ template "progressBar" (dict "Value" .CPU "Max" 100) }}
```

### Sparkline Template
```template
{{ define "sparkline" }}
  {{ $chars := list "▁" "▂" "▃" "▄" "▅" "▆" "▇" "█" }}
  {{ range .Values }}
    {{ $index := div (mul . 7) .Max }}
    {{ index $chars $index }}
  {{ end }}
{{ end }}

# Usage
{{ template "sparkline" (dict "Values" .CPUHistory "Max" 100) }}
```

### Status Badge Template
```template
{{ define "statusBadge" }}
  {{ $color := "gray" }}
  {{ $icon := "○" }}
  {{ if eq .Status "Running" }}
    {{ $color = "green" }}
    {{ $icon = "●" }}
  {{ else if eq .Status "Error" }}
    {{ $color = "red" }}
    {{ $icon = "✗" }}
  {{ end }}
  {{ style $color "bold" (printf "[%s %s]" $icon .Status) }}
{{ end }}
```

## Implementation

### Template Engine
```go
// internal/config/template/engine.go
package template

import (
    "text/template"
    "github.com/Masterminds/sprig/v3"
)

type Engine struct {
    funcMap    template.FuncMap
    templates  map[string]*template.Template
    cache      *TemplateCache
}

func NewEngine() *Engine {
    e := &Engine{
        funcMap:   make(template.FuncMap),
        templates: make(map[string]*template.Template),
    }
    e.registerBuiltinFuncs()
    return e
}

func (e *Engine) registerBuiltinFuncs() {
    // Add sprig functions as base
    for k, v := range sprig.TxtFuncMap() {
        e.funcMap[k] = v
    }
    
    // Add our custom functions
    e.funcMap["color"] = e.colorFunc
    e.funcMap["gradient"] = e.gradientFunc
    e.funcMap["icon"] = e.iconFunc
    e.funcMap["humanize"] = e.humanizeFunc
    e.funcMap["bytes"] = e.bytesFunc
    e.funcMap["millicores"] = e.millicoresFunc
    // ... more custom functions
}

func (e *Engine) Execute(tmpl string, data interface{}) (FormattedValue, error) {
    t, err := e.getOrParseTemplate(tmpl)
    if err != nil {
        return FormattedValue{}, err
    }
    
    var buf bytes.Buffer
    if err := t.Execute(&buf, data); err != nil {
        return FormattedValue{}, err
    }
    
    return e.parseFormattedOutput(buf.String()), nil
}
```

### Formatter Functions
```go
// internal/config/template/formatters.go

func (e *Engine) colorFunc(color, text string) string {
    return fmt.Sprintf("\x1b[color:%s]%s\x1b[reset]", color, text)
}

func (e *Engine) gradientFunc(value, min, max float64, colors ...string) string {
    // Calculate which color to use based on value position
    position := (value - min) / (max - min)
    colorIndex := int(position * float64(len(colors)-1))
    if colorIndex >= len(colors) {
        colorIndex = len(colors) - 1
    }
    return e.colorFunc(colors[colorIndex], fmt.Sprintf("%.0f", value))
}

func (e *Engine) humanizeBytes(bytes int64) string {
    units := []string{"B", "Ki", "Mi", "Gi", "Ti"}
    value := float64(bytes)
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
```

### Template Validation
```go
// internal/config/template/validator.go

type Validator struct {
    engine *Engine
}

func (v *Validator) Validate(tmpl string) error {
    // Parse template
    _, err := template.New("validate").Funcs(v.engine.funcMap).Parse(tmpl)
    if err != nil {
        return fmt.Errorf("template syntax error: %w", err)
    }
    
    // Check for required functions
    if err := v.checkFunctions(tmpl); err != nil {
        return err
    }
    
    // Validate color codes
    if err := v.validateColors(tmpl); err != nil {
        return err
    }
    
    return nil
}

func (v *Validator) ValidateColumnConfig(config *ColumnConfig) error {
    for _, col := range config.Columns {
        if col.Template != "" {
            if err := v.Validate(col.Template); err != nil {
                return fmt.Errorf("column %s: %w", col.Name, err)
            }
        }
    }
    return nil
}
```

### Template Caching
```go
// internal/config/template/cache.go

type TemplateCache struct {
    compiled map[string]*template.Template
    results  map[string]map[uint64]FormattedValue // LRU cache of results
    mu       sync.RWMutex
}

func (c *TemplateCache) Get(tmpl string, data interface{}) (FormattedValue, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    hash := hashData(data)
    if results, ok := c.results[tmpl]; ok {
        if value, ok := results[hash]; ok {
            return value, true
        }
    }
    return FormattedValue{}, false
}
```

## Benefits Over Lua/Starlark

1. **Familiar Syntax**: Similar to Go templates, Helm charts, Hugo
2. **Type Safety**: Templates are validated at load time
3. **Performance**: Compiled templates with caching
4. **Security**: No arbitrary code execution
5. **Declarative**: Easier to understand and audit
6. **IDE Support**: Better syntax highlighting and completion
7. **Testability**: Templates can be unit tested easily

## Migration Examples

### From Current Hard-coded
```go
// Before: Hard-coded in Go
if pod.Status.Phase == "Running" {
    style = style.Foreground(lipgloss.Color("2"))
} else if pod.Status.Phase == "Pending" {
    style = style.Foreground(lipgloss.Color("3"))
}

// After: Template
{{ if eq .Status.Phase "Running" }}
  {{ color "green" .Status.Phase }}
{{ else if eq .Status.Phase "Pending" }}
  {{ color "yellow" .Status.Phase }}
{{ end }}
```

### From Lua
```lua
-- Before: Lua formatter
function format_memory(value)
    local mb = parse_memory_to_mb(value)
    if mb > 512 then
        return {text = humanize(value), color = "red"}
    else
        return {text = humanize(value), color = "green"}
    end
end

-- After: Template
{{ $mb := .Memory | toMB }}
{{ if gt $mb 512 }}
  {{ color "red" (.Memory | humanize) }}
{{ else }}
  {{ color "green" (.Memory | humanize) }}
{{ end }}
```

## Conclusion

This template-based formatting system provides:
- **Power**: Full conditional logic, loops, and functions
- **Safety**: No arbitrary code execution
- **Familiarity**: Similar to existing Go/Helm templates
- **Performance**: Compiled and cached templates
- **Extensibility**: Easy to add new functions
- **User-friendly**: More approachable than scripting languages