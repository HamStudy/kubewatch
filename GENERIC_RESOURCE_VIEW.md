# Generic Resource View Implementation

## Overview

The Generic Resource View is a template-driven implementation that replaces hardcoded resource handling in KubeWatch TUI. It uses the resource configuration system to dynamically render Kubernetes resources based on YAML definitions.

## Components

### 1. GenericResourceView (`internal/ui/views/generic_resource_view.go`)

The main view component that:
- Uses the resource registry to get resource definitions
- Uses k8s.io/client-go/dynamic client to list resources
- Uses the template engine to render columns
- Supports all existing features: sorting, filtering, selection tracking, multi-context

### 2. Key Methods

#### NewGenericResourceView
Creates a new generic resource view with dependencies:
```go
func NewGenericResourceView(
    state *core.State,
    registry *resource.Registry,
    engine *template.Engine,
    dynamicClient dynamic.Interface,
) (*GenericResourceView, error)
```

#### RefreshResources
Fetches resources using the dynamic client:
```go
func (v *GenericResourceView) RefreshResources(ctx context.Context) error
```

#### renderColumns
Executes templates for each column:
```go
func (v *GenericResourceView) renderColumns(
    resource *unstructured.Unstructured,
    definition *resource.ResourceDefinition,
) ([]string, error)
```

#### updateTable
Builds table rows using templates:
```go
func (v *GenericResourceView) updateTable() error
```

## Integration with ResourceView

To integrate the generic view with the existing ResourceView, you can:

1. Add a `UseGenericView` feature flag to ResourceView
2. Create a `genericView` field of type `*GenericResourceView`
3. In the Update method, delegate to generic view when the flag is enabled
4. In the View method, render using generic view when available

### Example Integration

```go
type ResourceView struct {
    // ... existing fields ...
    
    // Feature flag for generic view
    UseGenericView bool
    
    // Generic view instance
    genericView *GenericResourceView
}

func (v *ResourceView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if v.UseGenericView && v.genericView != nil {
        // Delegate to generic view for supported operations
        return v.genericView.Update(msg)
    }
    // Fall back to legacy implementation
    // ... existing code ...
}
```

## Resource Definition Format

Resources are defined using the ResourceDefinition structure:

```go
type ResourceDefinition struct {
    APIVersion string   // Must be "kubewatch.io/v1"
    Kind       string   // Must be "ResourceDefinition"
    Metadata   Metadata // Name and description
    Spec       Spec     // Kubernetes info and columns
}
```

### Example Pod Definition

```go
podDef := &resource.ResourceDefinition{
    APIVersion: "kubewatch.io/v1",
    Kind:       "ResourceDefinition",
    Metadata: resource.Metadata{
        Name:        "pods",
        Description: "Kubernetes Pods",
    },
    Spec: resource.Spec{
        Kubernetes: resource.KubernetesSpec{
            Group:      "",
            Version:    "v1",
            Kind:       "Pod",
            Plural:     "pods",
            Namespaced: true,
        },
        Columns: []resource.Column{
            {Name: "Name", Template: "{{ .metadata.name }}", Width: 30},
            {Name: "Status", Template: "{{ .status.phase }}", Width: 20},
            {Name: "Ready", Template: "{{ .status.containerStatuses | readyContainers }}/{{ .spec.containers | len }}", Width: 10},
            {Name: "Age", Template: "{{ .metadata.creationTimestamp | age }}", Width: 10},
        },
    },
}
```

## Features Supported

✅ **Dynamic Resource Rendering**: Resources are rendered based on template definitions
✅ **Sorting**: Sort by any column in ascending/descending order
✅ **Filtering**: Filter resources by text across all columns
✅ **Multi-Context**: Support for multiple Kubernetes contexts
✅ **Selection Tracking**: Track selected items across updates
✅ **Real-time Updates**: Handle resource updates from event channels
✅ **Responsive Layout**: Adapt to terminal size changes

## Testing

The implementation includes comprehensive tests in `internal/ui/views/generic_resource_view_test.go`:

- `TestGenericResourceView_NewGenericResourceView`: Tests view creation
- `TestGenericResourceView_RefreshResources`: Tests resource fetching
- `TestGenericResourceView_RenderColumns`: Tests template rendering
- `TestGenericResourceView_UpdateTable`: Tests table building
- `TestGenericResourceView_Sorting`: Tests sorting functionality
- `TestGenericResourceView_Filtering`: Tests filtering functionality
- `TestGenericResourceView_MultiContext`: Tests multi-context support
- `TestGenericResourceView_Update`: Tests message handling

All tests pass successfully.

## Migration Path

1. **Phase 1**: Implement GenericResourceView alongside existing ResourceView
2. **Phase 2**: Add feature flag to enable generic view for testing
3. **Phase 3**: Migrate resource types one by one to use definitions
4. **Phase 4**: Remove legacy hardcoded implementations
5. **Phase 5**: Make generic view the default

## Benefits

- **Extensibility**: Add new resource types without code changes
- **Consistency**: All resources follow the same rendering pattern
- **Maintainability**: Template-based columns are easier to modify
- **Testability**: Generic implementation is thoroughly tested
- **Performance**: Efficient rendering with viewport-based updates

## Next Steps

1. Load resource definitions from embedded YAML files
2. Implement custom template functions for common patterns
3. Add support for custom resource definitions (CRDs)
4. Create a resource definition validator tool
5. Build a UI for editing resource definitions