package views

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

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
	logReaders []io.ReadCloser  // Multiple readers for multiple containers
	scanners   []*bufio.Scanner // Multiple scanners
	containers []string         // Container names
	following  bool             // Auto-scroll to bottom
	tailing    bool             // Keep reading new logs (always true while streaming)
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
	case tea.KeyMsg:
		switch msg.String() {
		case "f":
			// Toggle follow mode
			v.following = !v.following
			if v.following {
				v.viewport.GotoBottom()
			}
			return v, nil
		case "g", "home":
			v.following = false
			v.viewport.GotoTop()
			return v, nil
		case "G", "end":
			v.following = true
			v.viewport.GotoBottom()
			return v, nil
		}

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if !v.ready {
			v.viewport = viewport.New(msg.Width, msg.Height-3) // Extra line for status
			v.viewport.YPosition = 0
			v.ready = true
		} else {
			v.viewport.Width = msg.Width
			v.viewport.Height = msg.Height - 3
		}

	case logStreamStartedMsg:
		// Stream has been initialized, start reading from all containers
		var cmds []tea.Cmd
		for i := range v.scanners {
			cmds = append(cmds, v.readNextLine(i))
		}
		return v, tea.Batch(cmds...)

	case logLineMsg:
		// Format line with container prefix if multiple containers
		line := msg.line
		if len(v.containers) > 1 {
			line = fmt.Sprintf("[%s] %s", msg.container, msg.line)
		}
		v.content = append(v.content, line)
		if len(v.content) > 10000 {
			// Keep last 10000 lines
			v.content = v.content[len(v.content)-10000:]
		}
		v.viewport.SetContent(strings.Join(v.content, "\n"))
		if v.following {
			v.viewport.GotoBottom()
		}
		// Continue reading from the container that sent this message
		for i, container := range v.containers {
			if container == msg.container {
				return v, v.readNextLine(i)
			}
		}
		return v, nil

	case errMsg:
		// Display error in the log view
		v.content = append(v.content, fmt.Sprintf("Error: %v", msg.err))
		v.viewport.SetContent(strings.Join(v.content, "\n"))
		return v, nil
	}

	// Check if user scrolled manually (disable auto-follow)
	oldY := v.viewport.YOffset
	v.viewport, cmd = v.viewport.Update(msg)
	if oldY != v.viewport.YOffset && v.viewport.YOffset < v.viewport.TotalLineCount()-v.viewport.Height {
		v.following = false
	}

	return v, cmd
}

// View renders the view
func (v *LogView) View() string {
	if !v.ready {
		return "Loading logs..."
	}

	// Build header with status
	followStatus := "FOLLOWING"
	if !v.following {
		followStatus = "SCROLLING (logs still streaming)"
	}

	containerInfo := ""
	if len(v.containers) > 1 {
		containerInfo = fmt.Sprintf(" | %d containers", len(v.containers))
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render(fmt.Sprintf("📜 Logs [%s]%s", followStatus, containerInfo))

	// Build status line
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	status := statusStyle.Render(fmt.Sprintf(
		"Lines: %d | Position: %d/%d | Keys: ↑↓/PgUp/PgDn/Home/End | f: toggle follow | Esc: back",
		len(v.content),
		v.viewport.YOffset+1,
		v.viewport.TotalLineCount(),
	))

	return fmt.Sprintf("%s\n%s\n%s", header, v.viewport.View(), status)
}

// SetSize updates the view size
func (v *LogView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height - 3 // Account for header and status line
	v.ready = true
}

// StartStreaming starts streaming logs for the selected resource
func (v *LogView) StartStreaming(ctx context.Context, client *k8s.Client, state *core.State, selectedResourceName string) tea.Cmd {
	v.ctx, v.cancelFunc = context.WithCancel(ctx)
	v.content = []string{}
	v.following = true // Start with auto-follow enabled
	v.tailing = true   // Always tail while streaming
	v.viewport.SetContent("Loading logs...")

	// Reset readers and scanners
	v.logReaders = []io.ReadCloser{}
	v.scanners = []*bufio.Scanner{}
	v.containers = []string{}

	return func() tea.Msg {
		var readers []io.ReadCloser
		var containerNames []string
		var err error

		switch state.CurrentResourceType {
		case core.ResourceTypePod:
			// Find the pod by name
			for _, pod := range state.Pods {
				if pod.Name == selectedResourceName {
					// Stream logs from ALL containers
					for _, container := range pod.Spec.Containers {
						reader, err := client.GetPodLogs(v.ctx, pod.Namespace, pod.Name, container.Name, true, 100)
						if err != nil {
							v.content = append(v.content, fmt.Sprintf("[%s] Error: %v", container.Name, err))
							continue
						}
						readers = append(readers, reader)
						containerNames = append(containerNames, container.Name)
					}
					if len(pod.Spec.Containers) > 1 {
						v.content = append(v.content, fmt.Sprintf("=== Streaming logs from %d containers: %v ===", len(containerNames), containerNames))
					}
					break
				}
			}
		case core.ResourceTypeDeployment:
			// Find the deployment by name
			for _, deployment := range state.Deployments {
				if deployment.Name == selectedResourceName {
					// Get pods for deployment
					pods, err := client.GetPodsForDeployment(v.ctx, deployment.Namespace, deployment.Name)
					if err == nil && len(pods) > 0 {
						// Stream from all containers of first pod
						for _, container := range pods[0].Spec.Containers {
							reader, err := client.GetPodLogs(v.ctx, pods[0].Namespace, pods[0].Name, container.Name, true, 100)
							if err != nil {
								v.content = append(v.content, fmt.Sprintf("[%s] Error: %v", container.Name, err))
								continue
							}
							readers = append(readers, reader)
							containerNames = append(containerNames, container.Name)
						}
						if len(pods[0].Spec.Containers) > 1 {
							v.content = append(v.content, fmt.Sprintf("=== Streaming logs from %d containers in pod %s: %v ===", len(containerNames), pods[0].Name, containerNames))
						}
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
						// Stream from all containers of first pod
						for _, container := range pods[0].Spec.Containers {
							reader, err := client.GetPodLogs(v.ctx, pods[0].Namespace, pods[0].Name, container.Name, true, 100)
							if err != nil {
								v.content = append(v.content, fmt.Sprintf("[%s] Error: %v", container.Name, err))
								continue
							}
							readers = append(readers, reader)
							containerNames = append(containerNames, container.Name)
						}
						if len(pods[0].Spec.Containers) > 1 {
							v.content = append(v.content, fmt.Sprintf("=== Streaming logs from %d containers in pod %s: %v ===", len(containerNames), pods[0].Name, containerNames))
						}
					}
					break
				}
			}
		}

		if err != nil && len(readers) == 0 {
			return errMsg{err}
		}

		if len(readers) > 0 {
			v.logReaders = readers
			v.containers = containerNames
			// Create scanners for each reader
			for _, reader := range readers {
				v.scanners = append(v.scanners, bufio.NewScanner(reader))
			}
			// Return a message to trigger the first read
			return logStreamStartedMsg{containerCount: len(readers)}
		}

		return errMsg{fmt.Errorf("no logs available for selected resource")}
	}
}

// StopStreaming stops streaming logs
func (v *LogView) StopStreaming() tea.Cmd {
	if v.cancelFunc != nil {
		v.cancelFunc()
	}
	// Close all readers
	for _, reader := range v.logReaders {
		if reader != nil {
			reader.Close()
		}
	}
	v.logReaders = nil
	v.scanners = nil
	v.tailing = false
	return nil
}

// readNextLine reads the next line from a specific container's log stream
func (v *LogView) readNextLine(containerIndex int) tea.Cmd {
	return func() tea.Msg {
		if containerIndex >= len(v.scanners) || containerIndex >= len(v.containers) {
			return nil
		}

		scanner := v.scanners[containerIndex]
		containerName := v.containers[containerIndex]

		if scanner == nil {
			return nil
		}

		// Check if context is cancelled
		select {
		case <-v.ctx.Done():
			return nil
		default:
		}

		// Read in a goroutine to avoid blocking
		lineChan := make(chan string, 1)
		errChan := make(chan error, 1)

		go func() {
			if scanner.Scan() {
				lineChan <- scanner.Text()
			} else if err := scanner.Err(); err != nil {
				errChan <- err
			} else {
				// EOF or stream closed
				errChan <- io.EOF
			}
		}()

		// Wait for result with timeout
		select {
		case <-v.ctx.Done():
			return nil
		case line := <-lineChan:
			return logLineMsg{container: containerName, line: line}
		case err := <-errChan:
			if err == io.EOF {
				// Stream ended, this shouldn't happen with follow=true
				// but can happen if pod terminates
				return logLineMsg{container: containerName, line: "--- End of logs (pod may have terminated) ---"}
			}
			return errMsg{err}
		case <-time.After(100 * time.Millisecond):
			// No data yet, check again
			return v.readNextLine(containerIndex)()
		}
	}
}

// Message types
type logLineMsg struct {
	container string
	line      string
}
type logStreamStartedMsg struct {
	containerCount int
}
