# Default Template Formatters System

## Overview
All default formatters are defined as templates embedded in the binary but can be overridden by user configurations. Users can interactively edit formatters and save customizations.

## Default Templates Structure

### Embedded Templates
```go
// internal/config/template/defaults/embed.go
package defaults

import _ "embed"

//go:embed templates/*.yaml
var defaultTemplates embed.FS

// Load all default templates at startup
func LoadDefaults() map[string]*TemplateDefinition {
    templates := make(map[string]*TemplateDefinition)
    
    files, _ := defaultTemplates.ReadDir("templates")
    for _, file := range files {
        content, _ := defaultTemplates.ReadFile("templates/" + file.Name())
        def := parseTemplateDefinition(content)
        templates[def.Name] = def
    }
    
    return templates
}
```

## Core Default Templates

### Pod Status Formatter
```yaml
# internal/config/template/defaults/templates/pod-status.yaml
apiVersion: kubewatch.io/v1
kind: FormatterTemplate
metadata:
  name: pod-status
  description: Default pod status formatter with icons and colors
spec:
  template: |
    {{- /* Pod Status Formatter - Shows status with appropriate icon and color */ -}}
    {{- $status := .Status.Phase -}}
    {{- $reason := "" -}}
    
    {{- /* Check for more specific status from conditions */ -}}
    {{- range .Status.Conditions -}}
      {{- if and (eq .Type "Ready") (ne .Status "True") .Reason -}}
        {{- $reason = .Reason -}}
      {{- end -}}
    {{- end -}}
    
    {{- /* Check container statuses for waiting/terminated states */ -}}
    {{- range .Status.ContainerStatuses -}}
      {{- if .State.Waiting -}}
        {{- $status = .State.Waiting.Reason -}}
      {{- else if .State.Terminated -}}
        {{- $status = .State.Terminated.Reason -}}
      {{- end -}}
    {{- end -}}
    
    {{- /* Apply appropriate styling based on status */ -}}
    {{- if eq $status "Running" -}}
      {{- color "green" "●" }} {{ color "green" "Running" -}}
    {{- else if eq $status "Succeeded" -}}
      {{- color "green" "✓" }} {{ color "green" "Succeeded" -}}
    {{- else if eq $status "Pending" -}}
      {{- color "yellow" "◐" }} {{ color "yellow" "Pending" -}}
    {{- else if eq $status "ContainerCreating" -}}
      {{- color "yellow" "◑" }} {{ color "yellow" "Creating" -}}
    {{- else if eq $status "Terminating" -}}
      {{- color "magenta" "◉" }} {{ color "magenta" "Terminating" -}}
    {{- else if or (eq $status "Failed") (eq $status "Error") -}}
      {{- color "red" "✗" }} {{ color "red" $status -}}
    {{- else if eq $status "CrashLoopBackOff" -}}
      {{- color "red" "↻" }} {{ color "red" "CrashLoop" -}}
    {{- else if eq $status "ImagePullBackOff" -}}
      {{- color "red" "⬇" }} {{ color "red" "ImagePull" -}}
    {{- else if eq $status "ErrImagePull" -}}
      {{- color "red" "⬇" }} {{ color "red" "ImageErr" -}}
    {{- else if eq $status "Completed" -}}
      {{- color "blue" "☐" }} {{ color "blue" "Completed" -}}
    {{- else if eq $status "Evicted" -}}
      {{- color "yellow" "⚠" }} {{ color "yellow" "Evicted" -}}
    {{- else -}}
      {{- color "gray" "○" }} {{ color "gray" $status -}}
    {{- end -}}
```

### CPU Formatter
```yaml
# internal/config/template/defaults/templates/cpu.yaml
apiVersion: kubewatch.io/v1
kind: FormatterTemplate
metadata:
  name: cpu
  description: CPU usage with color coding and optional request comparison
spec:
  template: |
    {{- /* CPU Formatter - Shows CPU usage with appropriate coloring */ -}}
    {{- $cpu := .Metrics.CPU | default 0 -}}
    {{- $cpuMilli := $cpu | toMillicores -}}
    {{- $requested := 0 -}}
    {{- $limit := 0 -}}
    
    {{- /* Get request and limit if available */ -}}
    {{- if .Spec.Containers -}}
      {{- range .Spec.Containers -}}
        {{- $requested = add $requested (.Resources.Requests.cpu | toMillicores | default 0) -}}
        {{- $limit = add $limit (.Resources.Limits.cpu | toMillicores | default 0) -}}
      {{- end -}}
    {{- end -}}
    
    {{- /* Calculate percentage if request is set */ -}}
    {{- $percent := 0 -}}
    {{- if gt $requested 0 -}}
      {{- $percent = div (mul $cpuMilli 100) $requested -}}
    {{- end -}}
    
    {{- /* Format based on value */ -}}
    {{- if eq $cpuMilli 0 -}}
      {{- color "gray" "-" -}}
    {{- else if lt $cpuMilli 1000 -}}
      {{- /* Show millicores for small values */ -}}
      {{- if and (gt $percent 0) (gt $percent 90) -}}
        {{- color "red" (printf "%dm" $cpuMilli) -}}
      {{- else if and (gt $percent 0) (gt $percent 70) -}}
        {{- color "yellow" (printf "%dm" $cpuMilli) -}}
      {{- else -}}
        {{- color "green" (printf "%dm" $cpuMilli) -}}
      {{- end -}}
    {{- else -}}
      {{- /* Show cores for large values */ -}}
      {{- $cores := div $cpuMilli 1000.0 -}}
      {{- if and (gt $percent 0) (gt $percent 90) -}}
        {{- color "red" (printf "%.2f" $cores) -}}
      {{- else if and (gt $percent 0) (gt $percent 70) -}}
        {{- color "yellow" (printf "%.2f" $cores) -}}
      {{- else -}}
        {{- color "green" (printf "%.2f" $cores) -}}
      {{- end -}}
    {{- end -}}
```

### Memory Formatter
```yaml
# internal/config/template/defaults/templates/memory.yaml
apiVersion: kubewatch.io/v1
kind: FormatterTemplate
metadata:
  name: memory
  description: Memory usage with automatic unit scaling
spec:
  template: |
    {{- /* Memory Formatter - Shows memory with appropriate units and coloring */ -}}
    {{- $memory := .Metrics.Memory | default 0 -}}
    {{- $memoryMB := $memory | toMB -}}
    {{- $requested := 0 -}}
    {{- $limit := 0 -}}
    
    {{- /* Get request and limit if available */ -}}
    {{- if .Spec.Containers -}}
      {{- range .Spec.Containers -}}
        {{- $requested = add $requested (.Resources.Requests.memory | toMB | default 0) -}}
        {{- $limit = add $limit (.Resources.Limits.memory | toMB | default 0) -}}
      {{- end -}}
    {{- end -}}
    
    {{- /* Calculate percentage if request is set */ -}}
    {{- $percent := 0 -}}
    {{- if gt $requested 0 -}}
      {{- $percent = div (mul $memoryMB 100) $requested -}}
    {{- end -}}
    
    {{- /* Format with appropriate units */ -}}
    {{- if eq $memoryMB 0 -}}
      {{- color "gray" "-" -}}
    {{- else -}}
      {{- $formatted := $memory | humanizeBytes -}}
      {{- if and (gt $percent 0) (gt $percent 90) -}}
        {{- color "red" $formatted -}}
      {{- else if and (gt $percent 0) (gt $percent 70) -}}
        {{- color "yellow" $formatted -}}
      {{- else if lt $memoryMB 128 -}}
        {{- color "green" $formatted -}}
      {{- else if lt $memoryMB 512 -}}
        {{- color "yellow" $formatted -}}
      {{- else -}}
        {{- color "red" $formatted -}}
      {{- end -}}
    {{- end -}}
```

### Ready Formatter
```yaml
# internal/config/template/defaults/templates/ready.yaml
apiVersion: kubewatch.io/v1
kind: FormatterTemplate
metadata:
  name: ready
  description: Ready count for pods and deployments
spec:
  template: |
    {{- /* Ready Formatter - Shows ready/total with coloring */ -}}
    {{- $ready := 0 -}}
    {{- $total := 0 -}}
    
    {{- /* Handle different resource types */ -}}
    {{- if .Status.ContainerStatuses -}}
      {{- /* Pod */ -}}
      {{- $total = len .Status.ContainerStatuses -}}
      {{- range .Status.ContainerStatuses -}}
        {{- if .Ready -}}{{- $ready = add $ready 1 -}}{{- end -}}
      {{- end -}}
    {{- else if .Status.ReadyReplicas -}}
      {{- /* Deployment/StatefulSet */ -}}
      {{- $ready = .Status.ReadyReplicas | default 0 -}}
      {{- $total = .Spec.Replicas | default 1 -}}
    {{- end -}}
    
    {{- /* Format with color based on readiness */ -}}
    {{- $text := printf "%d/%d" $ready $total -}}
    {{- if eq $ready $total -}}
      {{- color "green" $text -}}
    {{- else if eq $ready 0 -}}
      {{- color "red" $text -}}
    {{- else -}}
      {{- color "yellow" $text -}}
    {{- end -}}
```

### Restart Formatter
```yaml
# internal/config/template/defaults/templates/restarts.yaml
apiVersion: kubewatch.io/v1
kind: FormatterTemplate
metadata:
  name: restarts
  description: Container restart count with last restart time
spec:
  template: |
    {{- /* Restart Formatter - Shows restart count with timing */ -}}
    {{- $restarts := 0 -}}
    {{- $lastRestart := "" -}}
    
    {{- /* Sum up all container restarts */ -}}
    {{- range .Status.ContainerStatuses -}}
      {{- $restarts = add $restarts .RestartCount -}}
      {{- if .LastTerminationState.Terminated -}}
        {{- $lastRestart = .LastTerminationState.Terminated.FinishedAt | ago -}}
      {{- end -}}
    {{- end -}}
    
    {{- /* Format based on restart count */ -}}
    {{- if eq $restarts 0 -}}
      {{- color "gray" "0" -}}
    {{- else -}}
      {{- $text := toString $restarts -}}
      {{- if $lastRestart -}}
        {{- $text = printf "%d (%s)" $restarts $lastRestart -}}
      {{- end -}}
      
      {{- if lt $restarts 3 -}}
        {{- color "yellow" $text -}}
      {{- else if lt $restarts 10 -}}
        {{- color "orange" (printf "⚠ %s" $text) -}}
      {{- else -}}
        {{- color "red" (printf "‼ %s" $text) -}}
      {{- end -}}
    {{- end -}}
```

### Age Formatter
```yaml
# internal/config/template/defaults/templates/age.yaml
apiVersion: kubewatch.io/v1
kind: FormatterTemplate
metadata:
  name: age
  description: Resource age in human-readable format
spec:
  template: |
    {{- /* Age Formatter - Shows age with optional coloring */ -}}
    {{- $age := .Metadata.CreationTimestamp | ago -}}
    {{- $ageSeconds := .Metadata.CreationTimestamp | ageInSeconds -}}
    
    {{- /* Color based on age (optional) */ -}}
    {{- if lt $ageSeconds 300 -}}
      {{- /* Less than 5 minutes - new */ -}}
      {{- color "cyan" (printf "✨ %s" $age) -}}
    {{- else if lt $ageSeconds 3600 -}}
      {{- /* Less than 1 hour */ -}}
      {{- color "green" $age -}}
    {{- else if lt $ageSeconds 86400 -}}
      {{- /* Less than 1 day */ -}}
      {{- color "white" $age -}}
    {{- else if lt $ageSeconds 604800 -}}
      {{- /* Less than 1 week */ -}}
      {{- color "gray" $age -}}
    {{- else -}}
      {{- /* Older than 1 week */ -}}
      {{- color "darkgray" $age -}}
    {{- end -}}
```

### Service Type Formatter
```yaml
# internal/config/template/defaults/templates/service-type.yaml
apiVersion: kubewatch.io/v1
kind: FormatterTemplate
metadata:
  name: service-type
  description: Service type with icons
spec:
  template: |
    {{- /* Service Type Formatter */ -}}
    {{- if eq .Spec.Type "LoadBalancer" -}}
      {{- color "blue" "🌐" }} {{ .Spec.Type -}}
    {{- else if eq .Spec.Type "NodePort" -}}
      {{- color "cyan" "📡" }} {{ .Spec.Type -}}
    {{- else if eq .Spec.Type "ClusterIP" -}}
      {{- color "green" "🔒" }} {{ .Spec.Type -}}
    {{- else if eq .Spec.Type "ExternalName" -}}
      {{- color "magenta" "🔗" }} {{ .Spec.Type -}}
    {{- else -}}
      {{- color "gray" .Spec.Type -}}
    {{- end -}}
```

### Namespace Formatter
```yaml
# internal/config/template/defaults/templates/namespace.yaml
apiVersion: kubewatch.io/v1
kind: FormatterTemplate
metadata:
  name: namespace
  description: Namespace with special handling for system namespaces
spec:
  template: |
    {{- /* Namespace Formatter */ -}}
    {{- if hasPrefix .Namespace "kube-" -}}
      {{- color "blue" (printf "⚙ %s" .Namespace) -}}
    {{- else if eq .Namespace "default" -}}
      {{- color "gray" .Namespace -}}
    {{- else if contains .Namespace "prod" -}}
      {{- color "red" (printf "🔴 %s" .Namespace) -}}
    {{- else if contains .Namespace "staging" -}}
      {{- color "yellow" (printf "🟡 %s" .Namespace) -}}
    {{- else if contains .Namespace "dev" -}}
      {{- color "green" (printf "🟢 %s" .Namespace) -}}
    {{- else -}}
      {{- .Namespace -}}
    {{- end -}}
```

## Interactive Template Editor

### Editor Interface
```go
// internal/ui/views/template_editor.go
package views

type TemplateEditor struct {
    BaseView
    template     string
    originalTemplate string
    formatter    *config.FormatterTemplate
    testData     interface{}
    preview      string
    error        string
    cursorPos    int
    viewport     viewport.Model
    syntaxHighlighter *SyntaxHighlighter
}

func (e *TemplateEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+s":
            // Save template
            return e, e.saveTemplate()
        case "ctrl+t":
            // Test template with sample data
            return e, e.testTemplate()
        case "ctrl+r":
            // Reset to original
            e.template = e.originalTemplate
            return e, e.renderPreview()
        case "ctrl+d":
            // Reset to default
            return e, e.loadDefault()
        case "ctrl+p":
            // Toggle preview
            e.showPreview = !e.showPreview
            return e, nil
        case "ctrl+h":
            // Show help/function reference
            return e, e.showHelp()
        case "tab":
            // Auto-complete
            return e, e.autoComplete()
        }
    }
    // Handle text input
    return e, nil
}

func (e *TemplateEditor) View() string {
    // Split view: editor on left, preview on right
    editor := e.renderEditor()
    preview := e.renderPreview()
    help := e.renderHelp()
    
    return lipgloss.JoinHorizontal(
        lipgloss.Top,
        editor,
        preview,
    ) + "\n" + help
}
```

### Template Editor Features

#### Syntax Highlighting
```go
type SyntaxHighlighter struct {
    keywords   []string
    functions  []string
    variables  []string
}

func (sh *SyntaxHighlighter) Highlight(text string) string {
    // Highlight template syntax
    text = sh.highlightDelimiters(text)    // {{ }}
    text = sh.highlightKeywords(text)       // if, else, range, end
    text = sh.highlightFunctions(text)      // color, humanize, etc
    text = sh.highlightVariables(text)      // $var, .Field
    text = sh.highlightStrings(text)        // "string"
    text = sh.highlightComments(text)       // {{/* comment */}}
    return text
}
```

#### Auto-completion
```go
type AutoCompleter struct {
    engine      *template.Engine
    schema      *ResourceSchema
    functions   []FunctionDoc
}

func (ac *AutoCompleter) Complete(text string, pos int) []Completion {
    context := ac.getContext(text, pos)
    
    switch context.Type {
    case "function":
        return ac.completeFunctions(context)
    case "field":
        return ac.completeFields(context)
    case "color":
        return ac.completeColors(context)
    case "icon":
        return ac.completeIcons(context)
    }
    
    return nil
}

type Completion struct {
    Text        string
    Description string
    Example     string
    Type        string
}
```

#### Live Preview
```go
func (e *TemplateEditor) renderPreview() tea.Cmd {
    return func() tea.Msg {
        // Parse and execute template with test data
        result, err := e.engine.Execute(e.template, e.testData)
        if err != nil {
            return templateErrorMsg{err}
        }
        
        // Render the result as it would appear in the table
        preview := e.renderFormattedValue(result)
        return templatePreviewMsg{preview}
    }
}
```

### Template Management

#### Save System
```go
// internal/config/template/manager.go

type TemplateManager struct {
    defaults  map[string]*FormatterTemplate
    overrides map[string]*FormatterTemplate
    custom    map[string]*FormatterTemplate
    configDir string
}

func (tm *TemplateManager) SaveOverride(name string, template *FormatterTemplate) error {
    // Save to ~/.config/kubewatch/templates/overrides/
    path := filepath.Join(tm.configDir, "templates", "overrides", name+".yaml")
    
    // Create directory if needed
    os.MkdirAll(filepath.Dir(path), 0755)
    
    // Marshal and save
    data, err := yaml.Marshal(template)
    if err != nil {
        return err
    }
    
    return os.WriteFile(path, data, 0644)
}

func (tm *TemplateManager) LoadTemplate(name string) (*FormatterTemplate, error) {
    // Check for override first
    if tmpl, ok := tm.overrides[name]; ok {
        return tmpl, nil
    }
    
    // Check custom templates
    if tmpl, ok := tm.custom[name]; ok {
        return tmpl, nil
    }
    
    // Fall back to default
    if tmpl, ok := tm.defaults[name]; ok {
        return tmpl, nil
    }
    
    return nil, fmt.Errorf("template %s not found", name)
}

func (tm *TemplateManager) ResetToDefault(name string) error {
    // Remove override file
    path := filepath.Join(tm.configDir, "templates", "overrides", name+".yaml")
    os.Remove(path)
    
    // Remove from overrides map
    delete(tm.overrides, name)
    
    return nil
}
```

### User Workflow

#### 1. View Current Formatter
```bash
# In the TUI, press 'F' to open formatter menu
# Shows list of formatters for current column
┌─ Formatters ──────────────────────────┐
│ > pod-status    (default)             │
│   cpu           (modified) ✏️          │
│   memory        (default)              │
│   ready         (custom) ⚡            │
└────────────────────────────────────────┘
```

#### 2. Edit Formatter
```bash
# Press Enter to edit selected formatter
┌─ Edit Template: pod-status ─────────────────────────────────┐
│ {{- if eq .Status.Phase "Running" -}}                      │
│   {{- color "green" "●" }} Running                         │
│ {{- else if eq .Status.Phase "Pending" -}}                 │
│   {{- color "yellow" "◐" }} Pending                        │
│ {{- end -}}                                                │
├─────────────────────────────────────────────────────────────┤
│ Preview:                                                    │
│ ● Running                                                   │
├─────────────────────────────────────────────────────────────┤
│ Ctrl+S: Save | Ctrl+T: Test | Ctrl+D: Default | Ctrl+H: Help│
└──────────────────────────────────────────────────────────────┘
```

#### 3. Test with Live Data
```bash
# Press Ctrl+T to test with current selected resource
┌─ Test Results ──────────────────────────────────────────────┐
│ Input: Pod "nginx-7c5ddbdf4-abc123"                        │
│ Status.Phase: "Running"                                     │
│                                                             │
│ Output: ● Running                                          │
│         (with green color applied)                         │
└──────────────────────────────────────────────────────────────┘
```

#### 4. Save Changes
```bash
# Changes are saved to ~/.config/kubewatch/templates/overrides/
# Original defaults remain in binary
# Can reset to default at any time with Ctrl+D
```

### Template Documentation

#### Built-in Help System
```go
type FunctionDoc struct {
    Name        string
    Signature   string
    Description string
    Examples    []Example
    Category    string
}

var BuiltinFunctions = []FunctionDoc{
    {
        Name:        "color",
        Signature:   "color(color string, text string) string",
        Description: "Apply color to text",
        Examples: []Example{
            {Code: `{{ color "red" "Error" }}`, Output: "Error (in red)"},
            {Code: `{{ color "#FF5733" "Custom" }}`, Output: "Custom (in #FF5733)"},
        },
        Category: "Formatting",
    },
    {
        Name:        "humanizeBytes",
        Signature:   "humanizeBytes(bytes int) string",
        Description: "Convert bytes to human readable format",
        Examples: []Example{
            {Code: `{{ 1536870912 | humanizeBytes }}`, Output: "1.4Gi"},
            {Code: `{{ .Memory | humanizeBytes }}`, Output: "256Mi"},
        },
        Category: "Memory",
    },
    // ... more functions
}
```

### Template Sharing

#### Export/Import
```yaml
# Export current configuration
$ kubewatch config export > my-formatters.yaml

# Import configuration
$ kubewatch config import my-formatters.yaml

# Share specific formatter
$ kubewatch formatter export pod-status > pod-status-formatter.yaml
```

#### Community Templates
```go
// internal/config/template/community.go

type CommunityTemplates struct {
    registry string // URL to template registry
}

func (ct *CommunityTemplates) Browse() ([]TemplatePackage, error)
func (ct *CommunityTemplates) Install(packageName string) error
func (ct *CommunityTemplates) Share(template *FormatterTemplate) error
```

## Benefits

1. **Transparency**: All formatting logic is visible and modifiable
2. **Consistency**: Same template language everywhere
3. **Discoverability**: Users can see exactly how things work
4. **Customization**: Easy to modify without coding
5. **Sharing**: Templates can be shared as simple YAML files
6. **Version Control**: User customizations can be tracked in git
7. **Learning**: Users learn by examining defaults
8. **Testing**: Templates can be tested interactively
9. **Documentation**: Self-documenting with examples

## Migration Path

1. **Phase 1**: Convert all hardcoded formatters to templates
2. **Phase 2**: Embed defaults in binary
3. **Phase 3**: Add template editor UI
4. **Phase 4**: Implement override system
5. **Phase 5**: Add community sharing features

This approach makes Kubewatch's formatting completely transparent and customizable while maintaining excellent defaults that "just work" out of the box.