package template

import (
	"strings"
	"testing"
)

func TestEngine_StyleFunction(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		contains bool // if true, check contains instead of exact match
	}{
		// Basic text coloring
		{
			name:     "text color only",
			template: `{{ style "" "red" "" "hello" }}`,
			data:     nil,
			want:     "hello",
			contains: true,
		},
		{
			name:     "background color only",
			template: `{{ style "blue" "" "" "world" }}`,
			data:     nil,
			want:     "world",
			contains: true,
		},
		{
			name:     "both colors",
			template: `{{ style "yellow" "black" "" "warning" }}`,
			data:     nil,
			want:     "warning",
			contains: true,
		},

		// Text decorations
		{
			name:     "underline only",
			template: `{{ style "" "" "underline" "underlined" }}`,
			data:     nil,
			want:     "underlined",
			contains: true,
		},
		{
			name:     "bold only",
			template: `{{ style "" "" "bold" "bold text" }}`,
			data:     nil,
			want:     "bold text",
			contains: true,
		},
		{
			name:     "italic only",
			template: `{{ style "" "" "italic" "italic text" }}`,
			data:     nil,
			want:     "italic text",
			contains: true,
		},
		{
			name:     "multiple decorations",
			template: `{{ style "" "" "bold,underline" "important" }}`,
			data:     nil,
			want:     "important",
			contains: true,
		},

		// Combined styling
		{
			name:     "full styling - red bg, white text, underline",
			template: `{{ style "red" "white" "underline" "100%" }}`,
			data:     nil,
			want:     "100%",
			contains: true,
		},
		{
			name:     "yellow text with bold",
			template: `{{ style "" "yellow" "bold" "90%" }}`,
			data:     nil,
			want:     "90%",
			contains: true,
		},
		{
			name:     "green text no decoration",
			template: `{{ style "" "green" "" "70%" }}`,
			data:     nil,
			want:     "70%",
			contains: true,
		},

		// Edge cases
		{
			name:     "empty text",
			template: `{{ style "red" "white" "bold" "" }}`,
			data:     nil,
			want:     "",
		},
		{
			name:     "nil/empty parameters",
			template: `{{ style "" "" "" "plain text" }}`,
			data:     nil,
			want:     "plain text",
		},
		{
			name:     "with variable",
			template: `{{ style "red" "white" "underline" .Value }}`,
			data:     map[string]string{"Value": "critical"},
			want:     "critical",
			contains: true,
		},

		// Color names support
		{
			name:     "red color",
			template: `{{ style "" "red" "" "text" }}`,
			data:     nil,
			want:     "text",
			contains: true,
		},
		{
			name:     "green color",
			template: `{{ style "" "green" "" "text" }}`,
			data:     nil,
			want:     "text",
			contains: true,
		},
		{
			name:     "yellow color",
			template: `{{ style "" "yellow" "" "text" }}`,
			data:     nil,
			want:     "text",
			contains: true,
		},
		{
			name:     "blue color",
			template: `{{ style "" "blue" "" "text" }}`,
			data:     nil,
			want:     "text",
			contains: true,
		},
		{
			name:     "magenta color",
			template: `{{ style "" "magenta" "" "text" }}`,
			data:     nil,
			want:     "text",
			contains: true,
		},
		{
			name:     "cyan color",
			template: `{{ style "" "cyan" "" "text" }}`,
			data:     nil,
			want:     "text",
			contains: true,
		},
		{
			name:     "white color",
			template: `{{ style "black" "white" "" "text" }}`,
			data:     nil,
			want:     "text",
			contains: true,
		},
		{
			name:     "black color",
			template: `{{ style "white" "black" "" "text" }}`,
			data:     nil,
			want:     "text",
			contains: true,
		},
		{
			name:     "gray color",
			template: `{{ style "" "gray" "" "text" }}`,
			data:     nil,
			want:     "text",
			contains: true,
		},

		// Invalid color handling (should still render text)
		{
			name:     "invalid color name",
			template: `{{ style "invalid" "invalid" "" "text" }}`,
			data:     nil,
			want:     "text",
			contains: true,
		},

		// Hex color support
		{
			name:     "hex color",
			template: `{{ style "#FF0000" "#FFFFFF" "" "hex colors" }}`,
			data:     nil,
			want:     "hex colors",
			contains: true,
		},

		// ANSI color codes
		{
			name:     "ansi color code",
			template: `{{ style "196" "231" "" "ansi colors" }}`,
			data:     nil,
			want:     "ansi colors",
			contains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if tt.contains {
				if !strings.Contains(got, tt.want) {
					t.Errorf("Execute() = %v, want to contain %v", got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("Execute() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestEngine_StyleFunction_PodResourceColoring(t *testing.T) {
	engine := NewEngine()

	// Test the specific use cases for Pod CPU/Memory coloring
	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		contains bool
	}{
		{
			name: "CPU >100% - red bg, white text, underline",
			template: `{{- if gt .CPUPercent 100.0 -}}
				{{- style "red" "white" "underline" (printf "%.0f%%" .CPUPercent) -}}
			{{- else if ge .CPUPercent 90.0 -}}
				{{- style "" "red" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- else if ge .CPUPercent 70.0 -}}
				{{- style "" "yellow" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- else -}}
				{{- style "" "green" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- end -}}`,
			data:     map[string]float64{"CPUPercent": 120},
			want:     "120%",
			contains: true,
		},
		{
			name: "CPU 90-100% - red text",
			template: `{{- if gt .CPUPercent 100.0 -}}
				{{- style "red" "white" "underline" (printf "%.0f%%" .CPUPercent) -}}
			{{- else if ge .CPUPercent 90.0 -}}
				{{- style "" "red" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- else if ge .CPUPercent 70.0 -}}
				{{- style "" "yellow" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- else -}}
				{{- style "" "green" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- end -}}`,
			data:     map[string]float64{"CPUPercent": 95},
			want:     "95%",
			contains: true,
		},
		{
			name: "CPU 70-90% - yellow text",
			template: `{{- if gt .CPUPercent 100.0 -}}
				{{- style "red" "white" "underline" (printf "%.0f%%" .CPUPercent) -}}
			{{- else if ge .CPUPercent 90.0 -}}
				{{- style "" "red" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- else if ge .CPUPercent 70.0 -}}
				{{- style "" "yellow" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- else -}}
				{{- style "" "green" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- end -}}`,
			data:     map[string]float64{"CPUPercent": 80},
			want:     "80%",
			contains: true,
		},
		{
			name: "CPU <70% - green text",
			template: `{{- if gt .CPUPercent 100.0 -}}
				{{- style "red" "white" "underline" (printf "%.0f%%" .CPUPercent) -}}
			{{- else if ge .CPUPercent 90.0 -}}
				{{- style "" "red" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- else if ge .CPUPercent 70.0 -}}
				{{- style "" "yellow" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- else -}}
				{{- style "" "green" "" (printf "%.0f%%" .CPUPercent) -}}
			{{- end -}}`,
			data:     map[string]float64{"CPUPercent": 50},
			want:     "50%",
			contains: true,
		},
		{
			name: "Memory coloring with humanized bytes",
			template: `{{- $percent := div .MemoryUsed .MemoryRequest | mul 100.0 -}}
			{{- if gt $percent 100.0 -}}
				{{- style "red" "white" "underline" (humanizeBytes .MemoryUsed) -}}
			{{- else if ge $percent 90.0 -}}
				{{- style "" "red" "" (humanizeBytes .MemoryUsed) -}}
			{{- else if ge $percent 70.0 -}}
				{{- style "" "yellow" "" (humanizeBytes .MemoryUsed) -}}
			{{- else -}}
				{{- style "" "green" "" (humanizeBytes .MemoryUsed) -}}
			{{- end -}}`,
			data: map[string]interface{}{
				"MemoryUsed":    536870912, // 512Mi
				"MemoryRequest": 268435456, // 256Mi = 200%
			},
			want:     "512.0Mi",
			contains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			// Clean up whitespace for comparison
			got = strings.TrimSpace(got)
			if tt.contains {
				if !strings.Contains(got, tt.want) {
					t.Errorf("Execute() = %v, want to contain %v", got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("Execute() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestEngine_StyleFunction_ComplexScenarios(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		contains bool
	}{
		{
			name:     "nested with other functions",
			template: `{{ style "" "green" "bold" (icon "success") }} {{ style "" "green" "" "All tests passed" }}`,
			data:     nil,
			want:     "All tests passed",
			contains: true,
		},
		{
			name:     "with conditional logic",
			template: `{{ $color := choose (gt .Value 50) "red" "green" }}{{ style "" $color "bold" .Text }}`,
			data: map[string]interface{}{
				"Value": 75,
				"Text":  "High",
			},
			want:     "High",
			contains: true,
		},
		{
			name:     "chained styling",
			template: `{{ .Text | style "blue" "white" "underline" }}`,
			data: map[string]string{
				"Text": "Important",
			},
			want:     "Important",
			contains: true,
		},
		{
			name:     "with printf formatting",
			template: `{{ printf "CPU: %.2f cores" .CPU | style "" "cyan" "italic" }}`,
			data: map[string]float64{
				"CPU": 2.5,
			},
			want:     "CPU: 2.50 cores",
			contains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if tt.contains {
				if !strings.Contains(got, tt.want) {
					t.Errorf("Execute() = %v, want to contain %v", got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("Execute() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func BenchmarkEngine_StyleFunction(b *testing.B) {
	engine := NewEngine()
	template := `{{ style "red" "white" "underline" "100%" }}`
	data := map[string]string{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(template, data)
	}
}

func BenchmarkEngine_StyleFunction_Complex(b *testing.B) {
	engine := NewEngine()
	template := `{{- if gt .CPUPercent 100 -}}
		{{- style "red" "white" "underline" (printf "%.0f%%" .CPUPercent) -}}
	{{- else if ge .CPUPercent 90 -}}
		{{- style "" "red" "" (printf "%.0f%%" .CPUPercent) -}}
	{{- else if ge .CPUPercent 70 -}}
		{{- style "" "yellow" "" (printf "%.0f%%" .CPUPercent) -}}
	{{- else -}}
		{{- style "" "green" "" (printf "%.0f%%" .CPUPercent) -}}
	{{- end -}}`
	data := map[string]float64{"CPUPercent": 95}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(template, data)
	}
}
