package resource

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceDefinition represents a complete resource configuration
type ResourceDefinition struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

// Metadata contains resource metadata
type Metadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Icon        string `yaml:"icon"`
}

// Spec contains the resource specification
type Spec struct {
	Kubernetes KubernetesSpec `yaml:"kubernetes"`
	Columns    []Column       `yaml:"columns"`
	Operations []Operation    `yaml:"operations"`
	Grouping   Grouping       `yaml:"grouping"`
	Filters    []Filter       `yaml:"filters"`
}

// KubernetesSpec defines the Kubernetes API information
type KubernetesSpec struct {
	Group      string `yaml:"group"`
	Version    string `yaml:"version"`
	Kind       string `yaml:"kind"`
	Plural     string `yaml:"plural"`
	Namespaced bool   `yaml:"namespaced"`
}

// Column defines a table column
type Column struct {
	Name      string `yaml:"name"`
	Width     int    `yaml:"width"`
	Priority  int    `yaml:"priority"`
	Template  string `yaml:"template"`
	Sortable  bool   `yaml:"sortable"`
	Align     string `yaml:"align,omitempty"`
	Condition string `yaml:"condition,omitempty"`
	SortKey   string `yaml:"sortKey,omitempty"`
}

// Operation defines an available operation
type Operation struct {
	Name            string `yaml:"name"`
	Key             string `yaml:"key"`
	Description     string `yaml:"description"`
	Command         string `yaml:"command"`
	Confirm         bool   `yaml:"confirm"`
	ConfirmMessage  string `yaml:"confirmMessage,omitempty"`
	RequiresRunning bool   `yaml:"requiresRunning"`
	Interactive     bool   `yaml:"interactive"`
	Prompt          string `yaml:"prompt,omitempty"`
}

// Grouping defines grouping configuration
type Grouping struct {
	Enabled      bool          `yaml:"enabled"`
	GroupBy      []GroupBySpec `yaml:"groupBy"`
	Aggregations []Aggregation `yaml:"aggregations"`
}

// GroupBySpec defines a grouping field
type GroupBySpec struct {
	Field string `yaml:"field"`
	Name  string `yaml:"name"`
	Icon  string `yaml:"icon"`
}

// Aggregation defines an aggregation function
type Aggregation struct {
	Column   string `yaml:"column"`
	Function string `yaml:"function"`
	Format   string `yaml:"format"`
}

// Filter defines a filter option
type Filter struct {
	Name      string `yaml:"name"`
	Key       string `yaml:"key"`
	Condition string `yaml:"condition"`
}

// Validate validates the resource definition
func (rd *ResourceDefinition) Validate() error {
	// Validate API version and kind
	if rd.APIVersion != "kubewatch.io/v1" {
		return fmt.Errorf("apiVersion must be kubewatch.io/v1, got %s", rd.APIVersion)
	}
	if rd.Kind != "ResourceDefinition" {
		return fmt.Errorf("kind must be ResourceDefinition, got %s", rd.Kind)
	}

	// Validate metadata
	if rd.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	// Validate Kubernetes spec
	if rd.Spec.Kubernetes.Kind == "" {
		return fmt.Errorf("spec.kubernetes.kind is required")
	}
	if rd.Spec.Kubernetes.Version == "" {
		return fmt.Errorf("spec.kubernetes.version is required")
	}
	if rd.Spec.Kubernetes.Plural == "" {
		return fmt.Errorf("spec.kubernetes.plural is required")
	}

	// Validate columns
	if len(rd.Spec.Columns) == 0 {
		return fmt.Errorf("at least one column must be defined")
	}
	for i, col := range rd.Spec.Columns {
		if col.Name == "" {
			return fmt.Errorf("column[%d]: column name is required", i)
		}
		if col.Template == "" {
			return fmt.Errorf("column[%d]: column template is required", i)
		}
		if col.Width <= 0 {
			return fmt.Errorf("column[%d]: column width must be positive", i)
		}
		if col.Align != "" && col.Align != "left" && col.Align != "center" && col.Align != "right" {
			return fmt.Errorf("column[%d]: column align must be one of: left, center, right", i)
		}
	}

	// Validate operations
	for i, op := range rd.Spec.Operations {
		if op.Name == "" {
			return fmt.Errorf("operation[%d]: operation name is required", i)
		}
		if op.Key == "" {
			return fmt.Errorf("operation[%d]: operation key is required", i)
		}
		if op.Command == "" {
			return fmt.Errorf("operation[%d]: operation command is required", i)
		}
	}

	return nil
}

// GetGroupVersionKind returns the GroupVersionKind for this resource
func (rd *ResourceDefinition) GetGroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   rd.Spec.Kubernetes.Group,
		Version: rd.Spec.Kubernetes.Version,
		Kind:    rd.Spec.Kubernetes.Kind,
	}
}

// GetGroupVersionResource returns the GroupVersionResource for this resource
func (rd *ResourceDefinition) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    rd.Spec.Kubernetes.Group,
		Version:  rd.Spec.Kubernetes.Version,
		Resource: rd.Spec.Kubernetes.Plural,
	}
}

// IsNamespaced returns whether the resource is namespaced
func (rd *ResourceDefinition) IsNamespaced() bool {
	return rd.Spec.Kubernetes.Namespaced
}

// GetResourceKey returns a unique key for this resource (group/version/kind)
func (rd *ResourceDefinition) GetResourceKey() string {
	if rd.Spec.Kubernetes.Group == "" {
		return fmt.Sprintf("%s/%s", rd.Spec.Kubernetes.Version, strings.ToLower(rd.Spec.Kubernetes.Kind))
	}
	return fmt.Sprintf("%s/%s/%s",
		rd.Spec.Kubernetes.Group,
		rd.Spec.Kubernetes.Version,
		strings.ToLower(rd.Spec.Kubernetes.Kind))
}
