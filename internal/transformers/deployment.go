package transformers

import (
	"fmt"
	"strings"

	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/template"
	appsv1 "k8s.io/api/apps/v1"
)

// DeploymentTransformer handles Deployment resource transformation
type DeploymentTransformer struct{}

// NewDeploymentTransformer creates a new deployment transformer
func NewDeploymentTransformer() *DeploymentTransformer {
	return &DeploymentTransformer{}
}

// GetResourceType returns the resource type
func (t *DeploymentTransformer) GetResourceType() string {
	return "Deployment"
}

// GetHeaders returns the column headers for deployments
func (t *DeploymentTransformer) GetHeaders(showNamespace bool, multiContext bool) []string {
	var headers []string

	if multiContext {
		headers = append(headers, "CONTEXT")
	}

	headers = append(headers, "NAME")

	if showNamespace {
		headers = append(headers, "NAMESPACE")
	}

	headers = append(headers, "READY", "UP-TO-DATE", "AVAILABLE", "AGE", "CONTAINERS", "IMAGES", "SELECTOR")

	return headers
}

// TransformToRow converts a deployment to a table row
func (t *DeploymentTransformer) TransformToRow(resource interface{}, showNamespace bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	deployment, ok := resource.(appsv1.Deployment)
	if !ok {
		return nil, nil, fmt.Errorf("expected Deployment, got %T", resource)
	}

	// Create resource identity
	identity := &selection.ResourceIdentity{
		Context:   "", // Will be set by caller if multi-context
		Namespace: deployment.Namespace,
		Name:      deployment.Name,
		UID:       string(deployment.UID),
		Kind:      "Deployment",
	}

	// Build row data
	var row []string

	// NAME column
	row = append(row, deployment.Name)

	// NAMESPACE column (if requested)
	if showNamespace {
		row = append(row, deployment.Namespace)
	}

	// READY column
	replicas := int32(0)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}
	ready := fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, replicas)
	row = append(row, ready)

	// UP-TO-DATE column
	upToDate := fmt.Sprintf("%d", deployment.Status.UpdatedReplicas)
	row = append(row, upToDate)

	// AVAILABLE column
	available := fmt.Sprintf("%d", deployment.Status.AvailableReplicas)
	row = append(row, available)

	// AGE column
	age := getAge(deployment.CreationTimestamp.Time)
	row = append(row, age)

	// CONTAINERS column
	var containers []string
	for _, container := range deployment.Spec.Template.Spec.Containers {
		containers = append(containers, container.Name)
	}
	containersStr := strings.Join(containers, ",")
	row = append(row, containersStr)

	// IMAGES column
	var images []string
	for _, container := range deployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}
	imagesStr := strings.Join(images, ",")
	row = append(row, imagesStr)

	// SELECTOR column
	var selectors []string
	for k, v := range deployment.Spec.Selector.MatchLabels {
		selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
	}
	selectorStr := strings.Join(selectors, ",")
	row = append(row, selectorStr)

	return row, identity, nil
}

// GetSortValue returns the value to use for sorting on the given column
func (t *DeploymentTransformer) GetSortValue(resource interface{}, column string) interface{} {
	deployment, ok := resource.(appsv1.Deployment)
	if !ok {
		return ""
	}

	switch strings.ToUpper(column) {
	case "NAME":
		return deployment.Name
	case "NAMESPACE":
		return deployment.Namespace
	case "AGE":
		return deployment.CreationTimestamp.Time
	case "READY":
		replicas := int32(1)
		if deployment.Spec.Replicas != nil {
			replicas = *deployment.Spec.Replicas
		}
		if replicas == 0 {
			return 0.0
		}
		return float64(deployment.Status.ReadyReplicas) / float64(replicas)
	default:
		return ""
	}
}
