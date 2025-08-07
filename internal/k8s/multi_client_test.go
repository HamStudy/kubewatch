package k8s

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestNewMultiContextClient(t *testing.T) {
	tests := []struct {
		name          string
		contexts      []string
		setupConfig   func()
		expectError   bool
		errorContains string
	}{
		{
			name:          "No contexts provided",
			contexts:      []string{},
			expectError:   true,
			errorContains: "no contexts specified",
		},
		{
			name:        "Single context",
			contexts:    []string{"test-context"},
			expectError: true, // Will fail without valid kubeconfig
		},
		{
			name:        "Multiple contexts",
			contexts:    []string{"context1", "context2", "context3"},
			expectError: true, // Will fail without valid kubeconfig
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupConfig != nil {
				tt.setupConfig()
			}

			client, err := NewMultiContextClient(tt.contexts)

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
			}
		})
	}
}

func TestNewClientWithContext(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() (*clientcmd.ClientConfigLoadingRules, *clientcmd.ConfigOverrides)
		expectError bool
	}{
		{
			name: "Valid context configuration",
			setupConfig: func() (*clientcmd.ClientConfigLoadingRules, *clientcmd.ConfigOverrides) {
				rules := clientcmd.NewDefaultClientConfigLoadingRules()
				overrides := &clientcmd.ConfigOverrides{
					CurrentContext: "test-context",
				}
				return rules, overrides
			},
			expectError: true, // Will fail without valid kubeconfig
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, overrides := tt.setupConfig()
			_, err := NewClientWithContext(rules, overrides)

			// Note: Will fail without valid kubeconfig
			_ = err
		})
	}
}

func TestMultiContextClientGetContexts(t *testing.T) {
	// Create a mock multi-context client
	mc := &MultiContextClient{
		contexts: []string{"context1", "context2", "context3"},
		clients:  make(map[string]*Client),
	}

	contexts := mc.GetContexts()

	if len(contexts) != 3 {
		t.Errorf("Expected 3 contexts, got %d", len(contexts))
	}

	// Verify contexts are returned correctly
	expectedContexts := map[string]bool{
		"context1": false,
		"context2": false,
		"context3": false,
	}

	for _, ctx := range contexts {
		if _, exists := expectedContexts[ctx]; !exists {
			t.Errorf("Unexpected context: %s", ctx)
		}
		expectedContexts[ctx] = true
	}

	for ctx, found := range expectedContexts {
		if !found {
			t.Errorf("Missing expected context: %s", ctx)
		}
	}
}

func TestMultiContextClientGetClient(t *testing.T) {
	// Create fake clients
	fakeClient1 := &Client{
		clientset: fake.NewSimpleClientset(),
	}
	fakeClient2 := &Client{
		clientset: fake.NewSimpleClientset(),
	}

	mc := &MultiContextClient{
		contexts: []string{"context1", "context2"},
		clients: map[string]*Client{
			"context1": fakeClient1,
			"context2": fakeClient2,
		},
	}

	tests := []struct {
		name        string
		context     string
		expectError bool
	}{
		{
			name:        "Get existing client",
			context:     "context1",
			expectError: false,
		},
		{
			name:        "Get another existing client",
			context:     "context2",
			expectError: false,
		},
		{
			name:        "Get non-existent client",
			context:     "context3",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := mc.GetClient(tt.context)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if !strings.Contains(err.Error(), "no client for context") {
					t.Errorf("Expected error about missing client, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected client to be non-nil")
				}
			}
		})
	}
}

func TestListPodsAllContexts(t *testing.T) {
	// Create fake clients with different pods
	fakeClient1 := fake.NewSimpleClientset()
	fakeClient2 := fake.NewSimpleClientset()

	// Add pods to context1
	pod1 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
		},
	}
	pod2 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: "default",
		},
	}
	fakeClient1.CoreV1().Pods("default").Create(context.Background(), pod1, metav1.CreateOptions{})
	fakeClient1.CoreV1().Pods("default").Create(context.Background(), pod2, metav1.CreateOptions{})

	// Add pods to context2
	pod3 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod3",
			Namespace: "default",
		},
	}
	fakeClient2.CoreV1().Pods("default").Create(context.Background(), pod3, metav1.CreateOptions{})

	mc := &MultiContextClient{
		contexts: []string{"context1", "context2"},
		clients: map[string]*Client{
			"context1": {clientset: fakeClient1},
			"context2": {clientset: fakeClient2},
		},
	}

	tests := []struct {
		name        string
		namespace   string
		expectCount int
		expectError bool
	}{
		{
			name:        "List pods from all contexts",
			namespace:   "default",
			expectCount: 3, // 2 from context1 + 1 from context2
			expectError: false,
		},
		{
			name:        "List pods from non-existent namespace",
			namespace:   "non-existent",
			expectCount: 0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pods, err := mc.ListPodsAllContexts(context.Background(), tt.namespace)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil && !strings.Contains(err.Error(), "errors from") {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(pods) != tt.expectCount {
					t.Errorf("Expected %d pods, got %d", tt.expectCount, len(pods))
				}

				// Verify context information is included
				contextCounts := make(map[string]int)
				for _, pod := range pods {
					contextCounts[pod.Context]++
				}

				if tt.expectCount > 0 {
					if contextCounts["context1"] != 2 {
						t.Errorf("Expected 2 pods from context1, got %d", contextCounts["context1"])
					}
					if contextCounts["context2"] != 1 {
						t.Errorf("Expected 1 pod from context2, got %d", contextCounts["context2"])
					}
				}
			}
		})
	}
}

func TestListDeploymentsAllContexts(t *testing.T) {
	// Create fake clients with different deployments
	fakeClient1 := fake.NewSimpleClientset()
	fakeClient2 := fake.NewSimpleClientset()

	// Add deployments to context1
	deploy1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy1",
			Namespace: "default",
		},
	}
	fakeClient1.AppsV1().Deployments("default").Create(context.Background(), deploy1, metav1.CreateOptions{})

	// Add deployments to context2
	deploy2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy2",
			Namespace: "default",
		},
	}
	deploy3 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy3",
			Namespace: "default",
		},
	}
	fakeClient2.AppsV1().Deployments("default").Create(context.Background(), deploy2, metav1.CreateOptions{})
	fakeClient2.AppsV1().Deployments("default").Create(context.Background(), deploy3, metav1.CreateOptions{})

	mc := &MultiContextClient{
		contexts: []string{"context1", "context2"},
		clients: map[string]*Client{
			"context1": {clientset: fakeClient1},
			"context2": {clientset: fakeClient2},
		},
	}

	deployments, err := mc.ListDeploymentsAllContexts(context.Background(), "default")
	if err != nil && !strings.Contains(err.Error(), "errors from") {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(deployments) != 3 {
		t.Errorf("Expected 3 deployments, got %d", len(deployments))
	}

	// Verify context information
	contextCounts := make(map[string]int)
	for _, deployment := range deployments {
		contextCounts[deployment.Context]++
	}

	if contextCounts["context1"] != 1 {
		t.Errorf("Expected 1 deployment from context1, got %d", contextCounts["context1"])
	}
	if contextCounts["context2"] != 2 {
		t.Errorf("Expected 2 deployments from context2, got %d", contextCounts["context2"])
	}
}

func TestConcurrentContextOperations(t *testing.T) {
	// Create multiple fake clients
	numContexts := 5
	clients := make(map[string]*Client)
	contexts := make([]string, numContexts)

	for i := 0; i < numContexts; i++ {
		contextName := fmt.Sprintf("context%d", i)
		contexts[i] = contextName

		fakeClient := fake.NewSimpleClientset()
		// Add some pods to each context
		for j := 0; j < 3; j++ {
			pod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("pod-%d-%d", i, j),
					Namespace: "default",
				},
			}
			fakeClient.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
		}

		clients[contextName] = &Client{clientset: fakeClient}
	}

	mc := &MultiContextClient{
		contexts: contexts,
		clients:  clients,
	}

	// Test concurrent access to different contexts
	var wg sync.WaitGroup
	errors := make(chan error, numContexts)

	for _, ctx := range contexts {
		wg.Add(1)
		go func(contextName string) {
			defer wg.Done()

			client, err := mc.GetClient(contextName)
			if err != nil {
				errors <- err
				return
			}

			pods, err := client.ListPods(context.Background(), "default")
			if err != nil {
				errors <- err
				return
			}

			if len(pods) != 3 {
				errors <- fmt.Errorf("context %s: expected 3 pods, got %d", contextName, len(pods))
			}
		}(ctx)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

func TestListPodsAllContextsWithErrors(t *testing.T) {
	// Create one successful client and one that will fail
	fakeClient1 := fake.NewSimpleClientset()
	pod1 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
		},
	}
	fakeClient1.CoreV1().Pods("default").Create(context.Background(), pod1, metav1.CreateOptions{})

	mc := &MultiContextClient{
		contexts: []string{"context1", "context2"},
		clients: map[string]*Client{
			"context1": {clientset: fakeClient1},
			// context2 client is missing, will cause error
		},
	}

	pods, err := mc.ListPodsAllContexts(context.Background(), "default")

	// Should return partial results with error
	if err == nil {
		t.Error("Expected error for missing client")
	}
	if !strings.Contains(err.Error(), "errors from") {
		t.Errorf("Expected error about context errors, got: %v", err)
	}

	// Should still return pods from successful context
	if len(pods) != 1 {
		t.Errorf("Expected 1 pod from successful context, got %d", len(pods))
	}
}

func TestGetAvailableContexts(t *testing.T) {
	// This test would require a valid kubeconfig file
	// For unit testing, we'll just verify the function doesn't panic

	t.Run("Get available contexts", func(t *testing.T) {
		// Note: This will fail without a valid kubeconfig
		contexts, currentContext, err := GetAvailableContexts()

		// The function should at least not panic
		_ = contexts
		_ = currentContext
		_ = err
	})
}

func TestResourceWithContext(t *testing.T) {
	// Test the ResourceWithContext wrapper
	resource := ResourceWithContext{
		Context: "test-context",
		Resource: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
		},
	}

	if resource.Context != "test-context" {
		t.Errorf("Expected context 'test-context', got %s", resource.Context)
	}

	pod, ok := resource.Resource.(*v1.Pod)
	if !ok {
		t.Error("Resource should be a Pod")
	}
	if pod.Name != "test-pod" {
		t.Errorf("Expected pod name 'test-pod', got %s", pod.Name)
	}
}

func TestContextSpecificWrappers(t *testing.T) {
	t.Run("PodWithContext", func(t *testing.T) {
		pwc := PodWithContext{
			Context: "ctx1",
			Pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "pod1"},
			},
		}
		if pwc.Context != "ctx1" {
			t.Errorf("Expected context ctx1, got %s", pwc.Context)
		}
		if pwc.Pod.Name != "pod1" {
			t.Errorf("Expected pod name pod1, got %s", pwc.Pod.Name)
		}
	})

	t.Run("DeploymentWithContext", func(t *testing.T) {
		dwc := DeploymentWithContext{
			Context: "ctx2",
			Deployment: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "deploy1"},
			},
		}
		if dwc.Context != "ctx2" {
			t.Errorf("Expected context ctx2, got %s", dwc.Context)
		}
		if dwc.Deployment.Name != "deploy1" {
			t.Errorf("Expected deployment name deploy1, got %s", dwc.Deployment.Name)
		}
	})

	t.Run("ServiceWithContext", func(t *testing.T) {
		swc := ServiceWithContext{
			Context: "ctx3",
			Service: v1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "svc1"},
			},
		}
		if swc.Context != "ctx3" {
			t.Errorf("Expected context ctx3, got %s", swc.Context)
		}
		if swc.Service.Name != "svc1" {
			t.Errorf("Expected service name svc1, got %s", swc.Service.Name)
		}
	})

	t.Run("ConfigMapWithContext", func(t *testing.T) {
		cmwc := ConfigMapWithContext{
			Context: "ctx4",
			ConfigMap: v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "cm1"},
			},
		}
		if cmwc.Context != "ctx4" {
			t.Errorf("Expected context ctx4, got %s", cmwc.Context)
		}
		if cmwc.ConfigMap.Name != "cm1" {
			t.Errorf("Expected configmap name cm1, got %s", cmwc.ConfigMap.Name)
		}
	})

	t.Run("SecretWithContext", func(t *testing.T) {
		swc := SecretWithContext{
			Context: "ctx5",
			Secret: v1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "secret1"},
			},
		}
		if swc.Context != "ctx5" {
			t.Errorf("Expected context ctx5, got %s", swc.Context)
		}
		if swc.Secret.Name != "secret1" {
			t.Errorf("Expected secret name secret1, got %s", swc.Secret.Name)
		}
	})

	t.Run("IngressWithContext", func(t *testing.T) {
		iwc := IngressWithContext{
			Context: "ctx6",
			Ingress: networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{Name: "ing1"},
			},
		}
		if iwc.Context != "ctx6" {
			t.Errorf("Expected context ctx6, got %s", iwc.Context)
		}
		if iwc.Ingress.Name != "ing1" {
			t.Errorf("Expected ingress name ing1, got %s", iwc.Ingress.Name)
		}
	})

	t.Run("StatefulSetWithContext", func(t *testing.T) {
		sswc := StatefulSetWithContext{
			Context: "ctx7",
			StatefulSet: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "sts1"},
			},
		}
		if sswc.Context != "ctx7" {
			t.Errorf("Expected context ctx7, got %s", sswc.Context)
		}
		if sswc.StatefulSet.Name != "sts1" {
			t.Errorf("Expected statefulset name sts1, got %s", sswc.StatefulSet.Name)
		}
	})
}

func TestMultiContextClientThreadSafety(t *testing.T) {
	// Create a multi-context client with multiple contexts
	mc := &MultiContextClient{
		contexts: []string{"ctx1", "ctx2", "ctx3"},
		clients: map[string]*Client{
			"ctx1": {clientset: fake.NewSimpleClientset()},
			"ctx2": {clientset: fake.NewSimpleClientset()},
			"ctx3": {clientset: fake.NewSimpleClientset()},
		},
	}

	// Test concurrent reads
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			contexts := mc.GetContexts()
			if len(contexts) != 3 {
				t.Errorf("Expected 3 contexts, got %d", len(contexts))
			}
		}()
	}

	// Test concurrent GetClient calls
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			contextName := fmt.Sprintf("ctx%d", (idx%3)+1)
			client, err := mc.GetClient(contextName)
			if err != nil {
				t.Errorf("Failed to get client for %s: %v", contextName, err)
			}
			if client == nil {
				t.Errorf("Client for %s is nil", contextName)
			}
		}(i)
	}

	wg.Wait()
}

func TestErrorPropagation(t *testing.T) {
	// Test that errors from individual contexts are properly aggregated
	mc := &MultiContextClient{
		contexts: []string{"ctx1", "ctx2", "ctx3"},
		clients:  make(map[string]*Client),
	}

	// Only ctx1 has a valid client
	mc.clients["ctx1"] = &Client{clientset: fake.NewSimpleClientset()}
	// ctx2 and ctx3 are missing, will cause errors

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	pods, err := mc.ListPodsAllContexts(ctx, "default")

	// Should get an error mentioning multiple contexts
	if err == nil {
		t.Error("Expected error for missing clients")
	}
	if !strings.Contains(err.Error(), "errors from") {
		t.Errorf("Expected aggregated error message, got: %v", err)
	}

	// Should still return results from successful context
	if len(pods) != 0 {
		// ctx1 has no pods, so should be 0
		t.Errorf("Expected 0 pods, got %d", len(pods))
	}
}

func BenchmarkListPodsAllContexts(b *testing.B) {
	// Create multiple contexts with pods
	numContexts := 10
	clients := make(map[string]*Client)
	contexts := make([]string, numContexts)

	for i := 0; i < numContexts; i++ {
		contextName := fmt.Sprintf("context%d", i)
		contexts[i] = contextName

		fakeClient := fake.NewSimpleClientset()
		// Add pods to each context
		for j := 0; j < 10; j++ {
			pod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("pod-%d-%d", i, j),
					Namespace: "default",
				},
			}
			fakeClient.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
		}

		clients[contextName] = &Client{clientset: fakeClient}
	}

	mc := &MultiContextClient{
		contexts: contexts,
		clients:  clients,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mc.ListPodsAllContexts(context.Background(), "default")
		if err != nil && !strings.Contains(err.Error(), "errors from") {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentGetClient(b *testing.B) {
	// Create a multi-context client
	mc := &MultiContextClient{
		contexts: []string{"ctx1", "ctx2", "ctx3"},
		clients: map[string]*Client{
			"ctx1": {clientset: fake.NewSimpleClientset()},
			"ctx2": {clientset: fake.NewSimpleClientset()},
			"ctx3": {clientset: fake.NewSimpleClientset()},
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			contextName := fmt.Sprintf("ctx%d", (i%3)+1)
			_, err := mc.GetClient(contextName)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

// Helper function to create mock kubeconfig for testing
func createMockKubeconfig() *api.Config {
	return &api.Config{
		Clusters: map[string]*api.Cluster{
			"cluster1": {Server: "https://cluster1.example.com"},
			"cluster2": {Server: "https://cluster2.example.com"},
		},
		Contexts: map[string]*api.Context{
			"context1": {Cluster: "cluster1", AuthInfo: "user1"},
			"context2": {Cluster: "cluster2", AuthInfo: "user2"},
		},
		CurrentContext: "context1",
		AuthInfos: map[string]*api.AuthInfo{
			"user1": {Token: "token1"},
			"user2": {Token: "token2"},
		},
	}
}

// Test helper to verify error types
func isContextError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "context")
}

// Test helper to verify client errors
func isClientError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "client")
}

// TestMultiContextClientNamespaces tests namespace listing functionality
func TestMultiContextClientNamespaces(t *testing.T) {
	// Create fake clients
	fakeClient1 := fake.NewSimpleClientset()
	fakeClient2 := fake.NewSimpleClientset()

	// Add namespaces to context1
	ns1 := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	}
	ns2 := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
		},
	}
	ns3 := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "production",
		},
	}
	fakeClient1.CoreV1().Namespaces().Create(context.Background(), ns1, metav1.CreateOptions{})
	fakeClient1.CoreV1().Namespaces().Create(context.Background(), ns2, metav1.CreateOptions{})
	fakeClient1.CoreV1().Namespaces().Create(context.Background(), ns3, metav1.CreateOptions{})

	// Add namespaces to context2 (some overlap, some unique)
	ns4 := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default", // Duplicate
		},
	}
	ns5 := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "staging", // Unique to context2
		},
	}
	ns6 := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "development", // Unique to context2
		},
	}
	fakeClient2.CoreV1().Namespaces().Create(context.Background(), ns4, metav1.CreateOptions{})
	fakeClient2.CoreV1().Namespaces().Create(context.Background(), ns5, metav1.CreateOptions{})
	fakeClient2.CoreV1().Namespaces().Create(context.Background(), ns6, metav1.CreateOptions{})

	mc := &MultiContextClient{
		contexts: []string{"context1", "context2"},
		clients: map[string]*Client{
			"context1": {clientset: fakeClient1},
			"context2": {clientset: fakeClient2},
		},
	}

	t.Run("ListNamespacesAllContexts", func(t *testing.T) {
		namespacesWithContext, err := mc.ListNamespacesAllContexts(context.Background())
		if err != nil && !strings.Contains(err.Error(), "errors from") {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should have 6 total namespaces (3 from each context)
		if len(namespacesWithContext) != 6 {
			t.Errorf("Expected 6 namespaces with context, got %d", len(namespacesWithContext))
		}

		// Verify context information
		contextCounts := make(map[string]int)
		namespaceNames := make(map[string]bool)
		for _, nsWithCtx := range namespacesWithContext {
			contextCounts[nsWithCtx.Context]++
			namespaceNames[nsWithCtx.Namespace.Name] = true
		}

		if contextCounts["context1"] != 3 {
			t.Errorf("Expected 3 namespaces from context1, got %d", contextCounts["context1"])
		}
		if contextCounts["context2"] != 3 {
			t.Errorf("Expected 3 namespaces from context2, got %d", contextCounts["context2"])
		}

		// Verify all expected namespace names are present
		expectedNames := []string{"default", "kube-system", "production", "staging", "development"}
		for _, name := range expectedNames {
			if !namespaceNames[name] {
				t.Errorf("Expected namespace %s not found", name)
			}
		}
	})

	t.Run("GetUniqueNamespaces", func(t *testing.T) {
		uniqueNamespaces, err := mc.GetUniqueNamespaces(context.Background())
		if err != nil && !strings.Contains(err.Error(), "errors from") {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should have 5 unique namespaces (default appears in both contexts but should be deduplicated)
		if len(uniqueNamespaces) != 5 {
			t.Errorf("Expected 5 unique namespaces, got %d", len(uniqueNamespaces))
		}

		// Verify all expected unique namespace names are present
		namespaceNames := make(map[string]bool)
		for _, ns := range uniqueNamespaces {
			namespaceNames[ns.Name] = true
		}

		expectedNames := []string{"default", "kube-system", "production", "staging", "development"}
		for _, name := range expectedNames {
			if !namespaceNames[name] {
				t.Errorf("Expected unique namespace %s not found", name)
			}
		}

		// Verify no duplicates
		if len(namespaceNames) != len(uniqueNamespaces) {
			t.Errorf("Duplicate namespaces found in unique list")
		}
	})
}
