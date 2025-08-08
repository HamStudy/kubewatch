package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedResourcesLoad(t *testing.T) {
	tests := []struct {
		name              string
		expectedResources []string
	}{
		{
			name: "loads all embedded resources",
			expectedResources: []string{
				"pod",
				"deployment",
				"service",
				"configmap",
				"secret",
				"ingress",
				"statefulset",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader()
			err := loader.LoadEmbedded()
			require.NoError(t, err, "LoadEmbedded should not return an error")

			registry := loader.GetRegistry()
			require.NotNil(t, registry, "Registry should not be nil")

			// Check that all expected resources are loaded
			for _, resourceName := range tt.expectedResources {
				def := registry.GetByName(resourceName)
				assert.NotNil(t, def, "Resource %s should be loaded", resourceName)
				if def != nil {
					assert.Equal(t, resourceName, def.Metadata.Name, "Resource name should match")
					assert.NotEmpty(t, def.APIVersion, "APIVersion should not be empty for %s", resourceName)
					assert.NotEmpty(t, def.Kind, "Kind should not be empty for %s", resourceName)
				}
			}

			// Verify we have at least the expected number of resources
			allResources := registry.List()
			assert.GreaterOrEqual(t, len(allResources), len(tt.expectedResources),
				"Should have at least %d resources loaded", len(tt.expectedResources))
		})
	}
}

func TestEmbeddedResourcesContent(t *testing.T) {
	loader := NewLoader()
	err := loader.LoadEmbedded()
	require.NoError(t, err)

	registry := loader.GetRegistry()

	tests := []struct {
		name         string
		resourceName string
		validate     func(t *testing.T, def *ResourceDefinition)
	}{
		{
			name:         "Pod resource has correct structure",
			resourceName: "pod",
			validate: func(t *testing.T, def *ResourceDefinition) {
				assert.Equal(t, "kubewatch.io/v1", def.APIVersion)
				assert.Equal(t, "ResourceDefinition", def.Kind)
				assert.Equal(t, "v1", def.Spec.Kubernetes.Version)
				assert.Equal(t, "Pod", def.Spec.Kubernetes.Kind)
				assert.NotEmpty(t, def.Spec.Columns, "Pod should have columns defined")
				// Check for essential pod columns
				hasName := false
				hasStatus := false
				hasNamespace := false
				for _, col := range def.Spec.Columns {
					switch col.Name {
					case "NAME":
						hasName = true
					case "STATUS":
						hasStatus = true
					case "NAMESPACE":
						hasNamespace = true
					}
				}
				assert.True(t, hasName, "Pod should have Name column")
				assert.True(t, hasStatus, "Pod should have Status column")
				assert.True(t, hasNamespace, "Pod should have Namespace column")
			},
		},
		{
			name:         "Deployment resource has correct structure",
			resourceName: "deployment",
			validate: func(t *testing.T, def *ResourceDefinition) {
				assert.Equal(t, "kubewatch.io/v1", def.APIVersion)
				assert.Equal(t, "ResourceDefinition", def.Kind)
				assert.Equal(t, "apps", def.Spec.Kubernetes.Group)
				assert.Equal(t, "v1", def.Spec.Kubernetes.Version)
				assert.Equal(t, "Deployment", def.Spec.Kubernetes.Kind)
				assert.NotEmpty(t, def.Spec.Columns, "Deployment should have columns defined")
				// Check for essential deployment columns
				hasName := false
				hasReplicas := false
				for _, col := range def.Spec.Columns {
					switch col.Name {
					case "NAME":
						hasName = true
					case "REPLICAS", "READY":
						hasReplicas = true
					}
				}
				assert.True(t, hasName, "Deployment should have Name column")
				assert.True(t, hasReplicas, "Deployment should have Replicas or Ready column")
			},
		},
		{
			name:         "Service resource has correct structure",
			resourceName: "service",
			validate: func(t *testing.T, def *ResourceDefinition) {
				assert.Equal(t, "kubewatch.io/v1", def.APIVersion)
				assert.Equal(t, "ResourceDefinition", def.Kind)
				assert.Equal(t, "v1", def.Spec.Kubernetes.Version)
				assert.Equal(t, "Service", def.Spec.Kubernetes.Kind)
				assert.NotEmpty(t, def.Spec.Columns, "Service should have columns defined")
			},
		},
		{
			name:         "ConfigMap resource has correct structure",
			resourceName: "configmap",
			validate: func(t *testing.T, def *ResourceDefinition) {
				assert.Equal(t, "kubewatch.io/v1", def.APIVersion)
				assert.Equal(t, "ResourceDefinition", def.Kind)
				assert.Equal(t, "v1", def.Spec.Kubernetes.Version)
				assert.Equal(t, "ConfigMap", def.Spec.Kubernetes.Kind)
				assert.NotEmpty(t, def.Spec.Columns, "ConfigMap should have columns defined")
			},
		},
		{
			name:         "Secret resource has correct structure",
			resourceName: "secret",
			validate: func(t *testing.T, def *ResourceDefinition) {
				assert.Equal(t, "kubewatch.io/v1", def.APIVersion)
				assert.Equal(t, "ResourceDefinition", def.Kind)
				assert.Equal(t, "v1", def.Spec.Kubernetes.Version)
				assert.Equal(t, "Secret", def.Spec.Kubernetes.Kind)
				assert.NotEmpty(t, def.Spec.Columns, "Secret should have columns defined")
			},
		},
		{
			name:         "Ingress resource has correct structure",
			resourceName: "ingress",
			validate: func(t *testing.T, def *ResourceDefinition) {
				assert.Equal(t, "kubewatch.io/v1", def.APIVersion)
				assert.Equal(t, "ResourceDefinition", def.Kind)
				assert.Equal(t, "networking.k8s.io", def.Spec.Kubernetes.Group)
				assert.Equal(t, "v1", def.Spec.Kubernetes.Version)
				assert.Equal(t, "Ingress", def.Spec.Kubernetes.Kind)
				assert.NotEmpty(t, def.Spec.Columns, "Ingress should have columns defined")
			},
		},
		{
			name:         "StatefulSet resource has correct structure",
			resourceName: "statefulset",
			validate: func(t *testing.T, def *ResourceDefinition) {
				assert.Equal(t, "kubewatch.io/v1", def.APIVersion)
				assert.Equal(t, "ResourceDefinition", def.Kind)
				assert.Equal(t, "apps", def.Spec.Kubernetes.Group)
				assert.Equal(t, "v1", def.Spec.Kubernetes.Version)
				assert.Equal(t, "StatefulSet", def.Spec.Kubernetes.Kind)
				assert.NotEmpty(t, def.Spec.Columns, "StatefulSet should have columns defined")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := registry.GetByName(tt.resourceName)
			require.NotNil(t, def, "Resource %s should be loaded", tt.resourceName)
			tt.validate(t, def)
		})
	}
}

func TestEmbeddedResourcesNoErrors(t *testing.T) {
	// Test that loading embedded resources multiple times doesn't cause issues
	loader1 := NewLoader()
	err1 := loader1.LoadEmbedded()
	require.NoError(t, err1, "First load should succeed")

	loader2 := NewLoader()
	err2 := loader2.LoadEmbedded()
	require.NoError(t, err2, "Second load should succeed")

	// Both loaders should have the same resources
	registry1 := loader1.GetRegistry()
	registry2 := loader2.GetRegistry()

	resources1 := registry1.List()
	resources2 := registry2.List()

	assert.Equal(t, len(resources1), len(resources2), "Both loaders should have same number of resources")

	// Check that all resources from loader1 exist in loader2
	for _, res1 := range resources1 {
		res2 := registry2.GetByName(res1.Metadata.Name)
		assert.NotNil(t, res2, "Resource %s should exist in both loaders", res1.Metadata.Name)
		if res2 != nil {
			assert.Equal(t, res1.APIVersion, res2.APIVersion, "APIVersion should match for %s", res1.Metadata.Name)
			assert.Equal(t, res1.Kind, res2.Kind, "Kind should match for %s", res1.Metadata.Name)
		}
	}
}

func TestLoadAllWithEmbedded(t *testing.T) {
	loader := NewLoader()
	err := loader.LoadAll()
	require.NoError(t, err, "LoadAll should not return an error")

	registry := loader.GetRegistry()
	resources := registry.List()
	assert.NotEmpty(t, resources, "Should have loaded some resources")

	// Verify core resources are present
	coreResources := []string{"pod", "deployment", "service", "configmap", "secret"}
	for _, name := range coreResources {
		def := registry.GetByName(name)
		assert.NotNil(t, def, "Core resource %s should be loaded", name)
	}
}

func TestEmbeddedResourcesIntegrity(t *testing.T) {
	loader := NewLoader()
	err := loader.LoadEmbedded()
	require.NoError(t, err)

	registry := loader.GetRegistry()
	resources := registry.List()

	for _, def := range resources {
		t.Run(def.Metadata.Name, func(t *testing.T) {
			// Basic validation for all resources
			assert.NotEmpty(t, def.Metadata.Name, "Resource name should not be empty")
			assert.NotEmpty(t, def.APIVersion, "APIVersion should not be empty")
			assert.NotEmpty(t, def.Kind, "Kind should not be empty")
			assert.NotEmpty(t, def.Spec.Columns, "Columns should not be empty")

			// Validate columns
			for i, col := range def.Spec.Columns {
				assert.NotEmpty(t, col.Name, "Column %d name should not be empty", i)
				assert.NotEmpty(t, col.Template, "Column %d template should not be empty", i)
				// Width should be reasonable
				if col.Width > 0 {
					assert.LessOrEqual(t, col.Width, 100, "Column %d width should be reasonable", i)
				}
			}

			// If there are operations, validate them
			for i, op := range def.Spec.Operations {
				assert.NotEmpty(t, op.Name, "Operation %d name should not be empty", i)
				assert.NotEmpty(t, op.Key, "Operation %d key should not be empty", i)
				assert.NotEmpty(t, op.Description, "Operation %d description should not be empty", i)
			}
		})
	}
}
