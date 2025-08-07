package k8s

import (
	"k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// KubernetesInterface is an interface that both real and fake clientsets implement
type KubernetesInterface interface {
	kubernetes.Interface
}

// MetricsInterface is an interface for metrics client
type MetricsInterface interface {
	metricsclient.Interface
}
