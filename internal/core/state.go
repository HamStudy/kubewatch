package core

import (
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// ResourceType represents the type of Kubernetes resource
type ResourceType string

const (
	ResourceTypePod         ResourceType = "Pods"
	ResourceTypeDeployment  ResourceType = "Deployments"
	ResourceTypeStatefulSet ResourceType = "StatefulSets"
	ResourceTypeService     ResourceType = "Services"
	ResourceTypeIngress     ResourceType = "Ingresses"
	ResourceTypeConfigMap   ResourceType = "ConfigMaps"
	ResourceTypeSecret      ResourceType = "Secrets"
)

// State holds the application state
type State struct {
	mu sync.RWMutex

	// Current view state
	CurrentResourceType ResourceType
	CurrentNamespace    string
	CurrentContext      string
	SelectedIndex       int
	ScrollOffset        int

	// Multi-context support
	CurrentContexts  []string        // Active contexts
	ContextFilter    map[string]bool // Which contexts to show
	MultiContextMode bool            // Whether in multi-context mode

	// Resources cache (single context)
	Pods         []v1.Pod
	Deployments  []appsv1.Deployment
	StatefulSets []appsv1.StatefulSet
	Services     []v1.Service
	Ingresses    []networkingv1.Ingress
	ConfigMaps   []v1.ConfigMap
	Secrets      []v1.Secret

	// Multi-context resources cache
	PodsByContext         map[string][]v1.Pod
	DeploymentsByContext  map[string][]appsv1.Deployment
	StatefulSetsByContext map[string][]appsv1.StatefulSet
	ServicesByContext     map[string][]v1.Service
	IngressesByContext    map[string][]networkingv1.Ingress
	ConfigMapsByContext   map[string][]v1.ConfigMap
	SecretsByContext      map[string][]v1.Secret

	// UI state
	ShowHelp      bool
	ShowLogs      bool
	LogsTarget    string // pod or deployment name
	FilterString  string
	SortColumn    string
	SortAscending bool

	// Selection state
	SelectedItems map[string]bool // for multi-select

	config *Config
}

// NewState creates a new application state
func NewState(config *Config) *State {
	// Set initial resource type from config
	resourceType := ResourceTypePod
	if config.InitialResourceType != "" {
		switch config.InitialResourceType {
		case "deployment":
			resourceType = ResourceTypeDeployment
		case "statefulset":
			resourceType = ResourceTypeStatefulSet
		case "service":
			resourceType = ResourceTypeService
		case "ingress":
			resourceType = ResourceTypeIngress
		case "configmap":
			resourceType = ResourceTypeConfigMap
		case "secret":
			resourceType = ResourceTypeSecret
		default:
			resourceType = ResourceTypePod
		}
	}

	return &State{
		CurrentResourceType: resourceType,
		CurrentNamespace:    config.CurrentNamespace,
		CurrentContext:      config.CurrentContext,
		SelectedItems:       make(map[string]bool),
		config:              config,
		SortColumn:          "NAME", // Default sort by name
		SortAscending:       true,

		// Initialize multi-context fields
		CurrentContexts:       []string{},
		ContextFilter:         make(map[string]bool),
		MultiContextMode:      false,
		PodsByContext:         make(map[string][]v1.Pod),
		DeploymentsByContext:  make(map[string][]appsv1.Deployment),
		StatefulSetsByContext: make(map[string][]appsv1.StatefulSet),
		ServicesByContext:     make(map[string][]v1.Service),
		IngressesByContext:    make(map[string][]networkingv1.Ingress),
		ConfigMapsByContext:   make(map[string][]v1.ConfigMap),
		SecretsByContext:      make(map[string][]v1.Secret),
	}
}

// GetCurrentResourceCount returns the count of current resource type
func (s *State) GetCurrentResourceCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch s.CurrentResourceType {
	case ResourceTypePod:
		return len(s.Pods)
	case ResourceTypeDeployment:
		return len(s.Deployments)
	case ResourceTypeStatefulSet:
		return len(s.StatefulSets)
	case ResourceTypeService:
		return len(s.Services)
	case ResourceTypeIngress:
		return len(s.Ingresses)
	case ResourceTypeConfigMap:
		return len(s.ConfigMaps)
	case ResourceTypeSecret:
		return len(s.Secrets)
	default:
		return 0
	}
}

// SetNamespace updates the current namespace
func (s *State) SetNamespace(namespace string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentNamespace = namespace
	s.SelectedIndex = 0
	s.ScrollOffset = 0
	s.SelectedItems = make(map[string]bool)
}

// SetResourceType updates the current resource type
func (s *State) SetResourceType(resourceType ResourceType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentResourceType = resourceType
	s.SelectedIndex = 0
	s.ScrollOffset = 0
	s.SelectedItems = make(map[string]bool)
}

// UpdatePods updates the pods list
func (s *State) UpdatePods(pods []v1.Pod) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Pods = pods
}

// UpdateDeployments updates the deployments list
func (s *State) UpdateDeployments(deployments []appsv1.Deployment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Deployments = deployments
}

// UpdateStatefulSets updates the statefulsets list
func (s *State) UpdateStatefulSets(statefulsets []appsv1.StatefulSet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.StatefulSets = statefulsets
}

// UpdateServices updates the services list
func (s *State) UpdateServices(services []v1.Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Services = services
}

// UpdateIngresses updates the ingresses list
func (s *State) UpdateIngresses(ingresses []networkingv1.Ingress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Ingresses = ingresses
}

// UpdateConfigMaps updates the configmaps list
func (s *State) UpdateConfigMaps(configmaps []v1.ConfigMap) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConfigMaps = configmaps
}

// UpdateSecrets updates the secrets list
func (s *State) UpdateSecrets(secrets []v1.Secret) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Secrets = secrets
}

// SetMultiContextMode enables or disables multi-context mode
func (s *State) SetMultiContextMode(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MultiContextMode = enabled
}

// SetCurrentContexts updates the active contexts
func (s *State) SetCurrentContexts(contexts []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentContexts = contexts

	// Update context filter
	s.ContextFilter = make(map[string]bool)
	for _, ctx := range contexts {
		s.ContextFilter[ctx] = true
	}
}

// UpdatePodsByContext updates pods for a specific context
func (s *State) UpdatePodsByContext(context string, pods []v1.Pod) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PodsByContext[context] = pods
}

// UpdateDeploymentsByContext updates deployments for a specific context
func (s *State) UpdateDeploymentsByContext(context string, deployments []appsv1.Deployment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DeploymentsByContext[context] = deployments
}

// UpdateStatefulSetsByContext updates statefulsets for a specific context
func (s *State) UpdateStatefulSetsByContext(context string, statefulsets []appsv1.StatefulSet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.StatefulSetsByContext[context] = statefulsets
}

// UpdateServicesByContext updates services for a specific context
func (s *State) UpdateServicesByContext(context string, services []v1.Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ServicesByContext[context] = services
}

// UpdateIngressesByContext updates ingresses for a specific context
func (s *State) UpdateIngressesByContext(context string, ingresses []networkingv1.Ingress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IngressesByContext[context] = ingresses
}

// UpdateConfigMapsByContext updates configmaps for a specific context
func (s *State) UpdateConfigMapsByContext(context string, configmaps []v1.ConfigMap) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConfigMapsByContext[context] = configmaps
}

// UpdateSecretsByContext updates secrets for a specific context
func (s *State) UpdateSecretsByContext(context string, secrets []v1.Secret) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SecretsByContext[context] = secrets
}

// GetAggregatedPods returns pods from all active contexts
func (s *State) GetAggregatedPods() []v1.Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.MultiContextMode {
		return s.Pods
	}

	var allPods []v1.Pod
	for _, context := range s.CurrentContexts {
		if s.ContextFilter[context] {
			if pods, ok := s.PodsByContext[context]; ok {
				allPods = append(allPods, pods...)
			}
		}
	}
	return allPods
}

// GetAggregatedDeployments returns deployments from all active contexts
func (s *State) GetAggregatedDeployments() []appsv1.Deployment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.MultiContextMode {
		return s.Deployments
	}

	var allDeployments []appsv1.Deployment
	for _, context := range s.CurrentContexts {
		if s.ContextFilter[context] {
			if deployments, ok := s.DeploymentsByContext[context]; ok {
				allDeployments = append(allDeployments, deployments...)
			}
		}
	}
	return allDeployments
}

// GetSortState returns the current sort column and direction in a thread-safe manner
func (s *State) GetSortState() (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SortColumn, s.SortAscending
}

// SetSortState updates the sort column and direction in a thread-safe manner
func (s *State) SetSortState(column string, ascending bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SortColumn = column
	s.SortAscending = ascending
}

// GetCurrentNamespace returns the current namespace in a thread-safe manner
func (s *State) GetCurrentNamespace() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentNamespace
}
