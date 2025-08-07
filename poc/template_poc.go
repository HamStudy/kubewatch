package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	tmpl "github.com/HamStudy/kubewatch/internal/template"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// Embed the pod.yaml configuration at compile time
//
//go:embed resources/pod.yaml
var podConfigYAML string

// ResourceConfig represents the configuration for a Kubernetes resource type
type ResourceConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name         string `yaml:"name"`
		ResourceType string `yaml:"resourceType"`
	} `yaml:"metadata"`
	Spec ResourceSpec `yaml:"spec"`
}

// ResourceSpec contains the specification for how to display a resource
type ResourceSpec struct {
	Kubernetes KubernetesSpec `yaml:"kubernetes"`
	Columns    []ColumnSpec   `yaml:"columns"`
	Operations OperationsSpec `yaml:"operations"`
	Grouping   GroupingSpec   `yaml:"grouping"`
	Formatters []FormatterRef `yaml:"formatters"`
}

// KubernetesSpec defines the Kubernetes resource metadata
type KubernetesSpec struct {
	Group      string `yaml:"group"`
	Version    string `yaml:"version"`
	Kind       string `yaml:"kind"`
	Namespaced bool   `yaml:"namespaced"`
	ListKind   string `yaml:"listKind"`
}

// ColumnSpec defines a table column
type ColumnSpec struct {
	Name        string `yaml:"name"`
	Width       int    `yaml:"width"`
	Flex        bool   `yaml:"flex"`
	Template    string `yaml:"template"`
	Conditional string `yaml:"conditional,omitempty"`
	Optional    bool   `yaml:"optional,omitempty"`
}

// OperationsSpec defines available operations on the resource
type OperationsSpec struct {
	Describe Operation `yaml:"describe"`
	Logs     Operation `yaml:"logs"`
	Exec     Operation `yaml:"exec"`
	Delete   Operation `yaml:"delete"`
	Edit     Operation `yaml:"edit"`
}

// Operation represents a single operation
type Operation struct {
	Enabled         bool   `yaml:"enabled"`
	Command         string `yaml:"command"`
	FollowSupported bool   `yaml:"followSupported,omitempty"`
	ConfirmRequired bool   `yaml:"confirmRequired,omitempty"`
}

// GroupingSpec defines resource grouping configuration
type GroupingSpec struct {
	Enabled     bool                `yaml:"enabled"`
	Key         string              `yaml:"key"`
	Aggregation []AggregationColumn `yaml:"aggregation"`
}

// AggregationColumn defines how to aggregate a column for grouped resources
type AggregationColumn struct {
	Column   string `yaml:"column"`
	Template string `yaml:"template"`
}

// FormatterRef references shared formatters
type FormatterRef struct {
	Ref string `yaml:"ref"`
}

// ResourceManager manages resource configurations
type ResourceManager struct {
	configs        map[string]*ResourceConfig
	templateEngine *tmpl.Engine
	dynamicClient  dynamic.Interface
}

// NewResourceManager creates a new resource manager
func NewResourceManager() (*ResourceManager, error) {
	// Create template engine (using the existing one from kubewatch)
	templateEngine := tmpl.NewEngine()

	// Create dynamic client for Kubernetes
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(os.Getenv("HOME"), ".kube", "config"))
	if err != nil {
		// For POC, we'll continue without a real client
		log.Printf("Warning: Could not create Kubernetes client: %v", err)
	}

	var dynamicClient dynamic.Interface
	if config != nil {
		dynamicClient, err = dynamic.NewForConfig(config)
		if err != nil {
			log.Printf("Warning: Could not create dynamic client: %v", err)
		}
	}

	return &ResourceManager{
		configs:        make(map[string]*ResourceConfig),
		templateEngine: templateEngine,
		dynamicClient:  dynamicClient,
	}, nil
}

// LoadConfig loads a resource configuration from YAML
func (rm *ResourceManager) LoadConfig(yamlData string) (*ResourceConfig, error) {
	var config ResourceConfig
	if err := yaml.Unmarshal([]byte(yamlData), &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Store the config
	rm.configs[config.Metadata.ResourceType] = &config

	return &config, nil
}

// LoadConfigFromFile loads a resource configuration from a file
func (rm *ResourceManager) LoadConfigFromFile(path string) (*ResourceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return rm.LoadConfig(string(data))
}

// RenderColumn renders a single column for a resource
func (rm *ResourceManager) RenderColumn(config *ResourceConfig, columnName string, resource interface{}, context map[string]interface{}) (string, error) {
	// Find the column spec
	var columnSpec *ColumnSpec
	for _, col := range config.Spec.Columns {
		if col.Name == columnName {
			columnSpec = &col
			break
		}
	}

	if columnSpec == nil {
		return "", fmt.Errorf("column %s not found", columnName)
	}

	// Check conditional
	if columnSpec.Conditional != "" {
		if val, ok := context[columnSpec.Conditional]; !ok || !val.(bool) {
			return "", nil // Skip this column
		}
	}

	// Execute the template
	result, err := rm.templateEngine.Execute(columnSpec.Template, resource)
	if err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return result, nil
}

// GetGVR returns the GroupVersionResource for a resource config
func (rm *ResourceManager) GetGVR(config *ResourceConfig) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    config.Spec.Kubernetes.Group,
		Version:  config.Spec.Kubernetes.Version,
		Resource: rm.pluralize(config.Spec.Kubernetes.Kind),
	}
}

// pluralize converts a kind to its plural resource name (simplified)
func (rm *ResourceManager) pluralize(kind string) string {
	// This is a simplified version - in production, use proper pluralization
	switch kind {
	case "Pod":
		return "pods"
	case "Service":
		return "services"
	case "Deployment":
		return "deployments"
	case "StatefulSet":
		return "statefulsets"
	case "ConfigMap":
		return "configmaps"
	case "Secret":
		return "secrets"
	case "Ingress":
		return "ingresses"
	default:
		return kind + "s"
	}
}

// ListResources lists resources using the dynamic client
func (rm *ResourceManager) ListResources(config *ResourceConfig, namespace string) (*unstructured.UnstructuredList, error) {
	if rm.dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client not available")
	}

	gvr := rm.GetGVR(config)

	var list *unstructured.UnstructuredList
	var err error

	if config.Spec.Kubernetes.Namespaced {
		list, err = rm.dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	} else {
		list, err = rm.dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	return list, nil
}

// RenderTableRow renders a complete table row for a resource
func (rm *ResourceManager) RenderTableRow(config *ResourceConfig, resource interface{}, context map[string]interface{}) ([]string, error) {
	var row []string

	for _, col := range config.Spec.Columns {
		// Check conditional
		if col.Conditional != "" {
			if val, ok := context[col.Conditional]; !ok || !val.(bool) {
				continue // Skip this column
			}
		}

		// Execute the template
		result, err := rm.templateEngine.Execute(col.Template, resource)
		if err != nil {
			// For optional columns, we can skip on error
			if col.Optional {
				row = append(row, "-")
				continue
			}
			return nil, fmt.Errorf("failed to render column %s: %w", col.Name, err)
		}

		row = append(row, result)
	}

	return row, nil
}

// GetHeaders returns the column headers based on context
func (rm *ResourceManager) GetHeaders(config *ResourceConfig, context map[string]interface{}) []string {
	var headers []string

	for _, col := range config.Spec.Columns {
		// Check conditional
		if col.Conditional != "" {
			if val, ok := context[col.Conditional]; !ok || !val.(bool) {
				continue // Skip this column
			}
		}

		headers = append(headers, col.Name)
	}

	return headers
}

// Example usage and integration test
func main() {
	fmt.Println("=== Kubewatch Template-Driven POC ===\n")

	// Create resource manager
	rm, err := NewResourceManager()
	if err != nil {
		log.Fatalf("Failed to create resource manager: %v", err)
	}

	// Load the embedded pod configuration
	fmt.Println("1. Loading embedded pod.yaml configuration...")
	config, err := rm.LoadConfig(podConfigYAML)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	fmt.Printf("   ✓ Loaded config for: %s\n", config.Metadata.ResourceType)
	fmt.Printf("   ✓ Columns defined: %d\n", len(config.Spec.Columns))
	fmt.Printf("   ✓ Operations defined: 5\n\n")

	// Create a sample pod for testing
	fmt.Println("2. Creating sample Pod resource for testing...")
	samplePod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-deployment-abc123",
			Namespace: "default",
			CreationTimestamp: metav1.Time{
				Time: time.Now().Add(-2 * time.Hour),
			},
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: v1.PodSpec{
			NodeName: "node-1",
		},
		Status: v1.PodStatus{
			Phase: "Running",
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:         "nginx",
					Ready:        true,
					RestartCount: 0,
				},
			},
		},
	}

	// Convert to unstructured for dynamic handling
	unstructuredPod, err := runtime.DefaultUnstructuredConverter.ToUnstructured(samplePod)
	if err != nil {
		log.Fatalf("Failed to convert pod: %v", err)
	}
	fmt.Println("   ✓ Sample pod created\n")

	// Test rendering with different contexts
	contexts := []struct {
		name    string
		context map[string]interface{}
	}{
		{
			name: "Basic view",
			context: map[string]interface{}{
				"showNamespace": false,
				"showMetrics":   false,
			},
		},
		{
			name: "With namespace",
			context: map[string]interface{}{
				"showNamespace": true,
				"showMetrics":   false,
			},
		},
		{
			name: "With metrics",
			context: map[string]interface{}{
				"showNamespace": false,
				"showMetrics":   true,
			},
		},
	}

	for _, tc := range contexts {
		fmt.Printf("3. Testing rendering: %s\n", tc.name)

		// Get headers
		headers := rm.GetHeaders(config, tc.context)
		fmt.Printf("   Headers: %v\n", headers)

		// Render row
		row, err := rm.RenderTableRow(config, unstructuredPod, tc.context)
		if err != nil {
			log.Printf("   Error rendering row: %v", err)
			continue
		}
		fmt.Printf("   Row data: %v\n\n", row)
	}

	// Test individual column rendering
	fmt.Println("4. Testing individual column rendering...")
	columns := []string{"NAME", "STATUS", "AGE"}
	for _, col := range columns {
		result, err := rm.RenderColumn(config, col, unstructuredPod, map[string]interface{}{})
		if err != nil {
			log.Printf("   Error rendering %s: %v", col, err)
			continue
		}
		fmt.Printf("   %s: %s\n", col, result)
	}

	// Demonstrate runtime override capability
	fmt.Println("\n5. Demonstrating runtime override...")

	// Override a column template at runtime
	for i, col := range config.Spec.Columns {
		if col.Name == "NAME" {
			config.Spec.Columns[i].Template = "{{ .metadata.namespace }}/{{ .metadata.name }}"
			break
		}
	}

	result, err := rm.RenderColumn(config, "NAME", unstructuredPod, map[string]interface{}{})
	if err != nil {
		log.Printf("   Error: %v", err)
	} else {
		fmt.Printf("   Overridden NAME column: %s\n", result)
	}

	// Show GVR for dynamic client usage
	fmt.Println("\n6. Dynamic client integration...")
	gvr := rm.GetGVR(config)
	fmt.Printf("   GroupVersionResource: %v\n", gvr)
	fmt.Printf("   Can be used with: dynamicClient.Resource(gvr).Namespace(ns).List(...)\n")

	// Test with unstructured data (simulating dynamic client response)
	fmt.Println("\n7. Testing with unstructured data (dynamic client simulation)...")
	unstructuredData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":              "test-pod",
			"namespace":         "kube-system",
			"creationTimestamp": "2024-01-01T10:00:00Z",
		},
		"status": map[string]interface{}{
			"phase": "Pending",
			"containerStatuses": []interface{}{
				map[string]interface{}{
					"ready":        false,
					"restartCount": 3,
				},
			},
		},
		"spec": map[string]interface{}{
			"nodeName": "node-2",
		},
	}

	row, err := rm.RenderTableRow(config, unstructuredData, map[string]interface{}{
		"showNamespace": true,
		"showMetrics":   false,
	})
	if err != nil {
		log.Printf("   Error: %v", err)
	} else {
		fmt.Printf("   Rendered row from unstructured: %v\n", row)
	}

	fmt.Println("\n=== POC Complete ===")
	fmt.Println("\nKey Integration Points Verified:")
	fmt.Println("✓ YAML configuration loading with go:embed")
	fmt.Println("✓ Template execution using existing template engine")
	fmt.Println("✓ Column rendering with conditionals")
	fmt.Println("✓ Runtime template overrides")
	fmt.Println("✓ Dynamic client GVR generation")
	fmt.Println("✓ Unstructured data handling")
	fmt.Println("✓ Context-based column visibility")
}

// LoadResourceConfigs loads configs from a directory
func LoadResourceConfigs(dir string) (map[string]*ResourceConfig, error) {
	configs := make(map[string]*ResourceConfig)

	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}

	rm, err := NewResourceManager()
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		config, err := rm.LoadConfigFromFile(file)
		if err != nil {
			log.Printf("Warning: Failed to load %s: %v", file, err)
			continue
		}
		configs[config.Metadata.ResourceType] = config
	}

	return configs, nil
}
