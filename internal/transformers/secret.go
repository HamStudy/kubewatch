package transformers

import (
	"fmt"
	"strings"

	"github.com/HamStudy/kubewatch/internal/components/selection"
	"github.com/HamStudy/kubewatch/internal/template"
	corev1 "k8s.io/api/core/v1"
)

// SecretTransformer handles Secret resource transformation
type SecretTransformer struct{}

// NewSecretTransformer creates a new Secret transformer
func NewSecretTransformer() *SecretTransformer {
	return &SecretTransformer{}
}

// GetResourceType returns the resource type
func (t *SecretTransformer) GetResourceType() string {
	return "Secret"
}

// GetHeaders returns column headers for Secrets
func (t *SecretTransformer) GetHeaders(showNamespace bool, multiContext bool) []string {
	headers := []string{"NAME", "TYPE", "DATA", "AGE"}

	if showNamespace {
		headers = append([]string{"NAMESPACE"}, headers...)
	}

	if multiContext {
		headers = append([]string{"CONTEXT"}, headers...)
	}

	return headers
}

// TransformToRow converts a Secret to a table row
func (t *SecretTransformer) TransformToRow(resource interface{}, showNamespace bool, templateEngine *template.Engine) ([]string, *selection.ResourceIdentity, error) {
	secret, ok := resource.(*corev1.Secret)
	if !ok {
		return nil, nil, fmt.Errorf("expected *corev1.Secret, got %T", resource)
	}

	// Create resource identity
	identity := &selection.ResourceIdentity{
		Name:      secret.Name,
		Namespace: secret.Namespace,
		Kind:      "Secret",
		Context:   "", // Will be set by caller if needed
	}

	// Basic formatting
	age := getAge(secret.CreationTimestamp.Time)
	dataCount := fmt.Sprintf("%d", len(secret.Data))
	secretType := string(secret.Type)

	row := []string{
		secret.Name,
		secretType,
		dataCount,
		age,
	}

	if showNamespace {
		row = append([]string{secret.Namespace}, row...)
	}

	return row, identity, nil
}

// GetSortValue returns the value for sorting on a given column
func (t *SecretTransformer) GetSortValue(resource interface{}, column string) interface{} {
	secret, ok := resource.(*corev1.Secret)
	if !ok {
		return ""
	}

	switch strings.ToUpper(column) {
	case "NAME":
		return secret.Name
	case "NAMESPACE":
		return secret.Namespace
	case "TYPE":
		return string(secret.Type)
	case "DATA":
		return len(secret.Data)
	case "AGE":
		return secret.CreationTimestamp.Time
	default:
		return secret.Name
	}
}
