package ui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/HamStudy/kubewatch/internal/core"
	"github.com/HamStudy/kubewatch/internal/k8s"
	tea "github.com/charmbracelet/bubbletea"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// createTestApp creates a test app with minimal setup for testing UI logic
func createTestApp(t *testing.T) *App {
	state := &core.State{
		CurrentResourceType: core.ResourceTypePod,
		CurrentNamespace:    "default",
		CurrentContext:      "test-context",
	}

	config := &core.Config{
		RefreshInterval: 5,
	}

	// Create app with nil client for pure UI testing
	app := NewApp(context.Background(), nil, state, config)
	app.width = 80
	app.height = 24
	app.ready = true

	// For pure UI testing, ensure both clients are nil to trigger test paths
	app.k8sClient = nil
	app.multiClient = nil

	// Ensure modes are initialized (they should be from NewApp, but let's be explicit)
	if app.modes == nil {
		app.modes = map[ScreenModeType]ScreenMode{
			ModeList:              NewListMode(),
			ModeLog:               NewLogMode(),
			ModeDescribe:          NewDescribeMode(),
			ModeHelp:              NewHelpMode(),
			ModeContextSelector:   NewContextSelectorMode(),
			ModeNamespaceSelector: NewNamespaceSelectorMode(),
			ModeConfirmDialog:     NewConfirmDialogMode(),
			ModeResourceSelector:  NewResourceSelectorMode(),
		}
	}

	return app
}

// createMockPod creates a mock pod for testing
func createMockPod(name, phase, namespace string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID("mock-uid-" + name),
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase(phase),
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "main", Image: "nginx:latest"},
			},
		},
	}
}

// createMockDeployment creates a mock deployment for testing
func createMockDeployment(name, namespace string) interface{} {
	return map[string]interface{}{
		"name":      name,
		"namespace": namespace,
		"replicas":  3,
		"ready":     2,
	}
}

// createMockService creates a mock service for testing
func createMockService(name, namespace string) interface{} {
	return map[string]interface{}{
		"name":      name,
		"namespace": namespace,
		"type":      "ClusterIP",
		"ports":     []string{"80:8080/TCP"},
	}
}

// simulateKeyPress simulates a key press and returns the updated model and command
func simulateKeyPress(app *App, key string) (*App, tea.Cmd) {
	var keyMsg tea.KeyMsg

	switch key {
	case "up":
		keyMsg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		keyMsg = tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		keyMsg = tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		keyMsg = tea.KeyMsg{Type: tea.KeyRight}
	case "enter":
		keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		keyMsg = tea.KeyMsg{Type: tea.KeyTab}
	case "space":
		keyMsg = tea.KeyMsg{Type: tea.KeySpace}
	default:
		// Handle single character keys
		keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}

	model, cmd := app.Update(keyMsg)
	return model.(*App), cmd
}

// assertMode checks that the app is in the expected mode
func assertMode(t *testing.T, app *App, expectedMode ScreenModeType) {
	t.Helper()
	if app.currentMode != expectedMode {
		t.Errorf("Expected mode %v, got %v", expectedMode, app.currentMode)
	}
}

// assertViewContains checks that the view contains the expected text
func assertViewContains(t *testing.T, app *App, expectedText string) {
	t.Helper()
	view := app.View()
	if !containsText(view, expectedText) {
		t.Errorf("Expected view to contain '%s', but it didn't.\nActual view:\n%s", expectedText, view)
	}
}

// assertViewNotContains checks that the view does not contain the specified text
func assertViewNotContains(t *testing.T, app *App, unexpectedText string) {
	t.Helper()
	view := app.View()
	if containsText(view, unexpectedText) {
		t.Errorf("Expected view to NOT contain '%s', but it did.\nActual view:\n%s", unexpectedText, view)
	}
}

// containsText checks if the view contains the specified text (case-insensitive)
func containsText(view, text string) bool {
	// Simple substring check - could be enhanced with regex or fuzzy matching
	return len(view) > 0 && len(text) > 0 &&
		(view == text || findSubstring(view, text))
}

// findSubstring performs a simple substring search
func findSubstring(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}

	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// MockK8sClient is a mock implementation of the K8s client for testing
type MockK8sClient struct {
	pods         []*v1.Pod
	deployments  []*appsv1.Deployment
	services     []*v1.Service
	namespaces   []string
	contexts     []string
	logs         map[string][]string // pod -> logs
	streamingErr error
}

// createMockK8sClient creates a mock K8s client for testing
func createMockK8sClient() *MockK8sClient {
	return &MockK8sClient{
		pods:       createTestPods(),
		namespaces: []string{"default", "kube-system", "test-namespace"},
		contexts:   []string{"test-context"},
		logs:       make(map[string][]string),
	}
}

// createTestPods creates test pods for testing
func createTestPods() []*v1.Pod {
	return []*v1.Pod{
		createMockPod("test-pod-1", "Running", "default"),
		createMockPod("test-pod-2", "Pending", "default"),
		createMockPod("test-pod-3", "Failed", "default"),
	}
}

// GetPods returns mock pods
func (m *MockK8sClient) GetPods(ctx context.Context, namespace string) ([]*v1.Pod, error) {
	var result []*v1.Pod
	for _, pod := range m.pods {
		if namespace == "" || pod.Namespace == namespace {
			result = append(result, pod)
		}
	}
	return result, nil
}

// GetDeployments returns mock deployments
func (m *MockK8sClient) GetDeployments(ctx context.Context, namespace string) ([]*appsv1.Deployment, error) {
	return m.deployments, nil
}

// GetServices returns mock services
func (m *MockK8sClient) GetServices(ctx context.Context, namespace string) ([]*v1.Service, error) {
	return m.services, nil
}

// GetNamespaces returns mock namespaces
func (m *MockK8sClient) GetNamespaces(ctx context.Context) ([]string, error) {
	return m.namespaces, nil
}

// GetContexts returns mock contexts
func (m *MockK8sClient) GetContexts() ([]string, error) {
	return m.contexts, nil
}

// GetCurrentContext returns the current mock context
func (m *MockK8sClient) GetCurrentContext() string {
	if len(m.contexts) > 0 {
		return m.contexts[0]
	}
	return "test-context"
}

// SwitchContext switches to a different context
func (m *MockK8sClient) SwitchContext(context string) error {
	return nil
}

// GetPodLogs returns mock logs for a pod
func (m *MockK8sClient) GetPodLogs(ctx context.Context, namespace, podName, containerName string, follow bool, tailLines int64) (io.ReadCloser, error) {
	if m.streamingErr != nil {
		return nil, m.streamingErr
	}
	// Return a simple reader with test logs
	logs := fmt.Sprintf("Test log line 1 for %s\nTest log line 2 for %s\n", podName, podName)
	return io.NopCloser(strings.NewReader(logs)), nil
}

// DeletePod deletes a mock pod
func (m *MockK8sClient) DeletePod(ctx context.Context, namespace, name string) error {
	for i, pod := range m.pods {
		if pod.Name == name && pod.Namespace == namespace {
			m.pods = append(m.pods[:i], m.pods[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("pod not found")
}

// DescribePod returns mock pod description
func (m *MockK8sClient) DescribePod(ctx context.Context, namespace, name string) (string, error) {
	for _, pod := range m.pods {
		if pod.Name == name && pod.Namespace == namespace {
			return fmt.Sprintf("Pod: %s\nNamespace: %s\nStatus: %s", name, namespace, pod.Status.Phase), nil
		}
	}
	return "", fmt.Errorf("pod not found")
}

// StreamPodLogs streams mock logs for a pod
func (m *MockK8sClient) StreamPodLogs(ctx context.Context, namespace, podName, containerName string) (<-chan string, <-chan error, error) {
	if m.streamingErr != nil {
		return nil, nil, m.streamingErr
	}

	logCh := make(chan string)
	errCh := make(chan error)

	go func() {
		defer close(logCh)
		defer close(errCh)

		// Send some test log lines
		logs := []string{
			fmt.Sprintf("[%s] Starting application...", time.Now().Format("15:04:05")),
			fmt.Sprintf("[%s] Server listening on port 8080", time.Now().Format("15:04:05")),
			fmt.Sprintf("[%s] Ready to accept connections", time.Now().Format("15:04:05")),
		}

		for _, log := range logs {
			select {
			case logCh <- log:
				time.Sleep(100 * time.Millisecond) // Simulate streaming delay
			case <-ctx.Done():
				return
			}
		}

		// Keep streaming until cancelled
		<-ctx.Done()
	}()

	return logCh, errCh, nil
}

// MockMultiClient is a mock implementation of the multi-client for testing
type MockMultiClient struct {
	contexts map[string]*MockK8sClient
	active   []string
}

// createMockMultiClient creates a mock multi-client for testing
func createMockMultiClient(contexts []string) *MockMultiClient {
	m := &MockMultiClient{
		contexts: make(map[string]*MockK8sClient),
		active:   contexts,
	}

	for _, ctx := range contexts {
		m.contexts[ctx] = createMockK8sClient()
		m.contexts[ctx].contexts = []string{ctx}
	}

	return m
}

// GetActiveContexts returns the active contexts
func (m *MockMultiClient) GetActiveContexts() []string {
	return m.active
}

// GetClientForContext returns a mock client for a specific context
func (m *MockMultiClient) GetClientForContext(context string) (*k8s.Client, error) {
	// For testing, we just return nil since we're mocking
	return nil, nil
}

// GetPodsFromAllContexts returns pods from all contexts
func (m *MockMultiClient) GetPodsFromAllContexts(ctx context.Context, namespace string) (map[string][]*v1.Pod, map[string]error) {
	results := make(map[string][]*v1.Pod)
	errors := make(map[string]error)

	for _, ctxName := range m.active {
		if client, ok := m.contexts[ctxName]; ok {
			pods, err := client.GetPods(ctx, namespace)
			if err != nil {
				errors[ctxName] = err
			} else {
				results[ctxName] = pods
			}
		}
	}

	return results, errors
}

// StreamLogsFromAllContexts streams logs from all contexts
func (m *MockMultiClient) StreamLogsFromAllContexts(ctx context.Context, namespace, podName, containerName string) (map[string]<-chan string, map[string]<-chan error, map[string]error) {
	logChannels := make(map[string]<-chan string)
	errChannels := make(map[string]<-chan error)
	errors := make(map[string]error)

	for _, ctxName := range m.active {
		if client, ok := m.contexts[ctxName]; ok {
			logCh, errCh, err := client.StreamPodLogs(ctx, namespace, podName, containerName)
			if err != nil {
				errors[ctxName] = err
			} else {
				logChannels[ctxName] = logCh
				errChannels[ctxName] = errCh
			}
		}
	}

	return logChannels, errChannels, errors
}
