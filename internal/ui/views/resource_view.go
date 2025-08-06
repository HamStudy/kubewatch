package views

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/k8s"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// ResourceView displays a list of Kubernetes resources
type ResourceView struct {
	state            *core.State
	k8sClient        *k8s.Client
	width            int
	height           int
	wordWrap         bool
	showMetrics      bool
	podMetrics       map[string]*k8s.PodMetrics
	horizontalOffset int
	lastRefresh      time.Time
	compactMode      bool // For split view with logs

	// Custom table data
	headers        []string
	rows           [][]string
	columnWidths   []int
	selectedRow    int
	viewportStart  int
	viewportHeight int
}

// NewResourceView creates a new resource view
func NewResourceView(state *core.State, k8sClient *k8s.Client) *ResourceView {
	rv := &ResourceView{
		state:       state,
		k8sClient:   k8sClient,
		showMetrics: true, // Try to show metrics by default
		selectedRow: 0,
		lastRefresh: time.Now(), // Initialize with current time
	}

	// Set initial columns based on resource type
	rv.updateColumnsForResourceType()

	return rv
}

// Init initializes the view
func (v *ResourceView) Init() tea.Cmd {
	return v.RefreshResources()
}

// Update handles messages
func (v *ResourceView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			// Move down
			if v.selectedRow < len(v.rows)-1 {
				v.selectedRow++
			}
			return v, nil
		case "k", "up":
			// Move up
			if v.selectedRow > 0 {
				v.selectedRow--
			}
			return v, nil
		case "u":
			// Toggle word wrap
			v.wordWrap = !v.wordWrap
			return v, nil
		case "home":
			v.selectedRow = 0
			return v, nil
		case "end":
			if len(v.rows) > 0 {
				v.selectedRow = len(v.rows) - 1
			}
			return v, nil
		case "pgup":
			// Page up
			if v.selectedRow > v.viewportHeight {
				v.selectedRow -= v.viewportHeight
			} else {
				v.selectedRow = 0
			}
			return v, nil
		case "pgdown":
			// Page down
			if v.selectedRow < len(v.rows)-v.viewportHeight {
				v.selectedRow += v.viewportHeight
			} else if len(v.rows) > 0 {
				v.selectedRow = len(v.rows) - 1
			}
			return v, nil
		case "h", "left":
			// Scroll left
			if v.horizontalOffset > 0 {
				v.horizontalOffset -= 5
			}
			return v, nil
		case "l", "right":
			// Scroll right
			v.horizontalOffset += 5
			return v, nil
		}
	}

	return v, nil
}

// View renders the view
func (v *ResourceView) View() string {
	header := v.renderHeader()
	// Use custom renderer instead of table.View()
	tableView := v.renderCustomTable()
	return lipgloss.JoinVertical(lipgloss.Left, header, tableView)
}

// SetSize updates the view size
func (v *ResourceView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.compactMode {
		// In compact mode, ensure selected item stays visible with minimal context
		v.viewportHeight = height - 3 // Less space for header in compact mode
	} else {
		v.viewportHeight = height - 6 // Account for header and borders
	}
}

// SetCompactMode enables/disables compact mode for split view
func (v *ResourceView) SetCompactMode(compact bool) {
	v.compactMode = compact
	if compact {
		// Adjust viewport to keep selected item visible
		v.ensureSelectedVisible()
	}
}

// ensureSelectedVisible adjusts viewport to keep selected item in view
func (v *ResourceView) ensureSelectedVisible() {
	// First ensure selectedRow is within bounds
	if v.selectedRow >= len(v.rows) && len(v.rows) > 0 {
		v.selectedRow = len(v.rows) - 1
	}
	if v.selectedRow < 0 && len(v.rows) > 0 {
		v.selectedRow = 0
	}

	// Ensure viewportStart is within bounds
	if v.viewportStart >= len(v.rows) {
		v.viewportStart = 0
		if len(v.rows) > v.viewportHeight {
			v.viewportStart = len(v.rows) - v.viewportHeight
		}
	}
	if v.viewportStart < 0 {
		v.viewportStart = 0
	}

	// Adjust viewport to keep selected item visible
	if v.selectedRow < v.viewportStart {
		v.viewportStart = v.selectedRow
	} else if v.selectedRow >= v.viewportStart+v.viewportHeight {
		v.viewportStart = v.selectedRow - v.viewportHeight + 1
	}

	// Ensure we show at least 3 items around selected if possible
	contextRows := 3
	if v.viewportHeight > contextRows*2 && len(v.rows) > 0 {
		idealStart := v.selectedRow - contextRows
		if idealStart >= 0 && idealStart+v.viewportHeight <= len(v.rows) {
			v.viewportStart = idealStart
		}
	}
}

// RefreshResources fetches and updates the resource list
func (v *ResourceView) RefreshResources() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		switch v.state.CurrentResourceType {
		case core.ResourceTypePod:
			pods, err := v.k8sClient.ListPods(ctx, v.state.CurrentNamespace)
			if err != nil {
				return errMsg{err}
			}

			// Try to get metrics (don't fail if not available)
			metrics, _ := v.k8sClient.GetPodMetrics(ctx, v.state.CurrentNamespace)
			v.podMetrics = metrics

			v.state.UpdatePods(pods)
			v.updateTableWithPods(pods)

		case core.ResourceTypeDeployment:
			deployments, err := v.k8sClient.ListDeployments(ctx, v.state.CurrentNamespace)
			if err != nil {
				return errMsg{err}
			}
			v.state.UpdateDeployments(deployments)
			v.updateTableWithDeployments(deployments)

		case core.ResourceTypeStatefulSet:
			statefulsets, err := v.k8sClient.ListStatefulSets(ctx, v.state.CurrentNamespace)
			if err != nil {
				return errMsg{err}
			}
			v.state.UpdateStatefulSets(statefulsets)
			v.updateTableWithStatefulSets(statefulsets)

		case core.ResourceTypeService:
			services, err := v.k8sClient.ListServices(ctx, v.state.CurrentNamespace)
			if err != nil {
				return errMsg{err}
			}
			v.state.UpdateServices(services)
			v.updateTableWithServices(services)

		case core.ResourceTypeIngress:
			ingresses, err := v.k8sClient.ListIngresses(ctx, v.state.CurrentNamespace)
			if err != nil {
				return errMsg{err}
			}
			v.state.UpdateIngresses(ingresses)
			v.updateTableWithIngresses(ingresses)

		case core.ResourceTypeConfigMap:
			configmaps, err := v.k8sClient.ListConfigMaps(ctx, v.state.CurrentNamespace)
			if err != nil {
				return errMsg{err}
			}
			v.state.UpdateConfigMaps(configmaps)
			v.updateTableWithConfigMaps(configmaps)

		case core.ResourceTypeSecret:
			secrets, err := v.k8sClient.ListSecrets(ctx, v.state.CurrentNamespace)
			if err != nil {
				return errMsg{err}
			}
			v.state.UpdateSecrets(secrets)
			v.updateTableWithSecrets(secrets)
		}

		// Update last refresh time
		v.lastRefresh = time.Now()

		return refreshCompleteMsg{}
	}
}

// GetSelectedResourceName returns the name of the currently selected resource
func (v *ResourceView) GetSelectedResourceName() string {
	if v.selectedRow >= 0 && v.selectedRow < len(v.rows) && len(v.rows) > 0 {
		return v.rows[v.selectedRow][0] // First column is always NAME
	}
	return ""
}

// DeleteSelected deletes the selected resource(s)
func (v *ResourceView) DeleteSelected() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Check if we have a selected row
		if v.selectedRow >= len(v.rows) || v.selectedRow < 0 {
			return nil
		}

		selectedRow := v.rows[v.selectedRow]
		if len(selectedRow) == 0 {
			return nil
		}

		name := selectedRow[0]
		namespace := v.state.CurrentNamespace

		var err error
		switch v.state.CurrentResourceType {
		case core.ResourceTypePod:
			err = v.k8sClient.DeletePod(ctx, namespace, name)
		case core.ResourceTypeDeployment:
			err = v.k8sClient.DeleteDeployment(ctx, namespace, name)
		case core.ResourceTypeStatefulSet:
			err = v.k8sClient.DeleteStatefulSet(ctx, namespace, name)
		case core.ResourceTypeService:
			err = v.k8sClient.DeleteService(ctx, namespace, name)
		case core.ResourceTypeIngress:
			err = v.k8sClient.DeleteIngress(ctx, namespace, name)
		case core.ResourceTypeConfigMap:
			err = v.k8sClient.DeleteConfigMap(ctx, namespace, name)
		case core.ResourceTypeSecret:
			err = v.k8sClient.DeleteSecret(ctx, namespace, name)
		}

		if err != nil {
			return errMsg{err}
		}

		return deleteCompleteMsg{name}
	}
}

// renderCustomTable renders the table using lipgloss styling
func (v *ResourceView) renderCustomTable() string {
	if len(v.headers) == 0 || len(v.rows) == 0 {
		return "No resources found"
	}

	// Ensure selectedRow is within bounds
	if v.selectedRow >= len(v.rows) {
		v.selectedRow = len(v.rows) - 1
	}
	if v.selectedRow < 0 {
		v.selectedRow = 0
	}

	// Ensure columnWidths is initialized and matches headers
	if len(v.columnWidths) != len(v.headers) {
		v.calculateColumnWidths()
	}

	// Render header
	var headerCells []string
	for i, header := range v.headers {
		width := 15 // default width
		if i < len(v.columnWidths) {
			width = v.columnWidths[i]
		}
		cell := v.styleHeaderCell(header, width)
		headerCells = append(headerCells, cell)
	}
	headerRow := strings.Join(headerCells, " ")

	// Style the header with border
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("7")).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(v.width)
	styledHeader := headerStyle.Render(headerRow)

	// Calculate viewport
	if v.viewportHeight == 0 {
		v.viewportHeight = v.height - 6 // Account for header and borders
	}

	// Ensure selected row is within bounds
	if v.selectedRow >= len(v.rows) && len(v.rows) > 0 {
		v.selectedRow = len(v.rows) - 1
	}
	if v.selectedRow < 0 && len(v.rows) > 0 {
		v.selectedRow = 0
	}

	// Ensure viewportStart is within bounds
	if v.viewportStart >= len(v.rows) {
		v.viewportStart = 0
		if len(v.rows) > v.viewportHeight {
			v.viewportStart = len(v.rows) - v.viewportHeight
		}
	}
	if v.viewportStart < 0 {
		v.viewportStart = 0
	}

	// Ensure selected row is visible
	if v.selectedRow < v.viewportStart {
		v.viewportStart = v.selectedRow
	} else if v.selectedRow >= v.viewportStart+v.viewportHeight {
		v.viewportStart = v.selectedRow - v.viewportHeight + 1
	}

	// Render visible rows
	var renderedRows []string
	endRow := v.viewportStart + v.viewportHeight
	if endRow > len(v.rows) {
		endRow = len(v.rows)
	}

	for i := v.viewportStart; i < endRow && i < len(v.rows); i++ {
		if i < 0 || i >= len(v.rows) {
			continue // Skip invalid indices
		}
		row := v.rows[i]
		isSelected := i == v.selectedRow
		var cells []string

		for j, cell := range row {
			if j < len(v.headers) {
				width := 15 // default width
				if j < len(v.columnWidths) {
					width = v.columnWidths[j]
				}
				styledCell := v.styleCellByColumn(v.headers[j], cell, width, isSelected)
				cells = append(cells, styledCell)
			}
		}

		rowStr := strings.Join(cells, " ")
		renderedRows = append(renderedRows, rowStr)
	}

	// Join all rows
	tableContent := strings.Join(renderedRows, "\n")

	// Add scroll indicators if needed
	if v.viewportStart > 0 || endRow < len(v.rows) {
		scrollInfo := fmt.Sprintf(" [%d-%d of %d]", v.viewportStart+1, endRow, len(v.rows))
		scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		tableContent += "\n" + scrollStyle.Render(scrollInfo)
	}

	// Combine header and content
	return lipgloss.JoinVertical(lipgloss.Left, styledHeader, tableContent)
}

// styleHeaderCell styles a header cell
func (v *ResourceView) styleHeaderCell(header string, width int) string {
	style := lipgloss.NewStyle().Width(width).Bold(true)

	// Right-align numeric columns
	if header == "CPU" || header == "MEMORY" || header == "READY" ||
		header == "RESTARTS" || header == "DATA" || header == "UP-TO-DATE" ||
		header == "AVAILABLE" {
		style = style.Align(lipgloss.Right)
	}

	return style.Render(header)
}

// styleCellByColumn applies appropriate styling based on column type
func (v *ResourceView) styleCellByColumn(columnName, value string, width int, isSelected bool) string {
	// Handle word wrap
	displayValue := value
	actualWidth := width

	if v.wordWrap {
		// When wrap is ON, respect the column width and truncate if needed
		if len(value) > width-2 && width > 5 {
			displayValue = value[:width-5] + "..."
		}
	} else {
		// When wrap is OFF, don't truncate - show full content
		// Adjust width if content is longer
		if len(value) > width {
			actualWidth = len(value) + 2
		}
	}

	switch columnName {
	case "STATUS":
		return v.styleStatusCell(displayValue, actualWidth, isSelected)
	case "CPU":
		return v.styleMetricCell(displayValue, actualWidth, isSelected, true)
	case "MEMORY":
		return v.styleMetricCell(displayValue, actualWidth, isSelected, false)
	case "RESTARTS":
		return v.styleRestartsCell(displayValue, actualWidth, isSelected)
	case "READY", "UP-TO-DATE", "AVAILABLE", "DATA":
		// Right-align numeric columns
		style := lipgloss.NewStyle().Width(actualWidth).Align(lipgloss.Right)
		if isSelected {
			style = style.Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
		}
		return style.Render(displayValue)
	default:
		// Default left-aligned
		style := lipgloss.NewStyle().Width(actualWidth)
		if isSelected {
			style = style.Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
		}
		return style.Render(displayValue)
	}
}

// styleStatusCell applies color based on pod status
func (v *ResourceView) styleStatusCell(status string, width int, isSelected bool) string {
	style := lipgloss.NewStyle().Width(width)

	// Apply selection background
	if isSelected {
		style = style.Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
		return style.Render(status)
	}

	// Apply status-based colors
	switch status {
	case "Running":
		style = style.Foreground(lipgloss.Color("2")) // Green
	case "Pending", "ContainerCreating":
		style = style.Foreground(lipgloss.Color("3")) // Yellow
	case "Failed", "Error", "CrashLoopBackOff", "ImagePullBackOff":
		style = style.Foreground(lipgloss.Color("1")) // Red
	case "Completed":
		style = style.Foreground(lipgloss.Color("4")) // Blue
	case "Terminating":
		style = style.Foreground(lipgloss.Color("5")) // Magenta
	default:
		style = style.Foreground(lipgloss.Color("7")) // Default
	}

	return style.Render(status)
}

// styleMetricCell applies color based on resource usage
func (v *ResourceView) styleMetricCell(value string, width int, isSelected bool, isCPU bool) string {
	style := lipgloss.NewStyle().Width(width).Align(lipgloss.Right)

	// Apply selection background
	if isSelected {
		style = style.Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
		return style.Render(value)
	}

	// Skip if no value or "-"
	if value == "-" || value == "" {
		style = style.Foreground(lipgloss.Color("241")) // Gray for no data
		return style.Render(value)
	}

	// Parse the numeric value
	var numValue float64
	if isCPU {
		// CPU values like "100m", "1", "2500m"
		if strings.HasSuffix(value, "m") {
			// Millicores
			numStr := strings.TrimSuffix(value, "m")
			if val, err := strconv.ParseFloat(numStr, 64); err == nil {
				numValue = val / 1000.0 // Convert to cores
			}
		} else {
			// Cores
			if val, err := strconv.ParseFloat(value, 64); err == nil {
				numValue = val
			}
		}

		// Color based on CPU usage (in cores)
		if numValue < 0.1 {
			style = style.Foreground(lipgloss.Color("2")) // Green for low
		} else if numValue < 0.5 {
			style = style.Foreground(lipgloss.Color("3")) // Yellow for medium
		} else {
			style = style.Foreground(lipgloss.Color("1")) // Red for high
		}
	} else {
		// Memory values like "128Mi", "1Gi", "512Ki"
		var multiplier float64 = 1
		cleanValue := value

		if strings.HasSuffix(value, "Gi") {
			multiplier = 1024
			cleanValue = strings.TrimSuffix(value, "Gi")
		} else if strings.HasSuffix(value, "Mi") {
			multiplier = 1
			cleanValue = strings.TrimSuffix(value, "Mi")
		} else if strings.HasSuffix(value, "Ki") {
			multiplier = 1.0 / 1024.0
			cleanValue = strings.TrimSuffix(value, "Ki")
		}

		if val, err := strconv.ParseFloat(cleanValue, 64); err == nil {
			numValue = val * multiplier // Convert to Mi
		}

		// Color based on memory usage (in Mi)
		if numValue < 128 {
			style = style.Foreground(lipgloss.Color("2")) // Green for low
		} else if numValue < 512 {
			style = style.Foreground(lipgloss.Color("3")) // Yellow for medium
		} else {
			style = style.Foreground(lipgloss.Color("1")) // Red for high
		}
	}

	return style.Render(value)
}

// styleRestartsCell applies color based on restart count
func (v *ResourceView) styleRestartsCell(value string, width int, isSelected bool) string {
	style := lipgloss.NewStyle().Width(width).Align(lipgloss.Right)

	// Apply selection background
	if isSelected {
		style = style.Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
		return style.Render(value)
	}

	// Extract number from format like "5 (2m ago)"
	numStr := strings.Split(value, " ")[0]
	restarts, err := strconv.Atoi(numStr)

	if err == nil {
		if restarts == 0 {
			style = style.Foreground(lipgloss.Color("241")) // Gray for zero
		} else if restarts < 5 {
			style = style.Foreground(lipgloss.Color("3")) // Yellow for low
		} else {
			style = style.Foreground(lipgloss.Color("1")) // Red for high
		}
	}

	return style.Render(value)
}

// restoreSelection intelligently restores the selection after updating rows
func (v *ResourceView) restoreSelection(newSelectedRow, previousSelectedRow int) {
	if newSelectedRow >= 0 {
		// Found the same resource, select it
		v.selectedRow = newSelectedRow
	} else if previousSelectedRow < len(v.rows) {
		// Keep the same position if possible
		v.selectedRow = previousSelectedRow
	} else if len(v.rows) > 0 {
		// Select the last item if previous position is out of bounds
		v.selectedRow = len(v.rows) - 1
	} else {
		// No items left
		v.selectedRow = 0
		v.viewportStart = 0
		return
	}

	// Ensure viewport is within bounds first
	if v.viewportStart >= len(v.rows) {
		v.viewportStart = 0
		if len(v.rows) > v.viewportHeight {
			v.viewportStart = len(v.rows) - v.viewportHeight
		}
	}
	if v.viewportStart < 0 {
		v.viewportStart = 0
	}

	// Adjust viewport to keep selection visible
	if v.selectedRow >= v.viewportStart+v.viewportHeight {
		v.viewportStart = v.selectedRow - v.viewportHeight + 1
		if v.viewportStart < 0 {
			v.viewportStart = 0
		}
	} else if v.selectedRow < v.viewportStart {
		v.viewportStart = v.selectedRow
	}
}

// calculateColumnWidths calculates the width for each column based on content
func (v *ResourceView) calculateColumnWidths() {
	if len(v.headers) == 0 {
		return
	}

	// Initialize with header widths
	v.columnWidths = make([]int, len(v.headers))
	for i, header := range v.headers {
		v.columnWidths[i] = len(header) + 2
	}

	// Check all rows for max width
	for _, row := range v.rows {
		for i, cell := range row {
			if i < len(v.columnWidths) {
				cellLen := len(cell) + 2
				if cellLen > v.columnWidths[i] {
					v.columnWidths[i] = cellLen
				}
			}
		}
	}

	// Apply limits based on word wrap setting
	for i := range v.columnWidths {
		if v.columnWidths[i] < 7 {
			v.columnWidths[i] = 7
		}
		// If word wrap is enabled, limit column width to prevent overly wide columns
		if v.wordWrap && v.columnWidths[i] > 50 {
			v.columnWidths[i] = 50
		}
		// When word wrap is off, no maximum limit - show full content
	}
}

func (v *ResourceView) renderHeader() string {
	title := fmt.Sprintf("KubeWatch TUI - %s", v.state.CurrentResourceType)
	namespace := fmt.Sprintf("Namespace: %s", v.state.CurrentNamespace)
	count := fmt.Sprintf("Count: %d", v.state.GetCurrentResourceCount())

	// Add word wrap indicator
	wrapStatus := "Wrap: OFF"
	if v.wordWrap {
		wrapStatus = "Wrap: ON"
	}

	// Add last refresh time
	refreshStatus := "Never"
	if !v.lastRefresh.IsZero() {
		elapsed := time.Since(v.lastRefresh)
		if elapsed < time.Minute {
			refreshStatus = fmt.Sprintf("%ds ago", int(elapsed.Seconds()))
		} else {
			refreshStatus = fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
		}
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	wrapStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))    // Yellow for wrap status
	refreshStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // Green for refresh

	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		titleStyle.Render(title),
		strings.Repeat(" ", 10),
		infoStyle.Render(namespace),
		strings.Repeat(" ", 5),
		infoStyle.Render(count),
		strings.Repeat(" ", 5),
		wrapStyle.Render(wrapStatus),
		strings.Repeat(" ", 5),
		refreshStyle.Render("↻ "+refreshStatus),
	)

	return header + "\n"
}

// updateColumnsForResourceType sets the appropriate columns for the current resource type
func (v *ResourceView) updateColumnsForResourceType() {
	// Check if we're viewing all namespaces or a specific one
	showNamespace := v.state.CurrentNamespace == "" || v.state.CurrentNamespace == "all"

	switch v.state.CurrentResourceType {
	case core.ResourceTypePod:
		v.headers = []string{"NAME"}
		if showNamespace {
			v.headers = append(v.headers, "NAMESPACE")
		}
		v.headers = append(v.headers, "READY", "STATUS", "RESTARTS", "AGE", "CPU", "MEMORY", "IP", "NODE")

	case core.ResourceTypeDeployment:
		v.headers = []string{"NAME"}
		if showNamespace {
			v.headers = append(v.headers, "NAMESPACE")
		}
		v.headers = append(v.headers, "READY", "UP-TO-DATE", "AVAILABLE", "AGE", "CONTAINERS", "IMAGES", "SELECTOR")

	case core.ResourceTypeStatefulSet:
		v.headers = []string{"NAME"}
		if showNamespace {
			v.headers = append(v.headers, "NAMESPACE")
		}
		v.headers = append(v.headers, "READY", "AGE", "CONTAINERS", "IMAGES")

	case core.ResourceTypeService:
		v.headers = []string{"NAME"}
		if showNamespace {
			v.headers = append(v.headers, "NAMESPACE")
		}
		v.headers = append(v.headers, "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE")

	case core.ResourceTypeIngress:
		v.headers = []string{"NAME"}
		if showNamespace {
			v.headers = append(v.headers, "NAMESPACE")
		}
		v.headers = append(v.headers, "CLASS", "HOSTS", "ADDRESS", "PORTS", "AGE")

	case core.ResourceTypeConfigMap:
		v.headers = []string{"NAME"}
		if showNamespace {
			v.headers = append(v.headers, "NAMESPACE")
		}
		v.headers = append(v.headers, "DATA", "AGE")

	case core.ResourceTypeSecret:
		v.headers = []string{"NAME"}
		if showNamespace {
			v.headers = append(v.headers, "NAMESPACE")
		}
		v.headers = append(v.headers, "TYPE", "DATA", "AGE")
	}
}

func (v *ResourceView) updateTableWithPods(pods []v1.Pod) {
	// Update columns for pods
	v.updateColumnsForResourceType()

	showNamespace := v.state.CurrentNamespace == "" || v.state.CurrentNamespace == "all"

	// Preserve the currently selected resource name and position
	var selectedResourceName string
	previousSelectedRow := v.selectedRow
	if v.selectedRow >= 0 && v.selectedRow < len(v.rows) && len(v.rows) > 0 {
		selectedResourceName = v.rows[v.selectedRow][0] // First column is always NAME
	}

	// Clear and rebuild rows
	v.rows = [][]string{}
	newSelectedRow := -1 // Will update this if we find the previously selected resource

	for _, pod := range pods {
		// Calculate ready containers
		readyContainers := 0
		totalContainers := len(pod.Status.ContainerStatuses)
		restartCount := int32(0)
		var lastRestartTime *time.Time

		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyContainers++
			}
			restartCount += cs.RestartCount
			if cs.LastTerminationState.Terminated != nil {
				t := cs.LastTerminationState.Terminated.FinishedAt.Time
				if lastRestartTime == nil || t.After(*lastRestartTime) {
					lastRestartTime = &t
				}
			}
		}

		ready := fmt.Sprintf("%d/%d", readyContainers, totalContainers)
		status := string(pod.Status.Phase)

		// Get more detailed status if available
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status != v1.ConditionTrue {
				if condition.Reason != "" {
					status = condition.Reason
				}
			}
		}

		// Check container statuses for more specific states
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				status = cs.State.Waiting.Reason
				break
			}
			if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
				status = cs.State.Terminated.Reason
				break
			}
		}

		// Format restart count with time if available
		restartStr := fmt.Sprintf("%d", restartCount)
		if restartCount > 0 && lastRestartTime != nil {
			restartAge := getAge(*lastRestartTime)
			restartStr = fmt.Sprintf("%d (%s ago)", restartCount, restartAge)
		}

		age := getAge(pod.CreationTimestamp.Time)

		// Get metrics if available
		cpu := "-"
		memory := "-"
		if v.podMetrics != nil {
			if metrics, ok := v.podMetrics[pod.Name]; ok {
				cpu = metrics.CPU
				memory = metrics.Memory
			}
		}

		// Get IP and Node
		ip := pod.Status.PodIP
		if ip == "" {
			ip = "-"
		}
		node := pod.Spec.NodeName
		if node == "" {
			node = "-"
		}

		// Build row data
		rowData := []string{pod.Name}
		if showNamespace {
			rowData = append(rowData, pod.Namespace)
		}
		rowData = append(rowData, ready, status, restartStr, age, cpu, memory, ip, node)
		v.rows = append(v.rows, rowData)

		// Check if this was the previously selected resource
		if selectedResourceName != "" && pod.Name == selectedResourceName {
			newSelectedRow = len(v.rows) - 1
		}
	}

	// Restore selection intelligently
	v.restoreSelection(newSelectedRow, previousSelectedRow)

	// Adjust viewport to keep selection visible
	if v.selectedRow >= v.viewportStart+v.viewportHeight {
		v.viewportStart = v.selectedRow - v.viewportHeight + 1
	} else if v.selectedRow < v.viewportStart {
		v.viewportStart = v.selectedRow
	}

	// Calculate column widths
	v.calculateColumnWidths()
}

func (v *ResourceView) updateTableWithDeployments(deployments []appsv1.Deployment) {
	// Update columns for deployments
	v.updateColumnsForResourceType()

	showNamespace := v.state.CurrentNamespace == "" || v.state.CurrentNamespace == "all"

	// Preserve the currently selected resource name
	var selectedResourceName string
	if v.selectedRow >= 0 && v.selectedRow < len(v.rows) && len(v.rows) > 0 {
		selectedResourceName = v.rows[v.selectedRow][0] // First column is always NAME
	}

	// Clear and rebuild rows
	v.rows = [][]string{}
	newSelectedRow := 0 // Will update this if we find the previously selected resource

	for _, dep := range deployments {
		replicas := int32(0)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}
		ready := fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, replicas)
		upToDate := fmt.Sprintf("%d", dep.Status.UpdatedReplicas)
		available := fmt.Sprintf("%d", dep.Status.AvailableReplicas)
		age := getAge(dep.CreationTimestamp.Time)

		// Get containers and images
		var containers []string
		var images []string
		for _, container := range dep.Spec.Template.Spec.Containers {
			containers = append(containers, container.Name)
			images = append(images, container.Image)
		}
		containersStr := strings.Join(containers, ",")
		imagesStr := strings.Join(images, ",")

		// Get selector
		var selectors []string
		for k, v := range dep.Spec.Selector.MatchLabels {
			selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
		}
		selectorStr := strings.Join(selectors, ",")

		// Build row data
		rowData := []string{dep.Name}
		if showNamespace {
			rowData = append(rowData, dep.Namespace)
		}
		rowData = append(rowData, ready, upToDate, available, age, containersStr, imagesStr, selectorStr)
		v.rows = append(v.rows, rowData)

		// Check if this was the previously selected resource
		if selectedResourceName != "" && dep.Name == selectedResourceName {
			newSelectedRow = len(v.rows) - 1
		}
	}

	// Restore selection
	v.selectedRow = newSelectedRow

	// Adjust viewport to keep selection visible
	if v.selectedRow >= v.viewportStart+v.viewportHeight {
		v.viewportStart = v.selectedRow - v.viewportHeight + 1
	} else if v.selectedRow < v.viewportStart {
		v.viewportStart = v.selectedRow
	}

	// Calculate column widths
	v.calculateColumnWidths()
}

func (v *ResourceView) updateTableWithStatefulSets(statefulsets []appsv1.StatefulSet) {
	// Update columns for statefulsets
	v.updateColumnsForResourceType()

	showNamespace := v.state.CurrentNamespace == "" || v.state.CurrentNamespace == "all"

	// Preserve the currently selected resource name
	var selectedResourceName string
	if v.selectedRow >= 0 && v.selectedRow < len(v.rows) && len(v.rows) > 0 {
		selectedResourceName = v.rows[v.selectedRow][0] // First column is always NAME
	}

	// Clear and rebuild rows
	v.rows = [][]string{}
	newSelectedRow := 0 // Will update this if we find the previously selected resource

	for _, sts := range statefulsets {
		replicas := int32(0)
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}
		ready := fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, replicas)
		age := getAge(sts.CreationTimestamp.Time)

		// Get containers and images
		var containers []string
		var images []string
		for _, container := range sts.Spec.Template.Spec.Containers {
			containers = append(containers, container.Name)
			images = append(images, container.Image)
		}
		containersStr := strings.Join(containers, ",")
		imagesStr := strings.Join(images, ",")

		// Build row data
		rowData := []string{sts.Name}
		if showNamespace {
			rowData = append(rowData, sts.Namespace)
		}
		rowData = append(rowData, ready, age, containersStr, imagesStr)
		v.rows = append(v.rows, rowData)

		// Check if this was the previously selected resource
		if selectedResourceName != "" && sts.Name == selectedResourceName {
			newSelectedRow = len(v.rows) - 1
		}
	}

	// Restore selection
	v.selectedRow = newSelectedRow

	// Adjust viewport to keep selection visible
	if v.selectedRow >= v.viewportStart+v.viewportHeight {
		v.viewportStart = v.selectedRow - v.viewportHeight + 1
	} else if v.selectedRow < v.viewportStart {
		v.viewportStart = v.selectedRow
	}

	// Calculate column widths
	v.calculateColumnWidths()
}

func (v *ResourceView) updateTableWithServices(services []v1.Service) {
	// Update columns for services
	v.updateColumnsForResourceType()

	showNamespace := v.state.CurrentNamespace == "" || v.state.CurrentNamespace == "all"

	// Preserve the currently selected resource name
	var selectedResourceName string
	if v.selectedRow >= 0 && v.selectedRow < len(v.rows) && len(v.rows) > 0 {
		selectedResourceName = v.rows[v.selectedRow][0] // First column is always NAME
	}

	// Clear and rebuild rows
	v.rows = [][]string{}
	newSelectedRow := 0 // Will update this if we find the previously selected resource

	for _, svc := range services {
		svcType := string(svc.Spec.Type)
		clusterIP := svc.Spec.ClusterIP
		if clusterIP == "" {
			clusterIP = "None"
		}

		// Get external IPs
		externalIP := "<none>"
		if len(svc.Spec.ExternalIPs) > 0 {
			externalIP = strings.Join(svc.Spec.ExternalIPs, ",")
		} else if svc.Spec.Type == v1.ServiceTypeLoadBalancer && len(svc.Status.LoadBalancer.Ingress) > 0 {
			var ips []string
			for _, ingress := range svc.Status.LoadBalancer.Ingress {
				if ingress.IP != "" {
					ips = append(ips, ingress.IP)
				} else if ingress.Hostname != "" {
					ips = append(ips, ingress.Hostname)
				}
			}
			if len(ips) > 0 {
				externalIP = strings.Join(ips, ",")
			}
		}

		// Get ports
		var ports []string
		for _, port := range svc.Spec.Ports {
			portStr := fmt.Sprintf("%d", port.Port)
			if port.NodePort != 0 {
				portStr = fmt.Sprintf("%d:%d", port.Port, port.NodePort)
			}
			if port.Protocol != "" && port.Protocol != "TCP" {
				portStr = fmt.Sprintf("%s/%s", portStr, port.Protocol)
			}
			if port.Name != "" {
				portStr = fmt.Sprintf("%s(%s)", portStr, port.Name)
			}
			ports = append(ports, portStr)
		}
		portStr := "<none>"
		if len(ports) > 0 {
			portStr = strings.Join(ports, ",")
			// Don't truncate - show full port information
		}

		age := getAge(svc.CreationTimestamp.Time)

		// Build row data
		rowData := []string{svc.Name}
		if showNamespace {
			rowData = append(rowData, svc.Namespace)
		}
		rowData = append(rowData, svcType, clusterIP, externalIP, portStr, age)
		v.rows = append(v.rows, rowData)

		// Check if this was the previously selected resource
		if selectedResourceName != "" && svc.Name == selectedResourceName {
			newSelectedRow = len(v.rows) - 1
		}
	}

	// Restore selection
	v.selectedRow = newSelectedRow

	// Adjust viewport to keep selection visible
	if v.selectedRow >= v.viewportStart+v.viewportHeight {
		v.viewportStart = v.selectedRow - v.viewportHeight + 1
	} else if v.selectedRow < v.viewportStart {
		v.viewportStart = v.selectedRow
	}

	// Calculate column widths
	v.calculateColumnWidths()
}

func (v *ResourceView) updateTableWithIngresses(ingresses []networkingv1.Ingress) {
	// Update columns for ingresses
	v.updateColumnsForResourceType()

	showNamespace := v.state.CurrentNamespace == "" || v.state.CurrentNamespace == "all"

	// Preserve the currently selected resource name
	var selectedResourceName string
	previousSelectedRow := v.selectedRow
	if v.selectedRow >= 0 && v.selectedRow < len(v.rows) && len(v.rows) > 0 {
		selectedResourceName = v.rows[v.selectedRow][0] // First column is always NAME
	}

	// Clear and rebuild rows
	v.rows = [][]string{}
	newSelectedRow := -1 // Will update this if we find the previously selected resource

	for _, ing := range ingresses {
		// Get ingress class
		className := "<none>"
		if ing.Spec.IngressClassName != nil {
			className = *ing.Spec.IngressClassName
		}

		// Get hosts
		var hosts []string
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		hostsStr := "<none>"
		if len(hosts) > 0 {
			hostsStr = strings.Join(hosts, ",")
		}

		// Get addresses
		var addresses []string
		for _, ingStatus := range ing.Status.LoadBalancer.Ingress {
			if ingStatus.IP != "" {
				addresses = append(addresses, ingStatus.IP)
			} else if ingStatus.Hostname != "" {
				addresses = append(addresses, ingStatus.Hostname)
			}
		}
		addressStr := "<none>"
		if len(addresses) > 0 {
			addressStr = strings.Join(addresses, ",")
		}

		// Get ports
		ports := "80"
		if len(ing.Spec.TLS) > 0 {
			ports = "80, 443"
		}

		age := getAge(ing.CreationTimestamp.Time)

		// Build row data
		rowData := []string{ing.Name}
		if showNamespace {
			rowData = append(rowData, ing.Namespace)
		}
		rowData = append(rowData, className, hostsStr, addressStr, ports, age)
		v.rows = append(v.rows, rowData)

		// Check if this was the previously selected resource
		if selectedResourceName != "" && ing.Name == selectedResourceName {
			newSelectedRow = len(v.rows) - 1
		}
	}

	// Restore selection intelligently
	v.restoreSelection(newSelectedRow, previousSelectedRow)

	// Calculate column widths
	v.calculateColumnWidths()
}

func (v *ResourceView) updateTableWithConfigMaps(configmaps []v1.ConfigMap) {
	// Update columns for configmaps
	v.updateColumnsForResourceType()

	showNamespace := v.state.CurrentNamespace == "" || v.state.CurrentNamespace == "all"

	// Preserve the currently selected resource name
	var selectedResourceName string
	if v.selectedRow >= 0 && v.selectedRow < len(v.rows) && len(v.rows) > 0 {
		selectedResourceName = v.rows[v.selectedRow][0] // First column is always NAME
	}

	// Clear and rebuild rows
	v.rows = [][]string{}
	newSelectedRow := 0 // Will update this if we find the previously selected resource

	for _, cm := range configmaps {
		dataCount := fmt.Sprintf("%d", len(cm.Data)+len(cm.BinaryData))
		age := getAge(cm.CreationTimestamp.Time)

		// Build row data
		rowData := []string{cm.Name}
		if showNamespace {
			rowData = append(rowData, cm.Namespace)
		}
		rowData = append(rowData, dataCount, age)
		v.rows = append(v.rows, rowData)

		// Check if this was the previously selected resource
		if selectedResourceName != "" && cm.Name == selectedResourceName {
			newSelectedRow = len(v.rows) - 1
		}
	}

	// Restore selection
	v.selectedRow = newSelectedRow

	// Adjust viewport to keep selection visible
	if v.selectedRow >= v.viewportStart+v.viewportHeight {
		v.viewportStart = v.selectedRow - v.viewportHeight + 1
	} else if v.selectedRow < v.viewportStart {
		v.viewportStart = v.selectedRow
	}

	// Calculate column widths
	v.calculateColumnWidths()
}

func (v *ResourceView) updateTableWithSecrets(secrets []v1.Secret) {
	// Update columns for secrets
	v.updateColumnsForResourceType()

	showNamespace := v.state.CurrentNamespace == "" || v.state.CurrentNamespace == "all"

	// Preserve the currently selected resource name
	var selectedResourceName string
	if v.selectedRow >= 0 && v.selectedRow < len(v.rows) && len(v.rows) > 0 {
		selectedResourceName = v.rows[v.selectedRow][0] // First column is always NAME
	}

	// Clear and rebuild rows
	v.rows = [][]string{}
	newSelectedRow := 0 // Will update this if we find the previously selected resource

	for _, secret := range secrets {
		secretType := string(secret.Type)
		dataCount := fmt.Sprintf("%d", len(secret.Data))
		age := getAge(secret.CreationTimestamp.Time)

		// Build row data
		rowData := []string{secret.Name}
		if showNamespace {
			rowData = append(rowData, secret.Namespace)
		}
		rowData = append(rowData, secretType, dataCount, age)
		v.rows = append(v.rows, rowData)

		// Check if this was the previously selected resource
		if selectedResourceName != "" && secret.Name == selectedResourceName {
			newSelectedRow = len(v.rows) - 1
		}
	}

	// Restore selection
	v.selectedRow = newSelectedRow

	// Adjust viewport to keep selection visible
	if v.selectedRow >= v.viewportStart+v.viewportHeight {
		v.viewportStart = v.selectedRow - v.viewportHeight + 1
	} else if v.selectedRow < v.viewportStart {
		v.viewportStart = v.selectedRow
	}

	// Calculate column widths
	v.calculateColumnWidths()
}

// Helper functions

func getAge(t time.Time) string {
	duration := time.Since(t)
	if duration.Hours() > 24*365 {
		years := int(duration.Hours() / (24 * 365))
		return fmt.Sprintf("%dy", years)
	} else if duration.Hours() > 24*30 {
		months := int(duration.Hours() / (24 * 30))
		return fmt.Sprintf("%dmo", months)
	} else if duration.Hours() > 24 {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	} else if duration.Hours() > 1 {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else if duration.Minutes() > 1 {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	return fmt.Sprintf("%ds", int(duration.Seconds()))
}

// Message types
type refreshCompleteMsg struct{}
type deleteCompleteMsg struct{ name string }
type errMsg struct{ err error }
