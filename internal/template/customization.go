package template

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// CustomizationManager handles user template customization
type CustomizationManager struct {
	engine        *Engine
	configDir     string
	userTemplates map[string]*UserTemplate
	watchers      map[string]*fsnotify.Watcher
	mu            sync.RWMutex
	hotReload     bool
	callbacks     []func(string, *UserTemplate)
}

// UserTemplate represents a user-customizable template
type UserTemplate struct {
	Name         string            `yaml:"name"`
	ResourceType string            `yaml:"resourceType"`
	Description  string            `yaml:"description"`
	Template     string            `yaml:"template"`
	Columns      []string          `yaml:"columns"`
	Variables    map[string]string `yaml:"variables"`
	CreatedAt    time.Time         `yaml:"createdAt"`
	UpdatedAt    time.Time         `yaml:"updatedAt"`
	Version      string            `yaml:"version"`
	Author       string            `yaml:"author"`
	Tags         []string          `yaml:"tags"`
}

// TemplateValidationError represents a template validation error
type TemplateValidationError struct {
	Template string
	Line     int
	Column   int
	Message  string
}

func (e *TemplateValidationError) Error() string {
	return fmt.Sprintf("template validation error in %s at line %d, column %d: %s",
		e.Template, e.Line, e.Column, e.Message)
}

// NewCustomizationManager creates a new template customization manager
func NewCustomizationManager(engine *Engine, configDir string) *CustomizationManager {
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config", "kubewatch", "templates")
	}

	return &CustomizationManager{
		engine:        engine,
		configDir:     configDir,
		userTemplates: make(map[string]*UserTemplate),
		watchers:      make(map[string]*fsnotify.Watcher),
		hotReload:     false,
		callbacks:     make([]func(string, *UserTemplate), 0),
	}
}

// EnableHotReload enables hot-reloading of templates for development
func (cm *CustomizationManager) EnableHotReload() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.hotReload = true

	// Set up file watchers for existing templates
	return cm.setupWatchers()
}

// DisableHotReload disables hot-reloading
func (cm *CustomizationManager) DisableHotReload() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.hotReload = false

	// Close all watchers
	for _, watcher := range cm.watchers {
		watcher.Close()
	}
	cm.watchers = make(map[string]*fsnotify.Watcher)
}

// LoadUserTemplates loads all user templates from the config directory
func (cm *CustomizationManager) LoadUserTemplates() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Ensure config directory exists
	if err := os.MkdirAll(cm.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Walk through template files
	return filepath.WalkDir(cm.configDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		template, err := cm.loadTemplateFile(path)
		if err != nil {
			return fmt.Errorf("failed to load template %s: %w", path, err)
		}

		cm.userTemplates[template.Name] = template

		// Set up watcher if hot reload is enabled
		if cm.hotReload {
			if err := cm.watchTemplate(path, template.Name); err != nil {
				// Log error but don't fail loading
				fmt.Printf("Warning: failed to watch template %s: %v\n", path, err)
			}
		}

		return nil
	})
}

// loadTemplateFile loads a single template file
func (cm *CustomizationManager) loadTemplateFile(path string) (*UserTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse YAML (we'll use a simple parser for now)
	template := &UserTemplate{}
	// TODO: Implement YAML parsing
	// For now, assume it's a simple template file
	template.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	template.Template = string(data)
	template.UpdatedAt = time.Now()

	return template, nil
}

// SaveUserTemplate saves a user template to disk
func (cm *CustomizationManager) SaveUserTemplate(template *UserTemplate) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate template first
	if err := cm.validateTemplate(template); err != nil {
		return err
	}

	// Update timestamps
	now := time.Now()
	if template.CreatedAt.IsZero() {
		template.CreatedAt = now
	}
	template.UpdatedAt = now

	// Save to disk
	filename := fmt.Sprintf("%s.yaml", template.Name)
	path := filepath.Join(cm.configDir, filename)

	// TODO: Marshal to YAML
	data := template.Template // For now, just save the template content
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	// Update in-memory cache
	cm.userTemplates[template.Name] = template

	// Set up watcher if hot reload is enabled
	if cm.hotReload {
		if err := cm.watchTemplate(path, template.Name); err != nil {
			fmt.Printf("Warning: failed to watch template %s: %v\n", path, err)
		}
	}

	// Notify callbacks
	for _, callback := range cm.callbacks {
		callback(template.Name, template)
	}

	return nil
}

// GetUserTemplate retrieves a user template by name
func (cm *CustomizationManager) GetUserTemplate(name string) (*UserTemplate, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	template, exists := cm.userTemplates[name]
	return template, exists
}

// ListUserTemplates returns all user templates
func (cm *CustomizationManager) ListUserTemplates() map[string]*UserTemplate {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*UserTemplate)
	for k, v := range cm.userTemplates {
		result[k] = v
	}
	return result
}

// DeleteUserTemplate removes a user template
func (cm *CustomizationManager) DeleteUserTemplate(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Remove from disk
	filename := fmt.Sprintf("%s.yaml", name)
	path := filepath.Join(cm.configDir, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete template file: %w", err)
	}

	// Remove from memory
	delete(cm.userTemplates, name)

	// Close watcher if exists
	if watcher, exists := cm.watchers[path]; exists {
		watcher.Close()
		delete(cm.watchers, path)
	}

	return nil
}

// ValidateTemplate validates a template for syntax and function usage
func (cm *CustomizationManager) ValidateTemplate(template *UserTemplate) error {
	return cm.validateTemplate(template)
}

// validateTemplate performs template validation
func (cm *CustomizationManager) validateTemplate(template *UserTemplate) error {
	if template.Name == "" {
		return &TemplateValidationError{
			Template: template.Name,
			Message:  "template name cannot be empty",
		}
	}

	if template.Template == "" {
		return &TemplateValidationError{
			Template: template.Name,
			Message:  "template content cannot be empty",
		}
	}

	// Validate template syntax using the engine
	if err := cm.engine.Validate(template.Template); err != nil {
		return &TemplateValidationError{
			Template: template.Name,
			Message:  fmt.Sprintf("template syntax error: %v", err),
		}
	}

	// Test template execution with sample data
	sampleData := cm.generateSampleData(template.ResourceType)
	if _, err := cm.engine.Execute(template.Template, sampleData); err != nil {
		return &TemplateValidationError{
			Template: template.Name,
			Message:  fmt.Sprintf("template execution error: %v", err),
		}
	}

	return nil
}

// generateSampleData creates sample data for template testing
func (cm *CustomizationManager) generateSampleData(resourceType string) interface{} {
	switch resourceType {
	case "pod":
		return map[string]interface{}{
			"Name":      "sample-pod",
			"Namespace": "default",
			"Status":    "Running",
			"Ready":     "1/1",
			"Restarts":  0,
			"Age":       "5m",
		}
	case "deployment":
		return map[string]interface{}{
			"Name":      "sample-deployment",
			"Namespace": "default",
			"Ready":     "3/3",
			"UpToDate":  3,
			"Available": 3,
			"Age":       "10m",
		}
	default:
		return map[string]interface{}{
			"Name":      "sample-resource",
			"Namespace": "default",
			"Status":    "Active",
			"Age":       "5m",
		}
	}
}

// setupWatchers sets up file watchers for hot reload
func (cm *CustomizationManager) setupWatchers() error {
	return filepath.WalkDir(cm.configDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		templateName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		return cm.watchTemplate(path, templateName)
	})
}

// watchTemplate sets up a file watcher for a specific template
func (cm *CustomizationManager) watchTemplate(path, templateName string) error {
	// Don't create duplicate watchers
	if _, exists := cm.watchers[path]; exists {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := watcher.Add(path); err != nil {
		watcher.Close()
		return err
	}

	cm.watchers[path] = watcher

	// Start watching in a goroutine
	go func() {
		defer watcher.Close()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					// Reload template
					if template, err := cm.loadTemplateFile(path); err == nil {
						cm.mu.Lock()
						cm.userTemplates[templateName] = template
						cm.mu.Unlock()

						// Notify callbacks
						for _, callback := range cm.callbacks {
							callback(templateName, template)
						}
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("Template watcher error: %v\n", err)
			}
		}
	}()

	return nil
}

// OnTemplateChange registers a callback for template changes
func (cm *CustomizationManager) OnTemplateChange(callback func(string, *UserTemplate)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.callbacks = append(cm.callbacks, callback)
}

// CreateDefaultTemplates creates default templates for common resource types
func (cm *CustomizationManager) CreateDefaultTemplates() error {
	defaults := map[string]*UserTemplate{
		"pod-status": {
			Name:         "pod-status",
			ResourceType: "pod",
			Description:  "Pod status with color-coded indicators",
			Template: `{{.Name | printf "%-30s"}} {{.Namespace | printf "%-15s"}} ` +
				`{{colorIf (eq .Status "Running") "green" "red" .Status | printf "%-10s"}} ` +
				`{{.Ready | printf "%-8s"}} {{.Restarts | printf "%3d"}} {{.Age}}`,
			Columns: []string{"Name", "Namespace", "Status", "Ready", "Restarts", "Age"},
			Version: "1.0.0",
			Author:  "KubeWatch",
			Tags:    []string{"default", "pod", "status"},
		},
		"deployment-health": {
			Name:         "deployment-health",
			ResourceType: "deployment",
			Description:  "Deployment health with progress indicators",
			Template: `{{.Name | printf "%-30s"}} {{.Namespace | printf "%-15s"}} ` +
				`{{colorIf (eq .Ready .Replicas) "green" "yellow" .Ready | printf "%-8s"}} ` +
				`{{.UpToDate | printf "%3d"}} {{.Available | printf "%3d"}} {{.Age}}`,
			Columns: []string{"Name", "Namespace", "Ready", "Up-to-date", "Available", "Age"},
			Version: "1.0.0",
			Author:  "KubeWatch",
			Tags:    []string{"default", "deployment", "health"},
		},
	}

	for _, template := range defaults {
		if _, exists := cm.GetUserTemplate(template.Name); !exists {
			if err := cm.SaveUserTemplate(template); err != nil {
				return fmt.Errorf("failed to create default template %s: %w", template.Name, err)
			}
		}
	}

	return nil
}

// ExportTemplate exports a template to a file
func (cm *CustomizationManager) ExportTemplate(name, path string) error {
	template, exists := cm.GetUserTemplate(name)
	if !exists {
		return fmt.Errorf("template %s not found", name)
	}

	// TODO: Marshal to YAML with full metadata
	data := template.Template
	return os.WriteFile(path, []byte(data), 0644)
}

// ImportTemplate imports a template from a file
func (cm *CustomizationManager) ImportTemplate(path string) error {
	template, err := cm.loadTemplateFile(path)
	if err != nil {
		return err
	}

	return cm.SaveUserTemplate(template)
}
