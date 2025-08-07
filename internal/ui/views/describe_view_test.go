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
	cmd := view.Init()

	if cmd == nil {
		t.Error("Init should return a command to load describe content")
	}

	// Execute the command
	msg := cmd()
	if _, ok := msg.(describeLoadedMsg); !ok {
		t.Error("Init command should return describeLoadedMsg")
	}
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

	// Initial resize
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	model, _ := view.Update(msg)
	view = model.(*DescribeView)

	if view.width != 100 || view.height != 50 {
		t.Errorf("size = (%d, %d), want (100, 50)", view.width, view.height)
	}

	if !view.ready {
		t.Error("should be ready after window size message")
	}

	if view.viewport.Width != 100 {
		t.Errorf("viewport width = %d, want 100", view.viewport.Width)
	}

	if view.viewport.Height != 47 { // 50 - 3 for header and footer
		t.Errorf("viewport height = %d, want 47", view.viewport.Height)
	}

	// Resize again
	msg = tea.WindowSizeMsg{Width: 120, Height: 60}
	model, _ = view.Update(msg)
	view = model.(*DescribeView)

	if view.viewport.Width != 120 {
		t.Error("viewport should update on resize")
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
		wantContains []string
	}{
		{
			name:         "generates pod content",
			resourceType: "Pod",
			resourceName: "test-pod",
			namespace:    "default",
			wantContains: []string{"Name:", "test-pod", "Status:", "Containers:", "Events:"},
		},
		{
			name:         "generates deployment content",
			resourceType: "Deployment",
			resourceName: "test-deploy",
			namespace:    "production",
			wantContains: []string{"Name:", "test-deploy", "Replicas:", "Strategy:", "Pod Template:"},
		},
		{
			name:         "generates service content",
			resourceType: "Service",
			resourceName: "test-svc",
			namespace:    "default",
			wantContains: []string{"Name:", "test-svc", "Type:", "IP:", "Port:", "Endpoints:"},
		},
		{
			name:         "generates generic content",
			resourceType: "ConfigMap",
			resourceName: "test-cm",
			namespace:    "default",
			wantContains: []string{"Name:", "test-cm", "Type:", "ConfigMap"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewDescribeView(tt.resourceType, tt.resourceName, tt.namespace, tt.context)
			content := view.getDescribeContent()

			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Errorf("content does not contain %q", want)
				}
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
