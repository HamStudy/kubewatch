package transformers

import (
	"fmt"
	"strings"

	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/template"
	networkingv1 "k8s.io/api/networking/v1"
)

// IngressTransformer handles Ingress resource transformation
type IngressTransformer struct{}

// NewIngressTransformer creates a new Ingress transformer
func NewIngressTransformer() *IngressTransformer {
	return &IngressTransformer{}
}

// GetResourceType returns the resource type
func (t *IngressTransformer) GetResourceType() string {
	return "Ingress"
}

// GetHeaders returns column headers for Ingresses
func (t *IngressTransformer) GetHeaders(showNamespace bool, multiContext bool) []string {
	headers := []string{"NAME", "CLASS", "HOSTS", "ADDRESS", "PORTS", "AGE"}

	if showNamespace {
		headers = append([]string{"NAMESPACE"}, headers...)
	}

	if multiContext {
		headers = append([]string{"CONTEXT"}, headers...)
	}

	return headers
}

// TransformToRow converts an Ingress to a table row
func (t *IngressTransformer) TransformToRow(resource interface{}, showNamespace bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	ingress, ok := resource.(*networkingv1.Ingress)
	if !ok {
		return nil, nil, fmt.Errorf("expected *networkingv1.Ingress, got %T", resource)
	}

	// Create resource identity
	identity := &selection.ResourceIdentity{
		Name:      ingress.Name,
		Namespace: ingress.Namespace,
		Kind:      "Ingress",
		Context:   "", // Will be set by caller if needed
	}

	// Basic formatting (template support can be added later)
	age := getAge(ingress.CreationTimestamp.Time)
	class := "<none>"
	if ingress.Spec.IngressClassName != nil {
		class = *ingress.Spec.IngressClassName
	}

	hosts := getIngressHosts(ingress)
	address := getIngressAddress(ingress)
	ports := getIngressPorts(ingress)

	row := []string{
		ingress.Name,
		class,
		hosts,
		address,
		ports,
		age,
	}

	if showNamespace {
		row = append([]string{ingress.Namespace}, row...)
	}

	return row, identity, nil
}

// GetSortValue returns the value for sorting on a given column
func (t *IngressTransformer) GetSortValue(resource interface{}, column string) interface{} {
	ingress, ok := resource.(*networkingv1.Ingress)
	if !ok {
		return ""
	}

	switch strings.ToUpper(column) {
	case "NAME":
		return ingress.Name
	case "NAMESPACE":
		return ingress.Namespace
	case "CLASS":
		if ingress.Spec.IngressClassName != nil {
			return *ingress.Spec.IngressClassName
		}
		return ""
	case "AGE":
		return ingress.CreationTimestamp.Time
	default:
		return ingress.Name
	}
}

func getIngressHosts(ingress *networkingv1.Ingress) string {
	var hosts []string
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	if len(hosts) == 0 {
		return "*"
	}

	// Match kubectl format: show first few hosts then "+ X more..."
	if len(hosts) > 4 {
		// Show first 3 hosts and indicate how many more
		moreCount := len(hosts) - 3
		return fmt.Sprintf("%s + %d more...", strings.Join(hosts[:3], ","), moreCount)
	}

	return strings.Join(hosts, ",")
}

func getIngressAddress(ingress *networkingv1.Ingress) string {
	var addresses []string
	for _, lb := range ingress.Status.LoadBalancer.Ingress {
		if lb.IP != "" {
			addresses = append(addresses, lb.IP)
		} else if lb.Hostname != "" {
			addresses = append(addresses, lb.Hostname)
		}
	}
	if len(addresses) == 0 {
		return "<none>"
	}
	return strings.Join(addresses, ",")
}

func getIngressPorts(ingress *networkingv1.Ingress) string {
	hasTLS := len(ingress.Spec.TLS) > 0
	if hasTLS {
		return "80, 443"
	}
	return "80"
}

// GetUniqKey generates a unique key for resource grouping
func (t *IngressTransformer) GetUniqKey(resource interface{}, templateEngine *template.Engine) (string, error) {
	ingress, ok := resource.(*networkingv1.Ingress)
	if !ok {
		return "", fmt.Errorf("expected *networkingv1.Ingress, got %T", resource)
	}

	data := map[string]interface{}{
		"Metadata": map[string]interface{}{
			"Name": ingress.Name,
		},
	}

	return templateEngine.Execute("{{ .Metadata.Name }}", data)
}

// CanGroup returns true if this resource type supports grouping
func (t *IngressTransformer) CanGroup() bool {
	return false
}

// AggregateResources combines multiple resources with the same unique key
func (t *IngressTransformer) AggregateResources(resources []interface{}, showNamespace bool, multiContext bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	if len(resources) == 0 {
		return nil, nil, fmt.Errorf("no resources to aggregate")
	}
	return t.TransformToRow(resources[0], showNamespace, templateEngine)
}
