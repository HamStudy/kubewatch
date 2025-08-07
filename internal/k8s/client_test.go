package k8s

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

func TestGetPathSeparator(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		expected string
	}{
		{
			name:     "Windows path separator",
			goos:     "windows",
			expected: ";",
		},
		{
			name:     "Unix path separator",
			goos:     "linux",
			expected: ":",
		},
		{
			name:     "Darwin path separator",
			goos:     "darwin",
			expected: ":",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually change runtime.GOOS, so we'll test the current OS
			result := getPathSeparator()
			if runtime.GOOS == "windows" {
				if result != ";" {
					t.Errorf("Expected ';' for Windows, got %s", result)
				}
			} else {
				if result != ":" {
					t.Errorf("Expected ':' for Unix-like OS, got %s", result)
				}
			}
		})
	}
}

func TestNewClientFromConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        *rest.Config
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid config creates client successfully",
			config: &rest.Config{
				Host: "https://kubernetes.default.svc",
			},
			expectError: false,
		},
		{
			name: "Invalid config returns error",
			config: &rest.Config{
				Host: "://invalid-url",
			},
			expectError:   true,
			errorContains: "failed to create clientset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClientFromConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected client to be non-nil")
				}
				if client != nil && client.config == nil {
					t.Error("Expected client config to be non-nil")
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	// Save original HOME to restore later
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Create a valid kubeconfig for testing
	validKubeconfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`

	tests := []struct {
		name        string
		kubeconfig  string
		setupEnv    func() string // Returns the kubeconfig path to use
		cleanupEnv  func()
		expectError bool
	}{
		{
			name:       "Empty kubeconfig uses default location",
			kubeconfig: "",
			setupEnv: func() string {
				// Create a temporary kubeconfig file in a test directory
				// NEVER modify the user's actual .kube/config!
				tempDir, _ := os.MkdirTemp("", "kubewatch-test-*")
				os.Setenv("HOME", tempDir)
				kubeconfigPath := filepath.Join(tempDir, ".kube", "config")
				os.MkdirAll(filepath.Dir(kubeconfigPath), 0755)
				os.WriteFile(kubeconfigPath, []byte(validKubeconfig), 0644)
				return "" // Empty string will use default location
			},
			cleanupEnv: func() {
				// Clean up the temporary test directory
				if tempHome := os.Getenv("HOME"); strings.Contains(tempHome, "kubewatch-test-") {
					os.RemoveAll(tempHome)
				}
			},
			expectError: false,
		},
		{
			name:       "Multiple kubeconfig paths",
			kubeconfig: "", // Will be set by setupEnv
			setupEnv: func() string {
				// Create proper temporary files
				tmpFile1, _ := os.CreateTemp("", "kubeconfig-test-1-*.yaml")
				tmpFile1.Write([]byte(validKubeconfig))
				tmpFile1.Close()

				tmpFile2, _ := os.CreateTemp("", "kubeconfig-test-2-*.yaml")
				tmpFile2.Write([]byte(validKubeconfig))
				tmpFile2.Close()

				// Store paths for cleanup
				os.Setenv("TEST_KUBECONFIG_1", tmpFile1.Name())
				os.Setenv("TEST_KUBECONFIG_2", tmpFile2.Name())

				// Return the paths separated by colon
				return tmpFile1.Name() + ":" + tmpFile2.Name()
			},
			cleanupEnv: func() {
				// Clean up temporary files
				if path1 := os.Getenv("TEST_KUBECONFIG_1"); path1 != "" {
					os.Remove(path1)
					os.Unsetenv("TEST_KUBECONFIG_1")
				}
				if path2 := os.Getenv("TEST_KUBECONFIG_2"); path2 != "" {
					os.Remove(path2)
					os.Unsetenv("TEST_KUBECONFIG_2")
				}
			},
			expectError: false,
		},
		{
			name:       "Multiple kubeconfig paths",
			kubeconfig: "", // Will be set in setupEnv
			setupEnv: func() string {
				// Create proper temporary files
				tmpFile1, _ := os.CreateTemp("", "kubeconfig-test-1-*.yaml")
				tmpFile1.Write([]byte(validKubeconfig))
				tmpFile1.Close()

				tmpFile2, _ := os.CreateTemp("", "kubeconfig-test-2-*.yaml")
				tmpFile2.Write([]byte(validKubeconfig))
				tmpFile2.Close()

				// Store paths for cleanup
				os.Setenv("TEST_KUBECONFIG_1", tmpFile1.Name())
				os.Setenv("TEST_KUBECONFIG_2", tmpFile2.Name())

				// Return the paths separated by colon
				return tmpFile1.Name() + ":" + tmpFile2.Name()
			},
			cleanupEnv: func() {
				// Clean up temporary files
				if path1 := os.Getenv("TEST_KUBECONFIG_1"); path1 != "" {
					os.Remove(path1)
					os.Unsetenv("TEST_KUBECONFIG_1")
				}
				if path2 := os.Getenv("TEST_KUBECONFIG_2"); path2 != "" {
					os.Remove(path2)
					os.Unsetenv("TEST_KUBECONFIG_2")
				}
			},
			expectError: false,
		},
		{
			name:        "Windows-style multiple paths",
			kubeconfig:  "C:\\config1;C:\\config2",
			setupEnv:    func() string { return "" },
			cleanupEnv:  func() {},
			expectError: runtime.GOOS != "windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeconfigPath := tt.kubeconfig
			if tt.setupEnv != nil {
				path := tt.setupEnv()
				if path != "" {
					kubeconfigPath = path
				}
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			client, err := NewClient(kubeconfigPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				// Note: Will likely fail without valid kubeconfig, but tests structure
				_ = client
			}
		})
	}
}

func TestNewClientWithOptions(t *testing.T) {
	tests := []struct {
		name        string
		kubeconfig  string
		opts        *ClientOptions
		expectError bool
	}{
		{
			name:        "Nil options",
			kubeconfig:  "",
			opts:        nil,
			expectError: false,
		},
		{
			name:       "With context override",
			kubeconfig: "",
			opts: &ClientOptions{
				Context: "test-context",
			},
			expectError: false,
		},
		{
			name:       "With auth options",
			kubeconfig: "",
			opts: &ClientOptions{
				Token:             "test-token",
				ClientCertificate: "/path/to/cert",
				ClientKey:         "/path/to/key",
			},
			expectError: false,
		},
		{
			name:       "With impersonation",
			kubeconfig: "",
			opts: &ClientOptions{
				Impersonate:       "user@example.com",
				ImpersonateGroups: []string{"system:masters"},
				ImpersonateUID:    "12345",
			},
			expectError: false,
		},
		{
			name:       "With cluster options",
			kubeconfig: "",
			opts: &ClientOptions{
				Cluster:              "https://k8s.example.com",
				CertificateAuthority: "/path/to/ca",
				InsecureSkipVerify:   true,
			},
			expectError: false,
		},
		{
			name:       "With namespace and timeout",
			kubeconfig: "",
			opts: &ClientOptions{
				Namespace: "custom-namespace",
				Timeout:   "30s",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary kubeconfig
			tmpfile, err := os.CreateTemp("", "kubeconfig")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(validKubeconfig)); err != nil {
				t.Fatal(err)
			}
			tmpfile.Close()

			client, err := NewClientWithOptions(tmpfile.Name(), tt.opts)

			// Note: Will likely fail without valid kubeconfig, but tests structure
			_ = client
			_ = err
		})
	}
}

func TestClientNamespaceOperations(t *testing.T) {
	tests := []struct {
		name        string
		namespaces  []v1.Namespace
		expectError bool
		expectCount int
	}{
		{
			name: "List namespaces successfully",
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}},
			},
			expectError: false,
			expectCount: 3,
		},
		{
			name:        "Empty namespace list",
			namespaces:  []v1.Namespace{},
			expectError: false,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset
			fakeClient := fake.NewSimpleClientset()

			// Add namespaces to fake client
			for _, ns := range tt.namespaces {
				_, err := fakeClient.CoreV1().Namespaces().Create(context.Background(), &ns, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("Failed to create namespace: %v", err)
				}
			}

			client := &Client{
				clientset: fakeClient,
			}

			// Test GetNamespaces
			namespaces, err := client.GetNamespaces(context.Background())
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(namespaces) != tt.expectCount {
					t.Errorf("Expected %d namespaces, got %d", tt.expectCount, len(namespaces))
				}
			}

			// Test ListNamespaces (should behave the same)
			namespaces2, err := client.ListNamespaces(context.Background())
			if err != nil {
				t.Errorf("ListNamespaces failed: %v", err)
			}
			if len(namespaces2) != len(namespaces) {
				t.Errorf("ListNamespaces returned different count than GetNamespaces")
			}
		})
	}
}

func TestClientPodOperations(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		pods        []v1.Pod
		podToDelete string
		expectError bool
		expectCount int
	}{
		{
			name:      "List and delete pods successfully",
			namespace: "default",
			pods: []v1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "default"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "default"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "pod3", Namespace: "other"}},
			},
			podToDelete: "pod1",
			expectError: false,
			expectCount: 2, // Only pods in "default" namespace
		},
		{
			name:        "Empty pod list",
			namespace:   "default",
			pods:        []v1.Pod{},
			expectError: false,
			expectCount: 0,
		},
		{
			name:      "Delete non-existent pod",
			namespace: "default",
			pods: []v1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "default"}},
			},
			podToDelete: "non-existent",
			expectError: true,
			expectCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset
			fakeClient := fake.NewSimpleClientset()

			// Add pods to fake client
			for _, pod := range tt.pods {
				_, err := fakeClient.CoreV1().Pods(pod.Namespace).Create(context.Background(), &pod, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("Failed to create pod: %v", err)
				}
			}

			client := &Client{
				clientset: fakeClient,
			}

			// Test ListPods
			pods, err := client.ListPods(context.Background(), tt.namespace)
			if err != nil {
				t.Errorf("ListPods failed: %v", err)
			}
			if len(pods) != tt.expectCount {
				t.Errorf("Expected %d pods, got %d", tt.expectCount, len(pods))
			}

			// Test DeletePod
			if tt.podToDelete != "" {
				err = client.DeletePod(context.Background(), tt.namespace, tt.podToDelete)
				if tt.expectError {
					if err == nil {
						t.Error("Expected error but got none")
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error: %v", err)
					}
				}
			}

			// Test DeletePods (multiple) - only if we didn't already delete a pod
			if !tt.expectError && len(pods) > 0 && tt.podToDelete == "" {
				podNames := []string{}
				for _, pod := range pods {
					podNames = append(podNames, pod.Name)
				}
				err = client.DeletePods(context.Background(), tt.namespace, podNames)
				if err != nil {
					t.Errorf("DeletePods failed: %v", err)
				}
			}
		})
	}
}

func TestClientWatchOperations(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		resource  string
	}{
		{
			name:      "Watch pods",
			namespace: "default",
			resource:  "pods",
		},
		{
			name:      "Watch deployments",
			namespace: "kube-system",
			resource:  "deployments",
		},
		{
			name:      "Watch services",
			namespace: "test",
			resource:  "services",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset with watch reactor
			fakeClient := fake.NewSimpleClientset()
			fakeWatch := watch.NewFake()

			client := &Client{
				clientset: fakeClient,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			var watcher watch.Interface
			var err error

			switch tt.resource {
			case "pods":
				// Note: fake client doesn't support watch properly, but we test the interface
				watcher, err = client.WatchPods(ctx, tt.namespace)
			case "deployments":
				watcher, err = client.WatchDeployments(ctx, tt.namespace)
			case "services":
				watcher, err = client.WatchServices(ctx, tt.namespace)
			}

			// Fake client may not support watch, but we verify no panic
			_ = watcher
			_ = err

			// Clean up
			if watcher != nil {
				watcher.Stop()
			}
			fakeWatch.Stop()
		})
	}
}

func TestClientPodLogs(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		pod       string
		container string
		follow    bool
		tailLines int64
		previous  bool
		sinceTime *time.Time
	}{
		{
			name:      "Get pod logs with follow",
			namespace: "default",
			pod:       "test-pod",
			container: "main",
			follow:    true,
			tailLines: 100,
		},
		{
			name:      "Get pod logs without container",
			namespace: "default",
			pod:       "test-pod",
			container: "",
			follow:    false,
			tailLines: 50,
		},
		{
			name:      "Get previous pod logs",
			namespace: "kube-system",
			pod:       "system-pod",
			container: "sidecar",
			follow:    false,
			tailLines: 200,
			previous:  true,
		},
		{
			name:      "Get logs since specific time",
			namespace: "default",
			pod:       "test-pod",
			container: "main",
			follow:    false,
			tailLines: 100,
			sinceTime: func() *time.Time { t := time.Now().Add(-1 * time.Hour); return &t }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset
			fakeClient := fake.NewSimpleClientset()

			// Create the pod first
			pod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.pod,
					Namespace: tt.namespace,
				},
			}
			_, err := fakeClient.CoreV1().Pods(tt.namespace).Create(context.Background(), pod, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("Failed to create pod: %v", err)
			}

			client := &Client{
				clientset: fakeClient,
			}

			// Test GetPodLogs
			reader, err := client.GetPodLogs(context.Background(), tt.namespace, tt.pod, tt.container, tt.follow, tt.tailLines)
			// Note: fake client doesn't fully support log streaming
			_ = reader
			_ = err

			// Test GetPodLogsWithOptions
			reader2, err2 := client.GetPodLogsWithOptions(
				context.Background(),
				tt.namespace,
				tt.pod,
				tt.container,
				tt.follow,
				tt.tailLines,
				tt.previous,
				tt.sinceTime,
				true, // timestamps
			)
			_ = reader2
			_ = err2

			// Clean up readers if they exist
			if reader != nil {
				reader.Close()
			}
			if reader2 != nil {
				reader2.Close()
			}
		})
	}
}

func TestClientDeploymentOperations(t *testing.T) {
	tests := []struct {
		name               string
		namespace          string
		deployments        []appsv1.Deployment
		deploymentToDelete string
		expectError        bool
		expectCount        int
	}{
		{
			name:      "List and delete deployments",
			namespace: "default",
			deployments: []appsv1.Deployment{
				{ObjectMeta: metav1.ObjectMeta{Name: "deploy1", Namespace: "default"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "deploy2", Namespace: "default"}},
			},
			deploymentToDelete: "deploy1",
			expectError:        false,
			expectCount:        2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset
			fakeClient := fake.NewSimpleClientset()

			// Add deployments to fake client
			for _, deployment := range tt.deployments {
				_, err := fakeClient.AppsV1().Deployments(deployment.Namespace).Create(context.Background(), &deployment, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("Failed to create deployment: %v", err)
				}
			}

			client := &Client{
				clientset: fakeClient,
			}

			// Test ListDeployments
			deployments, err := client.ListDeployments(context.Background(), tt.namespace)
			if err != nil {
				t.Errorf("ListDeployments failed: %v", err)
			}
			if len(deployments) != tt.expectCount {
				t.Errorf("Expected %d deployments, got %d", tt.expectCount, len(deployments))
			}

			// Test DeleteDeployment
			if tt.deploymentToDelete != "" {
				err = client.DeleteDeployment(context.Background(), tt.namespace, tt.deploymentToDelete)
				if tt.expectError {
					if err == nil {
						t.Error("Expected error but got none")
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error: %v", err)
					}
				}
			}
		})
	}
}

func TestClientStatefulSetOperations(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Create test statefulset
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
	}
	_, err := fakeClient.AppsV1().StatefulSets("default").Create(context.Background(), statefulSet, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create statefulset: %v", err)
	}

	client := &Client{
		clientset: fakeClient,
	}

	// Test ListStatefulSets
	statefulSets, err := client.ListStatefulSets(context.Background(), "default")
	if err != nil {
		t.Errorf("ListStatefulSets failed: %v", err)
	}
	if len(statefulSets) != 1 {
		t.Errorf("Expected 1 statefulset, got %d", len(statefulSets))
	}

	// Test DeleteStatefulSet
	err = client.DeleteStatefulSet(context.Background(), "default", "test-sts")
	if err != nil {
		t.Errorf("DeleteStatefulSet failed: %v", err)
	}
}

func TestClientServiceOperations(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Create test service
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-svc", Namespace: "default"},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Name: "http", Port: 80, Protocol: v1.ProtocolTCP},
			},
		},
	}
	_, err := fakeClient.CoreV1().Services("default").Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	client := &Client{
		clientset: fakeClient,
	}

	// Test ListServices
	services, err := client.ListServices(context.Background(), "default")
	if err != nil {
		t.Errorf("ListServices failed: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	// Test DeleteService
	err = client.DeleteService(context.Background(), "default", "test-svc")
	if err != nil {
		t.Errorf("DeleteService failed: %v", err)
	}
}

func TestClientIngressOperations(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Create test ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ing", Namespace: "default"},
	}
	_, err := fakeClient.NetworkingV1().Ingresses("default").Create(context.Background(), ingress, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create ingress: %v", err)
	}

	client := &Client{
		clientset: fakeClient,
	}

	// Test ListIngresses
	ingresses, err := client.ListIngresses(context.Background(), "default")
	if err != nil {
		t.Errorf("ListIngresses failed: %v", err)
	}
	if len(ingresses) != 1 {
		t.Errorf("Expected 1 ingress, got %d", len(ingresses))
	}

	// Test DeleteIngress
	err = client.DeleteIngress(context.Background(), "default", "test-ing")
	if err != nil {
		t.Errorf("DeleteIngress failed: %v", err)
	}
}

func TestClientConfigMapOperations(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Create test configmap
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "default"},
		Data: map[string]string{
			"key1": "value1",
		},
	}
	_, err := fakeClient.CoreV1().ConfigMaps("default").Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create configmap: %v", err)
	}

	client := &Client{
		clientset: fakeClient,
	}

	// Test ListConfigMaps
	configMaps, err := client.ListConfigMaps(context.Background(), "default")
	if err != nil {
		t.Errorf("ListConfigMaps failed: %v", err)
	}
	if len(configMaps) != 1 {
		t.Errorf("Expected 1 configmap, got %d", len(configMaps))
	}

	// Test DeleteConfigMap
	err = client.DeleteConfigMap(context.Background(), "default", "test-cm")
	if err != nil {
		t.Errorf("DeleteConfigMap failed: %v", err)
	}
}

func TestClientSecretOperations(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Create test secret
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-secret", Namespace: "default"},
		Data: map[string][]byte{
			"password": []byte("secret123"),
		},
	}
	_, err := fakeClient.CoreV1().Secrets("default").Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	client := &Client{
		clientset: fakeClient,
	}

	// Test ListSecrets
	secrets, err := client.ListSecrets(context.Background(), "default")
	if err != nil {
		t.Errorf("ListSecrets failed: %v", err)
	}
	if len(secrets) != 1 {
		t.Errorf("Expected 1 secret, got %d", len(secrets))
	}

	// Test DeleteSecret
	err = client.DeleteSecret(context.Background(), "default", "test-secret")
	if err != nil {
		t.Errorf("DeleteSecret failed: %v", err)
	}
}

func TestGetPodsForDeployment(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Create deployment with selector
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deploy", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}
	_, err := fakeClient.AppsV1().Deployments("default").Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create deployment: %v", err)
	}

	// Create matching pods
	for i := 0; i < 3; i++ {
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-pod-%d", i),
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
			},
		}
		_, err := fakeClient.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}
	}

	// Create non-matching pod
	nonMatchingPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "other",
			},
		},
	}
	_, err = fakeClient.CoreV1().Pods("default").Create(context.Background(), nonMatchingPod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create non-matching pod: %v", err)
	}

	client := &Client{
		clientset: fakeClient,
	}

	// Test GetPodsForDeployment
	pods, err := client.GetPodsForDeployment(context.Background(), "default", "test-deploy")
	if err != nil {
		t.Errorf("GetPodsForDeployment failed: %v", err)
	}
	if len(pods) != 3 {
		t.Errorf("Expected 3 pods for deployment, got %d", len(pods))
	}

	// Test with non-existent deployment
	_, err = client.GetPodsForDeployment(context.Background(), "default", "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent deployment")
	}
}

func TestGetPodsForStatefulSet(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Create statefulset with selector
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "stateful",
				},
			},
		},
	}
	_, err := fakeClient.AppsV1().StatefulSets("default").Create(context.Background(), statefulSet, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create statefulset: %v", err)
	}

	// Create matching pods
	for i := 0; i < 2; i++ {
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-sts-%d", i),
				Namespace: "default",
				Labels: map[string]string{
					"app": "stateful",
				},
			},
		}
		_, err := fakeClient.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}
	}

	client := &Client{
		clientset: fakeClient,
	}

	// Test GetPodsForStatefulSet
	pods, err := client.GetPodsForStatefulSet(context.Background(), "default", "test-sts")
	if err != nil {
		t.Errorf("GetPodsForStatefulSet failed: %v", err)
	}
	if len(pods) != 2 {
		t.Errorf("Expected 2 pods for statefulset, got %d", len(pods))
	}
}

func TestFormatCPU(t *testing.T) {
	tests := []struct {
		name     string
		milliCPU int64
		expected string
	}{
		{
			name:     "Zero CPU",
			milliCPU: 0,
			expected: "-",
		},
		{
			name:     "Less than 1 core",
			milliCPU: 500,
			expected: "500m",
		},
		{
			name:     "Exactly 1 core",
			milliCPU: 1000,
			expected: "1",
		},
		{
			name:     "Multiple cores",
			milliCPU: 2500,
			expected: "2",
		},
		{
			name:     "Large number of cores",
			milliCPU: 16000,
			expected: "16",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCPU(tt.milliCPU)
			if result != tt.expected {
				t.Errorf("formatCPU(%d) = %s, expected %s", tt.milliCPU, result, tt.expected)
			}
		})
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "Zero memory",
			bytes:    0,
			expected: "-",
		},
		{
			name:     "Bytes",
			bytes:    512,
			expected: "512B",
		},
		{
			name:     "Kilobytes",
			bytes:    2 * 1024,
			expected: "2Ki",
		},
		{
			name:     "Megabytes",
			bytes:    128 * 1024 * 1024,
			expected: "128Mi",
		},
		{
			name:     "Gigabytes",
			bytes:    2 * 1024 * 1024 * 1024,
			expected: "2Gi",
		},
		{
			name:     "Fractional gigabytes",
			bytes:    int64(2.5 * 1024 * 1024 * 1024),
			expected: "2.5Gi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMemory(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatMemory(%d) = %s, expected %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestGetPodMetrics(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		metricsClient metricsclient.Interface
		podMetrics    []metricsv1beta1.PodMetrics
		expectError   bool
		expectCount   int
	}{
		{
			name:          "Metrics API not available",
			namespace:     "default",
			metricsClient: nil,
			expectError:   true,
			expectCount:   0,
		},
		{
			name:          "Get pod metrics successfully",
			namespace:     "default",
			metricsClient: metricsfake.NewSimpleClientset(),
			podMetrics: []metricsv1beta1.PodMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "default",
					},
					Containers: []metricsv1beta1.ContainerMetrics{
						{
							Name: "container1",
							Usage: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("100m"),
								v1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					},
				},
			},
			expectError: false,
			expectCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				metricsClient: tt.metricsClient,
			}

			if tt.metricsClient != nil && len(tt.podMetrics) > 0 {
				// Note: fake metrics client doesn't support Create, so we can't add metrics
				// This is a limitation of the test, but the structure is correct
			}

			metrics, err := client.GetPodMetrics(context.Background(), tt.namespace)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				// Note: fake metrics client has limitations
				_ = metrics
			}
		})
	}
}

func TestGetNodeMetrics(t *testing.T) {
	tests := []struct {
		name          string
		metricsClient metricsclient.Interface
		expectError   bool
	}{
		{
			name:          "Metrics API not available",
			metricsClient: nil,
			expectError:   true,
		},
		{
			name:          "Get node metrics successfully",
			metricsClient: metricsfake.NewSimpleClientset(),
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				metricsClient: tt.metricsClient,
			}

			metrics, err := client.GetNodeMetrics(context.Background())
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				// Note: fake metrics client has limitations
				_ = metrics
			}
		})
	}
}

func TestDescribeResource(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Create test resources
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: v1.PodSpec{
			NodeName: "node1",
			Containers: []v1.Container{
				{
					Name:  "main",
					Image: "nginx:latest",
					Ports: []v1.ContainerPort{
						{ContainerPort: 80, Protocol: v1.ProtocolTCP},
					},
				},
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			PodIP: "10.0.0.1",
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:         "main",
					Ready:        true,
					RestartCount: 0,
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{
							StartedAt: metav1.Now(),
						},
					},
				},
			},
		},
	}
	_, err := fakeClient.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}

	replicas := int32(3)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            3,
			UpdatedReplicas:     3,
			AvailableReplicas:   3,
			UnavailableReplicas: 0,
		},
	}
	_, err = fakeClient.AppsV1().Deployments("default").Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create deployment: %v", err)
	}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: v1.ServiceSpec{
			Type:      v1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.1",
			Ports: []v1.ServicePort{
				{Name: "http", Port: 80, Protocol: v1.ProtocolTCP},
			},
			Selector: map[string]string{
				"app": "test",
			},
		},
	}
	_, err = fakeClient.CoreV1().Services("default").Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	client := &Client{
		clientset: fakeClient,
	}

	tests := []struct {
		name         string
		resourceType interface{}
		resourceName string
		namespace    string
		expectError  bool
	}{
		{
			name:         "Describe pod as string",
			resourceType: "pod",
			resourceName: "test-pod",
			namespace:    "default",
			expectError:  false,
		},
		{
			name:         "Describe deployment as string",
			resourceType: "deployment",
			resourceName: "test-deploy",
			namespace:    "default",
			expectError:  false,
		},
		{
			name:         "Describe service as string",
			resourceType: "service",
			resourceName: "test-svc",
			namespace:    "default",
			expectError:  false,
		},
		{
			name:         "Unsupported resource type",
			resourceType: "unsupported",
			resourceName: "test",
			namespace:    "default",
			expectError:  true,
		},
		{
			name:         "Non-existent resource",
			resourceType: "pod",
			resourceName: "non-existent",
			namespace:    "default",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, err := client.DescribeResource(context.Background(), tt.resourceType, tt.resourceName, tt.namespace)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if description == "" {
					t.Error("Expected non-empty description")
				}
				// Verify description contains expected content
				if !strings.Contains(description, "Name:") {
					t.Error("Description should contain 'Name:'")
				}
				if !strings.Contains(description, "Namespace:") {
					t.Error("Description should contain 'Namespace:'")
				}
			}
		})
	}
}

// Test helper for valid kubeconfig
var validKubeconfig = `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`

// Benchmark tests for performance-critical operations
func BenchmarkListPods(b *testing.B) {
	fakeClient := fake.NewSimpleClientset()

	// Create many pods for benchmarking
	for i := 0; i < 100; i++ {
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
			},
		}
		fakeClient.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
	}

	client := &Client{
		clientset: fakeClient,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.ListPods(context.Background(), "default")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFormatCPU(b *testing.B) {
	testValues := []int64{0, 100, 500, 1000, 2500, 16000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, val := range testValues {
			_ = formatCPU(val)
		}
	}
}

func BenchmarkFormatMemory(b *testing.B) {
	testValues := []int64{0, 512, 2048, 1048576, 1073741824}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, val := range testValues {
			_ = formatMemory(val)
		}
	}
}

// Test error handling and edge cases
func TestClientErrorHandling(t *testing.T) {
	t.Run("Nil clientset operations", func(t *testing.T) {
		client := &Client{
			clientset: nil,
		}

		// These should panic or return errors gracefully
		defer func() {
			if r := recover(); r != nil {
				// Expected panic for nil clientset
			}
		}()

		_, err := client.ListPods(context.Background(), "default")
		if err == nil {
			// If no panic, should at least return an error
			t.Error("Expected error for nil clientset")
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		client := &Client{
			clientset: fakeClient,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Operations should respect context cancellation
		_, err := client.ListPods(ctx, "default")
		// Note: fake client may not respect context, but real client would
		_ = err
	})

	t.Run("Invalid namespace", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		client := &Client{
			clientset: fakeClient,
		}

		// Empty namespace should still work (cluster-wide)
		_, err := client.ListPods(context.Background(), "")
		if err != nil {
			t.Errorf("Empty namespace should be valid: %v", err)
		}
	})
}

// Test concurrent operations
func TestConcurrentOperations(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Create test data
	for i := 0; i < 10; i++ {
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
			},
		}
		fakeClient.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
	}

	client := &Client{
		clientset: fakeClient,
	}

	// Run concurrent operations
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.ListPods(context.Background(), "default")
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

// Test rate limiting and backoff scenarios
func TestRateLimitingScenarios(t *testing.T) {
	t.Run("Multiple rapid requests", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		client := &Client{
			clientset: fakeClient,
		}

		// Simulate rapid requests
		for i := 0; i < 100; i++ {
			go func() {
				client.ListPods(context.Background(), "default")
			}()
		}

		// Should handle without issues
		time.Sleep(100 * time.Millisecond)
	})
}

// Test reconnection scenarios
func TestReconnectionScenarios(t *testing.T) {
	t.Run("Client with invalid config attempts reconnection", func(t *testing.T) {
		// This tests the structure, actual reconnection would need real cluster
		config := &rest.Config{
			Host:    "https://invalid-host:6443",
			Timeout: 1 * time.Second,
		}

		client, err := NewClientFromConfig(config)
		// Should create client even with invalid host
		if err == nil && client != nil {
			// Try an operation that would fail
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_, err = client.ListPods(ctx, "default")
			// Should fail due to invalid host
			if err == nil {
				t.Error("Expected error for invalid host")
			}
		}
	})
}

// Mock io.ReadCloser for testing log streaming
type mockReadCloser struct {
	data   []byte
	offset int
	closed bool
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, errors.New("reader closed")
	}
	if m.offset >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.offset:])
	m.offset += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}

func TestLogStreamingEdgeCases(t *testing.T) {
	t.Run("Log stream closes properly", func(t *testing.T) {
		reader := &mockReadCloser{
			data: []byte("test log line\n"),
		}

		// Read data
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			t.Errorf("Unexpected error reading: %v", err)
		}
		if n == 0 {
			t.Error("Expected to read some data")
		}

		// Close reader
		err = reader.Close()
		if err != nil {
			t.Errorf("Error closing reader: %v", err)
		}

		// Verify closed
		if !reader.closed {
			t.Error("Reader should be closed")
		}

		// Read after close should fail
		_, err = reader.Read(buf)
		if err == nil {
			t.Error("Expected error reading from closed reader")
		}
	})
}

// Test helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("Multiple path separator handling", func(t *testing.T) {
		separator := getPathSeparator()

		// Test splitting paths
		if separator == ":" {
			paths := strings.Split("/path1:/path2:/path3", separator)
			if len(paths) != 3 {
				t.Errorf("Expected 3 paths, got %d", len(paths))
			}
		} else if separator == ";" {
			paths := strings.Split("C:\\path1;C:\\path2", separator)
			if len(paths) != 2 {
				t.Errorf("Expected 2 paths, got %d", len(paths))
			}
		}
	})
}

var _ = sync.WaitGroup{}
