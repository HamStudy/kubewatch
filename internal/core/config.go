package core

import (
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	KubeConfig          string
	CurrentContext      string
	CurrentNamespace    string
	InitialResourceType string
	RefreshInterval     int // in seconds
	LogTailLines        int
	MaxResourcesShown   int
	ColorScheme         string
}

// LoadConfig loads the application configuration
func LoadConfig() (*Config, error) {
	config := &Config{
		RefreshInterval:   2,
		LogTailLines:      100,
		MaxResourcesShown: 500,
		ColorScheme:       "default",
	}

	// Get kubeconfig path - pass the raw KUBECONFIG env var value
	// The k8s client will handle parsing multiple paths
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		// If KUBECONFIG is not set, use the default location
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	config.KubeConfig = kubeconfig

	// Get namespace from env or use default
	namespace := os.Getenv("KUBEWATCH_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	config.CurrentNamespace = namespace

	return config, nil
}
