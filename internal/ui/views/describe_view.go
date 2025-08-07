package views

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/HamStudy/kubewatch/internal/k8s"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DescribeView displays the kubectl describe output for a resource
type DescribeView struct {
	viewport     viewport.Model
	content      string
	resourceType string
	resourceName string
	namespace    string
	context      string
	width        int
	height       int
	ready        bool
	loading      bool
}

// NewDescribeView creates a new describe view for a resource
func NewDescribeView(resourceType, resourceName, namespace, context string) *DescribeView {
	return &DescribeView{
		viewport:     viewport.New(80, 20),
		resourceType: resourceType,
		resourceName: resourceName,
		namespace:    namespace,
		context:      context,
		loading:      true,
	}
}

// Init initializes the view
func (v *DescribeView) Init() tea.Cmd {
	return v.loadDescribe()
}

// loadDescribe loads the describe output for the resource
func (v *DescribeView) loadDescribe() tea.Cmd {
	return func() tea.Msg {
		// Use placeholder content for now - real implementation would need client access
		return describeLoadedMsg{
			content: v.getDescribeContent(),
		}
	}
}

// LoadDescribeWithClient loads the describe output using a real K8s client
func (v *DescribeView) LoadDescribeWithClient(ctx context.Context, client *k8s.Client) tea.Cmd {
	return func() tea.Msg {
		content, err := GetDescribeContent(ctx, client, v.resourceType, v.resourceName, v.namespace)
		return describeLoadedMsg{
			content: content,
			err:     err,
		}
	}
}

// getDescribeContent gets the describe content using the K8s client
func (v *DescribeView) getDescribeContent() string {
	// In a real implementation, this would use the K8s client to get detailed resource info
	// For now, return formatted placeholder content
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("Name:         %s\n", v.resourceName))
	buf.WriteString(fmt.Sprintf("Namespace:    %s\n", v.namespace))
	if v.context != "" {
		buf.WriteString(fmt.Sprintf("Context:      %s\n", v.context))
	}
	buf.WriteString(fmt.Sprintf("Type:         %s\n", v.resourceType))
	buf.WriteString("\n")

	// Add type-specific information
	switch v.resourceType {
	case "Pod":
		buf.WriteString("Status:       Running\n")
		buf.WriteString("IP:           10.244.1.5\n")
		buf.WriteString("Node:         node-1\n")
		buf.WriteString("\nContainers:\n")
		buf.WriteString("  app:\n")
		buf.WriteString("    Image:      nginx:latest\n")
		buf.WriteString("    Port:       80/TCP\n")
		buf.WriteString("    State:      Running\n")
		buf.WriteString("    Ready:      True\n")
		buf.WriteString("    Restart Count: 0\n")

	case "Deployment":
		buf.WriteString("Replicas:     3 desired | 3 updated | 3 total | 3 available\n")
		buf.WriteString("Strategy:     RollingUpdate\n")
		buf.WriteString("Selector:     app=nginx\n")
		buf.WriteString("\nPod Template:\n")
		buf.WriteString("  Labels:     app=nginx\n")
		buf.WriteString("  Containers:\n")
		buf.WriteString("    nginx:\n")
		buf.WriteString("      Image:    nginx:latest\n")
		buf.WriteString("      Port:     80/TCP\n")

	case "Service":
		buf.WriteString("Type:         ClusterIP\n")
		buf.WriteString("IP:           10.96.1.1\n")
		buf.WriteString("Port:         http 80/TCP\n")
		buf.WriteString("Endpoints:    10.244.1.5:80,10.244.1.6:80\n")

	default:
		buf.WriteString("(Detailed information would appear here)\n")
	}

	buf.WriteString("\nEvents:\n")
	buf.WriteString("  Type    Reason    Age   From               Message\n")
	buf.WriteString("  ----    ------    ----  ----               -------\n")
	buf.WriteString("  Normal  Scheduled 5m    default-scheduler  Successfully assigned to node-1\n")
	buf.WriteString("  Normal  Pulled    5m    kubelet            Container image already present\n")
	buf.WriteString("  Normal  Created   5m    kubelet            Created container\n")
	buf.WriteString("  Normal  Started   5m    kubelet            Started container\n")

	return buf.String()
}

// GetDescribeUsingClient gets actual describe content using the K8s client
func GetDescribeContent(ctx context.Context, client *k8s.Client, resourceType, resourceName, namespace string) (string, error) {
	return client.DescribeResource(ctx, resourceType, resourceName, namespace)
}

// Update handles messages
func (v *DescribeView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if !v.ready {
			v.viewport = viewport.New(msg.Width, msg.Height-3) // Leave room for header and footer
			v.viewport.YPosition = 0
			v.ready = true
		} else {
			v.viewport.Width = msg.Width
			v.viewport.Height = msg.Height - 3
		}
		if v.content != "" {
			v.viewport.SetContent(v.content)
		}

	case describeLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.content = fmt.Sprintf("Error loading description: %v", msg.err)
		} else {
			v.content = msg.content
		}
		v.viewport.SetContent(v.content)
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
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

// View renders the describe view
func (v *DescribeView) View() string {
	if !v.ready {
		return "Loading..."
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	resourceInfo := fmt.Sprintf("%s/%s", v.resourceType, v.resourceName)
	if v.namespace != "" {
		resourceInfo = fmt.Sprintf("%s/%s", v.namespace, resourceInfo)
	}
	if v.context != "" {
		resourceInfo = fmt.Sprintf("[%s] %s", v.context, resourceInfo)
	}

	header := fmt.Sprintf("📋 Describe: %s", resourceInfo)

	// Footer with controls
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	footer := "↑↓/PgUp/PgDn: Scroll | g/G: Top/Bottom | Esc: Close"

	// Loading indicator
	if v.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("229"))
		return fmt.Sprintf(
			"%s\n%s\n%s",
			headerStyle.Render(header),
			loadingStyle.Render("Loading describe information..."),
			footerStyle.Render(footer),
		)
	}

	return fmt.Sprintf(
		"%s\n%s\n%s",
		headerStyle.Render(header),
		v.viewport.View(),
		footerStyle.Render(footer),
	)
}

// SetSize updates the view size
func (v *DescribeView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height - 3
	v.ready = true
}

// describeLoadedMsg is sent when describe content is loaded
type describeLoadedMsg struct {
	content string
	err     error
}

// FormatResourceType formats the resource type for display
func FormatResourceType(resourceType string) string {
	// Remove trailing 's' for singular form
	if strings.HasSuffix(resourceType, "s") && resourceType != "ingress" {
		return resourceType[:len(resourceType)-1]
	}
	return resourceType
}
