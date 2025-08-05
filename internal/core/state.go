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

	// Resources cache
	Pods         []v1.Pod
	Deployments  []appsv1.Deployment
	StatefulSets []appsv1.StatefulSet
	Services     []v1.Service
	Ingresses    []networkingv1.Ingress
	ConfigMaps   []v1.ConfigMap
	Secrets      []v1.Secret

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
		SelectedItems:       make(map[string]bool),
		config:              config,
		SortAscending:       true,
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
