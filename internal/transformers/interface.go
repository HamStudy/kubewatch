package transformers

import (
	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/template"
)

// ResourceTransformer defines the interface for transforming K8s resources to table data
type ResourceTransformer interface {
	// GetHeaders returns the column headers for this resource type
	GetHeaders(showNamespace bool, multiContext bool) []string

	// TransformToRow converts a resource to a table row
	TransformToRow(resource interface{}, showNamespace bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error)

	// GetSortValue returns the value to use for sorting on the given column
	GetSortValue(resource interface{}, column string) interface{}

	// GetResourceType returns the resource type this transformer handles
	GetResourceType() string

	// GetUniqKey generates a unique key for resource grouping
	GetUniqKey(resource interface{}, templateEngine *template.Engine) (string, error)

	// CanGroup returns true if this resource type supports grouping
	CanGroup() bool

	// AggregateResources combines multiple resources with the same unique key
	AggregateResources(resources []interface{}, showNamespace bool, multiContext bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error)
}

// Registry manages resource transformers
type Registry struct {
	transformers map[string]ResourceTransformer
}

// NewRegistry creates a new transformer registry
func NewRegistry() *Registry {
	return &Registry{
		transformers: make(map[string]ResourceTransformer),
	}
}

// Register registers a transformer for a resource type
func (r *Registry) Register(resourceType string, transformer ResourceTransformer) {
	r.transformers[resourceType] = transformer
}

// Get returns the transformer for a resource type
func (r *Registry) Get(resourceType string) (ResourceTransformer, bool) {
	transformer, exists := r.transformers[resourceType]
	return transformer, exists
}

// GetAll returns all registered transformers
func (r *Registry) GetAll() map[string]ResourceTransformer {
	result := make(map[string]ResourceTransformer)
	for k, v := range r.transformers {
		result[k] = v
	}
	return result
}

// GetDefaultRegistry returns a registry with all default transformers
func GetDefaultRegistry() *Registry {
	registry := NewRegistry()

	// Register all default transformers
	registry.Register("Pod", NewPodTransformer())
	registry.Register("Deployment", NewDeploymentTransformer())
	registry.Register("StatefulSet", NewStatefulSetTransformer())
	registry.Register("Service", NewServiceTransformer())
	registry.Register("Ingress", NewIngressTransformer())
	registry.Register("ConfigMap", NewConfigMapTransformer())
	registry.Register("Secret", NewSecretTransformer())

	return registry
}
