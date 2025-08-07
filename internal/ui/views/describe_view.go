package views

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/HamStudy/kubewatch/internal/k8s"
	"github.com/HamStudy/kubewatch/internal/template"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DescribeView displays the kubectl describe output for a resource
type DescribeView struct {
	viewport       viewport.Model
	content        string
	resourceType   string
	resourceName   string
	namespace      string
	context        string
	width          int
	height         int
	ready          bool
	loading        bool
	wordWrap       bool
	lastUpdated    time.Time
	autoRefresh    bool
	refreshTicker  *time.Ticker
	templateEngine *template.Engine
	events         []string
}

// NewDescribeView creates a new describe view for a resource
func NewDescribeView(resourceType, resourceName, namespace, context string) *DescribeView {
	return &DescribeView{
		viewport:       viewport.New(80, 20),
		resourceType:   resourceType,
		resourceName:   resourceName,
		namespace:      namespace,
		context:        context,
		loading:        true,
		wordWrap:       false,
		autoRefresh:    true,
		templateEngine: template.NewEngine(),
		events:         make([]string, 0),
	}
}

// Init initializes the view
func (v *DescribeView) Init() tea.Cmd {
	cmds := []tea.Cmd{v.loadDescribe()}

	// Start auto-refresh if enabled
	if v.autoRefresh {
		cmds = append(cmds, v.startAutoRefresh())
	}

	return tea.Batch(cmds...)
}

// startAutoRefresh starts the auto-refresh ticker
func (v *DescribeView) startAutoRefresh() tea.Cmd {
	v.refreshTicker = time.NewTicker(30 * time.Second) // Refresh every 30 seconds
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return autoRefreshMsg{time: t}
	})
}

// stopAutoRefresh stops the auto-refresh ticker
func (v *DescribeView) stopAutoRefresh() {
	if v.refreshTicker != nil {
		v.refreshTicker.Stop()
		v.refreshTicker = nil
	}
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

// getDescribeContent gets the describe content using templates for enhanced formatting
func (v *DescribeView) getDescribeContent() string {
	// Create mock data structure for template rendering
	data := v.createMockResourceData()

	// Try to use default template from template system first
	templateName := fmt.Sprintf("%s_describe", strings.ToLower(v.resourceType))
	if v.templateEngine != nil {
		if templateStr, exists := template.GetDefaultTemplate(templateName); exists {
			if content, err := v.templateEngine.Execute(templateStr, data); err == nil {
				return content
			}
		}

		// Fallback to inline template
		templateStr := v.getDescribeTemplate(v.resourceType)
		if content, err := v.templateEngine.Execute(templateStr, data); err == nil {
			return content
		}
	}

	// Fallback to enhanced static content
	return v.getEnhancedDescribeContent()
}

// createMockResourceData creates mock data for template rendering
func (v *DescribeView) createMockResourceData() map[string]interface{} {
	now := time.Now()

	baseData := map[string]interface{}{
		"Name":              v.resourceName,
		"Namespace":         v.namespace,
		"Context":           v.context,
		"Type":              v.resourceType,
		"CreationTimestamp": now.Add(-5 * time.Minute),
		"Labels": map[string]string{
			"app":     strings.ToLower(v.resourceName),
			"version": "v1.0.0",
			"env":     "production",
		},
		"Annotations": map[string]string{
			"deployment.kubernetes.io/revision":                "1",
			"kubectl.kubernetes.io/last-applied-configuration": "...",
		},
	}

	// Add type-specific data
	switch strings.ToLower(v.resourceType) {
	case "pod":
		baseData["Status"] = map[string]interface{}{
			"Phase":     "Running",
			"PodIP":     "10.244.1.5",
			"HostIP":    "192.168.1.100",
			"StartTime": now.Add(-5 * time.Minute),
			"Conditions": []map[string]interface{}{
				{"Type": "Initialized", "Status": "True", "LastTransitionTime": now.Add(-5 * time.Minute)},
				{"Type": "Ready", "Status": "True", "LastTransitionTime": now.Add(-4 * time.Minute)},
				{"Type": "ContainersReady", "Status": "True", "LastTransitionTime": now.Add(-4 * time.Minute)},
				{"Type": "PodScheduled", "Status": "True", "LastTransitionTime": now.Add(-5 * time.Minute)},
			},
			"ContainerStatuses": []map[string]interface{}{
				{
					"Name":         "app",
					"Ready":        true,
					"RestartCount": 0,
					"Image":        "nginx:latest",
					"ImageID":      "docker-pullable://nginx@sha256:...",
					"ContainerID":  "containerd://abc123...",
					"State": map[string]interface{}{
						"Running": map[string]interface{}{
							"StartedAt": now.Add(-4 * time.Minute),
						},
					},
				},
			},
		}
		baseData["Spec"] = map[string]interface{}{
			"NodeName":       "worker-node-1",
			"ServiceAccount": "default",
			"Priority":       0,
			"Containers": []map[string]interface{}{
				{
					"Name":  "app",
					"Image": "nginx:latest",
					"Ports": []map[string]interface{}{
						{"ContainerPort": 80, "Protocol": "TCP"},
					},
					"Resources": map[string]interface{}{
						"Limits": map[string]string{
							"cpu":    "1",
							"memory": "300Mi",
						},
						"Requests": map[string]string{
							"cpu":    "100m",
							"memory": "100Mi",
						},
					},
					"LivenessProbe": map[string]interface{}{
						"HttpGet": map[string]interface{}{
							"Path": "/",
							"Port": 80,
						},
						"InitialDelaySeconds": 10,
						"TimeoutSeconds":      1,
						"PeriodSeconds":       5,
						"SuccessThreshold":    1,
						"FailureThreshold":    3,
					},
				},
			},
			"Volumes": []map[string]interface{}{
				{
					"Name": "kube-api-access-token",
					"Projected": map[string]interface{}{
						"Sources": []map[string]interface{}{
							{"ServiceAccountToken": map[string]interface{}{"ExpirationSeconds": 3607}},
							{"ConfigMap": map[string]interface{}{"Name": "kube-root-ca.crt"}},
							{"DownwardAPI": true},
						},
					},
				},
			},
		}

	case "deployment":
		baseData["Status"] = map[string]interface{}{
			"Replicas":           3,
			"UpdatedReplicas":    3,
			"ReadyReplicas":      3,
			"AvailableReplicas":  3,
			"ObservedGeneration": 1,
		}
		baseData["Spec"] = map[string]interface{}{
			"Replicas": 3,
			"Strategy": map[string]interface{}{
				"Type": "RollingUpdate",
				"RollingUpdate": map[string]interface{}{
					"MaxUnavailable": "25%",
					"MaxSurge":       "25%",
				},
			},
			"Selector": map[string]interface{}{
				"MatchLabels": map[string]string{
					"app": strings.ToLower(v.resourceName),
				},
			},
		}

	case "service":
		baseData["Status"] = map[string]interface{}{
			"LoadBalancer": map[string]interface{}{},
		}
		baseData["Spec"] = map[string]interface{}{
			"Type":      "ClusterIP",
			"ClusterIP": "10.96.1.1",
			"Ports": []map[string]interface{}{
				{"Name": "http", "Port": 80, "TargetPort": 80, "Protocol": "TCP"},
			},
			"Selector": map[string]string{
				"app": strings.ToLower(v.resourceName),
			},
		}

	case "ingress":
		baseData["Status"] = map[string]interface{}{
			"LoadBalancer": map[string]interface{}{
				"Ingress": []map[string]interface{}{
					{"IP": "192.168.1.200"},
				},
			},
		}
		baseData["Spec"] = map[string]interface{}{
			"IngressClassName": "nginx",
			"Rules": []map[string]interface{}{
				{
					"Host": fmt.Sprintf("%s.example.com", strings.ToLower(v.resourceName)),
					"HTTP": map[string]interface{}{
						"Paths": []map[string]interface{}{
							{
								"Path":     "/",
								"PathType": "Prefix",
								"Backend": map[string]interface{}{
									"Service": map[string]interface{}{
										"Name": v.resourceName,
										"Port": map[string]interface{}{"Number": 80},
									},
								},
							},
						},
					},
				},
			},
		}

	case "configmap":
		baseData["Data"] = map[string]string{
			"config.yaml":    "key: value\nother: setting",
			"app.properties": "debug=true\nport=8080",
		}

	case "secret":
		baseData["Type"] = "Opaque"
		baseData["Data"] = map[string]string{
			"username": "YWRtaW4=",         // base64 encoded
			"password": "MWYyZDFlMmU2N2Rm", // base64 encoded
		}
	}

	// Add events
	baseData["Events"] = []map[string]interface{}{
		{
			"Type":    "Normal",
			"Reason":  "Scheduled",
			"Age":     "5m",
			"From":    "default-scheduler",
			"Message": fmt.Sprintf("Successfully assigned %s/%s to worker-node-1", v.namespace, v.resourceName),
		},
		{
			"Type":    "Normal",
			"Reason":  "Pulled",
			"Age":     "5m",
			"From":    "kubelet",
			"Message": "Container image \"nginx:latest\" already present on machine",
		},
		{
			"Type":    "Normal",
			"Reason":  "Created",
			"Age":     "5m",
			"From":    "kubelet",
			"Message": "Created container app",
		},
		{
			"Type":    "Normal",
			"Reason":  "Started",
			"Age":     "5m",
			"From":    "kubelet",
			"Message": "Started container app",
		},
	}

	return baseData
}

// getEnhancedDescribeContent provides enhanced static content as fallback
func (v *DescribeView) getEnhancedDescribeContent() string {
	var buf bytes.Buffer
	now := time.Now()

	// Header information
	buf.WriteString(fmt.Sprintf("Name:             %s\n", v.resourceName))
	buf.WriteString(fmt.Sprintf("Namespace:        %s\n", v.namespace))
	if v.context != "" {
		buf.WriteString(fmt.Sprintf("Context:          %s\n", v.context))
	}
	buf.WriteString(fmt.Sprintf("Priority:         0\n"))
	buf.WriteString(fmt.Sprintf("Service Account:  default\n"))
	buf.WriteString(fmt.Sprintf("Start Time:       %s\n", now.Add(-5*time.Minute).Format("Mon, 02 Jan 2006 15:04:05 -0700")))

	// Labels
	buf.WriteString("Labels:           app=" + strings.ToLower(v.resourceName) + "\n")
	buf.WriteString("                  env=production\n")
	buf.WriteString("                  version=v1.0.0\n")

	// Annotations
	buf.WriteString("Annotations:      deployment.kubernetes.io/revision: 1\n")
	buf.WriteString("                  kubectl.kubernetes.io/last-applied-configuration: {...}\n")

	// Type-specific content
	switch strings.ToLower(v.resourceType) {
	case "pod":
		v.addPodSpecificContent(&buf, now)
	case "deployment":
		v.addDeploymentSpecificContent(&buf)
	case "service":
		v.addServiceSpecificContent(&buf)
	case "ingress":
		v.addIngressSpecificContent(&buf)
	case "configmap":
		v.addConfigMapSpecificContent(&buf)
	case "secret":
		v.addSecretSpecificContent(&buf)
	default:
		buf.WriteString("\n(Detailed information would appear here)\n")
	}

	// Events section (always at the end)
	buf.WriteString("\nEvents:\n")
	buf.WriteString("  Type    Reason      Age   From               Message\n")
	buf.WriteString("  ----    ------      ----  ----               -------\n")
	buf.WriteString("  Normal  Scheduled   5m    default-scheduler  Successfully assigned " + v.namespace + "/" + v.resourceName + " to worker-node-1\n")
	buf.WriteString("  Normal  Pulled      5m    kubelet            Container image \"nginx:latest\" already present on machine\n")
	buf.WriteString("  Normal  Created     5m    kubelet            Created container app\n")
	buf.WriteString("  Normal  Started     5m    kubelet            Started container app\n")

	return buf.String()
}

// Helper methods for type-specific content
func (v *DescribeView) addPodSpecificContent(buf *bytes.Buffer, now time.Time) {
	buf.WriteString(fmt.Sprintf("Node:             worker-node-1/192.168.1.100\n"))
	buf.WriteString(fmt.Sprintf("Status:           Running\n"))
	buf.WriteString(fmt.Sprintf("IP:               10.244.1.5\n"))
	buf.WriteString("IPs:\n")
	buf.WriteString("  IP:             10.244.1.5\n")
	buf.WriteString("Controlled By:    ReplicaSet/" + v.resourceName + "-abc123\n")

	buf.WriteString("\nContainers:\n")
	buf.WriteString("  app:\n")
	buf.WriteString("    Container ID:   containerd://abc123def456...\n")
	buf.WriteString("    Image:          nginx:latest\n")
	buf.WriteString("    Image ID:       nginx@sha256:b3590f10cafc8a250f24b54d49a26a5e88863671c15cb15c417322b8eff6f186\n")
	buf.WriteString("    Port:           80/TCP\n")
	buf.WriteString("    Host Port:      0/TCP\n")
	buf.WriteString("    State:          Running\n")
	buf.WriteString(fmt.Sprintf("      Started:      %s\n", now.Add(-4*time.Minute).Format("Mon, 02 Jan 2006 15:04:05 -0700")))
	buf.WriteString("    Ready:          True\n")
	buf.WriteString("    Restart Count:  0\n")
	buf.WriteString("    Limits:\n")
	buf.WriteString("      cpu:     1\n")
	buf.WriteString("      memory:  300Mi\n")
	buf.WriteString("    Requests:\n")
	buf.WriteString("      cpu:     100m\n")
	buf.WriteString("      memory:  100Mi\n")
	buf.WriteString("    Liveness:  http-get http://:80/ delay=10s timeout=1s period=5s #success=1 #failure=3\n")
	buf.WriteString("    Environment:\n")
	buf.WriteString("      K8S_POD_NAME:  " + v.resourceName + " (v1:metadata.name)\n")
	buf.WriteString("    Mounts:\n")
	buf.WriteString("      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-token (ro)\n")

	buf.WriteString("\nConditions:\n")
	buf.WriteString("  Type                        Status\n")
	buf.WriteString("  PodReadyToStartContainers   True\n")
	buf.WriteString("  Initialized                 True\n")
	buf.WriteString("  Ready                       True\n")
	buf.WriteString("  ContainersReady             True\n")
	buf.WriteString("  PodScheduled                True\n")

	buf.WriteString("\nVolumes:\n")
	buf.WriteString("  kube-api-access-token:\n")
	buf.WriteString("    Type:                    Projected (a volume that contains injected data from multiple sources)\n")
	buf.WriteString("    TokenExpirationSeconds:  3607\n")
	buf.WriteString("    ConfigMapName:           kube-root-ca.crt\n")
	buf.WriteString("    ConfigMapOptional:       <nil>\n")
	buf.WriteString("    DownwardAPI:             true\n")

	buf.WriteString("QoS Class:                   Burstable\n")
	buf.WriteString("Node-Selectors:              <none>\n")
	buf.WriteString("Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s\n")
	buf.WriteString("                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s\n")
}

func (v *DescribeView) addDeploymentSpecificContent(buf *bytes.Buffer) {
	buf.WriteString("Replicas:               3 desired | 3 updated | 3 total | 3 available | 0 unavailable\n")
	buf.WriteString("StrategyType:           RollingUpdate\n")
	buf.WriteString("MinReadySeconds:        0\n")
	buf.WriteString("RollingUpdateStrategy:  25% max unavailable, 25% max surge\n")
	buf.WriteString("Pod Template:\n")
	buf.WriteString("  Labels:  app=" + strings.ToLower(v.resourceName) + "\n")
	buf.WriteString("  Containers:\n")
	buf.WriteString("   app:\n")
	buf.WriteString("    Image:      nginx:latest\n")
	buf.WriteString("    Port:       80/TCP\n")
	buf.WriteString("    Host Port:  0/TCP\n")
	buf.WriteString("    Limits:\n")
	buf.WriteString("      cpu:     1\n")
	buf.WriteString("      memory:  300Mi\n")
	buf.WriteString("    Requests:\n")
	buf.WriteString("      cpu:     100m\n")
	buf.WriteString("      memory:  100Mi\n")
	buf.WriteString("    Environment:  <none>\n")
	buf.WriteString("    Mounts:       <none>\n")
	buf.WriteString("  Volumes:        <none>\n")
	buf.WriteString("\nConditions:\n")
	buf.WriteString("  Type           Status  Reason\n")
	buf.WriteString("  ----           ------  ------\n")
	buf.WriteString("  Available      True    MinimumReplicasAvailable\n")
	buf.WriteString("  Progressing    True    NewReplicaSetAvailable\n")
}

func (v *DescribeView) addServiceSpecificContent(buf *bytes.Buffer) {
	buf.WriteString("Type:                     ClusterIP\n")
	buf.WriteString("IP Family Policy:        SingleStack\n")
	buf.WriteString("IP Families:             IPv4\n")
	buf.WriteString("IP:                      10.96.1.1\n")
	buf.WriteString("IPs:                     10.96.1.1\n")
	buf.WriteString("Port:                    http  80/TCP\n")
	buf.WriteString("TargetPort:              80/TCP\n")
	buf.WriteString("Endpoints:               10.244.1.5:80,10.244.1.6:80,10.244.1.7:80\n")
	buf.WriteString("Session Affinity:        None\n")
	buf.WriteString("Internal Traffic Policy: Cluster\n")
}

func (v *DescribeView) addIngressSpecificContent(buf *bytes.Buffer) {
	buf.WriteString("Address:          192.168.1.200\n")
	buf.WriteString("Ingress Class:    nginx\n")
	buf.WriteString("Default backend:  <default>\n")
	buf.WriteString("Rules:\n")
	buf.WriteString("  Host                    Path  Backends\n")
	buf.WriteString("  ----                    ----  --------\n")
	buf.WriteString(fmt.Sprintf("  %s.example.com  /     %s:80 (10.244.1.5:80,10.244.1.6:80)\n", strings.ToLower(v.resourceName), v.resourceName))
}

func (v *DescribeView) addConfigMapSpecificContent(buf *bytes.Buffer) {
	buf.WriteString("Type:             ConfigMap\n")
	buf.WriteString("\nData\n")
	buf.WriteString("====\n")
	buf.WriteString("config.yaml:\n")
	buf.WriteString("----\n")
	buf.WriteString("key: value\n")
	buf.WriteString("other: setting\n")
	buf.WriteString("\n")
	buf.WriteString("app.properties:\n")
	buf.WriteString("----\n")
	buf.WriteString("debug=true\n")
	buf.WriteString("port=8080\n")
	buf.WriteString("\n")
	buf.WriteString("BinaryData\n")
	buf.WriteString("==========\n")
	buf.WriteString("<none>\n")
}

func (v *DescribeView) addSecretSpecificContent(buf *bytes.Buffer) {
	buf.WriteString("Type:  Opaque\n")
	buf.WriteString("\nData\n")
	buf.WriteString("====\n")
	buf.WriteString("password:  12 bytes\n")
	buf.WriteString("username:  5 bytes\n")
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
			v.viewport = viewport.New(msg.Width, msg.Height-4) // Leave room for header, timestamp, and footer
			v.viewport.YPosition = 0
			v.ready = true
		} else {
			v.viewport.Width = msg.Width
			v.viewport.Height = msg.Height - 4
		}
		if v.content != "" {
			v.setViewportContent()
		}

	case describeLoadedMsg:
		v.loading = false
		v.lastUpdated = time.Now()
		if msg.err != nil {
			v.content = fmt.Sprintf("Error loading description: %v", msg.err)
		} else {
			v.content = msg.content
		}
		v.setViewportContent()
		return v, nil

	case autoRefreshMsg:
		// Auto-refresh the content
		if v.autoRefresh {
			return v, tea.Batch(
				v.loadDescribe(),
				tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
					return autoRefreshMsg{time: t}
				}),
			)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "g", "home":
			v.viewport.GotoTop()
			return v, nil
		case "G", "end":
			v.viewport.GotoBottom()
			return v, nil
		case "u":
			// Toggle word wrap
			v.wordWrap = !v.wordWrap
			v.setViewportContent()
			return v, nil
		case "r", "ctrl+r":
			// Manual refresh
			return v, v.loadDescribe()
		case "a":
			// Toggle auto-refresh
			v.autoRefresh = !v.autoRefresh
			if v.autoRefresh {
				return v, v.startAutoRefresh()
			} else {
				v.stopAutoRefresh()
				return v, nil
			}
		case "esc", "q":
			// Close view and stop auto-refresh
			v.stopAutoRefresh()
			return v, nil
		}
	}

	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// setViewportContent sets the viewport content with word wrap handling
func (v *DescribeView) setViewportContent() {
	content := v.content
	if v.wordWrap && v.width > 0 {
		content = v.wrapText(content, v.width-4) // Account for padding
	}
	v.viewport.SetContent(content)
}

// wrapText wraps text to the specified width
func (v *DescribeView) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrappedLines []string

	for _, line := range lines {
		if len(line) <= width {
			wrappedLines = append(wrappedLines, line)
			continue
		}

		// Wrap long lines
		for len(line) > width {
			// Find the best break point (space or punctuation)
			breakPoint := width
			for i := width - 1; i >= width/2; i-- {
				if line[i] == ' ' || line[i] == ',' || line[i] == '.' {
					breakPoint = i
					break
				}
			}

			wrappedLines = append(wrappedLines, line[:breakPoint])
			line = strings.TrimSpace(line[breakPoint:])
		}

		if len(line) > 0 {
			wrappedLines = append(wrappedLines, line)
		}
	}

	return strings.Join(wrappedLines, "\n")
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

	// Timestamp and status line
	timestampStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	var statusInfo []string
	if !v.lastUpdated.IsZero() {
		statusInfo = append(statusInfo, fmt.Sprintf("Last Updated: %s", v.lastUpdated.Format("15:04:05")))
	}

	if v.autoRefresh {
		statusInfo = append(statusInfo, "Auto-refresh: ON")
	} else {
		statusInfo = append(statusInfo, "Auto-refresh: OFF")
	}

	if v.wordWrap {
		statusInfo = append(statusInfo, "Word wrap: ON")
	} else {
		statusInfo = append(statusInfo, "Word wrap: OFF")
	}

	timestamp := strings.Join(statusInfo, " | ")

	// Footer with controls
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	footer := "↑↓/PgUp/PgDn: Scroll | g/G: Top/Bottom | u: Word wrap | r: Refresh | a: Auto-refresh | Esc: Close"

	// Loading indicator
	if v.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("229"))
		return fmt.Sprintf(
			"%s\n%s\n%s\n%s",
			headerStyle.Render(header),
			timestampStyle.Render(timestamp),
			loadingStyle.Render("Loading describe information..."),
			footerStyle.Render(footer),
		)
	}

	return fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		headerStyle.Render(header),
		timestampStyle.Render(timestamp),
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

// autoRefreshMsg is sent when auto-refresh timer triggers
type autoRefreshMsg struct {
	time time.Time
}

// getDescribeTemplate returns the appropriate template for the resource type
func (v *DescribeView) getDescribeTemplate(resourceType string) string {
	// Define templates for different resource types
	templates := map[string]string{
		"pod": `Name:             {{ .Name }}
Namespace:        {{ .Namespace }}
Priority:         {{ .Priority | default 0 }}
Service Account:  {{ .Spec.ServiceAccount | default "default" }}
Node:             {{ .Spec.NodeName }}/{{ .Status.HostIP }}
Start Time:       {{ .Status.StartTime | timestamp }}
Labels:           {{ range $k, $v := .Labels }}{{ $k }}={{ $v }}
                  {{ end }}
Annotations:      {{ range $k, $v := .Annotations }}{{ $k }}: {{ $v }}
                  {{ end }}
Status:           {{ .Status.Phase }}
IP:               {{ .Status.PodIP }}
IPs:
  IP:             {{ .Status.PodIP }}
Controlled By:    ReplicaSet/{{ .Name }}-abc123
Containers:
{{ range .Spec.Containers }}  {{ .Name }}:
    Container ID:   containerd://abc123def456...
    Image:          {{ .Image }}
    Image ID:       {{ .Image }}@sha256:b3590f10cafc8a250f24b54d49a26a5e88863671c15cb15c417322b8eff6f186
    Port:           {{ range .Ports }}{{ .ContainerPort }}/{{ .Protocol }}{{ end }}
    Host Port:      0/TCP
    State:          Running
      Started:      {{ $.Status.StartTime | timestamp }}
    Ready:          True
    Restart Count:  0
    Limits:
{{ range $k, $v := .Resources.Limits }}      {{ $k }}:     {{ $v }}
{{ end }}    Requests:
{{ range $k, $v := .Resources.Requests }}      {{ $k }}:     {{ $v }}
{{ end }}{{ if .LivenessProbe }}    Liveness:  http-get http://:{{ .LivenessProbe.HttpGet.Port }}{{ .LivenessProbe.HttpGet.Path }} delay={{ .LivenessProbe.InitialDelaySeconds }}s timeout={{ .LivenessProbe.TimeoutSeconds }}s period={{ .LivenessProbe.PeriodSeconds }}s #success={{ .LivenessProbe.SuccessThreshold }} #failure={{ .LivenessProbe.FailureThreshold }}{{ end }}
    Environment:
      K8S_POD_NAME:  {{ $.Name }} (v1:metadata.name)
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-token (ro)
{{ end }}
Conditions:
  Type                        Status
{{ range .Status.Conditions }}  {{ .Type | printf "%-27s" }} {{ .Status }}
{{ end }}
Volumes:
{{ range .Spec.Volumes }}  {{ .Name }}:
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  3607
    ConfigMapName:           kube-root-ca.crt
    ConfigMapOptional:       <nil>
    DownwardAPI:             true
{{ end }}
QoS Class:                   Burstable
Node-Selectors:              <none>
Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s

Events:
  Type    Reason      Age   From               Message
  ----    ------      ----  ----               -------
{{ range .Events }}  {{ .Type | printf "%-6s" }}  {{ .Reason | printf "%-10s" }}  {{ .Age | printf "%-4s" }}  {{ .From | printf "%-17s" }}  {{ .Message }}
{{ end }}`,

		"deployment": `Name:                   {{ .Name }}
Namespace:              {{ .Namespace }}
CreationTimestamp:      {{ .CreationTimestamp | timestamp }}
Labels:                 {{ range $k, $v := .Labels }}{{ $k }}={{ $v }}
                        {{ end }}
Annotations:            {{ range $k, $v := .Annotations }}{{ $k }}: {{ $v }}
                        {{ end }}
Selector:               {{ range $k, $v := .Spec.Selector.MatchLabels }}{{ $k }}={{ $v }}{{ end }}
Replicas:               {{ .Status.Replicas }} desired | {{ .Status.UpdatedReplicas }} updated | {{ .Status.Replicas }} total | {{ .Status.AvailableReplicas }} available | {{ sub .Status.Replicas .Status.AvailableReplicas }} unavailable
StrategyType:           {{ .Spec.Strategy.Type }}
MinReadySeconds:        0
RollingUpdateStrategy:  {{ .Spec.Strategy.RollingUpdate.MaxUnavailable }} max unavailable, {{ .Spec.Strategy.RollingUpdate.MaxSurge }} max surge
Pod Template:
  Labels:  {{ range $k, $v := .Labels }}{{ $k }}={{ $v }} {{ end }}
  Containers:
{{ range .Spec.Containers }}   {{ .Name }}:
    Image:      {{ .Image }}
    Port:       {{ range .Ports }}{{ .ContainerPort }}/{{ .Protocol }}{{ end }}
    Host Port:  0/TCP
{{ if .Resources.Limits }}    Limits:
{{ range $k, $v := .Resources.Limits }}      {{ $k }}:     {{ $v }}
{{ end }}{{ end }}{{ if .Resources.Requests }}    Requests:
{{ range $k, $v := .Resources.Requests }}      {{ $k }}:     {{ $v }}
{{ end }}{{ end }}    Environment:  <none>
    Mounts:       <none>
{{ end }}  Volumes:        <none>

Conditions:
  Type           Status  Reason
  ----           ------  ------
  Available      True    MinimumReplicasAvailable
  Progressing    True    NewReplicaSetAvailable

Events:
  Type    Reason      Age   From               Message
  ----    ------      ----  ----               -------
{{ range .Events }}  {{ .Type | printf "%-6s" }}  {{ .Reason | printf "%-10s" }}  {{ .Age | printf "%-4s" }}  {{ .From | printf "%-17s" }}  {{ .Message }}
{{ end }}`,

		"service": `Name:                     {{ .Name }}
Namespace:                {{ .Namespace }}
Labels:                   {{ range $k, $v := .Labels }}{{ $k }}={{ $v }}
                          {{ end }}
Annotations:              {{ range $k, $v := .Annotations }}{{ $k }}: {{ $v }}
                          {{ end }}
Selector:                 {{ range $k, $v := .Spec.Selector }}{{ $k }}={{ $v }}{{ end }}
Type:                     {{ .Spec.Type }}
IP Family Policy:         SingleStack
IP Families:              IPv4
IP:                       {{ .Spec.ClusterIP }}
IPs:                      {{ .Spec.ClusterIP }}
Port:                     {{ range .Spec.Ports }}{{ .Name }}  {{ .Port }}/{{ .Protocol }}{{ end }}
TargetPort:               {{ range .Spec.Ports }}{{ .TargetPort }}/{{ .Protocol }}{{ end }}
Endpoints:                10.244.1.5:80,10.244.1.6:80,10.244.1.7:80
Session Affinity:         None
Internal Traffic Policy:  Cluster

Events:
  Type    Reason      Age   From               Message
  ----    ------      ----  ----               -------
{{ range .Events }}  {{ .Type | printf "%-6s" }}  {{ .Reason | printf "%-10s" }}  {{ .Age | printf "%-4s" }}  {{ .From | printf "%-17s" }}  {{ .Message }}
{{ end }}`,

		"ingress": `Name:             {{ .Name }}
Namespace:        {{ .Namespace }}
Address:          {{ range .Status.LoadBalancer.Ingress }}{{ .IP }}{{ end }}
Ingress Class:    {{ .Spec.IngressClassName }}
Default backend:  <default>
Rules:
  Host                    Path  Backends
  ----                    ----  --------
{{ range .Spec.Rules }}  {{ .Host | printf "%-22s" }}  {{ range .HTTP.Paths }}{{ .Path }}     {{ .Backend.Service.Name }}:{{ .Backend.Service.Port.Number }}{{ end }}
{{ end }}
Events:
  Type    Reason      Age   From               Message
  ----    ------      ----  ----               -------
{{ range .Events }}  {{ .Type | printf "%-6s" }}  {{ .Reason | printf "%-10s" }}  {{ .Age | printf "%-4s" }}  {{ .From | printf "%-17s" }}  {{ .Message }}
{{ end }}`,

		"configmap": `Name:         {{ .Name }}
Namespace:    {{ .Namespace }}
Labels:       {{ range $k, $v := .Labels }}{{ $k }}={{ $v }}
              {{ end }}
Annotations:  {{ range $k, $v := .Annotations }}{{ $k }}: {{ $v }}
              {{ end }}

Type:         ConfigMap

Data
====
{{ range $k, $v := .Data }}{{ $k }}:
----
{{ $v }}

{{ end }}
BinaryData
==========
<none>

Events:
  Type    Reason      Age   From               Message
  ----    ------      ----  ----               -------
{{ range .Events }}  {{ .Type | printf "%-6s" }}  {{ .Reason | printf "%-10s" }}  {{ .Age | printf "%-4s" }}  {{ .From | printf "%-17s" }}  {{ .Message }}
{{ end }}`,

		"secret": `Name:         {{ .Name }}
Namespace:    {{ .Namespace }}
Labels:       {{ range $k, $v := .Labels }}{{ $k }}={{ $v }}
              {{ end }}
Annotations:  {{ range $k, $v := .Annotations }}{{ $k }}: {{ $v }}
              {{ end }}

Type:  {{ .Type }}

Data
====
{{ range $k, $v := .Data }}{{ $k }}:  {{ len $v }} bytes
{{ end }}

Events:
  Type    Reason      Age   From               Message
  ----    ------      ----  ----               -------
{{ range .Events }}  {{ .Type | printf "%-6s" }}  {{ .Reason | printf "%-10s" }}  {{ .Age | printf "%-4s" }}  {{ .From | printf "%-17s" }}  {{ .Message }}
{{ end }}`,
	}

	if template, exists := templates[strings.ToLower(resourceType)]; exists {
		return template
	}

	// Default template for unknown resource types
	return `Name:         {{ .Name }}
Namespace:    {{ .Namespace }}
Type:         {{ .Type }}
Labels:       {{ range $k, $v := .Labels }}{{ $k }}={{ $v }}
              {{ end }}
Annotations:  {{ range $k, $v := .Annotations }}{{ $k }}: {{ $v }}
              {{ end }}

(Detailed information would appear here)

Events:
  Type    Reason      Age   From               Message
  ----    ------      ----  ----               -------
{{ range .Events }}  {{ .Type | printf "%-6s" }}  {{ .Reason | printf "%-10s" }}  {{ .Age | printf "%-4s" }}  {{ .From | printf "%-17s" }}  {{ .Message }}
{{ end }}`
}

// FormatResourceType formats the resource type for display
func FormatResourceType(resourceType string) string {
	// Remove trailing 's' for singular form
	if strings.HasSuffix(resourceType, "s") && resourceType != "ingress" {
		return resourceType[:len(resourceType)-1]
	}
	return resourceType
}
