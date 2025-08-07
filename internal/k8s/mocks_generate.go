package k8s

//go:generate mockgen -destination=mocks/mock_kubernetes.go -package=mocks k8s.io/client-go/kubernetes Interface
//go:generate mockgen -destination=mocks/mock_corev1.go -package=mocks k8s.io/client-go/kubernetes/typed/core/v1 CoreV1Interface,NamespaceInterface,PodInterface,ServiceInterface,ConfigMapInterface,SecretInterface
//go:generate mockgen -destination=mocks/mock_appsv1.go -package=mocks k8s.io/client-go/kubernetes/typed/apps/v1 AppsV1Interface,DeploymentInterface,StatefulSetInterface
//go:generate mockgen -destination=mocks/mock_networkingv1.go -package=mocks k8s.io/client-go/kubernetes/typed/networking/v1 NetworkingV1Interface,IngressInterface
//go:generate mockgen -destination=mocks/mock_metrics.go -package=mocks k8s.io/metrics/pkg/client/clientset/versioned Interface
//go:generate mockgen -destination=mocks/mock_metricsv1beta1.go -package=mocks k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1 MetricsV1beta1Interface,PodMetricsInterface,NodeMetricsInterface
//go:generate mockgen -destination=mocks/mock_rest.go -package=mocks k8s.io/client-go/rest Interface
//go:generate mockgen -destination=mocks/mock_watch.go -package=mocks k8s.io/apimachinery/pkg/watch Interface
