package k8s

import (
	"k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// NewTestClient creates a client for testing with interface-based clientsets
func NewTestClient(clientset kubernetes.Interface, metricsClient metricsclient.Interface) *Client {
	// This is a workaround for testing since fake.Clientset and kubernetes.Clientset are different types
	// In production, we use kubernetes.Clientset, but for testing we need to accept the interface
	return &Client{
		clientset:     clientset.(*kubernetes.Clientset),
		metricsClient: metricsClient.(*metricsclient.Clientset),
		config:        nil,
	}
}

// TestableClient wraps Client for testing with interface-based dependencies
type TestableClient struct {
	KubeClient    kubernetes.Interface
	MetricsClient metricsclient.Interface
	Config        interface{}
}
