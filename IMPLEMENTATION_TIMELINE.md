# Kubewatch Refactoring Implementation Timeline

## Overview
This document outlines the chronological implementation plan with parallel work streams. Testing is developed in parallel with features, and some components can start immediately with no dependencies.

## Phase 0: Immediate Start (No Dependencies)
**Goal**: Begin work on independent components that require no existing code changes

### Parallel Work Streams:
```
A. Template Engine (Standalone Package)
   - Design template syntax specification
   - Implement template parser (using text/template as base)
   - Extend text/template with custom functions
   - Build function registry for our domain-specific functions
   - Create formatting functions (color, humanize, etc.)
   - Implement template compiler/caching layer
   - Create template validator
   
   TESTS (developed in parallel):
   - Parser test suite with valid/invalid syntax cases
   - Function tests with edge cases (nil, empty, overflow)
   - Template compilation benchmarks
   - Cache hit/miss tests
   - Fuzzing tests for template robustness
   - Property-based tests for formatting functions
   
   Dependencies: 
   - text/template (Go standard library)
   - github.com/Masterminds/sprig/v3 (template functions)
   - github.com/charmbracelet/lipgloss (colors)
   
   Owner: Developer A
   Status: Can start IMMEDIATELY

B. Configuration System (Standalone Package)
   - Design configuration schema (what fields, structure, validation rules)
   - Implement config loader using YAML parser
   - Create validation layer (schema validation, not YAML parsing)
   - Build config merger (defaults + user overrides)
   - Add config migration system (v1 -> v2 format upgrades)
   - Implement config file watcher
   
   TESTS (developed in parallel):
   - Schema validation tests
   - Config merge precedence tests
   - Migration tests (v1 -> v2 format)
   - Invalid config handling (missing fields, wrong types)
   - Large config performance tests
   - File watcher tests
   
   Dependencies:
   - gopkg.in/yaml.v3 (YAML parsing)
   - github.com/fsnotify/fsnotify (file watching)
   - github.com/go-playground/validator/v10 (struct validation)
   
   Owner: Developer B
   Status: Can start IMMEDIATELY

C. Default Templates Library
   - Write all default templates in template language
   - Create template documentation
   - Build template test harness
   - Organize templates by resource type
   
   TESTS (developed in parallel):
   - Each template tested with sample K8s resources
   - Edge case data (missing fields, zero values)
   - Output validation against expected formatting
   - Cross-reference with current hardcoded output
   - Template performance benchmarks
   
   Dependencies:
   - None (just text files initially)
   
   Owner: Developer C
   Status: Can start IMMEDIATELY

D. Test Infrastructure & Fixtures
   - Create comprehensive K8s resource fixtures
   - Build test data generators
   - Set up golden file testing system
   - Create benchmark framework
   - Implement snapshot testing utilities
   - Build table-driven test helpers
   
   Dependencies:
   - k8s.io/api (for K8s types)
   - github.com/stretchr/testify (assertions)
   - github.com/bradleyjkemp/cupaloy/v2 (snapshot testing)
   
   Owner: Developer D
   Status: Can start IMMEDIATELY
```

## What We're Actually Building vs Using Libraries

### Using Existing Libraries For:
```yaml
# These are NOT being reimplemented:
yaml_parsing: gopkg.in/yaml.v3
template_base: text/template + html/template  
file_watching: github.com/fsnotify/fsnotify
validation: github.com/go-playground/validator/v10
testing: github.com/stretchr/testify
colors: github.com/charmbracelet/lipgloss
k8s_types: k8s.io/api
k8s_client: k8s.io/client-go
```

### What We ARE Building:
```yaml
# Our custom code:
template_extensions:
  - Custom template functions (humanizeBytes, millicores, etc.)
  - Template function registry
  - Template caching layer
  - Template validation specific to our use case

config_system:
  - Schema definition (our config structure)
  - Config loading logic (how we organize configs)
  - Override system (how user configs override defaults)
  - Migration logic (upgrading old configs)

formatters:
  - Default formatting templates
  - Template organization
  - Formatter registry

components:
  - Table component (using lipgloss for styling)
  - Selection tracker
  - Column manager
```

## Detailed Example: Configuration System

```go
// internal/config/loader.go
package config

import (
    "os"
    "path/filepath"
    
    "gopkg.in/yaml.v3"  // We're USING this, not reimplementing
    "github.com/fsnotify/fsnotify"  // We're USING this
    "github.com/go-playground/validator/v10"  // We're USING this
)

// Our custom config schema
type Config struct {
    Version  string                    `yaml:"version" validate:"required,semver"`
    Theme    string                    `yaml:"theme" validate:"required"`
    Columns  map[string]ColumnConfig   `yaml:"columns"`
    Templates map[string]string        `yaml:"templates"`
}

type ColumnConfig struct {
    Visible  bool     `yaml:"visible"`
    Width    int      `yaml:"width" validate:"min=0,max=200"`
    Order    int      `yaml:"order"`
    Template string   `yaml:"template"`
}

// Our custom loader using the yaml package
type Loader struct {
    parser    *yaml.Decoder  // Using yaml.v3's decoder
    validator *validator.Validate  // Using validator package
    watcher   *fsnotify.Watcher    // Using fsnotify
}

func (l *Loader) Load(path string) (*Config, error) {
    // Open file
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    // Use yaml.v3 to parse
    var config Config
    decoder := yaml.NewDecoder(file)
    if err := decoder.Decode(&config); err != nil {
        return nil, err
    }
    
    // Use validator to validate our schema
    if err := l.validator.Struct(&config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

## Detailed Example: Template Engine

```go
// internal/template/engine.go
package template

import (
    "text/template"  // We're EXTENDING this, not replacing
    "github.com/Masterminds/sprig/v3"  // We're USING these functions
    "github.com/charmbracelet/lipgloss"  // We're USING for colors
)

type Engine struct {
    base *template.Template  // Using Go's template engine
    funcMap template.FuncMap  // Extending with our functions
}

func NewEngine() *Engine {
    e := &Engine{
        funcMap: make(template.FuncMap),
    }
    
    // Start with sprig functions
    for k, v := range sprig.TxtFuncMap() {
        e.funcMap[k] = v
    }
    
    // Add our custom functions
    e.funcMap["color"] = e.colorFunc
    e.funcMap["humanizeBytes"] = e.humanizeBytesFunc
    e.funcMap["millicores"] = e.millicoresFunc
    // ... more custom functions
    
    return e
}

// Our custom function using lipgloss
func (e *Engine) colorFunc(color, text string) string {
    style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
    return style.Render(text)
}

// Our custom function for K8s-specific formatting
func (e *Engine) humanizeBytesFunc(bytes int64) string {
    // Our implementation for K8s-style byte formatting
    // This is custom logic we write
    units := []string{"", "Ki", "Mi", "Gi", "Ti"}
    // ... implementation
}
```

## Testing Our Custom Code

```go
// internal/config/loader_test.go
package config

import (
    "testing"
    "os"
    "path/filepath"
    
    "github.com/stretchr/testify/assert"  // Using for assertions
    "gopkg.in/yaml.v3"  // Using to create test data
)

func TestConfigLoader(t *testing.T) {
    // Test OUR schema validation, not yaml parsing
    tests := []struct {
        name    string
        config  string  // YAML string
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid config",
            config: `
version: "1.0.0"
theme: "dark"
columns:
  pod-name:
    visible: true
    width: 50
`,
            wantErr: false,
        },
        {
            name: "invalid version",
            config: `
version: "not-semver"
theme: "dark"
`,
            wantErr: true,
            errMsg: "version validation failed",
        },
        {
            name: "missing required field",
            config: `
version: "1.0.0"
# missing theme
`,
            wantErr: true,
            errMsg: "theme is required",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // We're testing OUR validation logic, not YAML parsing
            loader := NewLoader()
            config, err := loader.LoadFromString(tt.config)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, config)
            }
        })
    }
}
```

## Summary of What We're Building vs Using

### We're USING (not reimplementing):
- YAML parsing (gopkg.in/yaml.v3)
- Base template engine (text/template)
- File watching (fsnotify)
- Validation framework (go-playground/validator)
- Testing utilities (testify)
- Terminal colors (lipgloss)
- Kubernetes types (k8s.io/api)

### We're BUILDING:
- Template extensions for K8s formatting
- Configuration schema and loading logic
- Default formatting templates
- UI components (table, selection, etc.)
- Integration between all these pieces

### We're TESTING:
- Our custom template functions
- Our configuration schema validation
- Our component behavior
- Integration between components
- NOT testing the libraries themselves

This approach means we can start immediately because we're building on top of well-tested libraries, not reimplementing foundational functionality.