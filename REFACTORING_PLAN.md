# Kubewatch View Layer Refactoring Plan with User Configuration System

## Executive Summary
The view layer, particularly `resource_view.go` (2203 lines), has grown too large and contains significant duplication. This plan outlines a systematic approach to reduce complexity, improve maintainability, establish better separation of concerns, and **introduce a powerful user configuration system** for customization and extensibility.

## Current State Analysis

### File Sizes
- **resource_view.go**: 2203 lines (45 functions)
- **log_view.go**: 805 lines
- **context_view.go**: 294 lines
- **data_view.go**: 282 lines
- **namespace_view.go**: 262 lines
- **describe_view.go**: 249 lines
- **app.go**: 908 lines
- **modes.go**: 573 lines

### Key Issues Identified

1. **Monolithic ResourceView struct** - Handles too many responsibilities
2. **Duplicated Patterns** - 8 similar update methods, repeated styling logic
3. **Mixed Concerns** - Business logic mixed with presentation
4. **Lack of Abstraction** - No shared components or configuration
5. **No User Customization** - Hard-coded columns and formatting

## Configuration System Design

### Configuration Directory Structure
```
~/.config/kubewatch/
├── config.yaml                 # Main configuration
├── themes/                      # Custom themes
│   ├── default.yaml
│   ├── solarized.yaml
│   └── custom.yaml
├── columns/                     # Column definitions
│   ├── pods.yaml
│   ├── deployments.yaml
│   └── custom/                 # User-defined resource types
│       └── cronjobs.yaml
├── formatters/                  # Custom formatters (Lua/Starlark scripts)
│   ├── memory.lua
│   ├── cpu.lua
│   └── custom.lua
├── filters/                     # Saved filter presets
│   └── production.yaml
└── layouts/                     # Saved view layouts
    └── monitoring.yaml
```

### Core Configuration Features

#### 1. Column Configuration
**File**: `~/.config/kubewatch/columns/pods.yaml`
```yaml
apiVersion: kubewatch.io/v1
kind: ColumnConfig
metadata:
  resourceType: Pod
spec:
  # Define which columns are visible and their order
  columns:
    - name: NAME
      visible: true
      width: auto
      minWidth: 20
      maxWidth: 50
      priority: 1  # For responsive hiding
      
    - name: NAMESPACE
      visible: true
      condition: "namespace == 'all'"  # Conditional visibility
      width: 15
      priority: 2
      
    - name: STATUS
      visible: true
      width: 12
      formatter: status  # Use built-in formatter
      priority: 1
      
    - name: CPU
      visible: true
      width: 8
      align: right
      formatter: cpu_usage  # Custom formatter
      priority: 3
      
    - name: MEMORY
      visible: true
      width: 10
      align: right
      formatter: memory_usage
      priority: 3
      
    # Custom column definition
    - name: LABELS
      visible: false  # Hidden by default
      width: 30
      source: "metadata.labels"  # JSONPath to data
      formatter: label_list
      priority: 4
      
    # Computed column
    - name: EFFICIENCY
      visible: true
      width: 10
      computed: true
      expression: "cpu_request > 0 ? cpu_usage / cpu_request * 100 : 0"
      formatter: percentage
      priority: 3
      
  # Default sort configuration
  defaultSort:
    column: NAME
    ascending: true
    
  # Grouping configuration
  groupBy:
    enabled: false
    column: NAMESPACE
    collapsed: false
```

#### 2. Custom Formatters
**File**: `~/.config/kubewatch/formatters/custom.lua`
```lua
-- Custom formatter for memory with color coding
function format_memory_usage(value, row, config)
    local num = parse_memory(value)
    local color = "green"
    
    if num > config.thresholds.high then
        color = "red"
    elseif num > config.thresholds.medium then
        color = "yellow"
    end
    
    -- Return formatted value with color hint
    return {
        text = humanize_memory(num),
        color = color,
        bold = num > config.thresholds.critical
    }
end

-- Custom formatter for label display
function format_label_list(labels, row, config)
    local important = {"app", "version", "env"}
    local result = {}
    
    for _, key in ipairs(important) do
        if labels[key] then
            table.insert(result, key .. "=" .. labels[key])
        end
    end
    
    if #result > 3 then
        return table.concat(result, ",", 1, 3) .. "..."
    end
    return table.concat(result, ",")
end

-- Formatter for percentage with bar graph
function format_percentage_bar(value, row, config)
    local width = config.width or 10
    local filled = math.floor(value / 100 * width)
    local bar = string.rep("█", filled) .. string.rep("░", width - filled)
    
    return {
        text = string.format("%3d%% %s", value, bar),
        color = value > 80 and "red" or value > 60 and "yellow" or "green"
    }
end
```

#### 3. Theme Configuration
**File**: `~/.config/kubewatch/themes/custom.yaml`
```yaml
apiVersion: kubewatch.io/v1
kind: Theme
metadata:
  name: custom
spec:
  colors:
    # Base colors
    background: "#1e1e1e"
    foreground: "#d4d4d4"
    selection:
      background: "#264f78"
      foreground: "#ffffff"
    
    # Status colors
    status:
      running: "#4ec9b0"
      pending: "#dcdcaa"
      failed: "#f44747"
      completed: "#569cd6"
      terminating: "#c586c0"
      
    # Metric colors (gradients)
    metrics:
      cpu:
        low: "#4ec9b0"
        medium: "#dcdcaa"
        high: "#ce9178"
        critical: "#f44747"
      memory:
        low: "#4ec9b0"
        medium: "#dcdcaa"
        high: "#ce9178"
        critical: "#f44747"
        
    # UI elements
    borders:
      normal: "#3c3c3c"
      focused: "#007acc"
    headers:
      background: "#2d2d2d"
      foreground: "#cccccc"
      
  # Typography
  fonts:
    mono: "JetBrains Mono"
    ui: "Inter"
    
  # Icons (optional, for Nerd Font users)
  icons:
    pod: ""
    deployment: ""
    service: ""
    configmap: ""
    secret: ""
    namespace: ""
    context: ""
```

#### 4. Filter Presets
**File**: `~/.config/kubewatch/filters/production.yaml`
```yaml
apiVersion: kubewatch.io/v1
kind: FilterPreset
metadata:
  name: production
spec:
  filters:
    - type: namespace
      pattern: "prod-*"
      
    - type: label
      key: environment
      value: production
      
    - type: status
      exclude: ["Completed", "Succeeded"]
      
    - type: age
      operator: "<"
      value: "7d"
      
    - type: custom
      expression: "cpu_usage > 100m || memory_usage > 128Mi"
      
  # Quick filters (accessible via hotkeys)
  quickFilters:
    - key: "1"
      name: "Errors only"
      expression: "status in ['Failed', 'Error', 'CrashLoopBackOff']"
      
    - key: "2"
      name: "High CPU"
      expression: "cpu_usage > cpu_request * 0.8"
      
    - key: "3"
      name: "Recent"
      expression: "age < 1h"
```

#### 5. View Layouts
**File**: `~/.config/kubewatch/layouts/monitoring.yaml`
```yaml
apiVersion: kubewatch.io/v1
kind: Layout
metadata:
  name: monitoring
spec:
  splitView:
    enabled: true
    orientation: horizontal  # or vertical
    ratio: 0.6
    
  panels:
    - type: resource
      position: main
      config:
        resourceType: Pod
        columns: ["NAME", "STATUS", "CPU", "MEMORY", "RESTARTS"]
        autoRefresh: 5s
        
    - type: logs
      position: secondary
      config:
        follow: true
        timestamps: true
        wrap: true
        
    - type: metrics
      position: bottom
      height: 20%
      config:
        graphs:
          - metric: cpu
            sparkline: true
          - metric: memory
            sparkline: true
```

### Configuration API

#### Configuration Manager
**New file**: `internal/config/manager.go`
```go
package config

type ConfigManager struct {
    configDir    string
    cache        *ConfigCache
    watchers     []ConfigWatcher
    scriptEngine *ScriptEngine
}

type Config struct {
    Columns    map[string]*ColumnConfig
    Themes     map[string]*Theme
    Formatters map[string]Formatter
    Filters    map[string]*FilterPreset
    Layouts    map[string]*Layout
    Settings   *UserSettings
}

func NewConfigManager() *ConfigManager {
    cm := &ConfigManager{
        configDir: getConfigDir(),
        cache:     NewConfigCache(),
    }
    cm.scriptEngine = NewScriptEngine(cm)
    return cm
}

func (cm *ConfigManager) Load() error
func (cm *ConfigManager) Save() error
func (cm *ConfigManager) Watch() error
func (cm *ConfigManager) GetColumnConfig(resourceType string) *ColumnConfig
func (cm *ConfigManager) RegisterFormatter(name string, formatter Formatter)
func (cm *ConfigManager) ApplyTheme(name string) error
```

#### Column Definition System
**New file**: `internal/config/columns.go`
```go
package config

type ColumnDefinition struct {
    Name       string
    Visible    bool
    Width      ColumnWidth
    Priority   int
    Align      Alignment
    Formatter  string
    Source     string // JSONPath for custom columns
    Computed   bool
    Expression string // For computed columns
    Condition  string // For conditional visibility
}

type ColumnConfig struct {
    ResourceType string
    Columns      []*ColumnDefinition
    DefaultSort  SortConfig
    GroupBy      *GroupConfig
}

func (cc *ColumnConfig) GetVisibleColumns() []*ColumnDefinition
func (cc *ColumnConfig) ReorderColumns(order []string)
func (cc *ColumnConfig) ToggleColumn(name string)
func (cc *ColumnConfig) AddCustomColumn(def *ColumnDefinition)
```

#### Formatter System
**New file**: `internal/config/formatters.go`
```go
package config

type Formatter interface {
    Format(value interface{}, row map[string]interface{}, config FormatConfig) FormattedValue
}

type FormattedValue struct {
    Text      string
    Color     string
    Bold      bool
    Italic    bool
    Underline bool
    Icon      string
}

type ScriptFormatter struct {
    engine   *ScriptEngine
    function string
}

type FormatConfig struct {
    Width      int
    Thresholds map[string]float64
    Options    map[string]interface{}
}

// Built-in formatters
type CPUFormatter struct{}
type MemoryFormatter struct{}
type DurationFormatter struct{}
type PercentageFormatter struct{}
type StatusFormatter struct{}

// Formatter registry
type FormatterRegistry struct {
    formatters map[string]Formatter
    scripts    map[string]*ScriptFormatter
}

func (fr *FormatterRegistry) Register(name string, formatter Formatter)
func (fr *FormatterRegistry) LoadScript(name, path string) error
func (fr *FormatterRegistry) Get(name string) Formatter
```

#### Script Engine (Lua/Starlark)
**New file**: `internal/config/scripting.go`
```go
package config

type ScriptEngine struct {
    lua      *lua.LState
    starlark *starlark.Thread
    funcs    map[string]interface{}
}

func NewScriptEngine(config *ConfigManager) *ScriptEngine
func (se *ScriptEngine) LoadScript(path string) error
func (se *ScriptEngine) Execute(function string, args ...interface{}) (interface{}, error)
func (se *ScriptEngine) RegisterFunction(name string, fn interface{})

// Helper functions exposed to scripts
func (se *ScriptEngine) RegisterHelpers() {
    se.RegisterFunction("parse_memory", ParseMemory)
    se.RegisterFunction("parse_cpu", ParseCPU)
    se.RegisterFunction("humanize_duration", HumanizeDuration)
    se.RegisterFunction("get_color", GetThemeColor)
}
```

### Integration with Refactored Components

#### Enhanced Table Component
```go
type Table struct {
    config       *ColumnConfig
    formatters   *FormatterRegistry
    columns      []*ColumnDefinition
    // ... existing fields
}

func (t *Table) ApplyColumnConfig(config *ColumnConfig)
func (t *Table) RenderWithFormatters(width, height int) string
func (t *Table) AddCustomColumn(def *ColumnDefinition)
func (t *Table) SaveColumnState() // Persist column widths, order
```

#### Enhanced ResourceView
```go
type ResourceView struct {
    configManager *config.ConfigManager
    columnConfig  *config.ColumnConfig
    theme         *config.Theme
    formatters    *config.FormatterRegistry
    // ... existing fields
}

func (v *ResourceView) LoadUserConfig() error
func (v *ResourceView) ApplyFilter(preset string)
func (v *ResourceView) SaveLayout(name string)
func (v *ResourceView) LoadLayout(name string)
```

### User Interaction Features

#### 1. Interactive Column Management
- **`c`** - Open column selector (toggle visibility)
- **`C`** - Column configuration mode (resize, reorder)
- **`Alt+←/→`** - Reorder columns
- **`+/-`** - Increase/decrease column width

#### 2. Dynamic Filtering
- **`/`** - Quick filter input
- **`F`** - Load filter preset
- **`Ctrl+F`** - Advanced filter builder

#### 3. Theme Switching
- **`t`** - Cycle through themes
- **`T`** - Theme selector

#### 4. Layout Management
- **`L`** - Load saved layout
- **`Ctrl+S`** - Save current layout

### Configuration Examples

#### Example: DevOps Dashboard Configuration
```yaml
# ~/.config/kubewatch/config.yaml
apiVersion: kubewatch.io/v1
kind: UserConfig
spec:
  defaultTheme: solarized-dark
  defaultLayout: monitoring
  
  autoRefresh:
    enabled: true
    interval: 5s
    
  shortcuts:
    - key: "p"
      action: "filter:production"
    - key: "d"
      action: "filter:development"
    - key: "m"
      action: "layout:monitoring"
      
  plugins:
    - name: prometheus-metrics
      enabled: true
      config:
        endpoint: http://prometheus:9090
        
  extensions:
    - path: ~/.config/kubewatch/extensions/custom-resources.lua
    - path: ~/.config/kubewatch/extensions/alert-handler.py
```

#### Example: Custom Resource Type
```yaml
# ~/.config/kubewatch/columns/custom/tekton-pipelineruns.yaml
apiVersion: kubewatch.io/v1
kind: ColumnConfig
metadata:
  resourceType: PipelineRun
  apiGroup: tekton.dev/v1beta1
spec:
  columns:
    - name: NAME
      source: metadata.name
    - name: PIPELINE
      source: spec.pipelineRef.name
    - name: STATUS
      source: status.conditions[0].reason
      formatter: tekton_status
    - name: DURATION
      computed: true
      expression: "status.completionTime - status.startTime"
      formatter: duration
    - name: TRIGGER
      source: metadata.labels["triggers.tekton.dev/trigger"]
```

### Migration Path

1. **Phase 1**: Implement configuration system alongside refactoring
2. **Phase 2**: Provide default configurations matching current behavior
3. **Phase 3**: Add UI for configuration management
4. **Phase 4**: Document and share community configurations

### Benefits of Configuration System

1. **User Empowerment**: Users can customize without code changes
2. **Extensibility**: Support for custom resources and CRDs
3. **Productivity**: Save and share configurations for different workflows
4. **Accessibility**: Themes for different visual needs
5. **Integration**: Script custom behaviors and integrations
6. **Performance**: Only compute/display what users need

## Implementation Timeline

### Week 1: Foundation + Configuration System
1. Create configuration package structure
2. Implement ConfigManager and parsers
3. Design formatter interface
4. Create table component with config support

### Week 2: Core Refactoring + Formatters
1. Implement script engine (Lua/Starlark)
2. Create built-in formatters
3. Refactor ResourceView with config integration
4. Implement column management

### Week 3: User Features
1. Add interactive configuration UI
2. Implement filter system
3. Create theme engine
4. Add layout management

### Week 4: Polish & Extensions
1. Documentation and examples
2. Community configuration repository
3. Plugin system for external data sources
4. Performance optimizations

## Conclusion

This enhanced refactoring plan not only addresses code complexity but also transforms Kubewatch into a highly customizable platform. Users can:
- Define exactly what data they want to see
- Create custom views for their workflows
- Share configurations with their team
- Extend functionality without modifying core code
- Integrate with external systems via scripting

The configuration system makes Kubewatch adaptable to any Kubernetes workflow while maintaining performance and simplicity.