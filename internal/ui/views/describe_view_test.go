package views

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDescribeViewInitialization(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceName string
		namespace    string
		context      string
	}{
		{
			name:         "creates describe view for pod",
			resourceType: "Pod",
			resourceName: "test-pod",
			namespace:    "default",
			context:      "test-context",
		},
		{
			name:         "creates describe view for deployment",
			resourceType: "Deployment",
			resourceName: "test-deployment",
			namespace:    "kube-system",
			context:      "",
		},
		{
			name:         "creates describe view for service",
			resourceType: "Service",
			resourceName: "test-service",
			namespace:    "production",
			context:      "prod-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewDescribeView(tt.resourceType, tt.resourceName, tt.namespace, tt.context)

			if view == nil {
				t.Fatal("NewDescribeView returned nil")
			}

			if view.resourceType != tt.resourceType {
				t.Errorf("resourceType = %q, want %q", view.resourceType, tt.resourceType)
			}

			if view.resourceName != tt.resourceName {
				t.Errorf("resourceName = %q, want %q", view.resourceName, tt.resourceName)
			}

			if view.namespace != tt.namespace {
				t.Errorf("namespace = %q, want %q", view.namespace, tt.namespace)
			}

			if view.context != tt.context {
				t.Errorf("context = %q, want %q", view.context, tt.context)
			}

			if !view.loading {
				t.Error("view should start in loading state")
			}
		})
	}
}

func TestDescribeViewInit(t *testing.T) {
	view := NewDescribeView("Pod", "test-pod", "default", "")

	// Test behavior: Init should return a command that can be executed
	cmd := view.Init()
	if cmd == nil {
		t.Error("Init should return a command to initiate content loading")
	}

	// Test behavior: The command should be executable without panicking
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Init command should be executable without panicking: %v", r)
			}
		}()

		// Execute the command - this tests that it's a valid tea.Cmd
		if cmd != nil {
			_ = cmd() // Should not panic
		}
	}()

	// Test behavior: loadDescribe should return an executable command
	loadCmd := view.loadDescribe()
	if loadCmd == nil {
		t.Error("loadDescribe should return an executable command")
	}

	// Test behavior: loadDescribe command should execute without panicking
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("loadDescribe command should execute without panicking: %v", r)
			}
		}()

		if loadCmd != nil {
			msg := loadCmd()
			if msg == nil {
				t.Error("loadDescribe command should return a message")
			}
		}
	}()
}

func TestDescribeViewContentLoading(t *testing.T) {
	view := NewDescribeView("Pod", "test-pod", "default", "")

	// Simulate content loaded
	msg := describeLoadedMsg{
		content: "Test content",
		err:     nil,
	}

	model, _ := view.Update(msg)
	view = model.(*DescribeView)

	if view.loading {
		t.Error("should not be loading after content is loaded")
	}

	if view.content != "Test content" {
		t.Errorf("content = %q, want %q", view.content, "Test content")
	}
}

func TestDescribeViewErrorHandling(t *testing.T) {
	view := NewDescribeView("Pod", "test-pod", "default", "")
	view.ready = true

	// Simulate error loading content
	msg := describeLoadedMsg{
		content: "",
		err:     context.DeadlineExceeded,
	}

	model, _ := view.Update(msg)
	view = model.(*DescribeView)

	if view.loading {
		t.Error("should not be loading after error")
	}

	if !strings.Contains(view.content, "Error loading description") {
		t.Error("should show error message in content")
	}
}

func TestDescribeViewKeyHandling(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		keyType tea.KeyType
	}{
		{
			name: "home key",
			key:  "g",
		},
		{
			name:    "home key type",
			keyType: tea.KeyHome,
		},
		{
			name: "end key",
			key:  "G",
		},
		{
			name:    "end key type",
			keyType: tea.KeyEnd,
		},
		{
			name: "escape key",
			key:  "esc",
		},
		{
			name: "quit key",
			key:  "q",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewDescribeView("Pod", "test-pod", "default", "")
			view.ready = true
			view.SetSize(80, 24)
			view.content = strings.Repeat("Line\n", 100) // Long content for scrolling
			view.viewport.SetContent(view.content)

			var msg tea.KeyMsg
			if tt.keyType != 0 {
				msg = tea.KeyMsg{Type: tt.keyType}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			model, _ := view.Update(msg)
			view = model.(*DescribeView)

			// Should handle key without crashing
			// Specific behavior is tested by viewport
		})
	}
}

func TestDescribeViewWindowResize(t *testing.T) {
	view := NewDescribeView("Pod", "test-pod", "default", "")

	// Test behavior: Window resize should make view ready and update dimensions
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	model, _ := view.Update(msg)
	view = model.(*DescribeView)

	// Test behavior: View should track the window dimensions
	if view.width != 100 || view.height != 50 {
		t.Errorf("view should track window dimensions: got (%d, %d), want (100, 50)", view.width, view.height)
	}

	// Test behavior: View should become ready after receiving window size
	if !view.ready {
		t.Error("view should be ready after receiving window size message")
	}

	// Test behavior: Viewport should be sized appropriately for content area
	if view.viewport.Width != 100 {
		t.Errorf("viewport width should match window width: got %d, want 100", view.viewport.Width)
	}

	// Test behavior: Viewport height should be less than window height (room for UI elements)
	if view.viewport.Height >= 50 {
		t.Errorf("viewport height (%d) should be less than window height (50) to leave room for UI elements", view.viewport.Height)
	}

	if view.viewport.Height <= 0 {
		t.Errorf("viewport height (%d) should be positive", view.viewport.Height)
	}

	// Test behavior: Subsequent resizes should update dimensions
	initialViewportHeight := view.viewport.Height
	msg = tea.WindowSizeMsg{Width: 120, Height: 60}
	model, _ = view.Update(msg)
	view = model.(*DescribeView)

	if view.viewport.Width != 120 {
		t.Errorf("viewport should update width on resize: got %d, want 120", view.viewport.Width)
	}

	// Test behavior: Larger window should result in larger viewport
	if view.viewport.Height <= initialViewportHeight {
		t.Error("larger window should result in larger viewport height")
	}
}

func TestDescribeViewSetSize(t *testing.T) {
	view := NewDescribeView("Pod", "test-pod", "default", "")

	view.SetSize(150, 40)

	if view.width != 150 || view.height != 40 {
		t.Errorf("size = (%d, %d), want (150, 40)", view.width, view.height)
	}

	if !view.ready {
		t.Error("should be ready after SetSize")
	}

	if view.viewport.Width != 150 || view.viewport.Height != 37 {
		t.Error("viewport should be sized correctly")
	}
}

func TestDescribeViewRendering(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceName string
		namespace    string
		context      string
		loading      bool
		content      string
		wantContains []string
	}{
		{
			name:         "shows loading state",
			resourceType: "Pod",
			resourceName: "test-pod",
			namespace:    "default",
			loading:      true,
			wantContains: []string{"Loading describe information"},
		},
		{
			name:         "shows resource info in header",
			resourceType: "Pod",
			resourceName: "test-pod",
			namespace:    "default",
			context:      "test-context",
			loading:      false,
			content:      "Test content",
			wantContains: []string{"Describe:", "Pod/test-pod", "default", "test-context"},
		},
		{
			name:         "shows controls in footer",
			resourceType: "Pod",
			resourceName: "test-pod",
			namespace:    "default",
			loading:      false,
			content:      "Test content",
			wantContains: []string{"Scroll", "Top/Bottom", "Close"},
		},
		{
			name:         "renders without context",
			resourceType: "Service",
			resourceName: "test-svc",
			namespace:    "kube-system",
			context:      "",
			loading:      false,
			content:      "Service details",
			wantContains: []string{"Service/test-svc", "kube-system"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewDescribeView(tt.resourceType, tt.resourceName, tt.namespace, tt.context)
			view.loading = tt.loading
			view.content = tt.content
			view.ready = true
			view.SetSize(80, 24)

			if tt.content != "" {
				view.viewport.SetContent(tt.content)
			}

			output := view.View()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output does not contain %q", want)
				}
			}
		})
	}
}

func TestDescribeViewNotReady(t *testing.T) {
	view := NewDescribeView("Pod", "test-pod", "default", "")
	view.ready = false

	output := view.View()

	if output != "Loading..." {
		t.Errorf("not ready view = %q, want %q", output, "Loading...")
	}
}

func TestDescribeViewGetDescribeContent(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceName string
		namespace    string
		context      string
		wantBehavior string
	}{
		{
			name:         "generates pod content with essential fields",
			resourceType: "Pod",
			resourceName: "test-pod",
			namespace:    "default",
			wantBehavior: "should include resource identification and pod-specific information",
		},
		{
			name:         "generates deployment content with essential fields",
			resourceType: "Deployment",
			resourceName: "test-deploy",
			namespace:    "production",
			wantBehavior: "should include resource identification and deployment-specific information",
		},
		{
			name:         "generates service content with essential fields",
			resourceType: "Service",
			resourceName: "test-svc",
			namespace:    "default",
			wantBehavior: "should include resource identification and service-specific information",
		},
		{
			name:         "generates configmap content with essential fields",
			resourceType: "ConfigMap",
			resourceName: "test-cm",
			namespace:    "default",
			wantBehavior: "should include resource identification and configmap-specific information",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewDescribeView(tt.resourceType, tt.resourceName, tt.namespace, tt.context)
			content := view.getDescribeContent()

			// Test behavior: content should be non-empty and contain resource identification
			if content == "" {
				t.Error("getDescribeContent should return non-empty content")
			}

			// Test behavior: content should identify the resource
			if !strings.Contains(content, tt.resourceName) {
				t.Errorf("content should contain resource name %q", tt.resourceName)
			}

			if !strings.Contains(content, tt.namespace) {
				t.Errorf("content should contain namespace %q", tt.namespace)
			}

			// Test behavior: content should have structured information (contains "Name:" field)
			if !strings.Contains(content, "Name:") {
				t.Error("content should be structured with field labels like 'Name:'")
			}

			// Test behavior: content should include events section for all resources
			if !strings.Contains(content, "Events:") {
				t.Error("content should include Events section")
			}

			// Test behavior: content length should be reasonable (not just a stub)
			if len(content) < 100 {
				t.Errorf("content seems too short (%d chars), may be incomplete", len(content))
			}
		})
	}
}

func TestFormatResourceType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"pods", "pod"},
		{"deployments", "deployment"},
		{"services", "service"},
		{"ingress", "ingress"}, // Special case - doesn't remove 's'
		{"configmaps", "configmap"},
		{"pod", "pod"}, // Already singular
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FormatResourceType(tt.input)
			if got != tt.want {
				t.Errorf("FormatResourceType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
