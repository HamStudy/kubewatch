package resource

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Registry manages resource definitions
type Registry struct {
	mu          sync.RWMutex
	definitions map[string]*ResourceDefinition                  // key: resource name
	byGVK       map[schema.GroupVersionKind]*ResourceDefinition // key: GVK
	byName      map[string]*ResourceDefinition                  // key: resource name (same as definitions, for consistency)
}

// NewRegistry creates a new resource registry
func NewRegistry() *Registry {
	return &Registry{
		definitions: make(map[string]*ResourceDefinition),
		byGVK:       make(map[schema.GroupVersionKind]*ResourceDefinition),
		byName:      make(map[string]*ResourceDefinition),
	}
}

// Register adds or updates a resource definition in the registry
func (r *Registry) Register(def *ResourceDefinition) error {
	if def == nil {
		return fmt.Errorf("definition cannot be nil")
	}

	// Validate the definition
	if err := def.Validate(); err != nil {
		return fmt.Errorf("invalid resource definition: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Store by name
	r.definitions[def.Metadata.Name] = def
	r.byName[def.Metadata.Name] = def

	// Store by GVK
	gvk := def.GetGroupVersionKind()
	r.byGVK[gvk] = def

	return nil
}

// GetByGVK retrieves a resource definition by its GroupVersionKind
func (r *Registry) GetByGVK(gvk schema.GroupVersionKind) *ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.byGVK[gvk]
}

// GetByName retrieves a resource definition by its name
func (r *Registry) GetByName(name string) *ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.byName[name]
}

// GetForResource retrieves a resource definition for an unstructured resource
func (r *Registry) GetForResource(obj *unstructured.Unstructured) *ResourceDefinition {
	if obj == nil {
		return nil
	}

	gvk := obj.GroupVersionKind()
	return r.GetByGVK(gvk)
}

// List returns all registered resource definitions
func (r *Registry) List() []*ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ResourceDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		result = append(result, def)
	}
	return result
}

// Clear removes all resource definitions from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.definitions = make(map[string]*ResourceDefinition)
	r.byGVK = make(map[schema.GroupVersionKind]*ResourceDefinition)
	r.byName = make(map[string]*ResourceDefinition)
}

// Has checks if a resource definition exists by name
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.byName[name]
	return exists
}

// HasGVK checks if a resource definition exists by GVK
func (r *Registry) HasGVK(gvk schema.GroupVersionKind) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.byGVK[gvk]
	return exists
}

// Count returns the number of registered resource definitions
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.definitions)
}

// GetSupportedGVKs returns all supported GroupVersionKinds
func (r *Registry) GetSupportedGVKs() []schema.GroupVersionKind {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]schema.GroupVersionKind, 0, len(r.byGVK))
	for gvk := range r.byGVK {
		result = append(result, gvk)
	}
	return result
}

// GetSupportedResources returns all supported resource names
func (r *Registry) GetSupportedResources() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(r.definitions))
	for name := range r.definitions {
		result = append(result, name)
	}
	return result
}
