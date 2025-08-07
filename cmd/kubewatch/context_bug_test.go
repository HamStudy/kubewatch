package main

import (
	"testing"
)

// TestContextBugReproduction tests the specific bug reported:
// Command: kubewatch --context=cluster1
// Expected: Shows context as cluster1
// Actual: Shows context as hamstudy (default context)
func TestContextBugReproduction(t *testing.T) {
	// Test the individual components that should handle context override

	t.Run("parseContexts handles single context correctly", func(t *testing.T) {
		flags := &CLIFlags{
			context: "cluster1",
		}

		contexts, err := parseContexts(flags)
		if err != nil {
			t.Fatalf("parseContexts failed: %v", err)
		}

		if len(contexts) != 1 {
			t.Fatalf("Expected 1 context, got %d", len(contexts))
		}

		if contexts[0] != "cluster1" {
			t.Errorf("Expected context 'cluster1', got '%s'", contexts[0])
		}
	})

	t.Run("loadConfigWithFlags sets CurrentContext correctly", func(t *testing.T) {
		flags := &CLIFlags{
			context: "cluster1",
		}

		config, err := loadConfigWithFlags(flags)
		if err != nil {
			t.Fatalf("loadConfigWithFlags failed: %v", err)
		}

		if config.CurrentContext != "cluster1" {
			t.Errorf("Expected CurrentContext 'cluster1', got '%s'", config.CurrentContext)
		}
	})

	t.Run("single context mode logic", func(t *testing.T) {
		// Simulate the logic from main() function
		flags := &CLIFlags{
			context: "cluster1",
		}

		config, err := loadConfigWithFlags(flags)
		if err != nil {
			t.Fatalf("loadConfigWithFlags failed: %v", err)
		}

		contexts, err := parseContexts(flags)
		if err != nil {
			t.Fatalf("parseContexts failed: %v", err)
		}

		isMultiContext := len(contexts) > 1
		if isMultiContext {
			t.Fatal("Expected single context mode, got multi-context")
		}

		// This is the logic from main.go lines 291-294
		contextToUse := config.CurrentContext
		if len(contexts) == 1 {
			contextToUse = contexts[0]
		}

		if contextToUse != "cluster1" {
			t.Errorf("Expected contextToUse 'cluster1', got '%s'", contextToUse)
		}
	})
}

// TestEmptyContextBehavior tests what happens when no context is specified
func TestEmptyContextBehavior(t *testing.T) {
	t.Run("empty context flag", func(t *testing.T) {
		flags := &CLIFlags{
			context: "",
		}

		contexts, err := parseContexts(flags)
		if err != nil {
			t.Fatalf("parseContexts failed: %v", err)
		}

		if len(contexts) != 0 {
			t.Errorf("Expected 0 contexts for empty flag, got %d", len(contexts))
		}

		config, err := loadConfigWithFlags(flags)
		if err != nil {
			t.Fatalf("loadConfigWithFlags failed: %v", err)
		}

		// When no context is specified, CurrentContext should remain empty
		// (it will be set by the kubeconfig loader)
		if config.CurrentContext != "" {
			t.Errorf("Expected empty CurrentContext, got '%s'", config.CurrentContext)
		}
	})

	t.Run("empty context mode logic", func(t *testing.T) {
		// Simulate what happens when no --context flag is provided
		flags := &CLIFlags{
			context: "",
		}

		config, err := loadConfigWithFlags(flags)
		if err != nil {
			t.Fatalf("loadConfigWithFlags failed: %v", err)
		}

		contexts, err := parseContexts(flags)
		if err != nil {
			t.Fatalf("parseContexts failed: %v", err)
		}

		isMultiContext := len(contexts) > 1
		if isMultiContext {
			t.Fatal("Expected single context mode, got multi-context")
		}

		// This is the logic from main.go lines 291-294
		contextToUse := config.CurrentContext // This should be empty
		if len(contexts) == 1 {
			contextToUse = contexts[0]
		}

		// When no context is specified, contextToUse should be empty
		// and the kubeconfig loader will use the current-context from the file
		if contextToUse != "" {
			t.Errorf("Expected empty contextToUse, got '%s'", contextToUse)
		}
	})
}
