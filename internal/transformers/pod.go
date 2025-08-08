package transformers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/k8s"
	"github.com/HamStudy/kubewatch/internal/template"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	var cpuMillicores int64
	if t.metricsProvider != nil {
		if metrics, err := t.metricsProvider.GetPodMetrics(pod.Namespace); err == nil {
			if podMetrics, ok := metrics[pod.Name]; ok && podMetrics.CPU != "" && podMetrics.CPU != "-" {
				// Parse the CPU value (it comes as a string like "100m" or "1.5")
				cpuMillicores = parseCPUToMillicores(podMetrics.CPU)

				// Get total CPU requests from containers
				var totalRequestMillicores int64
				for _, container := range pod.Spec.Containers {
					if container.Resources.Requests != nil {
						if cpuReq, ok := container.Resources.Requests[v1.ResourceCPU]; ok {
							totalRequestMillicores += cpuReq.MilliValue()
						}
					}
				}

				// Format CPU with request-based coloring
				cpu = formatCPU(cpuMillicores, totalRequestMillicores, templateEngine)
			}
		}
	}
	row = append(row, cpu)

	// MEMORY column
	memory := "-"
	var memoryBytes int64
	if t.metricsProvider != nil {
		if metrics, err := t.metricsProvider.GetPodMetrics(pod.Namespace); err == nil {
			if podMetrics, ok := metrics[pod.Name]; ok && podMetrics.Memory != "" && podMetrics.Memory != "-" {
				// Parse the memory value (it comes as a string like "128Mi" or "1Gi")
				memoryBytes = parseMemoryToBytes(podMetrics.Memory)

				// Get total memory requests from containers
				var totalRequestBytes int64
				for _, container := range pod.Spec.Containers {
					if container.Resources.Requests != nil {
						if memReq, ok := container.Resources.Requests[v1.ResourceMemory]; ok {
							totalRequestBytes += memReq.Value()
						}
					}
				}

				// Format memory with request-based coloring
				memory = formatMemory(memoryBytes, totalRequestBytes, templateEngine)
			}
		}
	}
	row = append(row, memory)

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

// parseCPUToMillicores parses a CPU string to millicores
func parseCPUToMillicores(cpu string) int64 {
	if cpu == "" || cpu == "-" {
		return 0
	}

	// Handle millicores format (e.g., "100m")
	if strings.HasSuffix(cpu, "m") {
		val, err := strconv.ParseInt(strings.TrimSuffix(cpu, "m"), 10, 64)
		if err == nil {
			return val
		}
	}

	// Handle cores format (e.g., "1.5")
	if val, err := strconv.ParseFloat(cpu, 64); err == nil {
		return int64(val * 1000)
	}

	// Try parsing as Kubernetes quantity
	if quantity, err := resource.ParseQuantity(cpu); err == nil {
		return quantity.MilliValue()
	}

	return 0
}

// parseMemoryToBytes parses a memory string to bytes
func parseMemoryToBytes(memory string) int64 {
	if memory == "" || memory == "-" {
		return 0
	}

	// Try parsing as Kubernetes quantity
	if quantity, err := resource.ParseQuantity(memory); err == nil {
		return quantity.Value()
	}

	return 0
}

// formatCPU formats CPU value with request-based coloring
func formatCPU(millicores int64, requestMillicores int64, templateEngine *template.Engine) string {
	if millicores == 0 {
		return "-"
	}

	// Format the value
	var formatted string
	if millicores < 1000 {
		// Show as millicores for values < 1 core
		formatted = fmt.Sprintf("%dm", millicores)
	} else {
		// Show as cores for values >= 1 core
		cores := float64(millicores) / 1000.0
		// Use one decimal place for cleaner display
		if cores == float64(int(cores)) {
			formatted = fmt.Sprintf("%.0f", cores)
		} else {
			formatted = fmt.Sprintf("%.1f", cores)
		}
	}

	// Apply coloring based on percentage of requests if template engine is available
	if templateEngine != nil && requestMillicores > 0 {
		percentage := (millicores * 100) / requestMillicores

		var styled string
		var err error
		if percentage > 100 {
			// Over 100%: red background, white text, underlined
			styled, err = templateEngine.Execute(`{{ style "red" "white" "underline" . }}`, formatted)
		} else if percentage >= 90 {
			// 90-100%: red text
			styled, err = templateEngine.Execute(`{{ style "" "red" "" . }}`, formatted)
		} else if percentage >= 70 {
			// 70-90%: yellow text
			styled, err = templateEngine.Execute(`{{ style "" "yellow" "" . }}`, formatted)
		} else {
			// <70%: green text
			styled, err = templateEngine.Execute(`{{ style "" "green" "" . }}`, formatted)
		}

		if err == nil && styled != "" {
			formatted = styled
		}
	}
	return formatted
}

// formatMemory formats memory value with request-based coloring
func formatMemory(bytes int64, requestBytes int64, templateEngine *template.Engine) string {
	if bytes == 0 {
		return "-"
	}

	// Format using Kubernetes-style units
	const (
		Ki = 1024
		Mi = 1024 * Ki
		Gi = 1024 * Mi
	)

	var formatted string
	if bytes >= Gi {
		gb := float64(bytes) / float64(Gi)
		// Show fractional GB if significant
		if gb < 10 && (bytes%Gi) >= 100*Mi {
			formatted = fmt.Sprintf("%.1fGi", gb)
		} else {
			formatted = fmt.Sprintf("%.0fGi", gb)
		}
	} else if bytes >= Mi {
		mb := bytes / Mi
		formatted = fmt.Sprintf("%dMi", mb)
	} else if bytes >= Ki {
		kb := bytes / Ki
		formatted = fmt.Sprintf("%dKi", kb)
	} else {
		formatted = fmt.Sprintf("%d", bytes)
	}

	// Apply coloring based on percentage of requests if template engine is available
	if templateEngine != nil && requestBytes > 0 {
		percentage := (bytes * 100) / requestBytes

		var styled string
		var err error
		if percentage > 100 {
			// Over 100%: red background, white text, underlined
			styled, err = templateEngine.Execute(`{{ style "red" "white" "underline" . }}`, formatted)
		} else if percentage >= 90 {
			// 90-100%: red text
			styled, err = templateEngine.Execute(`{{ style "" "red" "" . }}`, formatted)
		} else if percentage >= 70 {
			// 70-90%: yellow text
			styled, err = templateEngine.Execute(`{{ style "" "yellow" "" . }}`, formatted)
		} else {
			// <70%: green text
			styled, err = templateEngine.Execute(`{{ style "" "green" "" . }}`, formatted)
		}

		if err == nil && styled != "" {
			formatted = styled
		}
	}
	return formatted
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
	case "CPU":
		// Sort by actual CPU usage if metrics are available
		if t.metricsProvider != nil {
			if metrics, err := t.metricsProvider.GetPodMetrics(pod.Namespace); err == nil {
				if podMetrics, ok := metrics[pod.Name]; ok && podMetrics.CPU != "" && podMetrics.CPU != "-" {
					return parseCPUToMillicores(podMetrics.CPU)
				}
			}
		}
		return int64(0)
	case "MEMORY":
		// Sort by actual memory usage if metrics are available
		if t.metricsProvider != nil {
			if metrics, err := t.metricsProvider.GetPodMetrics(pod.Namespace); err == nil {
				if podMetrics, ok := metrics[pod.Name]; ok && podMetrics.Memory != "" && podMetrics.Memory != "-" {
					return parseMemoryToBytes(podMetrics.Memory)
				}
			}
		}
		return int64(0)
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
