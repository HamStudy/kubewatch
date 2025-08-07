package transformers

import (
	"fmt"
	"strings"

	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/template"
	corev1 "k8s.io/api/core/v1"
)

// ConfigMapTransformer handles ConfigMap resource transformation
type ConfigMapTransformer struct{}

// NewConfigMapTransformer creates a new ConfigMap transformer
func NewConfigMapTransformer() *ConfigMapTransformer {
	return &ConfigMapTransformer{}
}

// GetResourceType returns the resource type
func (t *ConfigMapTransformer) GetResourceType() string {
	return "ConfigMap"
}

// GetHeaders returns column headers for ConfigMaps
func (t *ConfigMapTransformer) GetHeaders(showNamespace bool, multiContext bool) []string {
	headers := []string{"NAME", "DATA", "AGE"}

	if showNamespace {
		headers = append([]string{"NAMESPACE"}, headers...)
	}

	if multiContext {
		headers = append([]string{"CONTEXT"}, headers...)
	}

	return headers
}

// TransformToRow converts a ConfigMap to a table row
func (t *ConfigMapTransformer) TransformToRow(resource interface{}, showNamespace bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	configMap, ok := resource.(*corev1.ConfigMap)
	if !ok {
		return nil, nil, fmt.Errorf("expected *corev1.ConfigMap, got %T", resource)
	}

	// Create resource identity
	identity := &selection.ResourceIdentity{
		Name:      configMap.Name,
		Namespace: configMap.Namespace,
		Kind:      "ConfigMap",
		Context:   "", // Will be set by caller if needed
	}

	// Basic formatting
	age := getAge(configMap.CreationTimestamp.Time)
	dataCount := fmt.Sprintf("%d", len(configMap.Data))

	row := []string{
		configMap.Name,
		dataCount,
		age,
	}

	if showNamespace {
		row = append([]string{configMap.Namespace}, row...)
	}

	return row, identity, nil
}

// GetSortValue returns the value for sorting on a given column
func (t *ConfigMapTransformer) GetSortValue(resource interface{}, column string) interface{} {
	configMap, ok := resource.(*corev1.ConfigMap)
	if !ok {
		return ""
	}

	switch strings.ToUpper(column) {
	case "NAME":
		return configMap.Name
	case "NAMESPACE":
		return configMap.Namespace
	case "DATA":
		return len(configMap.Data)
	case "AGE":
		return configMap.CreationTimestamp.Time
	default:
		return configMap.Name
	}
}
