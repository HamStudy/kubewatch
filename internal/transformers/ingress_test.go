package transformers

import (
	"testing"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIngressTransformer_GetHeaders(t *testing.T) {
	transformer := NewIngressTransformer()

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
			expected:      []string{"NAME", "CLASS", "HOSTS", "ADDRESS", "PORTS", "AGE"},
		},
		{
			name:          "with namespace",
			showNamespace: true,
			multiContext:  false,
			expected:      []string{"NAMESPACE", "NAME", "CLASS", "HOSTS", "ADDRESS", "PORTS", "AGE"},
		},
		{
			name:          "with context",
			showNamespace: false,
			multiContext:  true,
			expected:      []string{"CONTEXT", "NAME", "CLASS", "HOSTS", "ADDRESS", "PORTS", "AGE"},
		},
		{
			name:          "with namespace and context",
			showNamespace: true,
			multiContext:  true,
			expected:      []string{"CONTEXT", "NAMESPACE", "NAME", "CLASS", "HOSTS", "ADDRESS", "PORTS", "AGE"},
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

func TestGetIngressHosts(t *testing.T) {
	tests := []struct {
		name     string
		ingress  *networkingv1.Ingress
		expected string
	}{
		{
			name: "no hosts",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{},
				},
			},
			expected: "*",
		},
		{
			name: "single host",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "example.com"},
					},
				},
			},
			expected: "example.com",
		},
		{
			name: "multiple hosts",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "test-richard.ham.dev"},
						{Host: "hamstudy-richard.ham.dev"},
						{Host: "examtools-richard.ham.dev"},
					},
				},
			},
			expected: "test-richard.ham.dev,hamstudy-richard.ham.dev,examtools-richard.ham.dev",
		},
		{
			name: "many hosts (should show + X more...)",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "test-richard.ham.dev"},
						{Host: "hamstudy-richard.ham.dev"},
						{Host: "examtools-richard.ham.dev"},
						{Host: "another-richard.ham.dev"},
						{Host: "fifth-richard.ham.dev"},
					},
				},
			},
			expected: "test-richard.ham.dev,hamstudy-richard.ham.dev,examtools-richard.ham.dev + 2 more...",
		},
		{
			name: "empty host (wildcard)",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: ""},
					},
				},
			},
			expected: "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIngressHosts(tt.ingress)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetIngressAddress(t *testing.T) {
	tests := []struct {
		name     string
		ingress  *networkingv1.Ingress
		expected string
	}{
		{
			name: "no address",
			ingress: &networkingv1.Ingress{
				Status: networkingv1.IngressStatus{},
			},
			expected: "<none>",
		},
		{
			name: "single IP",
			ingress: &networkingv1.Ingress{
				Status: networkingv1.IngressStatus{
					LoadBalancer: networkingv1.IngressLoadBalancerStatus{
						Ingress: []networkingv1.IngressLoadBalancerIngress{
							{IP: "192.168.1.100"},
						},
					},
				},
			},
			expected: "192.168.1.100",
		},
		{
			name: "multiple IPs",
			ingress: &networkingv1.Ingress{
				Status: networkingv1.IngressStatus{
					LoadBalancer: networkingv1.IngressLoadBalancerStatus{
						Ingress: []networkingv1.IngressLoadBalancerIngress{
							{IP: "192.168.1.100"},
							{IP: "192.168.1.101"},
						},
					},
				},
			},
			expected: "192.168.1.100,192.168.1.101",
		},
		{
			name: "hostname",
			ingress: &networkingv1.Ingress{
				Status: networkingv1.IngressStatus{
					LoadBalancer: networkingv1.IngressLoadBalancerStatus{
						Ingress: []networkingv1.IngressLoadBalancerIngress{
							{Hostname: "elb.amazonaws.com"},
						},
					},
				},
			},
			expected: "elb.amazonaws.com",
		},
		{
			name: "mixed IP and hostname",
			ingress: &networkingv1.Ingress{
				Status: networkingv1.IngressStatus{
					LoadBalancer: networkingv1.IngressLoadBalancerStatus{
						Ingress: []networkingv1.IngressLoadBalancerIngress{
							{IP: "192.168.1.100"},
							{Hostname: "elb.amazonaws.com"},
						},
					},
				},
			},
			expected: "192.168.1.100,elb.amazonaws.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIngressAddress(tt.ingress)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetIngressPorts(t *testing.T) {
	tests := []struct {
		name     string
		ingress  *networkingv1.Ingress
		expected string
	}{
		{
			name: "no TLS",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{},
			},
			expected: "80",
		},
		{
			name: "with TLS",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{
							Hosts:      []string{"example.com"},
							SecretName: "tls-secret",
						},
					},
				},
			},
			expected: "80, 443",
		},
		{
			name: "multiple TLS entries",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{
							Hosts:      []string{"example.com"},
							SecretName: "tls-secret-1",
						},
						{
							Hosts:      []string{"another.com"},
							SecretName: "tls-secret-2",
						},
					},
				},
			},
			expected: "80, 443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIngressPorts(tt.ingress)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIngressTransformer_TransformToRow(t *testing.T) {
	transformer := NewIngressTransformer()
	now := time.Now()
	className := "nginx"

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-ingress",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: now},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &className,
			Rules: []networkingv1.IngressRule{
				{Host: "example.com"},
				{Host: "www.example.com"},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"example.com", "www.example.com"},
					SecretName: "tls-secret",
				},
			},
		},
		Status: networkingv1.IngressStatus{
			LoadBalancer: networkingv1.IngressLoadBalancerStatus{
				Ingress: []networkingv1.IngressLoadBalancerIngress{
					{IP: "192.168.1.100"},
				},
			},
		},
	}

	// Test without namespace
	row, identity, err := transformer.TransformToRow(ingress, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if identity.Name != "test-ingress" {
		t.Errorf("expected identity name to be test-ingress, got %s", identity.Name)
	}

	if len(row) != 6 { // NAME, CLASS, HOSTS, ADDRESS, PORTS, AGE
		t.Errorf("expected 6 columns, got %d", len(row))
	}

	// Verify specific columns
	if row[0] != "test-ingress" {
		t.Errorf("expected NAME to be test-ingress, got %s", row[0])
	}

	if row[1] != "nginx" {
		t.Errorf("expected CLASS to be nginx, got %s", row[1])
	}

	if row[2] != "example.com,www.example.com" {
		t.Errorf("expected HOSTS to be example.com,www.example.com, got %s", row[2])
	}

	if row[3] != "192.168.1.100" {
		t.Errorf("expected ADDRESS to be 192.168.1.100, got %s", row[3])
	}

	if row[4] != "80, 443" {
		t.Errorf("expected PORTS to be '80, 443', got %s", row[4])
	}

	// Test with namespace
	row, _, err = transformer.TransformToRow(ingress, true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(row) != 7 { // NAMESPACE + 6 columns
		t.Errorf("expected 7 columns with namespace, got %d", len(row))
	}

	if row[0] != "default" {
		t.Errorf("expected NAMESPACE to be default, got %s", row[0])
	}
}

func TestIngressTransformer_NoClass(t *testing.T) {
	transformer := NewIngressTransformer()
	now := time.Now()

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-ingress",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: now},
		},
		Spec: networkingv1.IngressSpec{
			// No IngressClassName set
			Rules: []networkingv1.IngressRule{
				{Host: "example.com"},
			},
		},
	}

	row, _, err := transformer.TransformToRow(ingress, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CLASS column should show <none> when not set
	if row[1] != "<none>" {
		t.Errorf("expected CLASS to be <none>, got %s", row[1])
	}
}
