package resource

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/HamStudy/kubewatch/configs/resources"
	"gopkg.in/yaml.v3"
)

// Loader handles loading resource definitions from various sources
type Loader struct {
	registry *Registry
}

// NewLoader creates a new resource loader
func NewLoader() *Loader {
	return &Loader{
		registry: NewRegistry(),
	}
}

// LoadEmbedded loads all embedded resource definitions
func (l *Loader) LoadEmbedded() error {
	// Get the embedded filesystem
	embeddedFS, err := resources.GetFS()
	if err != nil {
		return fmt.Errorf("failed to get embedded filesystem: %w", err)
	}

	// Walk through the embedded filesystem
	err = fs.WalkDir(embeddedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process YAML files
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// Read the file
		data, err := fs.ReadFile(embeddedFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Parse and register the definition
		var def ResourceDefinition
		if err := yaml.Unmarshal(data, &def); err != nil {
			return fmt.Errorf("failed to parse embedded file %s: %w", path, err)
		}

		if err := l.registry.Register(&def); err != nil {
			return fmt.Errorf("failed to register embedded resource from %s: %w", path, err)
		}

		return nil
	})

	return err
}

// LoadFromFile loads a resource definition from a file
func (l *Loader) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var def ResourceDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return fmt.Errorf("failed to parse file %s: %w", path, err)
	}

	if err := l.registry.Register(&def); err != nil {
		return fmt.Errorf("failed to register resource from %s: %w", path, err)
	}

	return nil
}

// LoadFromDirectory loads all resource definitions from a directory
func (l *Loader) LoadFromDirectory(dir string, recursive bool) error {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("failed to access directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	if recursive {
		// Walk directory recursively
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Only process YAML files
			if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
				return nil
			}

			// Load the file (ignore errors for individual files)
			if err := l.LoadFromFile(path); err != nil {
				// Log the error but continue processing other files
				fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", path, err)
			}

			return nil
		})
	} else {
		// Read directory non-recursively
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			// Skip directories
			if entry.IsDir() {
				continue
			}

			// Only process YAML files
			name := entry.Name()
			if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
				continue
			}

			path := filepath.Join(dir, name)
			// Load the file (ignore errors for individual files)
			if err := l.LoadFromFile(path); err != nil {
				// Log the error but continue processing other files
				fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", path, err)
			}
		}
	}

	return err
}

// LoadUserOverrides loads user-defined resource overrides from ~/.config/kubewatch/resources/
func (l *Loader) LoadUserOverrides() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home directory, just skip user overrides
		return nil
	}

	configDir := filepath.Join(homeDir, ".config", "kubewatch", "resources")

	// Check if the directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// Directory doesn't exist, no overrides to load
		return nil
	}

	// Load all YAML files from the directory (recursively)
	return l.LoadFromDirectory(configDir, true)
}

// LoadAll loads embedded resources and then user overrides
func (l *Loader) LoadAll() error {
	// First load embedded resources
	if err := l.LoadEmbedded(); err != nil {
		return fmt.Errorf("failed to load embedded resources: %w", err)
	}

	// Then load user overrides (which can override embedded resources)
	if err := l.LoadUserOverrides(); err != nil {
		return fmt.Errorf("failed to load user overrides: %w", err)
	}

	return nil
}

// GetRegistry returns the resource registry
func (l *Loader) GetRegistry() *Registry {
	return l.registry
}

// Clear clears all loaded resources
func (l *Loader) Clear() {
	l.registry.Clear()
}

// LoadFromData loads a resource definition from raw YAML data
func (l *Loader) LoadFromData(data []byte) error {
	var def ResourceDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return fmt.Errorf("failed to parse YAML data: %w", err)
	}

	if err := l.registry.Register(&def); err != nil {
		return fmt.Errorf("failed to register resource: %w", err)
	}

	return nil
}

// GetDefaultLoader creates and initializes a loader with all default resources
func GetDefaultLoader() (*Loader, error) {
	loader := NewLoader()
	if err := loader.LoadAll(); err != nil {
		return nil, err
	}
	return loader, nil
}
