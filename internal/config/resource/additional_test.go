package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestResourceDefinition_GetResourceKey(t *testing.T) {
	tests := []struct {
		name string
		rd   ResourceDefinition
		want string
	}{
		{
			name: "core resource (empty group)",
			rd: ResourceDefinition{
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
				},
			},
			want: "v1/pod",
		},
		{
			name: "resource with group",
			rd: ResourceDefinition{
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Group:   "apps",
						Version: "v1",
						Kind:    "Deployment",
					},
				},
			},
			want: "apps/v1/deployment",
		},
		{
			name: "custom resource",
			rd: ResourceDefinition{
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Group:   "custom.io",
						Version: "v1beta1",
						Kind:    "CustomResource",
					},
				},
			},
			want: "custom.io/v1beta1/customresource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rd.GetResourceKey()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRegistry_Has(t *testing.T) {
	r := NewRegistry()

	// Register a resource
	def := &ResourceDefinition{
		APIVersion: "kubewatch.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: Metadata{
			Name: "test",
		},
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:      "",
				Version:    "v1",
				Kind:       "Test",
				Plural:     "tests",
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

	// Test Has method
	assert.True(t, r.Has("test"))
	assert.False(t, r.Has("nonexistent"))
	assert.False(t, r.Has(""))
}

func TestRegistry_HasGVK(t *testing.T) {
	r := NewRegistry()

	// Register a resource
	def := &ResourceDefinition{
		APIVersion: "kubewatch.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: Metadata{
			Name: "test",
		},
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:      "test.io",
				Version:    "v1",
				Kind:       "TestKind",
				Plural:     "testkinds",
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

	// Test HasGVK method
	assert.True(t, r.HasGVK(schema.GroupVersionKind{
		Group:   "test.io",
		Version: "v1",
		Kind:    "TestKind",
	}))
	assert.False(t, r.HasGVK(schema.GroupVersionKind{
		Group:   "test.io",
		Version: "v2",
		Kind:    "TestKind",
	}))
	assert.False(t, r.HasGVK(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}))
}

func TestRegistry_Count(t *testing.T) {
	r := NewRegistry()

	// Initially empty
	assert.Equal(t, 0, r.Count())

	// Add resources
	for i := 0; i < 5; i++ {
		def := &ResourceDefinition{
			APIVersion: "kubewatch.io/v1",
			Kind:       "ResourceDefinition",
			Metadata: Metadata{
				Name: string(rune('a' + i)),
			},
			Spec: Spec{
				Kubernetes: KubernetesSpec{
					Group:      "",
					Version:    "v1",
					Kind:       "Kind" + string(rune('A'+i)),
					Plural:     "kinds",
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
	}

	assert.Equal(t, 5, r.Count())

	// Clear and check again
	r.Clear()
	assert.Equal(t, 0, r.Count())
}

func TestRegistry_GetSupportedGVKs(t *testing.T) {
	r := NewRegistry()

	// Register multiple resources
	defs := []struct {
		name  string
		group string
		kind  string
	}{
		{"pod", "", "Pod"},
		{"deployment", "apps", "Deployment"},
		{"service", "", "Service"},
	}

	for _, d := range defs {
		def := &ResourceDefinition{
			APIVersion: "kubewatch.io/v1",
			Kind:       "ResourceDefinition",
			Metadata: Metadata{
				Name: d.name,
			},
			Spec: Spec{
				Kubernetes: KubernetesSpec{
					Group:      d.group,
					Version:    "v1",
					Kind:       d.kind,
					Plural:     d.name + "s",
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
	}

	gvks := r.GetSupportedGVKs()
	assert.Len(t, gvks, 3)

	// Check that all expected GVKs are present
	gvkMap := make(map[schema.GroupVersionKind]bool)
	for _, gvk := range gvks {
		gvkMap[gvk] = true
	}

	assert.True(t, gvkMap[schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}])
	assert.True(t, gvkMap[schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}])
	assert.True(t, gvkMap[schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"}])
}

func TestRegistry_GetSupportedResources(t *testing.T) {
	r := NewRegistry()

	// Register multiple resources
	names := []string{"pod", "deployment", "service", "configmap"}

	for _, name := range names {
		def := &ResourceDefinition{
			APIVersion: "kubewatch.io/v1",
			Kind:       "ResourceDefinition",
			Metadata: Metadata{
				Name: name,
			},
			Spec: Spec{
				Kubernetes: KubernetesSpec{
					Group:      "",
					Version:    "v1",
					Kind:       "Kind",
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
		err := r.Register(def)
		require.NoError(t, err)
	}

	resources := r.GetSupportedResources()
	assert.Len(t, resources, 4)

	// Check that all expected resources are present
	resourceMap := make(map[string]bool)
	for _, res := range resources {
		resourceMap[res] = true
	}

	for _, name := range names {
		assert.True(t, resourceMap[name], "Expected resource %s to be in supported list", name)
	}
}

func TestLoader_LoadFromData(t *testing.T) {
	l := NewLoader()

	yamlData := []byte(`
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: fromdata
  description: Loaded from data
spec:
  kubernetes:
    group: test.io
    version: v1
    kind: FromData
    plural: fromdatas
    namespaced: true
  columns:
    - name: NAME
      width: 30
      priority: 1
      template: "{{ .metadata.name }}"
`)

	err := l.LoadFromData(yamlData)
	assert.NoError(t, err)

	// Verify it was loaded
	def := l.GetRegistry().GetByName("fromdata")
	assert.NotNil(t, def)
	assert.Equal(t, "fromdata", def.Metadata.Name)
	assert.Equal(t, "Loaded from data", def.Metadata.Description)
}

func TestLoader_LoadFromData_Invalid(t *testing.T) {
	l := NewLoader()

	// Invalid YAML
	err := l.LoadFromData([]byte("invalid: [yaml"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")

	// Invalid definition
	err = l.LoadFromData([]byte(`
apiVersion: v1
kind: ResourceDefinition
metadata:
  name: invalid
`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register")
}

func TestGetDefaultLoader(t *testing.T) {
	// This test might fail if HOME is not set or config dir doesn't exist
	// but it should at least load embedded resources
	loader, err := GetDefaultLoader()
	assert.NoError(t, err)
	assert.NotNil(t, loader)

	// Check that embedded resources are loaded
	registry := loader.GetRegistry()
	assert.NotNil(t, registry.GetByName("pod"))
	assert.NotNil(t, registry.GetByName("deployment"))
}
