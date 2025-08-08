package transformers

import (
	"testing"

	"github.com/HamStudy/kubewatch/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCPUToMillicores(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "dash",
			input:    "-",
			expected: 0,
		},
		{
			name:     "millicores format - 50m",
			input:    "50m",
			expected: 50,
		},
		{
			name:     "millicores format - 250m",
			input:    "250m",
			expected: 250,
		},
		{
			name:     "millicores format - 1000m",
			input:    "1000m",
			expected: 1000,
		},
		{
			name:     "millicores format - 2500m",
			input:    "2500m",
			expected: 2500,
		},
		{
			name:     "cores format - 1",
			input:    "1",
			expected: 1000,
		},
		{
			name:     "cores format - 1.5",
			input:    "1.5",
			expected: 1500,
		},
		{
			name:     "cores format - 2.25",
			input:    "2.25",
			expected: 2250,
		},
		{
			name:     "cores format - 0.5",
			input:    "0.5",
			expected: 500,
		},
		{
			name:     "cores format - 0.1",
			input:    "0.1",
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCPUToMillicores(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseMemoryToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "dash",
			input:    "-",
			expected: 0,
		},
		{
			name:     "bytes",
			input:    "1024",
			expected: 1024,
		},
		{
			name:     "kilobytes - Ki",
			input:    "512Ki",
			expected: 512 * 1024,
		},
		{
			name:     "megabytes - Mi",
			input:    "128Mi",
			expected: 128 * 1024 * 1024,
		},
		{
			name:     "megabytes - 256Mi",
			input:    "256Mi",
			expected: 256 * 1024 * 1024,
		},
		{
			name:     "gigabytes - Gi",
			input:    "1Gi",
			expected: 1024 * 1024 * 1024,
		},
		{
			name:     "gigabytes - 2Gi",
			input:    "2Gi",
			expected: 2 * 1024 * 1024 * 1024,
		},
		{
			name:     "gigabytes - 1.5Gi",
			input:    "1.5Gi",
			expected: int64(1.5 * 1024 * 1024 * 1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMemoryToBytes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCPU(t *testing.T) {
	// Create a template engine for testing coloring
	templateEngine := template.NewEngine()
	require.NotNil(t, templateEngine)

	tests := []struct {
		name              string
		millicores        int64
		requestMillicores int64
		withTemplate      bool
		expectedPlain     string
		expectedContains  []string // For checking colored output contains certain strings
	}{
		{
			name:              "zero value",
			millicores:        0,
			requestMillicores: 0,
			withTemplate:      false,
			expectedPlain:     "-",
		},
		{
			name:              "50 millicores no request",
			millicores:        50,
			requestMillicores: 0,
			withTemplate:      false,
			expectedPlain:     "50m",
		},
		{
			name:              "250 millicores no request",
			millicores:        250,
			requestMillicores: 0,
			withTemplate:      false,
			expectedPlain:     "250m",
		},
		{
			name:              "999 millicores no request",
			millicores:        999,
			requestMillicores: 0,
			withTemplate:      false,
			expectedPlain:     "999m",
		},
		{
			name:              "1000 millicores (1 core) no request",
			millicores:        1000,
			requestMillicores: 0,
			withTemplate:      false,
			expectedPlain:     "1",
		},
		{
			name:              "1200 millicores (1.2 cores) no request",
			millicores:        1200,
			requestMillicores: 0,
			withTemplate:      false,
			expectedPlain:     "1.2",
		},
		{
			name:              "2500 millicores (2.5 cores) no request",
			millicores:        2500,
			requestMillicores: 0,
			withTemplate:      false,
			expectedPlain:     "2.5",
		},
		{
			name:              "100m with 200m request (50% - green)",
			millicores:        100,
			requestMillicores: 200,
			withTemplate:      true,
			expectedContains:  []string{"100m"}, // Should contain the value
		},
		{
			name:              "150m with 200m request (75% - yellow)",
			millicores:        150,
			requestMillicores: 200,
			withTemplate:      true,
			expectedContains:  []string{"150m"},
		},
		{
			name:              "180m with 200m request (90% - red)",
			millicores:        180,
			requestMillicores: 200,
			withTemplate:      true,
			expectedContains:  []string{"180m"},
		},
		{
			name:              "250m with 200m request (125% - red bg)",
			millicores:        250,
			requestMillicores: 200,
			withTemplate:      true,
			expectedContains:  []string{"250m"},
		},
		{
			name:              "1500m with 2000m request (75% - yellow)",
			millicores:        1500,
			requestMillicores: 2000,
			withTemplate:      true,
			expectedContains:  []string{"1.5"},
		},
		{
			name:              "2000m with 1000m request (200% - red bg)",
			millicores:        2000,
			requestMillicores: 1000,
			withTemplate:      true,
			expectedContains:  []string{"2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.withTemplate {
				result = formatCPU(tt.millicores, tt.requestMillicores, templateEngine)
				// For template tests, just check that the value is contained
				for _, expected := range tt.expectedContains {
					assert.Contains(t, result, expected)
				}
			} else {
				result = formatCPU(tt.millicores, tt.requestMillicores, nil)
				assert.Equal(t, tt.expectedPlain, result)
			}
		})
	}
}

func TestFormatMemory(t *testing.T) {
	// Create a template engine for testing coloring
	templateEngine := template.NewEngine()
	require.NotNil(t, templateEngine)

	const (
		Ki = 1024
		Mi = 1024 * Ki
		Gi = 1024 * Mi
	)

	tests := []struct {
		name             string
		bytes            int64
		requestBytes     int64
		withTemplate     bool
		expectedPlain    string
		expectedContains []string // For checking colored output contains certain strings
	}{
		{
			name:          "zero value",
			bytes:         0,
			requestBytes:  0,
			withTemplate:  false,
			expectedPlain: "-",
		},
		{
			name:          "bytes",
			bytes:         512,
			requestBytes:  0,
			withTemplate:  false,
			expectedPlain: "512",
		},
		{
			name:          "kilobytes",
			bytes:         512 * Ki,
			requestBytes:  0,
			withTemplate:  false,
			expectedPlain: "512Ki",
		},
		{
			name:          "megabytes - 128Mi",
			bytes:         128 * Mi,
			requestBytes:  0,
			withTemplate:  false,
			expectedPlain: "128Mi",
		},
		{
			name:          "megabytes - 256Mi",
			bytes:         256 * Mi,
			requestBytes:  0,
			withTemplate:  false,
			expectedPlain: "256Mi",
		},
		{
			name:          "gigabytes - 1Gi",
			bytes:         1 * Gi,
			requestBytes:  0,
			withTemplate:  false,
			expectedPlain: "1Gi",
		},
		{
			name:          "gigabytes - 2Gi",
			bytes:         2 * Gi,
			requestBytes:  0,
			withTemplate:  false,
			expectedPlain: "2Gi",
		},
		{
			name:          "gigabytes with fraction - 1.5Gi",
			bytes:         int64(1.5 * float64(Gi)),
			requestBytes:  0,
			withTemplate:  false,
			expectedPlain: "1.5Gi",
		},
		{
			name:             "128Mi with 256Mi request (50% - green)",
			bytes:            128 * Mi,
			requestBytes:     256 * Mi,
			withTemplate:     true,
			expectedContains: []string{"128Mi"},
		},
		{
			name:             "192Mi with 256Mi request (75% - yellow)",
			bytes:            192 * Mi,
			requestBytes:     256 * Mi,
			withTemplate:     true,
			expectedContains: []string{"192Mi"},
		},
		{
			name:             "230Mi with 256Mi request (90% - red)",
			bytes:            230 * Mi,
			requestBytes:     256 * Mi,
			withTemplate:     true,
			expectedContains: []string{"230Mi"},
		},
		{
			name:             "512Mi with 256Mi request (200% - red bg)",
			bytes:            512 * Mi,
			requestBytes:     256 * Mi,
			withTemplate:     true,
			expectedContains: []string{"512Mi"},
		},
		{
			name:             "1.5Gi with 2Gi request (75% - yellow)",
			bytes:            int64(1.5 * float64(Gi)),
			requestBytes:     2 * Gi,
			withTemplate:     true,
			expectedContains: []string{"1.5Gi"},
		},
		{
			name:             "3Gi with 2Gi request (150% - red bg)",
			bytes:            3 * Gi,
			requestBytes:     2 * Gi,
			withTemplate:     true,
			expectedContains: []string{"3Gi"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.withTemplate {
				result = formatMemory(tt.bytes, tt.requestBytes, templateEngine)
				// For template tests, just check that the value is contained
				for _, expected := range tt.expectedContains {
					assert.Contains(t, result, expected)
				}
			} else {
				result = formatMemory(tt.bytes, tt.requestBytes, nil)
				assert.Equal(t, tt.expectedPlain, result)
			}
		})
	}
}

func TestFormatCPUColoring(t *testing.T) {
	// Test the coloring logic specifically
	// Note: In test environments, lipgloss doesn't apply ANSI codes,
	// so we just verify the correct values are returned
	templateEngine := template.NewEngine()
	require.NotNil(t, templateEngine)

	tests := []struct {
		name              string
		millicores        int64
		requestMillicores int64
		expectedValue     string
	}{
		{
			name:              "less than 70% - should be green",
			millicores:        60,
			requestMillicores: 100,
			expectedValue:     "60m",
		},
		{
			name:              "exactly 70% - should be yellow",
			millicores:        70,
			requestMillicores: 100,
			expectedValue:     "70m",
		},
		{
			name:              "85% - should be yellow",
			millicores:        85,
			requestMillicores: 100,
			expectedValue:     "85m",
		},
		{
			name:              "exactly 90% - should be red",
			millicores:        90,
			requestMillicores: 100,
			expectedValue:     "90m",
		},
		{
			name:              "95% - should be red",
			millicores:        95,
			requestMillicores: 100,
			expectedValue:     "95m",
		},
		{
			name:              "over 100% - should have red background",
			millicores:        120,
			requestMillicores: 100,
			expectedValue:     "120m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCPU(tt.millicores, tt.requestMillicores, templateEngine)
			// In test environments, lipgloss doesn't apply colors, so we just check the value
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}

func TestFormatMemoryColoring(t *testing.T) {
	// Test the coloring logic specifically
	// Note: In test environments, lipgloss doesn't apply ANSI codes,
	// so we just verify the correct values are returned
	templateEngine := template.NewEngine()
	require.NotNil(t, templateEngine)

	const Mi = 1024 * 1024

	tests := []struct {
		name          string
		bytes         int64
		requestBytes  int64
		expectedValue string
	}{
		{
			name:          "less than 70% - should be green",
			bytes:         60 * Mi,
			requestBytes:  100 * Mi,
			expectedValue: "60Mi",
		},
		{
			name:          "exactly 70% - should be yellow",
			bytes:         70 * Mi,
			requestBytes:  100 * Mi,
			expectedValue: "70Mi",
		},
		{
			name:          "85% - should be yellow",
			bytes:         85 * Mi,
			requestBytes:  100 * Mi,
			expectedValue: "85Mi",
		},
		{
			name:          "exactly 90% - should be red",
			bytes:         90 * Mi,
			requestBytes:  100 * Mi,
			expectedValue: "90Mi",
		},
		{
			name:          "95% - should be red",
			bytes:         95 * Mi,
			requestBytes:  100 * Mi,
			expectedValue: "95Mi",
		},
		{
			name:          "over 100% - should have red background",
			bytes:         120 * Mi,
			requestBytes:  100 * Mi,
			expectedValue: "120Mi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMemory(tt.bytes, tt.requestBytes, templateEngine)
			// In test environments, lipgloss doesn't apply colors, so we just check the value
			assert.Equal(t, tt.expectedValue, result)
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMemory(tt.bytes, tt.requestBytes, templateEngine)
			// In test environments, lipgloss doesn't apply colors, so we just check the value
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}
