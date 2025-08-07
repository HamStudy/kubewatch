package transformers

import (
	"fmt"
	"strings"

	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/template"
	corev1 "k8s.io/api/core/v1"
)

// ServiceTransformer handles Service resource transformation
type ServiceTransformer struct{}

// NewServiceTransformer creates a new Service transformer
func NewServiceTransformer() *ServiceTransformer {
	return &ServiceTransformer{}
}

// GetResourceType returns the resource type
func (t *ServiceTransformer) GetResourceType() string {
	return "Service"
}

// GetHeaders returns column headers for Services
func (t *ServiceTransformer) GetHeaders(showNamespace bool, multiContext bool) []string {
	headers := []string{"NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE"}

	if showNamespace {
		headers = append([]string{"NAMESPACE"}, headers...)
	}

	if multiContext {
		headers = append([]string{"CONTEXT"}, headers...)
	}

	return headers
}

// TransformToRow converts a Service to a table row
func (t *ServiceTransformer) TransformToRow(resource interface{}, showNamespace bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	service, ok := resource.(*corev1.Service)
	if !ok {
		return nil, nil, fmt.Errorf("expected *corev1.Service, got %T", resource)
	}

	// Create resource identity
	identity := &selection.ResourceIdentity{
		Name:      service.Name,
		Namespace: service.Namespace,
		Kind:      "Service",
		Context:   "", // Will be set by caller if needed
	}

	// Use template engine to format the row
	data := map[string]interface{}{
		"Name":       service.Name,
		"Namespace":  service.Namespace,
		"Type":       string(service.Spec.Type),
		"ClusterIP":  service.Spec.ClusterIP,
		"ExternalIP": getExternalIP(service),
		"Ports":      formatPorts(service.Spec.Ports),
		"Age":        service.CreationTimestamp.Time,
		"Service":    service,
	}

	// Get template for service row
	templateName := "service_row"
	if showNamespace {
		templateName = "service_row_with_namespace"
	}

	result, err := templateEngine.Execute(templateName, data)
	if err != nil {
		// Fallback to basic formatting if template fails
		return t.formatBasicRow(service, showNamespace), identity, nil
	}

	// Split template result into columns
	columns := strings.Split(strings.TrimSpace(result), "\t")
	return columns, identity, nil
}

// GetSortValue returns the value for sorting on a given column
func (t *ServiceTransformer) GetSortValue(resource interface{}, column string) interface{} {
	service, ok := resource.(*corev1.Service)
	if !ok {
		return ""
	}

	switch strings.ToUpper(column) {
	case "NAME":
		return service.Name
	case "NAMESPACE":
		return service.Namespace
	case "TYPE":
		return string(service.Spec.Type)
	case "CLUSTER-IP":
		return service.Spec.ClusterIP
	case "EXTERNAL-IP":
		return getExternalIP(service)
	case "PORT(S)", "PORTS":
		return formatPorts(service.Spec.Ports)
	case "AGE":
		return service.CreationTimestamp.Time
	default:
		return service.Name
	}
}

// formatBasicRow provides fallback formatting when templates fail
func (t *ServiceTransformer) formatBasicRow(service *corev1.Service, showNamespace bool) []string {
	age := getAge(service.CreationTimestamp.Time)

	row := []string{
		service.Name,
		string(service.Spec.Type),
		service.Spec.ClusterIP,
		getExternalIP(service),
		formatPorts(service.Spec.Ports),
		age,
	}

	if showNamespace {
		row = append([]string{service.Namespace}, row...)
	}

	return row
}

// getExternalIP returns the external IP(s) for a service
func getExternalIP(service *corev1.Service) string {
	if len(service.Spec.ExternalIPs) > 0 {
		return strings.Join(service.Spec.ExternalIPs, ",")
	}

	if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
		var ips []string
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				ips = append(ips, ingress.IP)
			} else if ingress.Hostname != "" {
				ips = append(ips, ingress.Hostname)
			}
		}
		if len(ips) > 0 {
			return strings.Join(ips, ",")
		}
		return "<pending>"
	}

	if service.Spec.Type == corev1.ServiceTypeNodePort {
		return "<nodes>"
	}

	return "<none>"
}

// GetUniqKey generates a unique key for resource grouping
func (t *ServiceTransformer) GetUniqKey(resource interface{}, templateEngine *template.Engine) (string, error) {
	service, ok := resource.(*corev1.Service)
	if !ok {
		return "", fmt.Errorf("expected *corev1.Service, got %T", resource)
	}

	data := map[string]interface{}{
		"Metadata": map[string]interface{}{
			"Name": service.Name,
		},
	}

	return templateEngine.Execute("{{ .Metadata.Name }}", data)
}

// CanGroup returns true if this resource type supports grouping
func (t *ServiceTransformer) CanGroup() bool {
	return false
}

// AggregateResources combines multiple resources with the same unique key
func (t *ServiceTransformer) AggregateResources(resources []interface{}, showNamespace bool, multiContext bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	if len(resources) == 0 {
		return nil, nil, fmt.Errorf("no resources to aggregate")
	}
	return t.TransformToRow(resources[0], showNamespace, templateEngine)
}

// formatPorts formats the service ports
func formatPorts(ports []corev1.ServicePort) string {
	if len(ports) == 0 {
		return "<none>"
	}

	var portStrings []string
	for _, port := range ports {
		portStr := fmt.Sprintf("%d", port.Port)
		if port.NodePort != 0 {
			portStr += fmt.Sprintf(":%d", port.NodePort)
		}
		if port.Protocol != corev1.ProtocolTCP {
			portStr += "/" + string(port.Protocol)
		}
		portStrings = append(portStrings, portStr)
	}

	return strings.Join(portStrings, ",")
}
