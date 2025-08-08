package transformers

import (
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceTransformer_GetHeaders(t *testing.T) {
	transformer := NewServiceTransformer()

	tests := []struct {
		name          string
		showNamespace bool
		multiContext  bool
		expected      []string
	}{
		{
			name:          "basic headers",
			showNamespace: false,
			multiContext:  false,
			expected:      []string{"NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE", "SELECTOR"},
		},
		{
			name:          "with namespace",
			showNamespace: true,
			multiContext:  false,
			expected:      []string{"NAMESPACE", "NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE", "SELECTOR"},
		},
		{
			name:          "with context",
			showNamespace: false,
			multiContext:  true,
			expected:      []string{"CONTEXT", "NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE", "SELECTOR"},
		},
		{
			name:          "with namespace and context",
			showNamespace: true,
			multiContext:  true,
			expected:      []string{"CONTEXT", "NAMESPACE", "NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE", "SELECTOR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := transformer.GetHeaders(tt.showNamespace, tt.multiContext)
			if len(headers) != len(tt.expected) {
				t.Errorf("expected %d headers, got %d", len(tt.expected), len(headers))
			}
			for i, header := range headers {
				if header != tt.expected[i] {
					t.Errorf("header[%d]: expected %s, got %s", i, tt.expected[i], header)
				}
			}
		})
	}
}

func TestFormatPorts(t *testing.T) {
	tests := []struct {
		name     string
		ports    []corev1.ServicePort
		expected string
	}{
		{
			name:     "no ports",
			ports:    []corev1.ServicePort{},
			expected: "<none>",
		},
		{
			name: "single TCP port",
			ports: []corev1.ServicePort{
				{Port: 80, Protocol: corev1.ProtocolTCP},
			},
			expected: "80/TCP",
		},
		{
			name: "single UDP port",
			ports: []corev1.ServicePort{
				{Port: 53, Protocol: corev1.ProtocolUDP},
			},
			expected: "53/UDP",
		},
		{
			name: "NodePort service",
			ports: []corev1.ServicePort{
				{Port: 10000, NodePort: 30963, Protocol: corev1.ProtocolTCP},
			},
			expected: "10000:30963/TCP",
		},
		{
			name: "multiple ports",
			ports: []corev1.ServicePort{
				{Port: 80, Protocol: corev1.ProtocolTCP},
				{Port: 443, Protocol: corev1.ProtocolTCP},
			},
			expected: "80/TCP,443/TCP",
		},
		{
			name: "LoadBalancer with multiple ports",
			ports: []corev1.ServicePort{
				{Port: 443, Protocol: corev1.ProtocolTCP},
				{Port: 80, Protocol: corev1.ProtocolTCP},
			},
			expected: "443/TCP,80/TCP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPorts(tt.ports)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector map[string]string
		expected string
	}{
		{
			name:     "no selector",
			selector: map[string]string{},
			expected: "<none>",
		},
		{
			name: "single selector",
			selector: map[string]string{
				"app": "nginx",
			},
			expected: "app=nginx",
		},
		{
			name: "two selectors",
			selector: map[string]string{
				"app":     "nginx",
				"version": "v1",
			},
			expected: "app=nginx,version=v1",
		},
		{
			name: "many selectors (should truncate)",
			selector: map[string]string{
				"app":                         "csi-hostpathplugin",
				"app.kubernetes.io/component": "socat",
				"app.kubernetes.io/instance":  "hostpath.csi.k8s.io",
				"environment":                 "production",
			},
			// Should show first two (alphabetically) and ellipsis
			expected: "app.kubernetes.io/component=socat,app.kubernetes.io/instance=hostpath.csi.k8s.io,...",
		},
		{
			name: "k8s-app selector",
			selector: map[string]string{
				"k8s-app": "dockerreg",
			},
			expected: "k8s-app=dockerreg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSelector(tt.selector)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetExternalIP(t *testing.T) {
	tests := []struct {
		name     string
		service  *corev1.Service
		expected string
	}{
		{
			name: "ClusterIP service",
			service: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
				},
			},
			expected: "<none>",
		},
		{
			name: "NodePort service",
			service: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
				},
			},
			expected: "<nodes>",
		},
		{
			name: "LoadBalancer with IPs",
			service: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: "2607:fa18:1000:21::40:166"},
							{IP: "44.40.48.166"},
						},
					},
				},
			},
			expected: "2607:fa18:1000:21::40:166,44.40.48.166",
		},
		{
			name: "LoadBalancer pending",
			service: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
			},
			expected: "<pending>",
		},
		{
			name: "Service with explicit external IPs",
			service: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type:        corev1.ServiceTypeClusterIP,
					ExternalIPs: []string{"192.168.1.100", "192.168.1.101"},
				},
			},
			expected: "192.168.1.100,192.168.1.101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getExternalIP(tt.service)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestServiceTransformer_TransformToRow(t *testing.T) {
	transformer := NewServiceTransformer()
	now := time.Now()

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-service",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: now},
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeLoadBalancer,
			ClusterIP: "10.96.24.72",
			Selector: map[string]string{
				"app":     "test",
				"version": "v1",
			},
			Ports: []corev1.ServicePort{
				{Port: 443, Protocol: corev1.ProtocolTCP},
			},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{IP: "44.40.48.166"},
				},
			},
		},
	}

	// Test without namespace
	row, identity, err := transformer.TransformToRow(service, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if identity.Name != "test-service" {
		t.Errorf("expected identity name to be test-service, got %s", identity.Name)
	}

	if len(row) != 7 { // NAME, TYPE, CLUSTER-IP, EXTERNAL-IP, PORT(S), AGE, SELECTOR
		t.Errorf("expected 7 columns, got %d", len(row))
	}

	// Verify specific columns
	if row[0] != "test-service" {
		t.Errorf("expected NAME to be test-service, got %s", row[0])
	}

	if row[1] != "LoadBalancer" {
		t.Errorf("expected TYPE to be LoadBalancer, got %s", row[1])
	}

	if row[2] != "10.96.24.72" {
		t.Errorf("expected CLUSTER-IP to be 10.96.24.72, got %s", row[2])
	}

	if row[3] != "44.40.48.166" {
		t.Errorf("expected EXTERNAL-IP to be 44.40.48.166, got %s", row[3])
	}

	if row[4] != "443/TCP" {
		t.Errorf("expected PORT(S) to be 443/TCP, got %s", row[4])
	}

	// Check selector column
	if !strings.Contains(row[6], "app=test") || !strings.Contains(row[6], "version=v1") {
		t.Errorf("expected SELECTOR to contain app=test and version=v1, got %s", row[6])
	}

	// Test with namespace
	row, _, err = transformer.TransformToRow(service, true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(row) != 8 { // NAMESPACE + 7 columns
		t.Errorf("expected 8 columns with namespace, got %d", len(row))
	}

	if row[0] != "default" {
		t.Errorf("expected NAMESPACE to be default, got %s", row[0])
	}
}
