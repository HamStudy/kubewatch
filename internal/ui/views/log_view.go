package views

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/k8s"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"
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

	// Search functionality
	searchMode    bool
	searchQuery   string
	searchResults []int // Line indices that match search
	currentMatch  int   // Current match index

	// Stream control
	showStdout        bool
	showStderr        bool
	selectedContainer int // -1 for all, 0+ for specific container

	// For deployments
	pods        []string
	selectedPod int // -1 for all, 0+ for specific pod

	// For restarting streams
	client       *k8s.Client
	state        *core.State
	resourceName string
	needsRestart bool
}

// NewLogView creates a new log view
func NewLogView() *LogView {
	return &LogView{
		viewport:          viewport.New(80, 20),
		content:           []string{},
		showStdout:        true,
		showStderr:        true,
		selectedContainer: -1, // Show all containers by default
		selectedPod:       -1, // Show all pods by default
		searchResults:     []int{},
	}
}

// IsSearchMode returns true if the log view is in search mode
func (v *LogView) IsSearchMode() bool {
	return v.searchMode
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
		// Handle search mode input
		if v.searchMode {
			switch msg.String() {
			case "enter":
				v.searchMode = false
				v.performSearch()
				return v, nil
			case "esc":
				// Cancel search mode
				v.searchMode = false
				v.searchQuery = ""
				v.searchResults = []int{}
				return v, nil
			case "backspace":
				if len(v.searchQuery) > 0 {
					v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
				}
				return v, nil
			default:
				if len(msg.String()) == 1 {
					v.searchQuery += msg.String()
				}
				return v, nil
			}
		}

		// In normal mode, don't handle ESC - let the app handle it
		if msg.String() == "esc" {
			// Let the parent app handle ESC to close the log view
			return v, nil
		}

		// Normal mode key handling
		switch msg.String() {
		case "/":
			// Start search mode
			v.searchMode = true
			v.searchQuery = ""
			return v, nil
		case "n":
			// Next search result
			if len(v.searchResults) > 0 {
				v.currentMatch = (v.currentMatch + 1) % len(v.searchResults)
				v.jumpToMatch()
			}
			return v, nil
		case "N":
			// Previous search result
			if len(v.searchResults) > 0 {
				v.currentMatch--
				if v.currentMatch < 0 {
					v.currentMatch = len(v.searchResults) - 1
				}
				v.jumpToMatch()
			}
			return v, nil
		case "f":
			// Toggle follow mode
			v.following = !v.following
			if v.following {
				v.viewport.GotoBottom()
			}
			return v, nil
		case "c":
			// Cycle through containers
			if len(v.containers) > 1 {
				v.selectedContainer++
				if v.selectedContainer >= len(v.containers) {
					v.selectedContainer = -1 // Back to all
				}
				// Restart streaming with selected container
				return v, v.restartStreaming()
			}
			return v, nil
		case "p":
			// Cycle through pods (for deployments)
			if len(v.pods) > 1 {
				v.selectedPod++
				if v.selectedPod >= len(v.pods) {
					v.selectedPod = -1 // Back to all
				}
				// Restart streaming with selected pod
				return v, v.restartStreaming()
			}
			return v, nil
		case "s":
			// Toggle stdout/stderr (Note: K8s API doesn't separate these streams)
			// This is kept for future implementation if we add log parsing
			if v.showStdout && v.showStderr {
				v.showStdout = true
				v.showStderr = false
			} else if v.showStdout && !v.showStderr {
				v.showStdout = false
				v.showStderr = true
			} else {
				v.showStdout = true
				v.showStderr = true
			}
			// Note: Kubernetes API combines stdout/stderr, so this doesn't actually filter
			// Would need to implement log parsing to detect stderr prefixes
			v.content = append(v.content, "Note: Kubernetes combines stdout/stderr streams - filtering not available")
			return v, nil
		case "C":
			// Clear log buffer
			v.content = []string{}
			v.viewport.SetContent("")
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
		followStatus = "SCROLLING"
	}

	// Container/Pod info
	streamInfo := ""
	if v.selectedContainer >= 0 && v.selectedContainer < len(v.containers) {
		streamInfo = fmt.Sprintf(" | Container: %s", v.containers[v.selectedContainer])
	} else if len(v.containers) > 1 {
		streamInfo = fmt.Sprintf(" | All %d containers", len(v.containers))
	}

	if v.selectedPod >= 0 && v.selectedPod < len(v.pods) {
		streamInfo += fmt.Sprintf(" | Pod: %s", v.pods[v.selectedPod])
	} else if len(v.pods) > 1 {
		streamInfo += fmt.Sprintf(" | All %d pods", len(v.pods))
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render(fmt.Sprintf("📜 Logs [%s]%s", followStatus, streamInfo))
	// Build status line
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	statusText := ""
	if v.searchMode {
		// Show search input
		searchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229"))
		statusText = searchStyle.Render(fmt.Sprintf("Search: %s_", v.searchQuery))
	} else if len(v.searchResults) > 0 {
		// Show search results
		statusText = fmt.Sprintf("Match %d/%d | n: next | N: prev | /: new search",
			v.currentMatch+1, len(v.searchResults))
	} else {
		// Normal status
		statusText = fmt.Sprintf(
			"Lines: %d | Pos: %d/%d | /: search | c: containers | p: pods | f: follow | ?: help",
			len(v.content),
			v.viewport.YOffset+1,
			v.viewport.TotalLineCount(),
		)
	}

	status := statusStyle.Render(statusText)

	// Apply search highlighting to the content if we have search results
	var viewportContent string
	if len(v.searchResults) > 0 && v.searchQuery != "" {
		viewportContent = v.getHighlightedContent()
	} else {
		viewportContent = v.viewport.View()
	}

	return fmt.Sprintf("%s\n%s\n%s", header, viewportContent, status)
}

// SetSize updates the view size
func (v *LogView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height - 3 // Account for header and status line
	v.ready = true
}

// performSearch searches for the query in the log content
func (v *LogView) performSearch() {
	v.searchResults = []int{}
	if v.searchQuery == "" {
		// Clear highlighting by resetting the viewport content
		v.viewport.SetContent(strings.Join(v.content, "\n"))
		return
	}

	query := strings.ToLower(v.searchQuery)
	for i, line := range v.content {
		if strings.Contains(strings.ToLower(line), query) {
			v.searchResults = append(v.searchResults, i)
		}
	}

	if len(v.searchResults) > 0 {
		v.currentMatch = 0
		v.jumpToMatch()
	}

	// The highlighting will be applied in the View() method
}

// jumpToMatch jumps to the current search match
func (v *LogView) jumpToMatch() {
	if v.currentMatch >= 0 && v.currentMatch < len(v.searchResults) {
		lineIndex := v.searchResults[v.currentMatch]
		// Calculate the position to jump to (center the match if possible)
		v.viewport.YOffset = lineIndex - v.viewport.Height/2

		// Ensure YOffset stays within valid bounds
		if v.viewport.YOffset < 0 {
			v.viewport.YOffset = 0
		}

		maxOffset := v.viewport.TotalLineCount() - v.viewport.Height
		if maxOffset < 0 {
			maxOffset = 0
		}
		if v.viewport.YOffset > maxOffset {
			v.viewport.YOffset = maxOffset
		}

		v.following = false // Disable following when jumping to search result
	}
}

// getHighlightedContent returns the viewport content with search matches highlighted
func (v *LogView) getHighlightedContent() string {
	// Get the visible lines from the viewport
	startLine := v.viewport.YOffset
	if startLine < 0 {
		startLine = 0
	}

	endLine := startLine + v.viewport.Height
	if endLine > len(v.content) {
		endLine = len(v.content)
	}

	// Style for highlighting matches
	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("226")). // Yellow background
		Foreground(lipgloss.Color("0"))    // Black text

	// Style for current match (different color)
	currentHighlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("202")). // Orange background
		Foreground(lipgloss.Color("15"))   // White text

	var result []string
	query := strings.ToLower(v.searchQuery)

	for i := startLine; i < endLine; i++ {
		if i < 0 || i >= len(v.content) {
			result = append(result, "")
			continue
		}

		line := v.content[i]
		lowerLine := strings.ToLower(line)

		// Check if this line contains the search query
		if strings.Contains(lowerLine, query) {
			// Check if this is the current match line
			isCurrentMatch := false
			if v.currentMatch >= 0 && v.currentMatch < len(v.searchResults) {
				isCurrentMatch = (i == v.searchResults[v.currentMatch])
			}

			// Highlight all occurrences in the line
			highlightedLine := ""
			lastEnd := 0

			for {
				idx := strings.Index(lowerLine[lastEnd:], query)
				if idx == -1 {
					// No more matches, append the rest
					highlightedLine += line[lastEnd:]
					break
				}

				// Add the part before the match
				actualIdx := lastEnd + idx
				highlightedLine += line[lastEnd:actualIdx]

				// Add the highlighted match
				matchText := line[actualIdx : actualIdx+len(v.searchQuery)]
				if isCurrentMatch {
					highlightedLine += currentHighlightStyle.Render(matchText)
				} else {
					highlightedLine += highlightStyle.Render(matchText)
				}

				lastEnd = actualIdx + len(v.searchQuery)
			}

			result = append(result, highlightedLine)
		} else {
			result = append(result, line)
		}
	}

	// Pad with empty lines if needed
	for len(result) < v.viewport.Height {
		result = append(result, "")
	}

	return strings.Join(result, "\n")
}

// StartStreaming starts streaming logs for the selected resource
func (v *LogView) StartStreaming(ctx context.Context, client *k8s.Client, state *core.State, selectedResourceName string) tea.Cmd {
	// Store for restarting
	v.client = client
	v.state = state
	v.resourceName = selectedResourceName

	v.ctx, v.cancelFunc = context.WithCancel(ctx)
	v.content = []string{}
	v.following = true // Start with auto-follow enabled
	v.tailing = true   // Always tail while streaming
	v.viewport.SetContent("Loading logs...")

	// Reset readers and scanners
	v.logReaders = []io.ReadCloser{}
	v.scanners = []*bufio.Scanner{}
	v.containers = []string{}

	// Reset pod list for new resource
	v.pods = []string{}

	return func() tea.Msg {
		var readers []io.ReadCloser
		var containerNames []string
		var err error

		switch state.CurrentResourceType {
		case core.ResourceTypePod:
			// Find the pod by name
			for _, pod := range state.Pods {
				if pod.Name == selectedResourceName {
					// Build list of all container names for selection
					allContainers := []string{}
					for _, container := range pod.Spec.Containers {
						allContainers = append(allContainers, container.Name)
					}
					v.containers = allContainers

					// Determine which containers to stream
					containersToStream := []string{}
					if v.selectedContainer >= 0 && v.selectedContainer < len(allContainers) {
						// Stream only selected container
						containersToStream = []string{allContainers[v.selectedContainer]}
					} else {
						// Stream all containers
						containersToStream = allContainers
					}

					// Stream logs from selected containers
					for _, containerName := range containersToStream {
						reader, err := client.GetPodLogs(v.ctx, pod.Namespace, pod.Name, containerName, true, 100)
						if err != nil {
							v.content = append(v.content, fmt.Sprintf("[%s] Error: %v", containerName, err))
							continue
						}
						readers = append(readers, reader)
						containerNames = append(containerNames, containerName)
					}

					// Show status message
					if v.selectedContainer >= 0 {
						v.content = append(v.content, fmt.Sprintf("=== Streaming logs from container: %s ===", containersToStream[0]))
					} else if len(pod.Spec.Containers) > 1 {
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
						// Store pod names for cycling
						v.pods = []string{}
						allPodNames := []string{}
						for _, pod := range pods {
							allPodNames = append(allPodNames, pod.Name)
						}
						v.pods = allPodNames

						// Build container list from first pod (assume all pods have same containers)
						if len(pods) > 0 {
							allContainers := []string{}
							for _, container := range pods[0].Spec.Containers {
								allContainers = append(allContainers, container.Name)
							}
							v.containers = allContainers
						}

						// Determine which pods to stream
						podsToStream := pods
						if v.selectedPod >= 0 && v.selectedPod < len(pods) {
							// Stream only selected pod
							podsToStream = []v1.Pod{pods[v.selectedPod]}
						}

						// Determine which containers to stream
						containersToStream := v.containers
						if v.selectedContainer >= 0 && v.selectedContainer < len(v.containers) {
							// Stream only selected container
							containersToStream = []string{v.containers[v.selectedContainer]}
						}

						// Stream from selected pods and containers
						for _, pod := range podsToStream {
							for _, containerName := range containersToStream {
								reader, err := client.GetPodLogs(v.ctx, pod.Namespace, pod.Name, containerName, true, 100)
								if err != nil {
									v.content = append(v.content, fmt.Sprintf("[%s/%s] Error: %v", pod.Name, containerName, err))
									continue
								}
								readers = append(readers, reader)
								// Include pod name in container identifier for deployments
								containerNames = append(containerNames, fmt.Sprintf("%s/%s", pod.Name, containerName))
							}
						}

						// Show status message
						statusMsg := ""
						if v.selectedPod >= 0 {
							statusMsg = fmt.Sprintf("Pod: %s", allPodNames[v.selectedPod])
						} else {
							statusMsg = fmt.Sprintf("%d pods", len(podsToStream))
						}
						if v.selectedContainer >= 0 {
							statusMsg += fmt.Sprintf(", Container: %s", v.containers[v.selectedContainer])
						} else {
							statusMsg += fmt.Sprintf(", %d containers", len(containersToStream))
						}
						v.content = append(v.content, fmt.Sprintf("=== Streaming logs: %s ===", statusMsg))
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
						// Store pod names for cycling
						v.pods = []string{}
						allPodNames := []string{}
						for _, pod := range pods {
							allPodNames = append(allPodNames, pod.Name)
						}
						v.pods = allPodNames

						// Build container list from first pod
						if len(pods) > 0 {
							allContainers := []string{}
							for _, container := range pods[0].Spec.Containers {
								allContainers = append(allContainers, container.Name)
							}
							v.containers = allContainers
						}

						// Determine which pods to stream
						podsToStream := pods
						if v.selectedPod >= 0 && v.selectedPod < len(pods) {
							// Stream only selected pod
							podsToStream = []v1.Pod{pods[v.selectedPod]}
						}

						// Determine which containers to stream
						containersToStream := v.containers
						if v.selectedContainer >= 0 && v.selectedContainer < len(v.containers) {
							// Stream only selected container
							containersToStream = []string{v.containers[v.selectedContainer]}
						}

						// Stream from selected pods and containers
						for _, pod := range podsToStream {
							for _, containerName := range containersToStream {
								reader, err := client.GetPodLogs(v.ctx, pod.Namespace, pod.Name, containerName, true, 100)
								if err != nil {
									v.content = append(v.content, fmt.Sprintf("[%s/%s] Error: %v", pod.Name, containerName, err))
									continue
								}
								readers = append(readers, reader)
								containerNames = append(containerNames, fmt.Sprintf("%s/%s", pod.Name, containerName))
							}
						}

						// Show status message
						statusMsg := ""
						if v.selectedPod >= 0 {
							statusMsg = fmt.Sprintf("Pod: %s", allPodNames[v.selectedPod])
						} else {
							statusMsg = fmt.Sprintf("%d pods", len(podsToStream))
						}
						if v.selectedContainer >= 0 {
							statusMsg += fmt.Sprintf(", Container: %s", v.containers[v.selectedContainer])
						} else {
							statusMsg += fmt.Sprintf(", %d containers", len(containersToStream))
						}
						v.content = append(v.content, fmt.Sprintf("=== Streaming logs: %s ===", statusMsg))
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

// restartStreaming stops current streams and restarts with current filter settings
func (v *LogView) restartStreaming() tea.Cmd {
	// Stop current streams
	if v.cancelFunc != nil {
		v.cancelFunc()
	}
	for _, reader := range v.logReaders {
		if reader != nil {
			reader.Close()
		}
	}
	v.logReaders = nil
	v.scanners = nil

	// Clear content but keep filter settings
	v.content = []string{"Restarting streams with new filters..."}
	v.viewport.SetContent(strings.Join(v.content, "\n"))

	// Restart with same resource but current filter settings
	if v.client != nil && v.state != nil && v.resourceName != "" {
		// Create new context for the new streams
		parentCtx := context.Background()
		v.ctx, v.cancelFunc = context.WithCancel(parentCtx)
		return v.StartStreaming(v.ctx, v.client, v.state, v.resourceName)
	}

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
