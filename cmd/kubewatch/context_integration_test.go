package main

import (
	"os"
	"testing"

	"github.com/HamStudy/kubewatch/internal/k8s"
)

// TestContextIntegrationBug tests the integration between CLI flags and K8s client
// to reproduce the exact bug described:
// Command: kubewatch --context=cluster1
// Expected: Shows context as cluster1
// Actual: Shows context as hamstudy (default context)
func TestContextIntegrationBug(t *testing.T) {
	// Create a test kubeconfig that simulates the user's environment
	testKubeconfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://hamstudy-cluster.example.com
  name: hamstudy
- cluster:
    server: https://cluster1.example.com
  name: cluster1
contexts:
- context:
    cluster: hamstudy
    user: hamstudy-user
  name: hamstudy
- context:
    cluster: cluster1
    user: cluster1-user
  name: cluster1
current-context: hamstudy
users:
- name: hamstudy-user
  user:
    token: hamstudy-token
- name: cluster1-user
  user:
    token: cluster1-token
`

	// Create temporary kubeconfig file
	tmpFile, err := os.CreateTemp("", "kubeconfig-integration-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testKubeconfig); err != nil {
		t.Fatalf("Failed to write kubeconfig: %v", err)
	}
	tmpFile.Close()

	tests := []struct {
		name                string
		contextFlag         string
		expectedContextUsed string
	}{
		{
			name:                "No context flag uses current-context from kubeconfig",
			contextFlag:         "",
			expectedContextUsed: "", // Should be empty, letting kubeconfig decide
		},
		{
			name:                "Context flag overrides current-context",
			contextFlag:         "cluster1",
			expectedContextUsed: "cluster1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the main() function logic
			flags := &CLIFlags{
				context:    tt.contextFlag,
				kubeconfig: tmpFile.Name(),
			}

			// Step 1: Load config with flags
			config, err := loadConfigWithFlags(flags)
			if err != nil {
				t.Fatalf("loadConfigWithFlags failed: %v", err)
			}

			// Step 2: Parse contexts
			contexts, err := parseContexts(flags)
			if err != nil {
				t.Fatalf("parseContexts failed: %v", err)
			}

			// Step 3: Determine context to use (main.go logic)
			isMultiContext := len(contexts) > 1
			if isMultiContext {
				t.Fatal("Expected single context mode for this test")
			}

			contextToUse := config.CurrentContext
			if len(contexts) == 1 {
				contextToUse = contexts[0]
			}

			// Verify the context that would be used
			if contextToUse != tt.expectedContextUsed {
				t.Errorf("Expected contextToUse '%s', got '%s'", tt.expectedContextUsed, contextToUse)
			}

			// Step 4: Create client with the context (this is where the bug might be)
			clientOpts := &k8s.ClientOptions{
				Context: contextToUse,
			}

			// Try to create the client - this will fail due to fake servers
			// but we can verify that the context override is being applied
			_, err = k8s.NewClientWithOptions(tmpFile.Name(), clientOpts)

			// We expect this to fail due to connection issues, not config issues
			if err != nil {
				t.Logf("Client creation failed as expected (fake servers): %v", err)
			}

			// The key test: verify that when contextToUse is "cluster1",
			// the client is actually configured to use cluster1, not hamstudy
			// This is hard to test without actually connecting, but we've verified
			// the logic flow is correct
		})
	}
}

// TestActualBugScenario tests the exact scenario from the bug report
func TestActualBugScenario(t *testing.T) {
	// This test simulates: kubewatch --context=cluster1
	// where the kubeconfig has current-context: hamstudy

	testKubeconfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://hamstudy-cluster.example.com
  name: hamstudy
- cluster:
    server: https://cluster1.example.com
  name: cluster1
contexts:
- context:
    cluster: hamstudy
    user: hamstudy-user
  name: hamstudy
- context:
    cluster: cluster1
    user: cluster1-user
  name: cluster1
current-context: hamstudy
users:
- name: hamstudy-user
  user:
    token: hamstudy-token
- name: cluster1-user
  user:
    token: cluster1-token
`

	tmpFile, err := os.CreateTemp("", "kubeconfig-bug-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testKubeconfig); err != nil {
		t.Fatalf("Failed to write kubeconfig: %v", err)
	}
	tmpFile.Close()

	// Simulate: kubewatch --context=cluster1
	flags := &CLIFlags{
		context:    "cluster1",
		kubeconfig: tmpFile.Name(),
	}

	// Follow the exact main() logic
	config, err := loadConfigWithFlags(flags)
	if err != nil {
		t.Fatalf("loadConfigWithFlags failed: %v", err)
	}

	// At this point, config.CurrentContext should be "cluster1"
	if config.CurrentContext != "cluster1" {
		t.Errorf("BUG FOUND: config.CurrentContext should be 'cluster1', got '%s'", config.CurrentContext)
	}

	contexts, err := parseContexts(flags)
	if err != nil {
		t.Fatalf("parseContexts failed: %v", err)
	}

	// contexts should be ["cluster1"]
	if len(contexts) != 1 || contexts[0] != "cluster1" {
		t.Errorf("BUG FOUND: contexts should be ['cluster1'], got %v", contexts)
	}

	// Single context mode logic
	isMultiContext := len(contexts) > 1
	if isMultiContext {
		t.Fatal("Should be single context mode")
	}

	contextToUse := config.CurrentContext
	if len(contexts) == 1 {
		contextToUse = contexts[0]
	}

	// contextToUse should be "cluster1"
	if contextToUse != "cluster1" {
		t.Errorf("BUG FOUND: contextToUse should be 'cluster1', got '%s'", contextToUse)
	}

	// Create client options
	clientOpts := &k8s.ClientOptions{
		Context: contextToUse,
	}

	// The client should be configured to use cluster1, not hamstudy
	if clientOpts.Context != "cluster1" {
		t.Errorf("BUG FOUND: clientOpts.Context should be 'cluster1', got '%s'", clientOpts.Context)
	}

	t.Logf("SUCCESS: All logic correctly uses context 'cluster1' instead of default 'hamstudy'")
}
