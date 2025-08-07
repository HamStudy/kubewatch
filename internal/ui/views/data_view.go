package views

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"
)

// DataView displays ConfigMap or Secret data
type DataView struct {
	viewport     viewport.Model
	resourceType string // "ConfigMap" or "Secret"
	resourceName string
	namespace    string
	data         map[string]string
	binaryData   map[string][]byte
	decoded      bool // For secrets, toggle between encoded/decoded
	selectedKey  int
	keys         []string
	width        int
	height       int
	ready        bool
}

// NewConfigMapView creates a view for a ConfigMap
func NewConfigMapView(cm *v1.ConfigMap) *DataView {
	v := &DataView{
		viewport:     viewport.New(80, 20),
		resourceType: "ConfigMap",
		resourceName: cm.Name,
		namespace:    cm.Namespace,
		data:         cm.Data,
		binaryData:   cm.BinaryData,
		decoded:      true, // ConfigMaps are always decoded
	}
	v.updateKeys()
	return v
}

// NewSecretView creates a view for a Secret
func NewSecretView(secret *v1.Secret) *DataView {
	// Convert byte data to strings for display
	data := make(map[string]string)
	for k, v := range secret.Data {
		data[k] = string(v)
	}

	v := &DataView{
		viewport:     viewport.New(80, 20),
		resourceType: "Secret",
		resourceName: secret.Name,
		namespace:    secret.Namespace,
		data:         data,
		decoded:      false, // Secrets start encoded
	}
	v.updateKeys()
	return v
}

// Init initializes the view
func (v *DataView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *DataView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if !v.ready {
			v.viewport = viewport.New(msg.Width, msg.Height-4) // Leave room for header and footer
			v.viewport.YPosition = 0
			v.ready = true
		} else {
			v.viewport.Width = msg.Width
			v.viewport.Height = msg.Height - 4
		}
		v.updateContent()

	case tea.KeyMsg:
		switch msg.String() {
		case "d":
			// Toggle decode for secrets
			if v.resourceType == "Secret" {
				v.decoded = !v.decoded
				v.updateContent()
			}
			return v, nil
		case "tab", "j":
			// Next key
			if v.selectedKey < len(v.keys)-1 {
				v.selectedKey++
				v.updateContent()
			}
			return v, nil
		case "shift+tab", "k":
			// Previous key
			if v.selectedKey > 0 {
				v.selectedKey--
				v.updateContent()
			}
			return v, nil
		case "g", "home":
			v.viewport.GotoTop()
			return v, nil
		case "G", "end":
			v.viewport.GotoBottom()
			return v, nil
		case "esc", "q":
			// Close view
			return v, nil
		}
	}

	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the data view
func (v *DataView) View() string {
	if !v.ready {
		return "Loading..."
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	header := fmt.Sprintf("%s: %s/%s", v.resourceType, v.resourceName, v.namespace)
	if v.resourceType == "Secret" {
		if v.decoded {
			header += " [DECODED]"
		} else {
			header += " [ENCODED]"
		}
	}

	// Footer with controls
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	footer := "Tab/j/k: Navigate keys | ↑↓: Scroll | g/G: Top/Bottom"
	if v.resourceType == "Secret" {
		footer += " | d: Toggle decode"
	}
	footer += " | Esc: Close"

	// Key list
	keyListStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true)

	keyInfo := ""
	if len(v.keys) > 0 {
		keyInfo = fmt.Sprintf(" | Key %d/%d: %s", v.selectedKey+1, len(v.keys), v.keys[v.selectedKey])
	}

	return fmt.Sprintf(
		"%s%s\n%s\n%s",
		headerStyle.Render(header),
		keyListStyle.Render(keyInfo),
		v.viewport.View(),
		footerStyle.Render(footer),
	)
}

// updateKeys updates the sorted list of keys
func (v *DataView) updateKeys() {
	v.keys = []string{}

	// Add regular data keys
	for k := range v.data {
		v.keys = append(v.keys, k)
	}

	// Add binary data keys
	for k := range v.binaryData {
		v.keys = append(v.keys, k+" (binary)")
	}

	sort.Strings(v.keys)

	if v.selectedKey >= len(v.keys) && len(v.keys) > 0 {
		v.selectedKey = len(v.keys) - 1
	}
}

// updateContent updates the viewport content based on selected key
func (v *DataView) updateContent() {
	if len(v.keys) == 0 {
		v.viewport.SetContent("No data")
		return
	}

	if v.selectedKey >= len(v.keys) {
		v.selectedKey = len(v.keys) - 1
	}

	key := v.keys[v.selectedKey]
	isBinary := strings.HasSuffix(key, " (binary)")
	if isBinary {
		key = strings.TrimSuffix(key, " (binary)")
	}

	var content string

	// Style for the content
	contentStyle := lipgloss.NewStyle().
		Padding(1, 2)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	// Build content for all keys or selected key
	if v.selectedKey < 0 {
		// Show all keys (overview mode)
		var lines []string
		for _, k := range v.keys {
			lines = append(lines, keyStyle.Render(k))
		}
		content = strings.Join(lines, "\n")
	} else {
		// Show selected key's value
		content = keyStyle.Render(fmt.Sprintf("Key: %s", key)) + "\n\n"

		if isBinary {
			// Binary data
			if data, ok := v.binaryData[key]; ok {
				if v.decoded && v.resourceType == "Secret" {
					// Try to decode and display as string
					content += string(data)
				} else {
					// Show as base64
					content += base64.StdEncoding.EncodeToString(data)
				}
			}
		} else {
			// Regular string data
			if value, ok := v.data[key]; ok {
				if v.resourceType == "Secret" && !v.decoded {
					// Show encoded (base64)
					content += base64.StdEncoding.EncodeToString([]byte(value))
				} else if v.resourceType == "Secret" && v.decoded {
					// Decode from base64
					decoded, err := base64.StdEncoding.DecodeString(value)
					if err != nil {
						content += fmt.Sprintf("Error decoding: %v\n\nRaw value:\n%s", err, value)
					} else {
						content += string(decoded)
					}
				} else {
					// ConfigMap - show as is
					content += value
				}
			}
		}
	}

	v.viewport.SetContent(contentStyle.Render(content))
}

// SetSize updates the view size
func (v *DataView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height - 4
	v.ready = true
	v.updateContent()
}
