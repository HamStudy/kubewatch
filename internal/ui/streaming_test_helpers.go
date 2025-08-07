package ui

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// LogStreamSimulator simulates log streaming for testing
type LogStreamSimulator struct {
	mu        sync.RWMutex
	pods      map[string]*PodLogSimulator    // podName -> simulator
	contexts  map[string]*LogStreamSimulator // context -> simulator
	streaming bool
	errors    map[string]error
}

// PodLogSimulator simulates logs for a single pod
type PodLogSimulator struct {
	podName    string
	namespace  string
	containers map[string][]string // container -> logs
	streaming  bool
	logChan    chan string
	errChan    chan error
}

// NewLogStreamSimulator creates a new log stream simulator
func NewLogStreamSimulator() *LogStreamSimulator {
	return &LogStreamSimulator{
		pods:     make(map[string]*PodLogSimulator),
		contexts: make(map[string]*LogStreamSimulator),
		errors:   make(map[string]error),
	}
}

// AddPod adds a pod with containers to the simulator
func (s *LogStreamSimulator) AddPod(name, namespace string, containers map[string][]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pods[name] = &PodLogSimulator{
		podName:    name,
		namespace:  namespace,
		containers: containers,
		logChan:    make(chan string, 100),
		errChan:    make(chan error, 10),
	}
}

// SimulatePodLogs simulates streaming logs for a specific pod/container
func (s *LogStreamSimulator) SimulatePodLogs(ctx context.Context, podName, container string, logs []string) (<-chan string, <-chan error) {
	s.mu.RLock()
	pod, exists := s.pods[podName]
	s.mu.RUnlock()

	if !exists {
		errCh := make(chan error, 1)
		errCh <- fmt.Errorf("pod %s not found", podName)
		close(errCh)
		return nil, errCh
	}

	// If logs not provided, use the stored logs for the container
	if logs == nil {
		if containerLogs, ok := pod.containers[container]; ok {
			logs = containerLogs
		} else {
			// If no container specified or found, use logs from first container
			for _, containerLogs := range pod.containers {
				logs = containerLogs
				break
			}
		}
	}

	logCh := make(chan string, len(logs))
	errCh := make(chan error, 1)

	go func() {
		defer close(logCh)
		defer close(errCh)

		for _, log := range logs {
			select {
			case logCh <- log:
				// Small delay for realism, but not so much it breaks performance tests
				if len(logs) < 100 {
					time.Sleep(10 * time.Millisecond) // Normal delay for small log sets
				} else {
					time.Sleep(1 * time.Millisecond) // Minimal delay for performance tests
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return logCh, errCh
}

// SimulateMultiPodLogs simulates logs from multiple pods (e.g., deployment)
func (s *LogStreamSimulator) SimulateMultiPodLogs(ctx context.Context, deployment string, podLogs map[string][]string) map[string]<-chan string {
	result := make(map[string]<-chan string)

	for podName, logs := range podLogs {
		logCh, _ := s.SimulatePodLogs(ctx, podName, "main", logs)
		result[podName] = logCh
	}

	return result
}

// SimulateError simulates an error for a specific pod
func (s *LogStreamSimulator) SimulateError(podName string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors[podName] = err
}

// StreamingExpectation defines expected streaming behavior
type StreamingExpectation struct {
	ShouldStream       bool
	ExpectedPods       []string
	ExpectedContainers []string
	ExpectedLogLines   []string
	ExpectedErrors     []string
}

// VerifyStreamingBehavior verifies that streaming behaves as expected
func (s *LogStreamSimulator) VerifyStreamingBehavior(t *testing.T, expected StreamingExpectation) {
	t.Helper()

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Verify expected pods
	for _, expectedPod := range expected.ExpectedPods {
		if _, exists := s.pods[expectedPod]; !exists {
			t.Errorf("Expected pod %s not found in simulator", expectedPod)
		}
	}

	// Verify streaming state
	if s.streaming != expected.ShouldStream {
		t.Errorf("Expected streaming=%v, got %v", expected.ShouldStream, s.streaming)
	}
}

// MultiContainerPodSimulator simulates a pod with multiple containers
type MultiContainerPodSimulator struct {
	pod        *v1.Pod
	containers map[string]*ContainerSimulator
}

// ContainerSimulator simulates a single container
type ContainerSimulator struct {
	name      string
	logs      []string
	streaming bool
	logIndex  int
}

// NewMultiContainerPodSimulator creates a simulator for multi-container pods
func NewMultiContainerPodSimulator(name, namespace string, containerNames []string) *MultiContainerPodSimulator {
	containers := make([]v1.Container, len(containerNames))
	containerSims := make(map[string]*ContainerSimulator)

	for i, cName := range containerNames {
		containers[i] = v1.Container{
			Name:  cName,
			Image: fmt.Sprintf("test/%s:latest", cName),
		}
		containerSims[cName] = &ContainerSimulator{
			name: cName,
			logs: []string{},
		}
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID(fmt.Sprintf("uid-%s", name)),
		},
		Spec: v1.PodSpec{
			Containers: containers,
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase("Running"),
		},
	}

	return &MultiContainerPodSimulator{
		pod:        pod,
		containers: containerSims,
	}
}

// AddContainerLogs adds logs to a specific container
func (m *MultiContainerPodSimulator) AddContainerLogs(containerName string, logs []string) {
	if c, exists := m.containers[containerName]; exists {
		c.logs = append(c.logs, logs...)
	}
}

// StreamContainerLogs streams logs for a specific container
func (m *MultiContainerPodSimulator) StreamContainerLogs(ctx context.Context, containerName string) (<-chan string, error) {
	c, exists := m.containers[containerName]
	if !exists {
		return nil, fmt.Errorf("container %s not found", containerName)
	}

	logCh := make(chan string, len(c.logs))

	go func() {
		defer close(logCh)
		c.streaming = true
		defer func() { c.streaming = false }()

		for _, log := range c.logs {
			select {
			case logCh <- log:
				time.Sleep(5 * time.Millisecond)
			case <-ctx.Done():
				return
			}
		}
	}()

	return logCh, nil
}

// DeploymentLogSimulator simulates logs for all pods in a deployment
type DeploymentLogSimulator struct {
	deploymentName string
	namespace      string
	replicas       int
	pods           []*MultiContainerPodSimulator
}

// NewDeploymentLogSimulator creates a deployment log simulator
func NewDeploymentLogSimulator(name, namespace string, replicas int) *DeploymentLogSimulator {
	pods := make([]*MultiContainerPodSimulator, replicas)

	for i := 0; i < replicas; i++ {
		podName := fmt.Sprintf("%s-%d", name, i)
		pods[i] = NewMultiContainerPodSimulator(podName, namespace, []string{"main", "sidecar"})

		// Add some default logs
		pods[i].AddContainerLogs("main", []string{
			fmt.Sprintf("[Pod %d] Starting main container", i),
			fmt.Sprintf("[Pod %d] Application initialized", i),
			fmt.Sprintf("[Pod %d] Ready to serve requests", i),
		})
		pods[i].AddContainerLogs("sidecar", []string{
			fmt.Sprintf("[Pod %d] Sidecar container started", i),
			fmt.Sprintf("[Pod %d] Monitoring enabled", i),
		})
	}

	return &DeploymentLogSimulator{
		deploymentName: name,
		namespace:      namespace,
		replicas:       replicas,
		pods:           pods,
	}
}

// StreamAllPodLogs streams logs from all pods in the deployment
func (d *DeploymentLogSimulator) StreamAllPodLogs(ctx context.Context) map[string]<-chan string {
	result := make(map[string]<-chan string)

	for _, pod := range d.pods {
		logCh, _ := pod.StreamContainerLogs(ctx, "main")
		result[pod.pod.Name] = logCh
	}

	return result
}

// MultiContextLogSimulator simulates log streaming across multiple contexts
type MultiContextLogSimulator struct {
	contexts map[string]*LogStreamSimulator
}

// NewMultiContextLogSimulator creates a multi-context log simulator
func NewMultiContextLogSimulator(contexts []string) *MultiContextLogSimulator {
	m := &MultiContextLogSimulator{
		contexts: make(map[string]*LogStreamSimulator),
	}

	for _, ctx := range contexts {
		m.contexts[ctx] = NewLogStreamSimulator()
	}

	return m
}

// AddPodToContext adds a pod to a specific context
func (m *MultiContextLogSimulator) AddPodToContext(context, podName, namespace string, containers map[string][]string) {
	if sim, exists := m.contexts[context]; exists {
		sim.AddPod(podName, namespace, containers)
	}
}

// StreamLogsFromContext streams logs from a specific context
func (m *MultiContextLogSimulator) StreamLogsFromContext(ctx context.Context, contextName, podName, container string) (<-chan string, <-chan error, error) {
	sim, exists := m.contexts[contextName]
	if !exists {
		return nil, nil, fmt.Errorf("context %s not found", contextName)
	}

	logCh, errCh := sim.SimulatePodLogs(ctx, podName, container, nil)
	return logCh, errCh, nil
}

// StreamLogsFromAllContexts streams logs from all contexts
func (m *MultiContextLogSimulator) StreamLogsFromAllContexts(ctx context.Context, podName, container string) map[string]<-chan string {
	result := make(map[string]<-chan string)

	for ctxName, sim := range m.contexts {
		if logCh, _ := sim.SimulatePodLogs(ctx, podName, container, nil); logCh != nil {
			result[ctxName] = logCh
		}
	}

	return result
}

// LogStreamTestHelper provides utilities for testing log streaming
type LogStreamTestHelper struct {
	t *testing.T
}

// NewLogStreamTestHelper creates a new test helper
func NewLogStreamTestHelper(t *testing.T) *LogStreamTestHelper {
	return &LogStreamTestHelper{t: t}
}

// AssertLogsReceived asserts that expected logs are received from a channel
func (h *LogStreamTestHelper) AssertLogsReceived(logCh <-chan string, expected []string, timeout time.Duration) {
	h.t.Helper()

	received := []string{}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for i := 0; i < len(expected); i++ {
		select {
		case log := <-logCh:
			received = append(received, log)
		case <-timer.C:
			h.t.Errorf("Timeout waiting for logs. Expected %d, got %d", len(expected), len(received))
			return
		}
	}

	// Verify received logs match expected
	for i, expectedLog := range expected {
		if i >= len(received) {
			h.t.Errorf("Missing log at index %d: expected %q", i, expectedLog)
			continue
		}
		if received[i] != expectedLog {
			h.t.Errorf("Log mismatch at index %d: expected %q, got %q", i, expectedLog, received[i])
		}
	}
}

// AssertNoErrors asserts that no errors are received from error channel
func (h *LogStreamTestHelper) AssertNoErrors(errCh <-chan error, timeout time.Duration) {
	h.t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-errCh:
		if err != nil {
			h.t.Errorf("Unexpected error received: %v", err)
		}
	case <-timer.C:
		// No error received, which is expected
	}
}

// AssertStreamingStops asserts that streaming stops within timeout
func (h *LogStreamTestHelper) AssertStreamingStops(logCh <-chan string, timeout time.Duration) {
	h.t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case _, ok := <-logCh:
		if ok {
			h.t.Error("Expected channel to be closed, but received a value")
		}
	case <-timer.C:
		h.t.Error("Timeout waiting for stream to stop")
	}
}
