package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)
	assert.NotNil(t, r.definitions)
	assert.NotNil(t, r.byGVK)
	assert.NotNil(t, r.byName)
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name    string
		def     *ResourceDefinition
		wantErr bool
		errMsg  string
	}{
		{
			name: "register valid pod definition",
			def: &ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name:        "pod",
					Description: "Pod resource",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Group:      "",
						Version:    "v1",
						Kind:       "Pod",
						Plural:     "pods",
						Namespaced: true,
					},
					Columns: []Column{
						{
							Name:     "NAME",
							Width:    30,
							Priority: 1,
							Template: "{{ .metadata.name }}",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "register deployment definition",
			def: &ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name:        "deployment",
					Description: "Deployment resource",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Group:      "apps",
						Version:    "v1",
						Kind:       "Deployment",
						Plural:     "deployments",
						Namespaced: true,
					},
					Columns: []Column{
						{
							Name:     "NAME",
							Width:    30,
							Priority: 1,
							Template: "{{ .metadata.name }}",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "register invalid definition",
			def: &ResourceDefinition{
				APIVersion: "v1", // Wrong API version
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "apiVersion must be kubewatch.io/v1",
		},
		{
			name:    "register nil definition",
			def:     nil,
			wantErr: true,
			errMsg:  "definition cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			err := r.Register(tt.def)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				// Verify it was registered
				assert.Contains(t, r.definitions, tt.def.Metadata.Name)
				gvk := tt.def.GetGroupVersionKind()
				assert.Contains(t, r.byGVK, gvk)
				assert.Contains(t, r.byName, tt.def.Metadata.Name)
			}
		})
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := NewRegistry()

	def1 := &ResourceDefinition{
		APIVersion: "kubewatch.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: Metadata{
			Name:        "pod",
			Description: "First pod definition",
		},
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:      "",
				Version:    "v1",
				Kind:       "Pod",
				Plural:     "pods",
				Namespaced: true,
			},
			Columns: []Column{
				{
					Name:     "NAME",
					Width:    30,
					Priority: 1,
					Template: "{{ .metadata.name }}",
				},
			},
		},
	}

	def2 := &ResourceDefinition{
		APIVersion: "kubewatch.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: Metadata{
			Name:        "pod",
			Description: "Second pod definition (override)",
		},
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:      "",
				Version:    "v1",
				Kind:       "Pod",
				Plural:     "pods",
				Namespaced: true,
			},
			Columns: []Column{
				{
					Name:     "NAME",
					Width:    40, // Different width
					Priority: 1,
					Template: "{{ .metadata.name }}",
				},
			},
		},
	}

	// Register first definition
	err := r.Register(def1)
	require.NoError(t, err)

	// Register second definition (should override)
	err = r.Register(def2)
	assert.NoError(t, err)

	// Verify the second definition is stored
	stored := r.GetByName("pod")
	assert.NotNil(t, stored)
	assert.Equal(t, "Second pod definition (override)", stored.Metadata.Description)
	assert.Equal(t, 40, stored.Spec.Columns[0].Width)
}

func TestRegistry_GetByGVK(t *testing.T) {
	r := NewRegistry()

	// Register a pod definition
	podDef := &ResourceDefinition{
		APIVersion: "kubewatch.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: Metadata{
			Name: "pod",
		},
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:      "",
				Version:    "v1",
				Kind:       "Pod",
				Plural:     "pods",
				Namespaced: true,
			},
			Columns: []Column{
				{
					Name:     "NAME",
					Width:    30,
					Priority: 1,
					Template: "{{ .metadata.name }}",
				},
			},
		},
	}

	err := r.Register(podDef)
	require.NoError(t, err)

	// Test getting by GVK
	tests := []struct {
		name  string
		gvk   schema.GroupVersionKind
		found bool
	}{
		{
			name: "find registered pod",
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
			found: true,
		},
		{
			name: "not found - different kind",
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Service",
			},
			found: false,
		},
		{
			name: "not found - different version",
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v2",
				Kind:    "Pod",
			},
			found: false,
		},
		{
			name: "not found - different group",
			gvk: schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Pod",
			},
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.GetByGVK(tt.gvk)
			if tt.found {
				assert.NotNil(t, result)
				assert.Equal(t, "pod", result.Metadata.Name)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestRegistry_GetByName(t *testing.T) {
	r := NewRegistry()

	// Register definitions
	podDef := &ResourceDefinition{
		APIVersion: "kubewatch.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: Metadata{
			Name: "pod",
		},
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:      "",
				Version:    "v1",
				Kind:       "Pod",
				Plural:     "pods",
				Namespaced: true,
			},
			Columns: []Column{
				{
					Name:     "NAME",
					Width:    30,
					Priority: 1,
					Template: "{{ .metadata.name }}",
				},
			},
		},
	}

	deployDef := &ResourceDefinition{
		APIVersion: "kubewatch.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: Metadata{
			Name: "deployment",
		},
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:      "apps",
				Version:    "v1",
				Kind:       "Deployment",
				Plural:     "deployments",
				Namespaced: true,
			},
			Columns: []Column{
				{
					Name:     "NAME",
					Width:    30,
					Priority: 1,
					Template: "{{ .metadata.name }}",
				},
			},
		},
	}

	err := r.Register(podDef)
	require.NoError(t, err)
	err = r.Register(deployDef)
	require.NoError(t, err)

	// Test getting by name
	tests := []struct {
		name      string
		lookupKey string
		found     bool
	}{
		{
			name:      "find pod",
			lookupKey: "pod",
			found:     true,
		},
		{
			name:      "find deployment",
			lookupKey: "deployment",
			found:     true,
		},
		{
			name:      "not found",
			lookupKey: "service",
			found:     false,
		},
		{
			name:      "empty name",
			lookupKey: "",
			found:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.GetByName(tt.lookupKey)
			if tt.found {
				assert.NotNil(t, result)
				assert.Equal(t, tt.lookupKey, result.Metadata.Name)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestRegistry_GetForResource(t *testing.T) {
	r := NewRegistry()

	// Register a pod definition
	podDef := &ResourceDefinition{
		APIVersion: "kubewatch.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: Metadata{
			Name: "pod",
		},
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:      "",
				Version:    "v1",
				Kind:       "Pod",
				Plural:     "pods",
				Namespaced: true,
			},
			Columns: []Column{
				{
					Name:     "NAME",
					Width:    30,
					Priority: 1,
					Template: "{{ .metadata.name }}",
				},
			},
		},
	}

	err := r.Register(podDef)
	require.NoError(t, err)

	// Test with unstructured resources
	tests := []struct {
		name  string
		obj   *unstructured.Unstructured
		found bool
	}{
		{
			name: "find pod definition",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
				},
			},
			found: true,
		},
		{
			name: "find pod definition with group",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "core/v1", // Some tools add core group
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
				},
			},
			found: false, // Won't match because group is different
		},
		{
			name: "not found - different kind",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name": "test-service",
					},
				},
			},
			found: false,
		},
		{
			name:  "nil object",
			obj:   nil,
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.GetForResource(tt.obj)
			if tt.found {
				assert.NotNil(t, result)
				assert.Equal(t, "pod", result.Metadata.Name)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	// Empty registry
	list := r.List()
	assert.Empty(t, list)

	// Add some definitions
	defs := []*ResourceDefinition{
		{
			APIVersion: "kubewatch.io/v1",
			Kind:       "ResourceDefinition",
			Metadata: Metadata{
				Name: "pod",
			},
			Spec: Spec{
				Kubernetes: KubernetesSpec{
					Group:      "",
					Version:    "v1",
					Kind:       "Pod",
					Plural:     "pods",
					Namespaced: true,
				},
				Columns: []Column{
					{
						Name:     "NAME",
						Width:    30,
						Priority: 1,
						Template: "{{ .metadata.name }}",
					},
				},
			},
		},
		{
			APIVersion: "kubewatch.io/v1",
			Kind:       "ResourceDefinition",
			Metadata: Metadata{
				Name: "deployment",
			},
			Spec: Spec{
				Kubernetes: KubernetesSpec{
					Group:      "apps",
					Version:    "v1",
					Kind:       "Deployment",
					Plural:     "deployments",
					Namespaced: true,
				},
				Columns: []Column{
					{
						Name:     "NAME",
						Width:    30,
						Priority: 1,
						Template: "{{ .metadata.name }}",
					},
				},
			},
		},
		{
			APIVersion: "kubewatch.io/v1",
			Kind:       "ResourceDefinition",
			Metadata: Metadata{
				Name: "service",
			},
			Spec: Spec{
				Kubernetes: KubernetesSpec{
					Group:      "",
					Version:    "v1",
					Kind:       "Service",
					Plural:     "services",
					Namespaced: true,
				},
				Columns: []Column{
					{
						Name:     "NAME",
						Width:    30,
						Priority: 1,
						Template: "{{ .metadata.name }}",
					},
				},
			},
		},
	}

	for _, def := range defs {
		err := r.Register(def)
		require.NoError(t, err)
	}

	// Get list
	list = r.List()
	assert.Len(t, list, 3)

	// Verify all definitions are in the list
	names := make(map[string]bool)
	for _, def := range list {
		names[def.Metadata.Name] = true
	}
	assert.True(t, names["pod"])
	assert.True(t, names["deployment"])
	assert.True(t, names["service"])
}

func TestRegistry_Clear(t *testing.T) {
	r := NewRegistry()

	// Add a definition
	def := &ResourceDefinition{
		APIVersion: "kubewatch.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: Metadata{
			Name: "pod",
		},
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:      "",
				Version:    "v1",
				Kind:       "Pod",
				Plural:     "pods",
				Namespaced: true,
			},
			Columns: []Column{
				{
					Name:     "NAME",
					Width:    30,
					Priority: 1,
					Template: "{{ .metadata.name }}",
				},
			},
		},
	}

	err := r.Register(def)
	require.NoError(t, err)

	// Verify it's registered
	assert.Len(t, r.List(), 1)
	assert.NotNil(t, r.GetByName("pod"))

	// Clear the registry
	r.Clear()

	// Verify it's empty
	assert.Empty(t, r.List())
	assert.Nil(t, r.GetByName("pod"))
	assert.Empty(t, r.definitions)
	assert.Empty(t, r.byGVK)
	assert.Empty(t, r.byName)
}

func TestRegistry_ThreadSafety(t *testing.T) {
	r := NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create multiple definitions
	createDef := func(name string, kind string) *ResourceDefinition {
		return &ResourceDefinition{
			APIVersion: "kubewatch.io/v1",
			Kind:       "ResourceDefinition",
			Metadata: Metadata{
				Name: name,
			},
			Spec: Spec{
				Kubernetes: KubernetesSpec{
					Group:      "",
					Version:    "v1",
					Kind:       kind,
					Plural:     name + "s",
					Namespaced: true,
				},
				Columns: []Column{
					{
						Name:     "NAME",
						Width:    30,
						Priority: 1,
						Template: "{{ .metadata.name }}",
					},
				},
			},
		}
	}

	// Run concurrent operations
	done := make(chan bool)

	// Writer goroutines
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				select {
				case <-ctx.Done():
					done <- true
					return
				default:
					name := "resource" + string(rune('a'+id))
					def := createDef(name, "Kind"+string(rune('A'+id)))
					_ = r.Register(def)
				}
			}
			done <- true
		}(i)
	}

	// Reader goroutines
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				select {
				case <-ctx.Done():
					done <- true
					return
				default:
					_ = r.List()
					_ = r.GetByName("resourcea")
					_ = r.GetByGVK(schema.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "KindA",
					})
				}
			}
			done <- true
		}(i)
	}

	// Clear goroutine
	go func() {
		for i := 0; i < 3; i++ {
			select {
			case <-ctx.Done():
				done <- true
				return
			default:
				r.Clear()
			}
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 11; i++ {
		<-done
	}

	// No assertions needed - test passes if no race conditions or panics
}
