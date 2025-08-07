package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Version   string                     `yaml:"version"`
	Theme     string                     `yaml:"theme"`
	Columns   map[string]*ColumnConfig   `yaml:"columns"`
	Templates map[string]*TemplateConfig `yaml:"templates"`
	Filters   map[string]*FilterConfig   `yaml:"filters"`
	Layouts   map[string]*LayoutConfig   `yaml:"layouts"`
	Settings  *Settings                  `yaml:"settings"`
}

// ColumnConfig defines column display configuration
type ColumnConfig struct {
	ResourceType string              `yaml:"resourceType"`
	Columns      []*ColumnDefinition `yaml:"columns"`
	DefaultSort  *SortConfig         `yaml:"defaultSort"`
	GroupBy      *GroupConfig        `yaml:"groupBy"`
}

// ColumnDefinition defines a single column
type ColumnDefinition struct {
	Name       string      `yaml:"name"`
	Visible    bool        `yaml:"visible"`
	Width      ColumnWidth `yaml:"width"`
	MinWidth   int         `yaml:"minWidth"`
	MaxWidth   int         `yaml:"maxWidth"`
	Priority   int         `yaml:"priority"`
	Align      string      `yaml:"align"`
	Template   string      `yaml:"template"`
	Formatter  string      `yaml:"formatter"`
	Source     string      `yaml:"source"`
	Computed   bool        `yaml:"computed"`
	Expression string      `yaml:"expression"`
	Condition  string      `yaml:"condition"`
}

// ColumnWidth can be auto, fixed, or percentage
type ColumnWidth struct {
	Type  string `yaml:"type"` // auto, fixed, percent
	Value int    `yaml:"value"`
}

// SortConfig defines sorting configuration
type SortConfig struct {
	Column    string `yaml:"column"`
	Ascending bool   `yaml:"ascending"`
	Secondary string `yaml:"secondary"`
}

// GroupConfig defines grouping configuration
type GroupConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Column    string `yaml:"column"`
	Collapsed bool   `yaml:"collapsed"`
}

// TemplateConfig defines a formatting template
type TemplateConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Template    string `yaml:"template"`
}

// FilterConfig defines a filter preset
type FilterConfig struct {
	Name         string         `yaml:"name"`
	Description  string         `yaml:"description"`
	Filters      []*Filter      `yaml:"filters"`
	QuickFilters []*QuickFilter `yaml:"quickFilters"`
}

// Filter defines a single filter rule
type Filter struct {
	Type       string      `yaml:"type"`
	Field      string      `yaml:"field"`
	Operator   string      `yaml:"operator"`
	Value      interface{} `yaml:"value"`
	Expression string      `yaml:"expression"`
}

// QuickFilter defines a hotkey-accessible filter
type QuickFilter struct {
	Key        string `yaml:"key"`
	Name       string `yaml:"name"`
	Expression string `yaml:"expression"`
}

// LayoutConfig defines a view layout
type LayoutConfig struct {
	Name      string         `yaml:"name"`
	SplitView *SplitConfig   `yaml:"splitView"`
	Panels    []*PanelConfig `yaml:"panels"`
}

// SplitConfig defines split view configuration
type SplitConfig struct {
	Enabled     bool    `yaml:"enabled"`
	Orientation string  `yaml:"orientation"`
	Ratio       float64 `yaml:"ratio"`
}

// PanelConfig defines a panel in a layout
type PanelConfig struct {
	Type     string                 `yaml:"type"`
	Position string                 `yaml:"position"`
	Height   string                 `yaml:"height"`
	Width    string                 `yaml:"width"`
	Config   map[string]interface{} `yaml:"config"`
}

// Settings defines user preferences
type Settings struct {
	AutoRefresh      *AutoRefreshConfig `yaml:"autoRefresh"`
	DefaultNamespace string             `yaml:"defaultNamespace"`
	DefaultContext   string             `yaml:"defaultContext"`
	ShowMetrics      bool               `yaml:"showMetrics"`
	WordWrap         bool               `yaml:"wordWrap"`
	Shortcuts        []*Shortcut        `yaml:"shortcuts"`
}

// AutoRefreshConfig defines auto-refresh settings
type AutoRefreshConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Interval string `yaml:"interval"`
}

// Shortcut defines a keyboard shortcut
type Shortcut struct {
	Key    string `yaml:"key"`
	Action string `yaml:"action"`
	Args   string `yaml:"args"`
}

// Loader handles configuration loading and management
type Loader struct {
	configDir string
	defaults  *Config
	user      *Config
	merged    *Config
	mu        sync.RWMutex
}

// NewLoader creates a new configuration loader
func NewLoader(configDir string) *Loader {
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config", "kubewatch")
	}

	return &Loader{
		configDir: configDir,
		defaults:  getDefaultConfig(),
	}
}

// Load loads the configuration from disk
func (l *Loader) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure config directory exists
	if err := os.MkdirAll(l.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load user config if it exists
	configPath := filepath.Join(l.configDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		userConfig, err := l.loadConfigFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to load user config: %w", err)
		}
		l.user = userConfig
	}

	// Merge configs
	l.merged = l.mergeConfigs(l.defaults, l.user)

	return nil
}

// LoadFile loads a specific configuration file
func (l *Loader) LoadFile(path string) (*Config, error) {
	return l.loadConfigFile(path)
}

// loadConfigFile loads a configuration from a file
func (l *Loader) loadConfigFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return l.parseConfig(file)
}

// LoadString loads configuration from a string
func (l *Loader) LoadString(content string) (*Config, error) {
	return l.parseConfig(strings.NewReader(content))
}

// parseConfig parses configuration from a reader
func (l *Loader) parseConfig(r io.Reader) (*Config, error) {
	var config Config
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true) // Strict parsing

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := l.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// validateConfig validates a configuration
func (l *Loader) validateConfig(config *Config) error {
	// Validate version
	if config.Version == "" {
		config.Version = "1.0.0"
	}

	// Validate theme
	if config.Theme == "" {
		config.Theme = "default"
	}

	// Validate columns
	for name, col := range config.Columns {
		if col.ResourceType == "" {
			col.ResourceType = name
		}

		// Validate column definitions
		for _, def := range col.Columns {
			if def.Name == "" {
				return fmt.Errorf("column must have a name")
			}
			if def.Priority == 0 {
				def.Priority = 5
			}
			if def.MinWidth == 0 {
				def.MinWidth = 5
			}
			if def.MaxWidth == 0 {
				def.MaxWidth = 100
			}
			if def.MinWidth > def.MaxWidth {
				return fmt.Errorf("column %s: minWidth > maxWidth", def.Name)
			}
		}
	}

	return nil
}

// mergeConfigs merges user config over defaults
func (l *Loader) mergeConfigs(defaults, user *Config) *Config {
	if user == nil {
		return defaults
	}
	if defaults == nil {
		return user
	}

	// Deep copy defaults
	merged := *defaults

	// Override with user settings
	if user.Version != "" {
		merged.Version = user.Version
	}
	if user.Theme != "" {
		merged.Theme = user.Theme
	}

	// Merge columns
	if merged.Columns == nil {
		merged.Columns = make(map[string]*ColumnConfig)
	}
	for k, v := range user.Columns {
		merged.Columns[k] = v
	}

	// Merge templates
	if merged.Templates == nil {
		merged.Templates = make(map[string]*TemplateConfig)
	}
	for k, v := range user.Templates {
		merged.Templates[k] = v
	}

	// Merge filters
	if merged.Filters == nil {
		merged.Filters = make(map[string]*FilterConfig)
	}
	for k, v := range user.Filters {
		merged.Filters[k] = v
	}

	// Merge layouts
	if merged.Layouts == nil {
		merged.Layouts = make(map[string]*LayoutConfig)
	}
	for k, v := range user.Layouts {
		merged.Layouts[k] = v
	}

	// Override settings
	if user.Settings != nil {
		merged.Settings = user.Settings
	}

	return &merged
}

// Get returns the current configuration
func (l *Loader) Get() *Config {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.merged != nil {
		return l.merged
	}
	return l.defaults
}

// GetColumnConfig returns column configuration for a resource type
func (l *Loader) GetColumnConfig(resourceType string) *ColumnConfig {
	config := l.Get()
	if config.Columns != nil {
		if col, ok := config.Columns[resourceType]; ok {
			return col
		}
	}
	return nil
}

// GetTemplate returns a template by name
func (l *Loader) GetTemplate(name string) *TemplateConfig {
	config := l.Get()
	if config.Templates != nil {
		if tmpl, ok := config.Templates[name]; ok {
			return tmpl
		}
	}
	return nil
}

// Save saves the current configuration to disk
func (l *Loader) Save() error {
	l.mu.RLock()
	config := l.user
	if config == nil {
		config = l.merged
	}
	l.mu.RUnlock()

	if config == nil {
		return fmt.Errorf("no configuration to save")
	}

	// Ensure config directory exists
	if err := os.MkdirAll(l.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	configPath := filepath.Join(l.configDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// SaveTemplate saves a template override
func (l *Loader) SaveTemplate(name string, template *TemplateConfig) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure user config exists
	if l.user == nil {
		l.user = &Config{
			Version:   "1.0.0",
			Templates: make(map[string]*TemplateConfig),
		}
	}
	if l.user.Templates == nil {
		l.user.Templates = make(map[string]*TemplateConfig)
	}

	// Add template
	l.user.Templates[name] = template

	// Re-merge configs
	l.merged = l.mergeConfigs(l.defaults, l.user)

	// Save to disk
	return l.Save()
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *Config {
	return &Config{
		Version: "1.0.0",
		Theme:   "default",
		Settings: &Settings{
			AutoRefresh: &AutoRefreshConfig{
				Enabled:  true,
				Interval: "5s",
			},
			ShowMetrics: true,
			WordWrap:    false,
		},
	}
}
