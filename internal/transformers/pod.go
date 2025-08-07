package transformers

import (
	"fmt"
	"strings"
	"time"

	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/k8s"
	"github.com/HamStudy/kubewatch/internal/template"
	v1 "k8s.io/api/core/v1"
)

// PodTransformer handles Pod resource transformation
type PodTransformer struct {
	metricsProvider MetricsProvider
}

// MetricsProvider interface for getting pod metrics
type MetricsProvider interface {
	GetPodMetrics(namespace string) (map[string]*k8s.PodMetrics, error)
}

// NewPodTransformer creates a new pod transformer
func NewPodTransformer() *PodTransformer {
	return &PodTransformer{}
}

// SetMetricsProvider sets the metrics provider
func (t *PodTransformer) SetMetricsProvider(provider MetricsProvider) {
	t.metricsProvider = provider
}

// GetResourceType returns the resource type
func (t *PodTransformer) GetResourceType() string {
	return "Pod"
}

// GetHeaders returns the column headers for pods
func (t *PodTransformer) GetHeaders(showNamespace bool, multiContext bool) []string {
	var headers []string

	if multiContext {
		headers = append(headers, "CONTEXT")
	}

	headers = append(headers, "NAME")

	if showNamespace {
		headers = append(headers, "NAMESPACE")
	}

	headers = append(headers, "READY", "STATUS", "RESTARTS", "AGE", "CPU", "MEMORY", "IP", "NODE")

	return headers
}

// TransformToRow converts a pod to a table row
func (t *PodTransformer) TransformToRow(resource interface{}, showNamespace bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	pod, ok := resource.(v1.Pod)
	if !ok {
		return nil, nil, fmt.Errorf("expected Pod, got %T", resource)
	}

	// Create resource identity
	identity := &selection.ResourceIdentity{
		Context:   "", // Will be set by caller if multi-context
		Namespace: pod.Namespace,
		Name:      pod.Name,
		UID:       string(pod.UID),
		Kind:      "Pod",
	}

	// Build row data
	var row []string

	// NAME column
	row = append(row, pod.Name)

	// NAMESPACE column (if requested)
	if showNamespace {
		if templateEngine != nil {
			if formatted, err := templateEngine.Execute("{{ . | namespace }}", map[string]string{"Namespace": pod.Namespace}); err == nil {
				row = append(row, formatted)
			} else {
				row = append(row, pod.Namespace)
			}
		} else {
			row = append(row, pod.Namespace)
		}
	}

	// READY column
	readyContainers := 0
	totalContainers := len(pod.Status.ContainerStatuses)
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			readyContainers++
		}
	}

	readyText := fmt.Sprintf("%d/%d", readyContainers, totalContainers)
	if templateEngine != nil {
		data := map[string]interface{}{
			"Status": map[string]interface{}{
				"ContainerStatuses": pod.Status.ContainerStatuses,
			},
		}
		if formatted, err := templateEngine.Execute("{{ . | ready }}", data); err == nil {
			row = append(row, formatted)
		} else {
			row = append(row, readyText)
		}
	} else {
		row = append(row, readyText)
	}

	// STATUS column
	status := string(pod.Status.Phase)

	// Get more detailed status if available
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodReady && condition.Status != v1.ConditionTrue {
			if condition.Reason != "" {
				status = condition.Reason
			}
		}
	}

	// Check container statuses for more specific states
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			status = cs.State.Waiting.Reason
			break
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			status = cs.State.Terminated.Reason
			break
		}
	}

	if templateEngine != nil {
		data := map[string]interface{}{
			"Status": map[string]interface{}{
				"Phase":      pod.Status.Phase,
				"Conditions": pod.Status.Conditions,
			},
		}
		if formatted, err := templateEngine.Execute("{{ . | pod-status }}", data); err == nil {
			row = append(row, formatted)
		} else {
			row = append(row, status)
		}
	} else {
		row = append(row, status)
	}

	// RESTARTS column
	restartCount := int32(0)
	var lastRestartTime *time.Time

	for _, cs := range pod.Status.ContainerStatuses {
		restartCount += cs.RestartCount
		if cs.LastTerminationState.Terminated != nil {
			t := cs.LastTerminationState.Terminated.FinishedAt.Time
			if lastRestartTime == nil || t.After(*lastRestartTime) {
				lastRestartTime = &t
			}
		}
	}

	restartText := fmt.Sprintf("%d", restartCount)
	if restartCount > 0 && lastRestartTime != nil {
		restartAge := getAge(*lastRestartTime)
		restartText = fmt.Sprintf("%d (%s ago)", restartCount, restartAge)
	}

	if templateEngine != nil {
		data := map[string]interface{}{
			"Status": map[string]interface{}{
				"ContainerStatuses": pod.Status.ContainerStatuses,
			},
		}
		if formatted, err := templateEngine.Execute("{{ . | restarts }}", data); err == nil {
			row = append(row, formatted)
		} else {
			row = append(row, restartText)
		}
	} else {
		row = append(row, restartText)
	}

	// AGE column
	age := getAge(pod.CreationTimestamp.Time)
	if templateEngine != nil {
		data := map[string]interface{}{
			"Metadata": map[string]interface{}{
				"CreationTimestamp": pod.CreationTimestamp.Time,
			},
		}
		if formatted, err := templateEngine.Execute("{{ . | age }}", data); err == nil {
			row = append(row, formatted)
		} else {
			row = append(row, age)
		}
	} else {
		row = append(row, age)
	}

	// CPU column
	cpu := "-"
	if t.metricsProvider != nil {
		if metrics, err := t.metricsProvider.GetPodMetrics(pod.Namespace); err == nil {
			if podMetrics, ok := metrics[pod.Name]; ok {
				cpu = podMetrics.CPU
			}
		}
	}

	if templateEngine != nil && cpu != "-" {
		data := map[string]interface{}{
			"Metrics": map[string]interface{}{
				"CPU": cpu,
			},
			"Spec": map[string]interface{}{
				"Containers": pod.Spec.Containers,
			},
		}
		if formatted, err := templateEngine.Execute("{{ . | cpu }}", data); err == nil {
			row = append(row, formatted)
		} else {
			row = append(row, cpu)
		}
	} else {
		row = append(row, cpu)
	}

	// MEMORY column
	memory := "-"
	if t.metricsProvider != nil {
		if metrics, err := t.metricsProvider.GetPodMetrics(pod.Namespace); err == nil {
			if podMetrics, ok := metrics[pod.Name]; ok {
				memory = podMetrics.Memory
			}
		}
	}

	if templateEngine != nil && memory != "-" {
		data := map[string]interface{}{
			"Metrics": map[string]interface{}{
				"Memory": memory,
			},
			"Spec": map[string]interface{}{
				"Containers": pod.Spec.Containers,
			},
		}
		if formatted, err := templateEngine.Execute("{{ . | memory }}", data); err == nil {
			row = append(row, formatted)
		} else {
			row = append(row, memory)
		}
	} else {
		row = append(row, memory)
	}

	// IP column
	ip := pod.Status.PodIP
	if ip == "" {
		ip = "-"
	}
	row = append(row, ip)

	// NODE column
	node := pod.Spec.NodeName
	if node == "" {
		node = "-"
	}
	row = append(row, node)

	return row, identity, nil
}

// GetSortValue returns the value to use for sorting on the given column
func (t *PodTransformer) GetSortValue(resource interface{}, column string) interface{} {
	pod, ok := resource.(v1.Pod)
	if !ok {
		return ""
	}

	switch strings.ToUpper(column) {
	case "NAME":
		return pod.Name
	case "NAMESPACE":
		return pod.Namespace
	case "STATUS":
		return string(pod.Status.Phase)
	case "AGE":
		return pod.CreationTimestamp.Time
	case "NODE":
		return pod.Spec.NodeName
	case "IP":
		return pod.Status.PodIP
	case "READY":
		readyContainers := 0
		totalContainers := len(pod.Status.ContainerStatuses)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyContainers++
			}
		}
		if totalContainers == 0 {
			return 0.0
		}
		return float64(readyContainers) / float64(totalContainers)
	case "RESTARTS":
		restartCount := int32(0)
		for _, cs := range pod.Status.ContainerStatuses {
			restartCount += cs.RestartCount
		}
		return restartCount
	default:
		return ""
	}
}

// GetUniqKey generates a unique key for resource grouping
func (t *PodTransformer) GetUniqKey(resource interface{}, templateEngine *template.Engine) (string, error) {
	pod, ok := resource.(v1.Pod)
	if !ok {
		return "", fmt.Errorf("expected Pod, got %T", resource)
	}

	// For pods, the unique key is just the name (pods don't typically get grouped)
	data := map[string]interface{}{
		"Metadata": map[string]interface{}{
			"Name": pod.Name,
		},
	}

	return templateEngine.Execute("{{ .Metadata.Name }}", data)
}

// CanGroup returns true if this resource type supports grouping
func (t *PodTransformer) CanGroup() bool {
	return false // Pods typically don't get grouped
}

// AggregateResources combines multiple pods with the same unique key
func (t *PodTransformer) AggregateResources(resources []interface{}, showNamespace bool, multiContext bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	// Since pods don't typically get grouped, just return the first one
	if len(resources) == 0 {
		return nil, nil, fmt.Errorf("no resources to aggregate")
	}

	// Use the first resource
	return t.TransformToRow(resources[0], showNamespace, templateEngine)
}

// getAge returns a human-readable age string
func getAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else if duration < 30*24*time.Hour {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	} else if duration < 365*24*time.Hour {
		return fmt.Sprintf("%dmo", int(duration.Hours()/24/30))
	}
	return fmt.Sprintf("%dy", int(duration.Hours()/24/365))
}
