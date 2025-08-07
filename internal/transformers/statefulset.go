package transformers

import (
	"fmt"
	"strings"

	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/template"
	appsv1 "k8s.io/api/apps/v1"
)

// StatefulSetTransformer handles StatefulSet resource transformation
type StatefulSetTransformer struct{}

// NewStatefulSetTransformer creates a new StatefulSet transformer
func NewStatefulSetTransformer() *StatefulSetTransformer {
	return &StatefulSetTransformer{}
}

// GetResourceType returns the resource type
func (t *StatefulSetTransformer) GetResourceType() string {
	return "StatefulSet"
}

// GetHeaders returns column headers for StatefulSets
func (t *StatefulSetTransformer) GetHeaders(showNamespace bool, multiContext bool) []string {
	headers := []string{"NAME", "READY", "AGE"}

	if showNamespace {
		headers = append([]string{"NAMESPACE"}, headers...)
	}

	if multiContext {
		headers = append([]string{"CONTEXT"}, headers...)
	}

	return headers
}

// TransformToRow converts a StatefulSet to a table row
func (t *StatefulSetTransformer) TransformToRow(resource interface{}, showNamespace bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	statefulSet, ok := resource.(*appsv1.StatefulSet)
	if !ok {
		return nil, nil, fmt.Errorf("expected *appsv1.StatefulSet, got %T", resource)
	}

	// Create resource identity
	identity := &selection.ResourceIdentity{
		Name:      statefulSet.Name,
		Namespace: statefulSet.Namespace,
		Kind:      "StatefulSet",
		Context:   "", // Will be set by caller if needed
	}

	// Use template engine to format the row
	data := map[string]interface{}{
		"Name":        statefulSet.Name,
		"Namespace":   statefulSet.Namespace,
		"Ready":       fmt.Sprintf("%d/%d", statefulSet.Status.ReadyReplicas, *statefulSet.Spec.Replicas),
		"Age":         statefulSet.CreationTimestamp.Time,
		"StatefulSet": statefulSet,
	}

	// Get template for statefulset row
	templateName := "statefulset_row"
	if showNamespace {
		templateName = "statefulset_row_with_namespace"
	}

	result, err := templateEngine.Execute(templateName, data)
	if err != nil {
		// Fallback to basic formatting if template fails
		return t.formatBasicRow(statefulSet, showNamespace), identity, nil
	}

	// Split template result into columns
	columns := strings.Split(strings.TrimSpace(result), "\t")
	return columns, identity, nil
}

// GetSortValue returns the value for sorting on a given column
func (t *StatefulSetTransformer) GetSortValue(resource interface{}, column string) interface{} {
	statefulSet, ok := resource.(*appsv1.StatefulSet)
	if !ok {
		return ""
	}

	switch strings.ToUpper(column) {
	case "NAME":
		return statefulSet.Name
	case "NAMESPACE":
		return statefulSet.Namespace
	case "READY":
		return statefulSet.Status.ReadyReplicas
	case "AGE":
		return statefulSet.CreationTimestamp.Time
	default:
		return statefulSet.Name
	}
}

// formatBasicRow provides fallback formatting when templates fail
func (t *StatefulSetTransformer) formatBasicRow(statefulSet *appsv1.StatefulSet, showNamespace bool) []string {
	age := getAge(statefulSet.CreationTimestamp.Time)
	ready := fmt.Sprintf("%d/%d", statefulSet.Status.ReadyReplicas, *statefulSet.Spec.Replicas)

	row := []string{
		statefulSet.Name,
		ready,
		age,
	}

	if showNamespace {
		row = append([]string{statefulSet.Namespace}, row...)
	}

	return row
}
