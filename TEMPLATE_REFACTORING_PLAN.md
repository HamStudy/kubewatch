# Kubewatch Template-Driven Architecture Refactoring Plan

## Executive Summary

This document outlines a comprehensive refactoring plan to transform Kubewatch from a hardcoded resource handling system to a fully template-driven, extensible architecture that supports custom resource types (CRDs) through configuration files.

## Current Architecture Analysis

### Problems with Current Implementation

1. **Hardcoded Resource Logic** (`internal/ui/views/resource_view.go:1247-1877`)
   - Each resource type has dedicated update methods (updateTableWithPods, updateTableWithDeployments, etc.)
   - Column definitions are hardcoded in switch statements
   - Formatting logic is embedded in Go code
   - Adding new resource types requires code changes

2. **Limited Extensibility**
   - Only 7 built-in resource types supported
   - No support for Custom Resource Definitions (CRDs)
   - Cannot customize column layouts without recompiling
   - Resource-specific operations are tightly coupled

3. **Maintenance Burden**
   - 600+ lines of repetitive code for resource handling
   - Similar patterns duplicated across resource types
   - Changes require updating multiple locations
   - Testing requires mocking specific resource types

## Proposed Architecture

### Core Design Principles

1. **Configuration-Driven**: All resource types defined in YAML configs
2. **Template-Based Formatting**: Use template engine for all display logic
3. **Embedded Defaults**: Ship with embedded configs for standard resources
4. **Override Capability**: Allow runtime overrides without recompilation
5. **CRD Support**: Enable custom resources through config files
6. **Parallel Development**: Multiple teams can work on different resource configs

## Directory Structure

```
kubewatch/
├── configs/                          # Configuration root
│   ├── resources/                    # Resource type definitions
│   │   ├── embedded/                 # Built-in resources (embedded in binary)
│   │   │   ├── core/                 # Core K8s resources
│   │   │   │   ├── pod.yaml
│   │   │   │   ├── deployment.yaml
│   │   │   │   ├── statefulset.yaml
│   │   │   │   ├── service.yaml
│   │   │   │   ├── ingress.yaml
│   │   │   │   ├── configmap.yaml
│   │   │   │   └── secret.yaml
│   │   │   ├── apps/                 # Apps resources
│   │   │   │   ├── daemonset.yaml
│   │   │   │   ├── replicaset.yaml
│   │   │   │   └── job.yaml
│   │   │   ├── networking/           # Networking resources
│   │   │   │   ├── networkpolicy.yaml
│   │   │   │   └── endpoint.yaml
│   │   │   └── storage/              # Storage resources
│   │   │       ├── persistentvolume.yaml
│   │   │       └── persistentvolumeclaim.yaml
│   │   └── custom/                   # User-defined resources (runtime)
│   │       └── .gitkeep
│   ├── templates/                    # Reusable template fragments
│   │   ├── formatters/               # Column formatters
│   │   │   ├── status.tmpl
│   │   │   ├── ready.tmpl
│   │   │   ├── cpu.tmpl
│   │   │   ├── memory.tmpl
│   │   │   ├── age.tmpl
│   │   │   └── restarts.tmpl
│   │   └── operations/               # Operation templates
│   │       ├── describe.tmpl
│   │       ├── logs.tmpl
│   │       └── exec.tmpl
│   └── schemas/                      # JSON schemas for validation
│       ├── resource-config.schema.json
│       └── template.schema.json

~/.config/kubewatch/                  # User configuration directory
├── resources/                        # User resource overrides
│   ├── overrides/                    # Override built-in resources
│   │   └── pod.yaml                  # Override default pod config
│   └── custom/                       # Custom resource definitions
│       ├── my-crd.yaml
│       └── team-resource.yaml
└── templates/                        # User template overrides
    └── formatters/
        └── custom-status.tmpl
```

## Resource Configuration Format

### Resource Definition Schema

```yaml
# configs/resources/embedded/core/pod.yaml
apiVersion: kubewatch.io/v1
kind: ResourceConfig
metadata:
  name: pod
  displayName: Pods
  description: Kubernetes Pod resources
  category: core
spec:
  # API configuration
  api:
    group: ""  # core API group
    version: v1
    resource: pods
    namespaced: true
    
  # List operation configuration
  list:
    # Method to use for listing resources
    method: standard  # standard, watch, informer
    
    # Caching strategy
    cache:
      enabled: true
      ttl: 2s
      
    # Field selector for filtering
    fieldSelector: ""
    
    # Label selector for filtering
    labelSelector: ""
    
  # Column definitions
  columns:
    - name: NAME
      field: .metadata.name
      width: auto
      priority: 1  # Always show
      sortable: true
      
    - name: NAMESPACE
      field: .metadata.namespace
      width: auto
      priority: 2
      sortable: true
      showWhen: "{{ .ShowNamespace }}"
      template: "{{ .metadata.namespace | namespace }}"
      
    - name: READY
      width: 8
      priority: 1
      sortable: true
      sortType: numeric
      template: |
        {{- $ready := 0 -}}
        {{- $total := len .status.containerStatuses -}}
        {{- range .status.containerStatuses -}}
          {{- if .ready -}}{{- $ready = add $ready 1 -}}{{- end -}}
        {{- end -}}
        {{- if eq $ready $total -}}
          {{- color "green" (printf "%d/%d" $ready $total) -}}
        {{- else if eq $ready 0 -}}
          {{- color "red" (printf "%d/%d" $ready $total) -}}
        {{- else -}}
          {{- color "yellow" (printf "%d/%d" $ready $total) -}}
        {{- end -}}
      sortValue: |
        {{- $ready := 0 -}}
        {{- $total := len .status.containerStatuses -}}
        {{- range .status.containerStatuses -}}
          {{- if .ready -}}{{- $ready = add $ready 1 -}}{{- end -}}
        {{- end -}}
        {{- if eq $total 0 -}}0{{- else -}}{{ div $ready $total }}{{- end -}}
        
    - name: STATUS
      width: 12
      priority: 1
      sortable: true
      template: "{{ template \"pod-status\" . }}"
      
    - name: RESTARTS
      width: 10
      priority: 2
      sortable: true
      sortType: numeric
      template: "{{ template \"restarts\" . }}"
      
    - name: AGE
      field: .metadata.creationTimestamp
      width: 8
      priority: 2
      sortable: true
      sortType: time
      template: "{{ .metadata.creationTimestamp | age }}"
      
    - name: CPU
      width: 10
      priority: 3
      sortable: true
      sortType: numeric
      requiresMetrics: true
      template: "{{ template \"cpu\" . }}"
      
    - name: MEMORY
      width: 10
      priority: 3
      sortable: true
      sortType: numeric
      requiresMetrics: true
      template: "{{ template \"memory\" . }}"
      
    - name: IP
      field: .status.podIP
      width: 15
      priority: 4
      sortable: true
      template: "{{ .status.podIP | default \"-\" }}"
      
    - name: NODE
      field: .spec.nodeName
      width: auto
      priority: 4
      sortable: true
      template: "{{ .spec.nodeName | default \"-\" }}"
      
  # Operations available for this resource
  operations:
    - name: describe
      key: d
      description: Describe resource
      command: "kubectl describe pod {{ .metadata.name }} -n {{ .metadata.namespace }}"
      template: "{{ template \"describe-pod\" . }}"
      
    - name: logs
      key: l
      description: View logs
      command: "kubectl logs {{ .metadata.name }} -n {{ .metadata.namespace }}"
      available: "{{ gt (len .spec.containers) 0 }}"
      
    - name: exec
      key: e
      description: Execute shell
      command: "kubectl exec -it {{ .metadata.name }} -n {{ .metadata.namespace }} -- /bin/sh"
      available: "{{ and (eq .status.phase \"Running\") (gt (len .spec.containers) 0) }}"
      
    - name: delete
      key: x
      description: Delete resource
      command: "kubectl delete pod {{ .metadata.name }} -n {{ .metadata.namespace }}"
      confirm: true
      
    - name: edit
      key: E
      description: Edit resource
      command: "kubectl edit pod {{ .metadata.name }} -n {{ .metadata.namespace }}"
      
  # Grouping configuration (for aggregated views)
  grouping:
    enabled: false  # Pods typically aren't grouped
    
  # Multi-context behavior
  multiContext:
    supported: true
    mergeStrategy: append  # append, replace, merge
    contextColumn: true
    
  # Resource-specific features
  features:
    metrics: true
    logs: true
    exec: true
    portForward: true
    
  # Custom data enrichment
  enrichment:
    - name: metrics
      type: pod-metrics
      cache: 5s
      
  # Status indicators
  statusIndicators:
    healthy:
      conditions:
        - field: .status.phase
          operator: eq
          value: Running
    warning:
      conditions:
        - field: .status.phase
          operator: in
          values: [Pending, Unknown]
    error:
      conditions:
        - field: .status.phase
          operator: in
          values: [Failed, Evicted]
```

### Custom Resource Example

```yaml
# ~/.config/kubewatch/resources/custom/tekton-pipeline.yaml
apiVersion: kubewatch.io/v1
kind: ResourceConfig
metadata:
  name: tekton-pipeline
  displayName: Tekton Pipelines
  description: Tekton Pipeline CRD
  category: ci-cd
spec:
  api:
    group: tekton.dev
    version: v1beta1
    resource: pipelines
    namespaced: true
    
  columns:
    - name: NAME
      field: .metadata.name
      width: auto
      priority: 1
      
    - name: VERSION
      field: .metadata.labels["version"]
      width: 10
      priority: 2
      template: "{{ .metadata.labels.version | default \"latest\" }}"
      
    - name: TASKS
      width: 8
      priority: 2
      template: "{{ len .spec.tasks }}"
      
    - name: PARAMS
      width: 8
      priority: 3
      template: "{{ len .spec.params }}"
      
    - name: AGE
      field: .metadata.creationTimestamp
      width: 8
      priority: 2
      template: "{{ .metadata.creationTimestamp | age }}"
      
  operations:
    - name: describe
      key: d
      description: Describe pipeline
      template: "{{ template \"describe-generic\" . }}"
      
    - name: runs
      key: r
      description: Show pipeline runs
      command: "kubectl get pipelineruns -l tekton.dev/pipeline={{ .metadata.name }}"
```

## Implementation Architecture

### 1. Resource Registry

```go
// internal/config/resources/registry.go
package resources

import (
    "embed"
    "sync"
)

//go:embed embedded/*/*.yaml
var embeddedConfigs embed.FS

type Registry struct {
    mu        sync.RWMutex
    configs   map[string]*ResourceConfig
    overrides map[string]*ResourceConfig
    custom    map[string]*ResourceConfig
}

func NewRegistry() (*Registry, error) {
    r := &Registry{
        configs:   make(map[string]*ResourceConfig),
        overrides: make(map[string]*ResourceConfig),
        custom:    make(map[string]*ResourceConfig),
    }
    
    // Load embedded configs
    if err := r.loadEmbedded(); err != nil {
        return nil, err
    }
    
    // Load user overrides
    if err := r.loadUserConfigs(); err != nil {
        return nil, err
    }
    
    return r, nil
}

func (r *Registry) Get(resourceType string) (*ResourceConfig, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    // Check overrides first
    if config, ok := r.overrides[resourceType]; ok {
        return config, nil
    }
    
    // Check custom resources
    if config, ok := r.custom[resourceType]; ok {
        return config, nil
    }
    
    // Fall back to embedded
    if config, ok := r.configs[resourceType]; ok {
        return config, nil
    }
    
    return nil, fmt.Errorf("resource type %s not found", resourceType)
}

func (r *Registry) List() []ResourceConfig {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    var configs []ResourceConfig
    
    // Merge all sources
    seen := make(map[string]bool)
    
    // Add overrides first (highest priority)
    for name, config := range r.overrides {
        configs = append(configs, *config)
        seen[name] = true
    }
    
    // Add custom resources
    for name, config := range r.custom {
        if !seen[name] {
            configs = append(configs, *config)
            seen[name] = true
        }
    }
    
    // Add embedded resources
    for name, config := range r.configs {
        if !seen[name] {
            configs = append(configs, *config)
        }
    }
    
    return configs
}
```

### 2. Generic Resource Handler

```go
// internal/ui/views/resource_view_generic.go
package views

type GenericResourceView struct {
    config         *resources.ResourceConfig
    templateEngine *template.Engine
    client         dynamic.Interface
    state          *core.State
    cache          *cache.ResourceCache
}

func NewGenericResourceView(config *resources.ResourceConfig) *GenericResourceView {
    return &GenericResourceView{
        config:         config,
        templateEngine: template.NewEngine(),
        cache:          cache.NewResourceCache(config.Spec.List.Cache.TTL),
    }
}

func (v *GenericResourceView) GetHeaders() []string {
    var headers []string
    
    for _, col := range v.config.Spec.Columns {
        // Check if column should be shown
        if col.ShowWhen != "" {
            show, _ := v.templateEngine.ExecuteBool(col.ShowWhen, v.getContext())
            if !show {
                continue
            }
        }
        headers = append(headers, col.Name)
    }
    
    return headers
}

func (v *GenericResourceView) TransformToRow(resource unstructured.Unstructured) ([]string, error) {
    var row []string
    
    for _, col := range v.config.Spec.Columns {
        // Skip hidden columns
        if col.ShowWhen != "" {
            show, _ := v.templateEngine.ExecuteBool(col.ShowWhen, v.getContext())
            if !show {
                continue
            }
        }
        
        // Get value using template or field
        var value string
        if col.Template != "" {
            val, err := v.templateEngine.Execute(col.Template, resource.Object)
            if err != nil {
                value = "ERROR"
            } else {
                value = val
            }
        } else if col.Field != "" {
            val, _, err := unstructured.NestedString(resource.Object, strings.Split(col.Field, ".")...)
            if err != nil {
                value = "-"
            } else {
                value = val
            }
        }
        
        row = append(row, value)
    }
    
    return row, nil
}

func (v *GenericResourceView) List(namespace string) ([]unstructured.Unstructured, error) {
    // Use dynamic client to list resources
    gvr := schema.GroupVersionResource{
        Group:    v.config.Spec.API.Group,
        Version:  v.config.Spec.API.Version,
        Resource: v.config.Spec.API.Resource,
    }
    
    var list *unstructured.UnstructuredList
    var err error
    
    if v.config.Spec.API.Namespaced {
        list, err = v.client.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{
            FieldSelector: v.config.Spec.List.FieldSelector,
            LabelSelector: v.config.Spec.List.LabelSelector,
        })
    } else {
        list, err = v.client.Resource(gvr).List(context.TODO(), metav1.ListOptions{
            FieldSelector: v.config.Spec.List.FieldSelector,
            LabelSelector: v.config.Spec.List.LabelSelector,
        })
    }
    
    if err != nil {
        return nil, err
    }
    
    return list.Items, nil
}
```

### 3. Template Engine Extensions

```go
// internal/template/k8s_functions.go
package template

// Additional K8s-specific template functions
func (e *Engine) registerK8sFunctions() {
    // Resource field access
    e.funcMap["field"] = e.fieldFunc           // Access nested fields
    e.funcMap["annotation"] = e.annotationFunc // Get annotation
    e.funcMap["label"] = e.labelFunc          // Get label
    
    // Resource status
    e.funcMap["podStatus"] = e.podStatusFunc
    e.funcMap["deploymentStatus"] = e.deploymentStatusFunc
    e.funcMap["serviceStatus"] = e.serviceStatusFunc
    
    // Metrics
    e.funcMap["cpuUsage"] = e.cpuUsageFunc
    e.funcMap["memoryUsage"] = e.memoryUsageFunc
    e.funcMap["efficiency"] = e.efficiencyFunc
    
    // Conditions
    e.funcMap["hasCondition"] = e.hasConditionFunc
    e.funcMap["conditionStatus"] = e.conditionStatusFunc
    
    // Aggregations
    e.funcMap["sumField"] = e.sumFieldFunc
    e.funcMap["avgField"] = e.avgFieldFunc
    e.funcMap["countWhere"] = e.countWhereFunc
}

func (e *Engine) fieldFunc(obj interface{}, path string) (interface{}, error) {
    // Use unstructured to access nested fields
    if unstr, ok := obj.(map[string]interface{}); ok {
        parts := strings.Split(path, ".")
        val, found, err := unstructured.NestedFieldCopy(unstr, parts...)
        if err != nil || !found {
            return nil, err
        }
        return val, nil
    }
    return nil, fmt.Errorf("invalid object type")
}
```

### 4. Build-Time Embedding

```go
// internal/config/resources/embed.go
package resources

import (
    "embed"
    "io/fs"
    "path/filepath"
    "gopkg.in/yaml.v3"
)

//go:embed embedded/core/*.yaml
//go:embed embedded/apps/*.yaml
//go:embed embedded/networking/*.yaml
//go:embed embedded/storage/*.yaml
var embeddedFS embed.FS

func LoadEmbeddedConfigs() (map[string]*ResourceConfig, error) {
    configs := make(map[string]*ResourceConfig)
    
    err := fs.WalkDir(embeddedFS, "embedded", func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        
        if filepath.Ext(path) != ".yaml" {
            return nil
        }
        
        data, err := embeddedFS.ReadFile(path)
        if err != nil {
            return err
        }
        
        var config ResourceConfig
        if err := yaml.Unmarshal(data, &config); err != nil {
            return fmt.Errorf("failed to parse %s: %w", path, err)
        }
        
        configs[config.Metadata.Name] = &config
        return nil
    })
    
    return configs, err
}
```

### 5. Runtime Override System

```go
// internal/config/resources/overrides.go
package resources

import (
    "os"
    "path/filepath"
)

type OverrideManager struct {
    configDir string
    watcher   *fsnotify.Watcher
    onChange  func(string, *ResourceConfig)
}

func NewOverrideManager() (*OverrideManager, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }
    
    configDir := filepath.Join(home, ".config", "kubewatch")
    
    // Create directories if they don't exist
    dirs := []string{
        filepath.Join(configDir, "resources", "overrides"),
        filepath.Join(configDir, "resources", "custom"),
        filepath.Join(configDir, "templates", "formatters"),
    }
    
    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return nil, err
        }
    }
    
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }
    
    om := &OverrideManager{
        configDir: configDir,
        watcher:   watcher,
    }
    
    // Watch for changes
    om.watchDirectories()
    
    return om, nil
}

func (om *OverrideManager) LoadOverrides() (map[string]*ResourceConfig, error) {
    overrides := make(map[string]*ResourceConfig)
    
    overrideDir := filepath.Join(om.configDir, "resources", "overrides")
    files, err := os.ReadDir(overrideDir)
    if err != nil {
        if os.IsNotExist(err) {
            return overrides, nil
        }
        return nil, err
    }
    
    for _, file := range files {
        if filepath.Ext(file.Name()) != ".yaml" {
            continue
        }
        
        path := filepath.Join(overrideDir, file.Name())
        config, err := om.loadConfig(path)
        if err != nil {
            log.Printf("Failed to load override %s: %v", file.Name(), err)
            continue
        }
        
        overrides[config.Metadata.Name] = config
    }
    
    return overrides, nil
}

func (om *OverrideManager) SaveOverride(name string, config *ResourceConfig) error {
    path := filepath.Join(om.configDir, "resources", "overrides", name+".yaml")
    
    data, err := yaml.Marshal(config)
    if err != nil {
        return err
    }
    
    return os.WriteFile(path, data, 0644)
}
```

## Migration Strategy

### Phase 1: Foundation (Week 1)
**Goal**: Set up configuration infrastructure

1. **Create Configuration Schema**
   - Define ResourceConfig struct
   - Create JSON schema for validation
   - Implement YAML parser

2. **Build Template Engine Extensions**
   - Add K8s-specific functions
   - Implement field access helpers
   - Create formatting functions

3. **Implement Resource Registry**
   - Embed default configs
   - Load user overrides
   - Hot reload support

**Deliverables**:
- `internal/config/resources/` package
- Embedded YAML configs for all current resources
- Template engine with K8s functions

### Phase 2: Generic Handler (Week 2)
**Goal**: Replace hardcoded logic with generic handler

1. **Create Generic Resource View**
   - Dynamic column generation
   - Template-based formatting
   - Operation execution

2. **Implement Dynamic Client**
   - Unstructured resource handling
   - GVR resolution
   - Multi-context support

3. **Update UI Layer**
   - Replace resource-specific methods
   - Use generic handler
   - Maintain backward compatibility

**Deliverables**:
- Generic resource handler
- Dynamic client integration
- All existing resources working via configs

### Phase 3: CRD Support (Week 3)
**Goal**: Enable custom resource types

1. **CRD Discovery**
   - Auto-discover CRDs
   - Generate basic configs
   - Schema introspection

2. **Custom Resource Configs**
   - User-defined resource configs
   - Template customization
   - Operation definitions

3. **Testing & Documentation**
   - Example CRD configs
   - User documentation
   - Integration tests

**Deliverables**:
- CRD support
- Example configurations
- User documentation

### Phase 4: Advanced Features (Week 4)
**Goal**: Polish and optimize

1. **Performance Optimization**
   - Template caching
   - Resource caching
   - Lazy loading

2. **Developer Tools**
   - Config validation CLI
   - Template testing tool
   - Config generator

3. **UI Enhancements**
   - Config editor in TUI
   - Template preview
   - Live reload

**Deliverables**:
- Performance improvements
- Developer tools
- Enhanced UI features

## Parallel Development Plan

### Team Structure

**Team A: Core Infrastructure**
- Resource registry
- Template engine
- Configuration schema
- Embedding system

**Team B: Generic Handler**
- Generic resource view
- Dynamic client
- Operation executor
- Cache system

**Team C: Resource Configs**
- Convert existing resources to YAML
- Create template formatters
- Define operations
- Test configurations

**Team D: User Experience**
- Override system
- Hot reload
- Config editor UI
- Documentation

### Integration Points

1. **Week 1 Checkpoint**
   - Schema finalized
   - Template engine ready
   - Sample configs created

2. **Week 2 Checkpoint**
   - Generic handler working
   - 3+ resources converted
   - Override system functional

3. **Week 3 Checkpoint**
   - All resources converted
   - CRD support working
   - Documentation complete

4. **Week 4 Checkpoint**
   - Performance optimized
   - Tools completed
   - Full test coverage

## Benefits

1. **Extensibility**
   - Support any Kubernetes resource
   - Easy to add custom resources
   - No code changes for new types

2. **Customization**
   - Override any aspect via config
   - Custom formatters and templates
   - Per-user preferences

3. **Maintainability**
   - 80% less code
   - Declarative configuration
   - Easier to test

4. **Developer Experience**
   - Parallel development possible
   - Clear separation of concerns
   - Self-documenting configs

5. **User Experience**
   - Consistent interface
   - Predictable behavior
   - Easy customization

## Risk Mitigation

1. **Performance Risk**
   - Mitigation: Aggressive caching, compiled templates
   - Fallback: Hybrid approach for critical resources

2. **Compatibility Risk**
   - Mitigation: Maintain backward compatibility
   - Fallback: Legacy mode flag

3. **Complexity Risk**
   - Mitigation: Progressive rollout
   - Fallback: Keep old code during transition

## Success Metrics

- All 7 existing resource types working via configs
- Support for 3+ CRDs demonstrated
- <10ms template execution time
- 80% code reduction in resource_view.go
- Zero breaking changes for users
- 100% test coverage for generic handler

## Conclusion

This refactoring transforms Kubewatch into a truly extensible Kubernetes TUI that can adapt to any resource type through configuration. The template-driven approach reduces code complexity while increasing flexibility, making it easier to maintain and extend.