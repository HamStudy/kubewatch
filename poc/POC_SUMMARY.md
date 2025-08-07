# Template-Driven Approach POC Summary

## ✅ Proof of Concept Successful

The POC successfully demonstrates that a template-driven approach will work for kubewatch. All key integration points have been verified.

## Working Components Demonstrated

### 1. YAML Configuration Loading ✅
- Successfully loads resource configurations from YAML files
- Supports compile-time embedding with `go:embed`
- Can also load from filesystem at runtime for user customization

### 2. Template Execution ✅
- Integrates with existing `internal/template/Engine`
- Supports all existing template functions (color, humanizeBytes, ago, etc.)
- Templates can access nested resource fields using dot notation
- Conditional rendering based on context (showNamespace, showMetrics)

### 3. Dynamic Resource Handling ✅
- Generates correct GroupVersionResource (GVR) for dynamic client
- Works with both structured (v1.Pod) and unstructured data
- Can handle any Kubernetes resource type including CRDs
- No code changes needed to support new resource types

### 4. Integration Points Verified ✅

#### With Template Engine (`internal/template/`)
```go
// Existing template engine works perfectly
templateEngine := tmpl.NewEngine()
result, err := templateEngine.Execute(columnSpec.Template, resource)
```

#### With Table Rendering (`internal/ui/views/resource_view.go`)
```go
// Template-based transformer implements existing interface
type TemplateBasedTransformer struct {
    config  *ResourceConfig
    manager *ResourceManager
}

// Implements ResourceTransformer interface seamlessly
func (t *TemplateBasedTransformer) TransformToRow(resource interface{}, ...) ([]string, *selection.ResourceIdentity, error)
```

#### With Dynamic Kubernetes Client
```go
// Generate GVR from config
gvr := schema.GroupVersionResource{
    Group:    config.Spec.Kubernetes.Group,
    Version:  config.Spec.Kubernetes.Version,
    Resource: pluralize(config.Spec.Kubernetes.Kind),
}

// Use with dynamic client
dynamicClient.Resource(gvr).Namespace(ns).List(...)
```

## Key Features Demonstrated

### Column Configuration
```yaml
columns:
  - name: STATUS
    width: 15
    template: |
      {{- $status := .status.phase -}}
      {{- if eq $status "Running" -}}
        {{- color "green" $status -}}
      {{- else if eq $status "Pending" -}}
        {{- color "yellow" $status -}}
      {{- end -}}
```

### Conditional Columns
```yaml
- name: NAMESPACE
  width: 15
  template: "{{ .metadata.namespace }}"
  conditional: showNamespace  # Only shown when showNamespace=true
```

### Resource Grouping
```yaml
grouping:
  enabled: true
  key: "{{ .metadata.labels.app | default .metadata.name }}"
  aggregation:
    - column: NAME
      template: "{{ .group.key }} ({{ len .group.resources }})"
```

### Operations Configuration
```yaml
operations:
  logs:
    enabled: true
    command: "kubectl logs {{ .metadata.name }} -n {{ .metadata.namespace }}"
    followSupported: true
```

## Technical Challenges Identified

### 1. Template Function Completeness ⚠️
**Issue**: Some template functions like `add` are not yet implemented in the template engine.
**Solution**: Add missing functions to the template engine or use Sprig library for comprehensive function set.

### 2. Resource Pluralization
**Issue**: Need to convert Kind (e.g., "Pod") to resource name (e.g., "pods") for dynamic client.
**Solution**: Use Kubernetes discovery API or maintain a mapping table.

### 3. Unstructured Data Access
**Issue**: Accessing nested fields in unstructured data requires careful handling.
**Solution**: Template engine already handles this well with dot notation.

## Migration Path

### Phase 1: Parallel Implementation
- Keep existing hardcoded transformers
- Add template-based transformer as alternative
- Use feature flag to toggle between implementations

### Phase 2: Gradual Migration
- Convert one resource type at a time
- Start with simple resources (ConfigMap, Secret)
- Move to complex resources (Pod, Deployment)

### Phase 3: Full Template-Driven
- All resources use template configuration
- Ship default configs embedded in binary
- Allow user overrides in ~/.kubewatch/resources/

## Benefits Confirmed

1. **Extensibility**: Support any Kubernetes resource without code changes
2. **Customization**: Users can modify display without rebuilding
3. **CRD Support**: Automatic support for Custom Resource Definitions
4. **Maintainability**: Column definitions in YAML instead of Go code
5. **Consistency**: Single source of truth for resource display

## Performance Considerations

- Template compilation can be cached (already implemented in template.Cache)
- YAML configs can be embedded at compile time (no runtime overhead)
- Dynamic client adds minimal overhead compared to typed clients

## Recommendation

✅ **Proceed with template-driven approach**

The POC proves this approach is viable and offers significant benefits:
- Works with existing codebase
- No breaking changes required
- Enables powerful user customization
- Reduces maintenance burden
- Future-proofs for new Kubernetes resources

## Next Steps

1. Enhance template engine with missing functions
2. Create comprehensive default configurations for all resource types
3. Implement configuration loading system with override support
4. Add validation for YAML configurations
5. Create documentation for template syntax and customization