package views

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/kubewatch-tui/internal/core"
	"github.com/user/kubewatch-tui/internal/k8s"
)

// LogView displays logs from pods
type LogView struct {
	viewport viewport.Model
	content  []string
	width    int
	height   int
	ready    bool

	// Log streaming
	ctx        context.Context
	cancelFunc context.CancelFunc
	logReader  io.ReadCloser
}

// NewLogView creates a new log view
func NewLogView() *LogView {
	return &LogView{
		viewport: viewport.New(80, 20),
		content:  []string{},
	}
}

// Init initializes the view
func (v *LogView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *LogView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if !v.ready {
			v.viewport = viewport.New(msg.Width, msg.Height-2)
			v.viewport.YPosition = 0
			v.ready = true
		} else {
			v.viewport.Width = msg.Width
			v.viewport.Height = msg.Height - 2
		}

	case logLineMsg:
		v.content = append(v.content, string(msg))
		if len(v.content) > 1000 {
			v.content = v.content[len(v.content)-1000:]
		}
		v.viewport.SetContent(strings.Join(v.content, "\n"))
		v.viewport.GotoBottom()
		return v, waitForLogLine(v.ctx, v.logReader)
	}

	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the view
func (v *LogView) View() string {
	if !v.ready {
		return "Loading logs..."
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("📜 Logs")

	return fmt.Sprintf("%s\n%s", header, v.viewport.View())
}

// SetSize updates the view size
func (v *LogView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height - 2
	v.ready = true
}

// StartStreaming starts streaming logs for the selected resource
func (v *LogView) StartStreaming(ctx context.Context, client *k8s.Client, state *core.State, selectedResourceName string) tea.Cmd {
	v.ctx, v.cancelFunc = context.WithCancel(ctx)
	v.content = []string{fmt.Sprintf("Starting log stream for %s %s...", state.CurrentResourceType, selectedResourceName)}
	v.viewport.SetContent(strings.Join(v.content, "\n"))

	return func() tea.Msg {
		var reader io.ReadCloser
		var err error

		switch state.CurrentResourceType {
		case core.ResourceTypePod:
			// Find the pod by name
			for _, pod := range state.Pods {
				if pod.Name == selectedResourceName {
					reader, err = client.GetPodLogs(v.ctx, pod.Namespace, pod.Name, "", true, 100)
					break
				}
			}
		case core.ResourceTypeDeployment:
			// Find the deployment by name
			for _, deployment := range state.Deployments {
				if deployment.Name == selectedResourceName {
					// Get pods for deployment and aggregate logs
					pods, err := client.GetPodsForDeployment(v.ctx, deployment.Namespace, deployment.Name)
					if err == nil && len(pods) > 0 {
						// For simplicity, show logs from first pod
						reader, err = client.GetPodLogs(v.ctx, pods[0].Namespace, pods[0].Name, "", true, 100)
					}
					break
				}
			}
		case core.ResourceTypeStatefulSet:
			// Find the statefulset by name
			for _, sts := range state.StatefulSets {
				if sts.Name == selectedResourceName {
					// Get pods for statefulset
					pods, err := client.GetPodsForStatefulSet(v.ctx, sts.Namespace, sts.Name)
					if err == nil && len(pods) > 0 {
						// Show logs from first pod
						reader, err = client.GetPodLogs(v.ctx, pods[0].Namespace, pods[0].Name, "", true, 100)
					}
					break
				}
			}
		}

		if err != nil {
			return errMsg{err}
		}

		if reader != nil {
			v.logReader = reader
			return waitForLogLine(v.ctx, reader)
		}

		return errMsg{fmt.Errorf("no logs available for selected resource")}
	}
}

// StopStreaming stops streaming logs
func (v *LogView) StopStreaming() tea.Cmd {
	if v.cancelFunc != nil {
		v.cancelFunc()
	}
	if v.logReader != nil {
		v.logReader.Close()
	}
	return nil
}

// waitForLogLine waits for the next log line
func waitForLogLine(ctx context.Context, reader io.ReadCloser) tea.Cmd {
	return func() tea.Msg {
		if reader == nil {
			return nil
		}

		scanner := bufio.NewScanner(reader)
		if scanner.Scan() {
			return logLineMsg(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return errMsg{err}
		}

		return nil
	}
}

// Message types
type logLineMsg string
