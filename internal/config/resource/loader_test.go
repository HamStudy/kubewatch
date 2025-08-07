package resource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	l := NewLoader()
	assert.NotNil(t, l)
	assert.NotNil(t, l.registry)
}

func TestLoader_LoadEmbedded(t *testing.T) {
	l := NewLoader()

	err := l.LoadEmbedded()
	assert.NoError(t, err)

	// Check that core resources are loaded
	registry := l.GetRegistry()
	assert.NotNil(t, registry)

	// Verify pod definition is loaded
	podDef := registry.GetByName("pod")
	assert.NotNil(t, podDef)
	assert.Equal(t, "pod", podDef.Metadata.Name)
	assert.Equal(t, "Pod", podDef.Spec.Kubernetes.Kind)

	// Verify deployment definition is loaded
	deployDef := registry.GetByName("deployment")
	assert.NotNil(t, deployDef)
	assert.Equal(t, "deployment", deployDef.Metadata.Name)
	assert.Equal(t, "Deployment", deployDef.Spec.Kubernetes.Kind)
}

func TestLoader_LoadFromFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "kubewatch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test YAML file
	testYAML := `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: testresource
  description: Test Resource
spec:
  kubernetes:
    group: test.io
    version: v1
    kind: TestResource
    plural: testresources
    namespaced: true
  columns:
    - name: NAME
      width: 30
      priority: 1
      template: "{{ .metadata.name }}"
`

	testFile := filepath.Join(tmpDir, "test.yaml")
	err = os.WriteFile(testFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Load the file
	l := NewLoader()
	err = l.LoadFromFile(testFile)
	assert.NoError(t, err)

	// Verify it was loaded
	def := l.GetRegistry().GetByName("testresource")
	assert.NotNil(t, def)
	assert.Equal(t, "testresource", def.Metadata.Name)
	assert.Equal(t, "TestResource", def.Spec.Kubernetes.Kind)
}

func TestLoader_LoadFromFile_InvalidYAML(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "kubewatch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create an invalid YAML file
	testYAML := `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: [invalid
`

	testFile := filepath.Join(tmpDir, "invalid.yaml")
	err = os.WriteFile(testFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Try to load the file
	l := NewLoader()
	err = l.LoadFromFile(testFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestLoader_LoadFromFile_InvalidDefinition(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "kubewatch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a YAML file with invalid definition
	testYAML := `
apiVersion: v1  # Wrong API version
kind: ResourceDefinition
metadata:
  name: invalid
spec:
  kubernetes:
    kind: Test
`

	testFile := filepath.Join(tmpDir, "invalid.yaml")
	err = os.WriteFile(testFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Try to load the file
	l := NewLoader()
	err = l.LoadFromFile(testFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register")
}

func TestLoader_LoadFromFile_NonExistent(t *testing.T) {
	l := NewLoader()
	err := l.LoadFromFile("/non/existent/file.yaml")
	assert.Error(t, err)
}

func TestLoader_LoadFromDirectory(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "kubewatch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create multiple test YAML files
	testFiles := map[string]string{
		"resource1.yaml": `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: resource1
  description: Resource 1
spec:
  kubernetes:
    group: test.io
    version: v1
    kind: Resource1
    plural: resource1s
    namespaced: true
  columns:
    - name: NAME
      width: 30
      priority: 1
      template: "{{ .metadata.name }}"
`,
		"resource2.yml": `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: resource2
  description: Resource 2
spec:
  kubernetes:
    group: test.io
    version: v1
    kind: Resource2
    plural: resource2s
    namespaced: false
  columns:
    - name: NAME
      width: 30
      priority: 1
      template: "{{ .metadata.name }}"
`,
		"not-a-resource.txt": "This is not a YAML file",
		"skip.json":          `{"skip": "this"}`,
	}

	for filename, content := range testFiles {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create a subdirectory with another resource
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	subResource := `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: subresource
  description: Sub Resource
spec:
  kubernetes:
    group: test.io
    version: v1
    kind: SubResource
    plural: subresources
    namespaced: true
  columns:
    - name: NAME
      width: 30
      priority: 1
      template: "{{ .metadata.name }}"
`
	err = os.WriteFile(filepath.Join(subDir, "sub.yaml"), []byte(subResource), 0644)
	require.NoError(t, err)

	// Load from directory (non-recursive)
	l := NewLoader()
	err = l.LoadFromDirectory(tmpDir, false)
	assert.NoError(t, err)

	// Verify resources were loaded
	registry := l.GetRegistry()
	assert.NotNil(t, registry.GetByName("resource1"))
	assert.NotNil(t, registry.GetByName("resource2"))
	assert.Nil(t, registry.GetByName("subresource")) // Should not be loaded (non-recursive)

	// Load from directory (recursive)
	l2 := NewLoader()
	err = l2.LoadFromDirectory(tmpDir, true)
	assert.NoError(t, err)

	// Verify all resources were loaded including subdirectory
	registry2 := l2.GetRegistry()
	assert.NotNil(t, registry2.GetByName("resource1"))
	assert.NotNil(t, registry2.GetByName("resource2"))
	assert.NotNil(t, registry2.GetByName("subresource")) // Should be loaded (recursive)
}

func TestLoader_LoadFromDirectory_NonExistent(t *testing.T) {
	l := NewLoader()
	err := l.LoadFromDirectory("/non/existent/directory", false)
	assert.Error(t, err)
}

func TestLoader_LoadFromDirectory_EmptyDir(t *testing.T) {
	// Create an empty temporary directory
	tmpDir, err := os.MkdirTemp("", "kubewatch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	l := NewLoader()
	err = l.LoadFromDirectory(tmpDir, false)
	assert.NoError(t, err) // Should not error on empty directory

	// Registry should be empty
	assert.Equal(t, 0, l.GetRegistry().Count())
}

func TestLoader_LoadUserOverrides(t *testing.T) {
	// Create a temporary home directory
	tmpHome, err := os.MkdirTemp("", "kubewatch-home-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	// Set HOME environment variable
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Create the config directory structure
	configDir := filepath.Join(tmpHome, ".config", "kubewatch", "resources")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Create an override file
	overrideYAML := `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: pod
  description: Custom Pod Definition
spec:
  kubernetes:
    group: ""
    version: v1
    kind: Pod
    plural: pods
    namespaced: true
  columns:
    - name: CUSTOM
      width: 50
      priority: 1
      template: "{{ .metadata.name }}"
`

	err = os.WriteFile(filepath.Join(configDir, "pod.yaml"), []byte(overrideYAML), 0644)
	require.NoError(t, err)

	// Load with user overrides
	l := NewLoader()

	// First load embedded
	err = l.LoadEmbedded()
	require.NoError(t, err)

	// Then load user overrides
	err = l.LoadUserOverrides()
	assert.NoError(t, err)

	// Verify the override was applied
	podDef := l.GetRegistry().GetByName("pod")
	assert.NotNil(t, podDef)
	assert.Equal(t, "Custom Pod Definition", podDef.Metadata.Description)
	assert.Equal(t, "CUSTOM", podDef.Spec.Columns[0].Name)
	assert.Equal(t, 50, podDef.Spec.Columns[0].Width)
}

func TestLoader_LoadUserOverrides_NoConfigDir(t *testing.T) {
	// Create a temporary home directory without config
	tmpHome, err := os.MkdirTemp("", "kubewatch-home-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	// Set HOME environment variable
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Load with user overrides (should not error if directory doesn't exist)
	l := NewLoader()
	err = l.LoadUserOverrides()
	assert.NoError(t, err)
}

func TestLoader_LoadAll(t *testing.T) {
	// Create a temporary home directory
	tmpHome, err := os.MkdirTemp("", "kubewatch-home-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	// Set HOME environment variable
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Create the config directory structure
	configDir := filepath.Join(tmpHome, ".config", "kubewatch", "resources")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Create a custom resource
	customYAML := `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: customresource
  description: Custom Resource
spec:
  kubernetes:
    group: custom.io
    version: v1
    kind: CustomResource
    plural: customresources
    namespaced: true
  columns:
    - name: NAME
      width: 30
      priority: 1
      template: "{{ .metadata.name }}"
`

	err = os.WriteFile(filepath.Join(configDir, "custom.yaml"), []byte(customYAML), 0644)
	require.NoError(t, err)

	// Load all (embedded + user overrides)
	l := NewLoader()
	err = l.LoadAll()
	assert.NoError(t, err)

	registry := l.GetRegistry()

	// Verify embedded resources are loaded
	assert.NotNil(t, registry.GetByName("pod"))
	assert.NotNil(t, registry.GetByName("deployment"))

	// Verify custom resource is loaded
	assert.NotNil(t, registry.GetByName("customresource"))
}

func TestLoader_GetRegistry(t *testing.T) {
	l := NewLoader()
	registry := l.GetRegistry()
	assert.NotNil(t, registry)

	// Should return the same registry instance
	registry2 := l.GetRegistry()
	assert.Same(t, registry, registry2)
}

func TestLoader_Clear(t *testing.T) {
	l := NewLoader()

	// Load embedded resources
	err := l.LoadEmbedded()
	require.NoError(t, err)

	// Verify resources are loaded
	assert.True(t, l.GetRegistry().Count() > 0)

	// Clear the loader
	l.Clear()

	// Verify resources are cleared
	assert.Equal(t, 0, l.GetRegistry().Count())
}
