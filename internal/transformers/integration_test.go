package transformers

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HamStudy/kubewatch/internal/k8s"
	"github.com/HamStudy/kubewatch/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// MockMetricsProvider for testing
type MockMetricsProvider struct {
	metrics map[string]map[string]*k8s.PodMetrics
	mu      sync.RWMutex
}

func NewMockMetricsProvider() *MockMetricsProvider {
	return &MockMetricsProvider{
		metrics: make(map[string]map[string]*k8s.PodMetrics),
	}
}

func (m *MockMetricsProvider) SetPodMetrics(namespace, name string, metrics *k8s.PodMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.metrics[namespace] == nil {
		m.metrics[namespace] = make(map[string]*k8s.PodMetrics)
	}
	m.metrics[namespace][name] = metrics
}

func (m *MockMetricsProvider) GetPodMetrics(namespace string) (map[string]*k8s.PodMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if metrics, ok := m.metrics[namespace]; ok {
		return metrics, nil
	}
	return make(map[string]*k8s.PodMetrics), nil
}

// TestPodTransformerWithMetrics tests the complete pod transformation flow
// including CPU/Memory coloring based on requests - what users see
func TestPodTransformerWithMetrics(t *testing.T) {
	transformer := NewPodTransformer()
	templateEngine := template.NewEngine()
	metricsProvider := NewMockMetricsProvider()
	transformer.SetMetricsProvider(metricsProvider)

	// Create test pods with various states
	tests := []struct {
		name         string
		pod          v1.Pod
		metrics      *k8s.PodMetrics
		verifyOutput func(t *testing.T, row []string)
		scenario     string
	}{
		{
			name: "healthy pod with low resource usage",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "healthy-pod",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
				},
				Spec: v1.PodSpec{
					NodeName: "node-1",
					Containers: []v1.Container{
						{
							Name: "app",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    parseQuantity("500m"),
									v1.ResourceMemory: parseQuantity("256Mi"),
								},
							},
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					PodIP: "10.0.0.1",
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true, RestartCount: 0},
					},
				},
			},
			metrics: &k8s.PodMetrics{
				CPU:    "250m",  // 50% of request
				Memory: "128Mi", // 50% of request
			},
			verifyOutput: func(t *testing.T, row []string) {
				// Verify all columns are present
				assert.Equal(t, "healthy-pod", row[0], "NAME should be correct")
				assert.Equal(t, "1/1", row[2], "READY should show all containers ready")
				assert.Equal(t, "Running", row[3], "STATUS should be Running")
				assert.Equal(t, "0", row[4], "RESTARTS should be 0")
				assert.Contains(t, row[6], "250m", "CPU should show usage")
				assert.Contains(t, row[7], "128Mi", "MEMORY should show usage")
				assert.Equal(t, "10.0.0.1", row[8], "IP should be shown")
				assert.Equal(t, "node-1", row[9], "NODE should be shown")
			},
			scenario: "User sees healthy pod with green resource indicators",
		},
		{
			name: "pod with high CPU usage",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "high-cpu-pod",
					Namespace: "default",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    parseQuantity("100m"),
									v1.ResourceMemory: parseQuantity("128Mi"),
								},
							},
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						{Ready: true},
					},
				},
			},
			metrics: &k8s.PodMetrics{
				CPU:    "150m", // 150% of request - should be red with underline
				Memory: "64Mi", // 50% of request
			},
			verifyOutput: func(t *testing.T, row []string) {
				cpuCol := row[6]
				assert.Contains(t, cpuCol, "150m", "Should show actual CPU usage")
				// Should contain ANSI codes for styling (red background)
				assert.Contains(t, cpuCol, "\x1b[", "High CPU should be styled")
			},
			scenario: "User sees CPU over limit highlighted in red",
		},
		{
			name: "pod with restarts",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "restarting-pod",
					Namespace: "default",
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						{
							Ready:        false,
							RestartCount: 5,
							LastTerminationState: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									FinishedAt: metav1.NewTime(time.Now().Add(-30 * time.Minute)),
								},
							},
						},
						{
							Ready:        true,
							RestartCount: 3,
						},
					},
				},
			},
			verifyOutput: func(t *testing.T, row []string) {
				assert.Equal(t, "1/2", row[2], "Should show 1 of 2 containers ready")
				assert.Contains(t, row[4], "8", "Should show total restarts")
				assert.Contains(t, row[4], "ago", "Should show time since last restart")
			},
			scenario: "User sees restart count with time since last restart",
		},
		{
			name: "pending pod without metrics",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-pod",
					Namespace: "default",
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodScheduled,
							Status: v1.ConditionFalse,
							Reason: "Unschedulable",
						},
					},
				},
			},
			verifyOutput: func(t *testing.T, row []string) {
				assert.Equal(t, "Unschedulable", row[3], "Should show detailed status")
				assert.Equal(t, "-", row[6], "CPU should be dash when no metrics")
				assert.Equal(t, "-", row[7], "Memory should be dash when no metrics")
				assert.Equal(t, "-", row[8], "IP should be dash when pending")
				assert.Equal(t, "-", row[9], "NODE should be dash when not scheduled")
			},
			scenario: "User sees pending pod with clear status reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set metrics if provided
			if tt.metrics != nil {
				metricsProvider.SetPodMetrics(tt.pod.Namespace, tt.pod.Name, tt.metrics)
			}

			// Transform to row
			row, identity, err := transformer.TransformToRow(tt.pod, true, templateEngine)
			require.NoError(t, err, "Should transform without error")
			require.NotNil(t, identity, "Should return resource identity")

			// Verify identity
			assert.Equal(t, tt.pod.Name, identity.Name)
			assert.Equal(t, tt.pod.Namespace, identity.Namespace)
			assert.Equal(t, "Pod", identity.Kind)

			// Verify output
			tt.verifyOutput(t, row)

			t.Logf("Scenario verified: %s", tt.scenario)
		})
	}
}

// TestServiceTransformer tests service transformation with selector formatting
func TestServiceTransformer(t *testing.T) {
	transformer := NewServiceTransformer()
	templateEngine := template.NewEngine()

	tests := []struct {
		name     string
		service  v1.Service
		verify   func(t *testing.T, row []string)
		scenario string
	}{
		{
			name: "clusterip service with selector",
			service: v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "web-service",
					Namespace: "default",
				},
				Spec: v1.ServiceSpec{
					Type:      v1.ServiceTypeClusterIP,
					ClusterIP: "10.96.0.1",
					Selector: map[string]string{
						"app":     "web",
						"version": "v1",
					},
					Ports: []v1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(8080),
							Protocol:   v1.ProtocolTCP,
						},
					},
				},
			},
			verify: func(t *testing.T, row []string) {
				assert.Equal(t, "web-service", row[0])
				assert.Equal(t, "ClusterIP", row[2])
				assert.Equal(t, "10.96.0.1", row[3])
				assert.Equal(t, "80/TCP", row[5])
				// Selector should be formatted
				assert.Contains(t, row[6], "app=web")
				assert.Contains(t, row[6], "version=v1")
			},
			scenario: "User sees service with formatted selector",
		},
		{
			name: "loadbalancer service with external IP",
			service: v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "lb-service",
					Namespace: "default",
				},
				Spec: v1.ServiceSpec{
					Type:      v1.ServiceTypeLoadBalancer,
					ClusterIP: "10.96.0.2",
					Selector: map[string]string{
						"app": "api",
					},
					Ports: []v1.ServicePort{
						{Port: 443, TargetPort: intstr.FromInt(8443)},
						{Port: 80, TargetPort: intstr.FromInt(8080)},
					},
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{IP: "203.0.113.1"},
						},
					},
				},
			},
			verify: func(t *testing.T, row []string) {
				assert.Equal(t, "LoadBalancer", row[2])
				assert.Equal(t, "203.0.113.1", row[4], "Should show external IP")
				assert.Contains(t, row[5], "443/TCP")
				assert.Contains(t, row[5], "80/TCP")
			},
			scenario: "User sees LoadBalancer with external IP",
		},
		{
			name: "service with very long selector",
			service: v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "complex-service",
					Namespace: "default",
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeClusterIP,
					Selector: map[string]string{
						"app":                       "complex-application-name",
						"version":                   "v2.1.0-beta",
						"environment":               "production",
						"tier":                      "backend",
						"managed-by":                "helm",
						"part-of":                   "microservices-platform",
						"component":                 "api-gateway",
						"release":                   "stable",
						"very-long-label-name-here": "very-long-label-value-here",
					},
				},
			},
			verify: func(t *testing.T, row []string) {
				selector := row[6]
				// Should truncate long selectors
				assert.LessOrEqual(t, len(selector), 100, "Selector should be truncated")
				assert.Contains(t, selector, "...", "Should indicate truncation")
			},
			scenario: "User sees truncated selector for readability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, identity, err := transformer.TransformToRow(tt.service, true, templateEngine)
			require.NoError(t, err)
			require.NotNil(t, identity)

			tt.verify(t, row)
			t.Logf("Scenario: %s", tt.scenario)
		})
	}
}

// TestIngressTransformer tests ingress transformation with host truncation
func TestIngressTransformer(t *testing.T) {
	transformer := NewIngressTransformer()
	templateEngine := template.NewEngine()

	tests := []struct {
		name     string
		ingress  networkingv1.Ingress
		verify   func(t *testing.T, row []string)
		scenario string
	}{
		{
			name: "simple ingress",
			ingress: networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "web-ingress",
					Namespace: "default",
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "example.com"},
					},
				},
				Status: networkingv1.IngressStatus{
					LoadBalancer: networkingv1.IngressLoadBalancerStatus{
						Ingress: []networkingv1.IngressLoadBalancerIngress{
							{IP: "203.0.113.1"},
						},
					},
				},
			},
			verify: func(t *testing.T, row []string) {
				assert.Equal(t, "web-ingress", row[0])
				assert.Equal(t, "example.com", row[3])
				assert.Equal(t, "203.0.113.1", row[4])
			},
			scenario: "User sees ingress with single host",
		},
		{
			name: "ingress with multiple hosts",
			ingress: networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multi-ingress",
					Namespace: "default",
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "api.example.com"},
						{Host: "www.example.com"},
						{Host: "admin.example.com"},
						{Host: "blog.example.com"},
						{Host: "shop.example.com"},
						{Host: "cdn.example.com"},
					},
				},
			},
			verify: func(t *testing.T, row []string) {
				hosts := row[3]
				// Should show multiple hosts
				assert.Contains(t, hosts, "api.example.com")
				// Should truncate if too many
				if len(hosts) > 80 {
					assert.Contains(t, hosts, "...", "Should truncate long host lists")
				}
			},
			scenario: "User sees multiple hosts with truncation if needed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, identity, err := transformer.TransformToRow(tt.ingress, true, templateEngine)
			require.NoError(t, err)
			require.NotNil(t, identity)

			tt.verify(t, row)
			t.Logf("Scenario: %s", tt.scenario)
		})
	}
}

// TestTransformerPerformance tests that transformers are fast enough
// for real-time updates in the UI
func TestTransformerPerformance(t *testing.T) {
	transformer := NewPodTransformer()
	templateEngine := template.NewEngine()
	metricsProvider := NewMockMetricsProvider()
	transformer.SetMetricsProvider(metricsProvider)

	// Create 500 pods with metrics
	pods := make([]v1.Pod, 500)
	for i := 0; i < 500; i++ {
		pods[i] = v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceCPU:    parseQuantity("100m"),
								v1.ResourceMemory: parseQuantity("128Mi"),
							},
						},
					},
				},
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				ContainerStatuses: []v1.ContainerStatus{
					{Ready: true, RestartCount: int32(i % 5)},
				},
			},
		}

		// Add metrics
		metricsProvider.SetPodMetrics("default", pods[i].Name, &k8s.PodMetrics{
			CPU:    fmt.Sprintf("%dm", 50+i%100),
			Memory: fmt.Sprintf("%dMi", 64+i%128),
		})
	}

	// Measure transformation time
	start := time.Now()
	for _, pod := range pods {
		row, _, err := transformer.TransformToRow(pod, true, templateEngine)
		require.NoError(t, err)
		require.NotEmpty(t, row)
	}
	elapsed := time.Since(start)

	// Should be fast enough for smooth UI updates
	assert.Less(t, elapsed.Milliseconds(), int64(100),
		"Transforming 500 pods should take less than 100ms, took %v", elapsed)

	avgTime := elapsed.Nanoseconds() / int64(len(pods)) / 1000 // microseconds
	assert.Less(t, avgTime, int64(200),
		"Average transform time should be under 200μs, was %dμs", avgTime)

	t.Logf("Transformed %d pods in %v (avg: %dμs per pod)",
		len(pods), elapsed, avgTime)
}

// TestTransformerEdgeCases tests edge cases that could occur in production
func TestTransformerEdgeCases(t *testing.T) {
	transformer := NewPodTransformer()
	templateEngine := template.NewEngine()

	tests := []struct {
		name     string
		pod      v1.Pod
		verify   func(t *testing.T, row []string, err error)
		scenario string
	}{
		{
			name: "pod with no containers",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-pod",
					Namespace: "default",
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			},
			verify: func(t *testing.T, row []string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "0/0", row[2], "Should handle no containers")
			},
			scenario: "Handle pod with no containers gracefully",
		},
		{
			name: "pod with nil status fields",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nil-status-pod",
					Namespace: "default",
				},
				Status: v1.PodStatus{
					// All fields nil/empty
				},
			},
			verify: func(t *testing.T, row []string, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, row[3], "Should have some status")
				assert.Equal(t, "-", row[8], "Should show dash for nil IP")
			},
			scenario: "Handle nil status fields without panic",
		},
		{
			name: "pod with very long name",
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      strings.Repeat("very-long-pod-name-", 20),
					Namespace: "default",
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
			verify: func(t *testing.T, row []string, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, row[0], "Should handle long names")
				// Name should be preserved as-is (truncation happens in UI)
				assert.Contains(t, row[0], "very-long-pod-name")
			},
			scenario: "Handle very long resource names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, _, err := transformer.TransformToRow(tt.pod, true, templateEngine)
			tt.verify(t, row, err)
			t.Logf("Edge case handled: %s", tt.scenario)
		})
	}
}

// TestConcurrentTransformations tests thread safety of transformers
func TestConcurrentTransformations(t *testing.T) {
	transformer := NewPodTransformer()
	templateEngine := template.NewEngine()
	metricsProvider := NewMockMetricsProvider()
	transformer.SetMetricsProvider(metricsProvider)

	// Create test pod
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "concurrent-pod",
			Namespace: "default",
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				{Ready: true},
			},
		},
	}

	// Set metrics
	metricsProvider.SetPodMetrics("default", "concurrent-pod", &k8s.PodMetrics{
		CPU:    "100m",
		Memory: "128Mi",
	})

	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				// Modify pod slightly for each iteration
				testPod := pod
				testPod.Name = fmt.Sprintf("pod-%d-%d", id, i)

				row, identity, err := transformer.TransformToRow(testPod, true, templateEngine)
				if err != nil {
					errors <- err
					continue
				}

				// Verify basic correctness
				if len(row) == 0 {
					errors <- fmt.Errorf("empty row returned")
				}
				if identity == nil {
					errors <- fmt.Errorf("nil identity returned")
				}
				if identity != nil && identity.Name != testPod.Name {
					errors <- fmt.Errorf("wrong name in identity")
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errCount int
	for err := range errors {
		t.Errorf("Concurrent transformation error: %v", err)
		errCount++
		if errCount > 10 {
			t.Fatal("Too many concurrent errors")
		}
	}

	assert.Equal(t, 0, errCount, "Should have no errors during concurrent transformations")
}

// Helper function to parse quantity
func parseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}
