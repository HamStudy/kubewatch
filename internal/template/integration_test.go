package template

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStyleFunction_BehavioralOutput tests that the style function produces
// the expected visual output for users, not implementation details
func TestStyleFunction_BehavioralOutput(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name        string
		template    string
		data        interface{}
		shouldColor bool // Whether output should contain ANSI codes
		contains    string
		scenario    string // What user scenario this tests
	}{
		{
			name:        "cpu over limit shows warning",
			template:    `{{ style "red" "white" "underline" "150%" }}`,
			data:        nil,
			shouldColor: true,
			contains:    "150%",
			scenario:    "User sees CPU usage over request limit highlighted",
		},
		{
			name:        "normal cpu usage shows green",
			template:    `{{ style "" "green" "" "50%" }}`,
			data:        nil,
			shouldColor: true,
			contains:    "50%",
			scenario:    "User sees normal CPU usage in green",
		},
		{
			name:        "empty text returns empty",
			template:    `{{ style "red" "white" "bold" "" }}`,
			data:        nil,
			shouldColor: false,
			contains:    "",
			scenario:    "No visual noise for empty values",
		},
		{
			name:        "multiple decorations work together",
			template:    `{{ style "" "yellow" "bold,underline" "WARNING" }}`,
			data:        nil,
			shouldColor: true,
			contains:    "WARNING",
			scenario:    "User sees important warnings with multiple emphasis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			require.NoError(t, err, "Template execution should not fail")

			assert.Contains(t, got, tt.contains,
				"Scenario: %s - Output should contain expected text", tt.scenario)

			if tt.shouldColor {
				// Check for ANSI escape codes without testing specific codes
				assert.True(t, strings.Contains(got, "\x1b["),
					"Scenario: %s - Output should be styled", tt.scenario)
			} else {
				assert.False(t, strings.Contains(got, "\x1b["),
					"Scenario: %s - Output should not be styled", tt.scenario)
			}
		})
	}
}

// TestResourceFormattingIntegration tests the complete flow of formatting
// Kubernetes resources with templates - what users actually see
func TestResourceFormattingIntegration(t *testing.T) {
	engine := NewEngine()

	// Simulate pod resource data structure
	podData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "nginx-deployment-abc123",
			"namespace": "production",
		},
		"status": map[string]interface{}{
			"phase": "Running",
			"containerStatuses": []interface{}{
				map[string]interface{}{
					"ready":        true,
					"restartCount": 0,
				},
				map[string]interface{}{
					"ready":        true,
					"restartCount": 2,
				},
			},
		},
		"metrics": map[string]interface{}{
			"cpu":    "850m",
			"memory": "512Mi",
		},
		"requests": map[string]interface{}{
			"cpu":    "1000m",
			"memory": "1Gi",
		},
	}

	tests := []struct {
		name     string
		template string
		verify   func(t *testing.T, output string)
		scenario string
	}{
		{
			name: "ready containers display",
			template: `{{- $ready := 0 -}}
{{- $total := len .status.containerStatuses -}}
{{- range .status.containerStatuses -}}
  {{- if .ready -}}{{- $ready = add $ready 1 -}}{{- end -}}
{{- end -}}
{{- $ready -}}/{{- $total -}}`,
			verify: func(t *testing.T, output string) {
				assert.Equal(t, "2/2", output, "Should show all containers ready")
			},
			scenario: "User sees container readiness at a glance",
		},
		{
			name: "restart count with details",
			template: `{{- $restarts := 0 -}}
{{- range .status.containerStatuses -}}
  {{- $restarts = add $restarts .restartCount -}}
{{- end -}}
{{- if gt $restarts 0 -}}
  {{- style "" "yellow" "" (toString $restarts) -}}
{{- else -}}
  {{- toString $restarts -}}
{{- end -}}`,
			verify: func(t *testing.T, output string) {
				assert.Contains(t, output, "2", "Should show total restart count")
				// Should be styled since restarts > 0
				assert.Contains(t, output, "\x1b[", "Restarts should be highlighted")
			},
			scenario: "User notices pods with restarts immediately",
		},
		{
			name: "cpu usage with threshold coloring",
			template: `{{- $cpu := toMillicores .metrics.cpu -}}
{{- $request := toMillicores .requests.cpu -}}
{{- $percent := div (mul $cpu 100) $request -}}
{{- if gt $percent 90 -}}
  {{- style "" "red" "" .metrics.cpu -}}
{{- else if gt $percent 70 -}}
  {{- style "" "yellow" "" .metrics.cpu -}}
{{- else -}}
  {{- style "" "green" "" .metrics.cpu -}}
{{- end -}}`,
			verify: func(t *testing.T, output string) {
				assert.Contains(t, output, "850m", "Should show CPU value")
				// 850/1000 = 85% so should be yellow
				assert.Contains(t, output, "\x1b[", "Should be colored based on usage")
			},
			scenario: "User sees CPU usage with color-coded thresholds",
		},
		{
			name:     "memory formatting",
			template: `{{ .metrics.memory }}`,
			verify: func(t *testing.T, output string) {
				assert.Equal(t, "512Mi", output, "Should show memory in readable format")
			},
			scenario: "User sees memory in familiar Kubernetes units",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := engine.Execute(tt.template, podData)
			require.NoError(t, err, "Template should execute without error")

			// Clean up whitespace for comparison
			output = strings.TrimSpace(output)

			tt.verify(t, output)
			t.Logf("Scenario verified: %s", tt.scenario)
		})
	}
}

// TestTemplatePerformance tests that templates render quickly enough
// for interactive use - users should not experience lag
func TestTemplatePerformance(t *testing.T) {
	engine := NewEngine()

	// Create a complex template similar to real pod display
	complexTemplate := `
{{- if eq .status.phase "Running" -}}
  {{- style "" "green" "" .status.phase -}}
{{- else if eq .status.phase "Pending" -}}
  {{- style "" "yellow" "" .status.phase -}}
{{- else -}}
  {{- style "" "red" "" .status.phase -}}
{{- end }} | 
CPU: {{ style "" (choose (gt (toMillicores .metrics.cpu) (toMillicores .requests.cpu)) "red" "green") "" .metrics.cpu }} | 
Memory: {{ .metrics.memory }}`

	// Generate test data for multiple resources
	resources := make([]map[string]interface{}, 500)
	for i := 0; i < 500; i++ {
		resources[i] = map[string]interface{}{
			"status": map[string]interface{}{
				"phase": []string{"Running", "Pending", "Failed"}[i%3],
			},
			"metrics": map[string]interface{}{
				"cpu":    fmt.Sprintf("%dm", 100+i%900),
				"memory": fmt.Sprintf("%dMi", 128+i%512),
			},
			"requests": map[string]interface{}{
				"cpu":    "1000m",
				"memory": "1Gi",
			},
		}
	}

	// Measure rendering time
	start := time.Now()
	for _, resource := range resources {
		output, err := engine.Execute(complexTemplate, resource)
		require.NoError(t, err)
		require.NotEmpty(t, output)
	}
	elapsed := time.Since(start)

	// Verify performance meets user expectations
	assert.Less(t, elapsed.Milliseconds(), int64(100),
		"Rendering 500 resources should take less than 100ms for smooth UI")

	avgTime := elapsed.Nanoseconds() / int64(len(resources)) / 1000 // microseconds
	assert.Less(t, avgTime, int64(200),
		"Average render time per resource should be under 200μs")

	t.Logf("Rendered %d resources in %v (avg: %dμs per resource)",
		len(resources), elapsed, avgTime)
}

// TestTemplateConcurrency tests that templates work correctly when
// multiple goroutines render simultaneously (common in watch mode)
func TestTemplateConcurrency(t *testing.T) {
	engine := NewEngine()
	template := `{{ style "" "green" "" .name }}: {{ .value }}`

	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				data := map[string]interface{}{
					"name":  fmt.Sprintf("pod-%d", id),
					"value": fmt.Sprintf("value-%d-%d", id, i),
				}

				output, err := engine.Execute(template, data)
				if err != nil {
					errors <- err
					return
				}

				// Verify output contains expected values
				if !strings.Contains(output, fmt.Sprintf("pod-%d", id)) {
					errors <- fmt.Errorf("output missing pod name: %s", output)
				}
				if !strings.Contains(output, fmt.Sprintf("value-%d-%d", id, i)) {
					errors <- fmt.Errorf("output missing value: %s", output)
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	var errCount int
	for err := range errors {
		t.Errorf("Concurrent execution error: %v", err)
		errCount++
		if errCount > 10 {
			t.Fatal("Too many concurrent errors, stopping")
		}
	}

	assert.Equal(t, 0, errCount, "Should have no errors during concurrent execution")
}

// TestTemplateErrorHandling tests that template errors provide useful
// feedback to users when templates are misconfigured
func TestTemplateErrorHandling(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name          string
		template      string
		data          interface{}
		errorContains string
		scenario      string
	}{
		{
			name:          "undefined variable",
			template:      `{{ .undefinedField }}`,
			data:          map[string]interface{}{},
			errorContains: "", // Go templates return empty for undefined fields
			scenario:      "User references non-existent field",
		},
		{
			name:          "invalid function",
			template:      `{{ unknownFunc .field }}`,
			data:          map[string]interface{}{"field": "value"},
			errorContains: "function \"unknownFunc\" not defined",
			scenario:      "User uses undefined template function",
		},
		{
			name:          "type mismatch in function",
			template:      `{{ add "not-a-number" 5 }}`,
			data:          nil,
			errorContains: "", // Our add function handles string conversion
			scenario:      "User passes wrong type to function",
		},
		{
			name:          "nil data handling",
			template:      `{{ default "N/A" .field }}`,
			data:          nil,
			errorContains: "", // Should handle gracefully
			scenario:      "Template handles nil data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := engine.Execute(tt.template, tt.data)

			if tt.errorContains != "" {
				require.Error(t, err, "Scenario: %s - Should produce error", tt.scenario)
				assert.Contains(t, err.Error(), tt.errorContains,
					"Error message should be helpful for: %s", tt.scenario)
			} else {
				// Should handle gracefully without error
				assert.NoError(t, err, "Scenario: %s - Should handle gracefully", tt.scenario)
				assert.NotNil(t, output, "Should return some output even with issues")
			}
		})
	}
}

// TestTemplateEdgeCases tests edge cases that could occur in production
func TestTemplateEdgeCases(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		verify   func(t *testing.T, output string, err error)
		scenario string
	}{
		{
			name:     "very long selector list truncation",
			template: `{{ $list := list }}{{ range $i := list 1 2 3 4 5 6 7 8 9 10 }}{{ $list = append $list (toString $i) }}{{ end }}{{ join $list "," }}`,
			data:     nil,
			verify: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "1,2,3,4,5,6,7,8,9,10", output)
			},
			scenario: "Handle long selector lists without breaking",
		},
		{
			name:     "empty container list",
			template: `{{ len .containers }}`,
			data:     map[string]interface{}{"containers": []interface{}{}},
			verify: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "0", output)
			},
			scenario: "Handle pods with no containers",
		},
		{
			name:     "division by zero in percentage",
			template: `{{ percent 50 0 }}`,
			data:     nil,
			verify: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "0%", output)
			},
			scenario: "Handle division by zero gracefully",
		},
		{
			name:     "nil values in math operations",
			template: `{{ add .missing 10 }}`,
			data:     map[string]interface{}{},
			verify: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "10", output) // nil converts to 0
			},
			scenario: "Handle nil values in calculations",
		},
		{
			name:     "deeply nested data access",
			template: `{{ .a.b.c.d.e }}`,
			data: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": map[string]interface{}{
							"d": map[string]interface{}{
								"e": "deep-value",
							},
						},
					},
				},
			},
			verify: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "deep-value", output)
			},
			scenario: "Access deeply nested Kubernetes resource fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := engine.Execute(tt.template, tt.data)
			tt.verify(t, output, err)
			t.Logf("Edge case handled: %s", tt.scenario)
		})
	}
}

// TestTemplateCacheEffectiveness verifies that caching improves performance
// This tests actual behavior users experience (faster repeated renders)
func TestTemplateCacheEffectiveness(t *testing.T) {
	engine := NewEngine()

	template := `{{ if gt .cpu 80 }}{{ style "" "red" "" "HIGH" }}{{ else }}{{ style "" "green" "" "OK" }}{{ end }}`
	data := map[string]interface{}{"cpu": 75}

	// First execution - not cached
	start := time.Now()
	result1, err := engine.Execute(template, data)
	require.NoError(t, err)
	firstTime := time.Since(start)

	// Second execution - should be cached
	start = time.Now()
	result2, err := engine.Execute(template, data)
	require.NoError(t, err)
	cachedTime := time.Since(start)

	// Verify same result
	assert.Equal(t, result1, result2, "Cached result should be identical")

	// Verify cache is faster (at least 2x faster is reasonable)
	// Note: We're testing behavior (faster renders) not implementation
	if firstTime > time.Microsecond*100 { // Only check if first execution took meaningful time
		assert.Less(t, cachedTime.Nanoseconds(), firstTime.Nanoseconds()/2,
			"Cached execution should be significantly faster for better UX")
	}

	// Different data should not use cache
	data2 := map[string]interface{}{"cpu": 95}
	result3, err := engine.Execute(template, data2)
	require.NoError(t, err)
	assert.NotEqual(t, result1, result3, "Different data should produce different result")
	assert.Contains(t, result3, "HIGH", "High CPU should show HIGH")
}

// TestRealWorldTemplates tests actual templates used in the application
func TestRealWorldTemplates(t *testing.T) {
	engine := NewEngine()

	// Test actual pod status template that would be used
	podStatusTemplate := `{{- if eq .phase "Running" -}}
  {{- $ready := 0 -}}
  {{- $total := len .containers -}}
  {{- range .containers -}}
    {{- if .ready -}}{{- $ready = add $ready 1 -}}{{- end -}}
  {{- end -}}
  {{- if eq $ready $total -}}
    {{- style "" "green" "" "Running" -}}
  {{- else -}}
    {{- style "" "yellow" "" "Running" -}} ({{- $ready -}}/{{- $total -}})
  {{- end -}}
{{- else if eq .phase "Pending" -}}
  {{- style "" "yellow" "" .phase -}}
{{- else if eq .phase "Succeeded" -}}
  {{- style "" "blue" "" .phase -}}
{{- else if eq .phase "Failed" -}}
  {{- style "" "red" "bold" .phase -}}
{{- else -}}
  {{- style "" "gray" "" (default "Unknown" .phase) -}}
{{- end -}}`

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected string // What the user should see (without ANSI codes)
		scenario string
	}{
		{
			name: "all containers ready",
			data: map[string]interface{}{
				"phase": "Running",
				"containers": []map[string]interface{}{
					{"ready": true},
					{"ready": true},
				},
			},
			expected: "Running",
			scenario: "Healthy pod shows green Running",
		},
		{
			name: "some containers not ready",
			data: map[string]interface{}{
				"phase": "Running",
				"containers": []map[string]interface{}{
					{"ready": true},
					{"ready": false},
					{"ready": true},
				},
			},
			expected: "Running (2/3)",
			scenario: "Partially ready pod shows count",
		},
		{
			name: "pending pod",
			data: map[string]interface{}{
				"phase":      "Pending",
				"containers": []map[string]interface{}{},
			},
			expected: "Pending",
			scenario: "Pending pod shows yellow",
		},
		{
			name: "failed pod",
			data: map[string]interface{}{
				"phase":      "Failed",
				"containers": []map[string]interface{}{},
			},
			expected: "Failed",
			scenario: "Failed pod shows red and bold",
		},
		{
			name: "unknown status",
			data: map[string]interface{}{
				"phase":      nil,
				"containers": []map[string]interface{}{},
			},
			expected: "Unknown",
			scenario: "Missing status shows Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := engine.Execute(podStatusTemplate, tt.data)
			require.NoError(t, err, "Template should execute without error")

			// Strip ANSI codes for comparison
			cleaned := stripANSI(output)
			cleaned = strings.TrimSpace(cleaned)

			assert.Equal(t, tt.expected, cleaned,
				"Scenario: %s - User should see correct status", tt.scenario)

			// Verify styling is present when expected
			if tt.data["phase"] != nil {
				assert.Contains(t, output, "\x1b[",
					"Status should be styled for better visibility")
			}
		})
	}
}

// stripANSI removes ANSI escape codes for testing text content
func stripANSI(s string) string {
	// Simple ANSI stripping for tests
	result := s
	for strings.Contains(result, "\x1b[") {
		start := strings.Index(result, "\x1b[")
		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return result
}
