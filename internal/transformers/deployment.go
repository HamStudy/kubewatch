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

// GetUniqKey generates a unique key for resource grouping
func (t *DeploymentTransformer) GetUniqKey(resource interface{}, templateEngine *template.Engine) (string, error) {
	deployment, ok := resource.(appsv1.Deployment)
	if !ok {
		return "", fmt.Errorf("expected Deployment, got %T", resource)
	}

	// Extract image list for the unique key
	var images []string
	for _, container := range deployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	// Create template data
	data := map[string]interface{}{
		"Metadata": map[string]interface{}{
			"Name": deployment.Name,
		},
		"ImageList": images,
	}

	// Use the deployment-specific unique key template
	return templateEngine.Execute("{{ .Metadata.Name }}_{{ join .ImageList \";\" }}", data)
}

// CanGroup returns true if this resource type supports grouping
func (t *DeploymentTransformer) CanGroup() bool {
	return true
}

// AggregateResources combines multiple deployments with the same unique key
func (t *DeploymentTransformer) AggregateResources(resources []interface{}, showNamespace bool, multiContext bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	if len(resources) == 0 {
		return nil, nil, fmt.Errorf("no resources to aggregate")
	}

	// Convert to deployments
	var deployments []appsv1.Deployment
	var contexts []string
	for _, resource := range resources {
		if dep, ok := resource.(appsv1.Deployment); ok {
			deployments = append(deployments, dep)
		} else if depWithContext, ok := resource.(map[string]interface{}); ok {
			// Handle multi-context resource format
			if dep, ok := depWithContext["resource"].(appsv1.Deployment); ok {
				deployments = append(deployments, dep)
				if ctx, ok := depWithContext["context"].(string); ok {
					contexts = append(contexts, ctx)
				}
			}
		}
	}

	if len(deployments) == 0 {
		return nil, nil, fmt.Errorf("no valid deployments found")
	}

	// Use the first deployment as the base
	baseDeployment := deployments[0]

	// Aggregate ready replicas
	totalReady := int32(0)
	totalReplicas := int32(0)
	totalUpdated := int32(0)
	totalAvailable := int32(0)

	for _, dep := range deployments {
		totalReady += dep.Status.ReadyReplicas
		if dep.Spec.Replicas != nil {
			totalReplicas += *dep.Spec.Replicas
		}
		totalUpdated += dep.Status.UpdatedReplicas
		totalAvailable += dep.Status.AvailableReplicas
	}

	// Create resource identity
	identity := &selection.ResourceIdentity{
		Context:   "", // Will be set to aggregated contexts if multi-context
		Namespace: baseDeployment.Namespace,
		Name:      baseDeployment.Name,
		UID:       string(baseDeployment.UID), // Use first deployment's UID
		Kind:      "Deployment",
	}

	// Build row data
	var row []string

	// CONTEXT column (if multi-context)
	if multiContext {
		if len(contexts) > 0 {
			contextStr := strings.Join(contexts, ",")
			row = append(row, contextStr)
			identity.Context = contextStr
		} else {
			row = append(row, "")
		}
	}

	// NAME column
	row = append(row, baseDeployment.Name)

	// NAMESPACE column (if requested)
	if showNamespace {
		row = append(row, baseDeployment.Namespace)
	}

	// READY column (aggregated)
	ready := fmt.Sprintf("%d/%d", totalReady, totalReplicas)
	row = append(row, ready)

	// UP-TO-DATE column (aggregated)
	upToDate := fmt.Sprintf("%d", totalUpdated)
	row = append(row, upToDate)

	// AVAILABLE column (aggregated)
	available := fmt.Sprintf("%d", totalAvailable)
	row = append(row, available)

	// AGE column (use oldest deployment)
	oldestTime := baseDeployment.CreationTimestamp.Time
	for _, dep := range deployments[1:] {
		if dep.CreationTimestamp.Time.Before(oldestTime) {
			oldestTime = dep.CreationTimestamp.Time
		}
	}
	age := getAge(oldestTime)
	row = append(row, age)

	// CONTAINERS column (from base deployment)
	var containers []string
	for _, container := range baseDeployment.Spec.Template.Spec.Containers {
		containers = append(containers, container.Name)
	}
	containersStr := strings.Join(containers, ",")
	row = append(row, containersStr)

	// IMAGES column (from base deployment)
	var images []string
	for _, container := range baseDeployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}
	imagesStr := strings.Join(images, ",")
	row = append(row, imagesStr)

	// SELECTOR column (from base deployment)
	var selectors []string
	for k, v := range baseDeployment.Spec.Selector.MatchLabels {
		selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
	}
	selectorStr := strings.Join(selectors, ",")
	row = append(row, selectorStr)

	return row, identity, nil
}
