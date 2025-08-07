package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescribeViewEnhancements(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceName string
		namespace    string
		context      string
	}{
		{
			name:         "Pod describe view",
			resourceType: "Pod",
			resourceName: "test-pod",
			namespace:    "default",
			context:      "test-context",
		},
		{
			name:         "Deployment describe view",
			resourceType: "Deployment",
			resourceName: "test-deployment",
			namespace:    "default",
			context:      "test-context",
		},
		{
			name:         "Service describe view",
			resourceType: "Service",
			resourceName: "test-service",
			namespace:    "default",
			context:      "test-context",
		},
		{
			name:         "Ingress describe view",
			resourceType: "Ingress",
			resourceName: "test-ingress",
			namespace:    "default",
			context:      "test-context",
		},
		{
			name:         "ConfigMap describe view",
			resourceType: "ConfigMap",
			resourceName: "test-configmap",
			namespace:    "default",
			context:      "test-context",
		},
		{
			name:         "Secret describe view",
			resourceType: "Secret",
			resourceName: "test-secret",
			namespace:    "default",
			context:      "test-context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewDescribeView(tt.resourceType, tt.resourceName, tt.namespace, tt.context)
			require.NotNil(t, view)

			// Test initial state
			assert.Equal(t, tt.resourceType, view.resourceType)
			assert.Equal(t, tt.resourceName, view.resourceName)
			assert.Equal(t, tt.namespace, view.namespace)
			assert.Equal(t, tt.context, view.context)
			assert.True(t, view.loading)
			assert.False(t, view.wordWrap)
			assert.True(t, view.autoRefresh)
			assert.NotNil(t, view.templateEngine)
			assert.NotNil(t, view.events)

			// Test initialization
			cmd := view.Init()
			assert.NotNil(t, cmd)

			// Test window size message
			model, cmd := view.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
			view = model.(*DescribeView)
			assert.NotNil(t, view)
			assert.Equal(t, 100, view.width)
			assert.Equal(t, 30, view.height)
			assert.True(t, view.ready)

			// Test describe loaded message
			model, cmd = view.Update(describeLoadedMsg{
				content: "Test content",
				err:     nil,
			})
			view = model.(*DescribeView)
			assert.NotNil(t, view)
			assert.False(t, view.loading)
			assert.Equal(t, "Test content", view.content)
			assert.False(t, view.lastUpdated.IsZero())

			// Test word wrap toggle
			model, cmd = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
			view = model.(*DescribeView)
			assert.NotNil(t, view)
			assert.True(t, view.wordWrap)

			// Test auto-refresh toggle
			model, cmd = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
			view = model.(*DescribeView)
			assert.NotNil(t, view)
			assert.False(t, view.autoRefresh)

			// Test manual refresh
			model, cmd = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
			view = model.(*DescribeView)
			assert.NotNil(t, view)
			assert.NotNil(t, cmd)

			// Test view rendering
			viewStr := view.View()
			assert.Contains(t, viewStr, tt.resourceName)
			assert.Contains(t, viewStr, "Word wrap: ON")
			assert.Contains(t, viewStr, "Auto-refresh: OFF")
		})
	}
}

func TestDescribeViewWordWrap(t *testing.T) {
	view := NewDescribeView("Pod", "test-pod", "default", "test-context")
	view.width = 50
	view.ready = true

	// Test word wrap functionality
	longText := "This is a very long line that should be wrapped when word wrap is enabled and should exceed the width limit"

	// Without word wrap
	view.wordWrap = false
	view.content = longText
	view.setViewportContent()

	// With word wrap
	view.wordWrap = true
	view.setViewportContent()

	// Test wrapText method directly
	wrapped := view.wrapText(longText, 30)
	lines := len(strings.Split(wrapped, "\n"))
	assert.Greater(t, lines, 1, "Text should be wrapped into multiple lines")
}

func TestDescribeViewAutoRefresh(t *testing.T) {
	view := NewDescribeView("Pod", "test-pod", "default", "test-context")

	// Test auto-refresh message
	view.autoRefresh = true
	model, cmd := view.Update(autoRefreshMsg{time: time.Now()})
	view = model.(*DescribeView)
	assert.NotNil(t, view)
	assert.NotNil(t, cmd)

	// Test auto-refresh disabled
	view.autoRefresh = false
	model, cmd = view.Update(autoRefreshMsg{time: time.Now()})
	view = model.(*DescribeView)
	assert.NotNil(t, view)
	// Should not trigger refresh when disabled
}
func TestDescribeViewTemplateGeneration(t *testing.T) {
	view := NewDescribeView("Pod", "test-pod", "default", "test-context")

	// Test template data generation
	data := view.createMockResourceData()
	assert.NotNil(t, data)
	assert.Equal(t, "test-pod", data["Name"])
	assert.Equal(t, "default", data["Namespace"])
	assert.Equal(t, "test-context", data["Context"])
	assert.Equal(t, "Pod", data["Type"])

	// Test that events are included
	events, ok := data["Events"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(events), 0)

	// Test template retrieval
	template := view.getDescribeTemplate("Pod")
	assert.NotEmpty(t, template)
	assert.Contains(t, template, "Name:")
	assert.Contains(t, template, "Events:")

	// Test unknown resource type
	unknownTemplate := view.getDescribeTemplate("UnknownType")
	assert.NotEmpty(t, unknownTemplate)
	assert.Contains(t, unknownTemplate, "(Detailed information would appear here)")
}

func TestDescribeViewEnhancedContent(t *testing.T) {
	resourceTypes := []string{"Pod", "Deployment", "Service", "Ingress", "ConfigMap", "Secret"}

	for _, resourceType := range resourceTypes {
		t.Run(resourceType, func(t *testing.T) {
			view := NewDescribeView(resourceType, "test-resource", "default", "test-context")

			// Test enhanced content generation
			content := view.getEnhancedDescribeContent()
			assert.NotEmpty(t, content)
			assert.Contains(t, content, "test-resource")
			assert.Contains(t, content, "default")
			assert.Contains(t, content, "Events:")

			// Test that content includes resource-specific information
			switch resourceType {
			case "Pod":
				assert.Contains(t, content, "Containers:")
				assert.Contains(t, content, "Conditions:")
				assert.Contains(t, content, "Volumes:")
			case "Deployment":
				assert.Contains(t, content, "Replicas:")
				assert.Contains(t, content, "Strategy")
			case "Service":
				assert.Contains(t, content, "Type:")
				assert.Contains(t, content, "Endpoints:")
			case "Ingress":
				assert.Contains(t, content, "Rules:")
				assert.Contains(t, content, "Host")
			case "ConfigMap":
				assert.Contains(t, content, "Data")
				assert.Contains(t, content, "BinaryData")
			case "Secret":
				assert.Contains(t, content, "Type:")
				assert.Contains(t, content, "Data")
			}
		})
	}
}

func TestDescribeViewKeyBindings(t *testing.T) {
	view := NewDescribeView("Pod", "test-pod", "default", "test-context")
	view.ready = true
	view.content = "test content"

	testCases := []struct {
		key      string
		expected func(*DescribeView) bool
	}{
		{"u", func(v *DescribeView) bool { return v.wordWrap }},
		{"a", func(v *DescribeView) bool { return !v.autoRefresh }}, // Should toggle to false
	}

	for _, tc := range testCases {
		t.Run("Key "+tc.key, func(t *testing.T) {
			model, _ := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(tc.key[0])}})
			view = model.(*DescribeView)
			assert.True(t, tc.expected(view))
		})
	}
}
