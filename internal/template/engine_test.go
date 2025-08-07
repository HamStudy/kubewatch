package template

import (
	"strings"
	"testing"
	"time"
)

func TestEngine_ColorFunc(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "basic color",
			template: `{{ color "red" "hello" }}`,
			data:     nil,
			want:     "hello", // Will contain ANSI codes
		},
		{
			name:     "empty text",
			template: `{{ color "blue" "" }}`,
			data:     nil,
			want:     "",
		},
		{
			name:     "color with variable",
			template: `{{ color "green" .Status }}`,
			data:     map[string]string{"Status": "Running"},
			want:     "Running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_HumanizeBytes(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			name:     "bytes",
			template: `{{ humanizeBytes 1024 }}`,
			data:     nil,
			want:     "1.0Ki",
		},
		{
			name:     "megabytes",
			template: `{{ humanizeBytes 1048576 }}`,
			data:     nil,
			want:     "1.0Mi",
		},
		{
			name:     "gigabytes",
			template: `{{ humanizeBytes 1073741824 }}`,
			data:     nil,
			want:     "1.0Gi",
		},
		{
			name:     "zero",
			template: `{{ humanizeBytes 0 }}`,
			data:     nil,
			want:     "0",
		},
		{
			name:     "from variable",
			template: `{{ humanizeBytes .Memory }}`,
			data:     map[string]int64{"Memory": 536870912},
			want:     "512.0Mi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_Millicores(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			name:     "from cores",
			template: `{{ millicores 0.25 }}`,
			data:     nil,
			want:     "250m",
		},
		{
			name:     "from millicores string",
			template: `{{ millicores "100m" }}`,
			data:     nil,
			want:     "100m",
		},
		{
			name:     "from cores string",
			template: `{{ millicores "2" }}`,
			data:     nil,
			want:     "2000m",
		},
		{
			name:     "zero",
			template: `{{ millicores 0 }}`,
			data:     nil,
			want:     "0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_ConditionalColor(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			name:     "if true",
			template: `{{ if eq .Status "Running" }}{{ color "green" .Status }}{{ else }}{{ color "red" .Status }}{{ end }}`,
			data:     map[string]string{"Status": "Running"},
			want:     "Running",
		},
		{
			name:     "if false",
			template: `{{ if eq .Status "Running" }}{{ color "green" .Status }}{{ else }}{{ color "red" .Status }}{{ end }}`,
			data:     map[string]string{"Status": "Failed"},
			want:     "Failed",
		},
		{
			name:     "colorIf function",
			template: `{{ colorIf (eq .Count 0) "gray" "green" .Text }}`,
			data:     map[string]interface{}{"Count": 0, "Text": "empty"},
			want:     "empty",
		},
		{
			name:     "choose function",
			template: `{{ choose (gt .Value 10) "high" "low" }}`,
			data:     map[string]int{"Value": 15},
			want:     "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_Icons(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			name:     "success icon",
			template: `{{ icon "success" }}`,
			data:     nil,
			want:     "✓",
		},
		{
			name:     "error icon",
			template: `{{ icon "error" }}`,
			data:     nil,
			want:     "✗",
		},
		{
			name:     "running icon",
			template: `{{ icon "running" }}`,
			data:     nil,
			want:     "●",
		},
		{
			name:     "unknown icon",
			template: `{{ icon "unknown" }}`,
			data:     nil,
			want:     "",
		},
		{
			name:     "iconIf true",
			template: `{{ iconIf true "success" "error" }}`,
			data:     nil,
			want:     "✓",
		},
		{
			name:     "iconIf false",
			template: `{{ iconIf false "success" "error" }}`,
			data:     nil,
			want:     "✗",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_MathFunctions(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			name:     "percent",
			template: `{{ percent 25 100 }}`,
			data:     nil,
			want:     "25%",
		},
		{
			name:     "percent zero total",
			template: `{{ percent 25 0 }}`,
			data:     nil,
			want:     "0%",
		},
		{
			name:     "div",
			template: `{{ div 10 2 }}`,
			data:     nil,
			want:     "5",
		},
		{
			name:     "mul",
			template: `{{ mul 5 3 }}`,
			data:     nil,
			want:     "15",
		},
		{
			name:     "sub",
			template: `{{ sub 10 3 }}`,
			data:     nil,
			want:     "7",
		},
		{
			name:     "min",
			template: `{{ min 5 3 8 1 }}`,
			data:     nil,
			want:     "1",
		},
		{
			name:     "max",
			template: `{{ max 5 3 8 1 }}`,
			data:     nil,
			want:     "8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_TimeFunctions(t *testing.T) {
	engine := NewEngine()
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	tests := []struct {
		name     string
		template string
		data     interface{}
		contains string
	}{
		{
			name:     "ago function",
			template: `{{ ago .Time }}`,
			data:     map[string]time.Time{"Time": oneHourAgo},
			contains: "1h",
		},
		{
			name:     "timestamp function",
			template: `{{ timestamp .Time }}`,
			data:     map[string]time.Time{"Time": now},
			contains: now.Format("2006"),
		},
		{
			name:     "ageInSeconds",
			template: `{{ ageInSeconds .Time }}`,
			data:     map[string]time.Time{"Time": oneHourAgo},
			contains: "3600", // approximately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if !strings.Contains(got, tt.contains) {
				t.Errorf("Execute() = %v, want to contain %v", got, tt.contains)
			}
		})
	}
}

func TestEngine_ComplexTemplate(t *testing.T) {
	engine := NewEngine()

	// Complex pod status template
	template := `{{- if eq .Status "Running" -}}
  {{- color "green" (icon "running") }} {{ color "green" .Status -}}
{{- else if eq .Status "Pending" -}}
  {{- color "yellow" (icon "pending") }} {{ color "yellow" .Status -}}
{{- else -}}
  {{- color "red" (icon "error") }} {{ color "red" .Status -}}
{{- end -}}`

	tests := []struct {
		name string
		data interface{}
		want string
	}{
		{
			name: "running pod",
			data: map[string]string{"Status": "Running"},
			want: "Running",
		},
		{
			name: "pending pod",
			data: map[string]string{"Status": "Pending"},
			want: "Pending",
		},
		{
			name: "failed pod",
			data: map[string]string{"Status": "Failed"},
			want: "Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(template, tt.data)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("Execute() = %v, want to contain %v", got, tt.want)
			}
		})
	}
}

func TestEngine_Validation(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "valid template",
			template: `{{ .Field }}`,
			wantErr:  false,
		},
		{
			name:     "invalid syntax",
			template: `{{ .Field }`,
			wantErr:  true,
		},
		{
			name:     "unknown function",
			template: `{{ unknownFunc .Field }}`,
			wantErr:  true,
		},
		{
			name:     "valid with functions",
			template: `{{ color "red" .Field }}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Validate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_Cache(t *testing.T) {
	engine := NewEngine()

	template := `{{ .Value }}`
	data := map[string]string{"Value": "test"}

	// First execution - should not be cached
	result1, err := engine.Execute(template, data)
	if err != nil {
		t.Fatalf("First execution failed: %v", err)
	}

	// Second execution - should be cached
	result2, err := engine.Execute(template, data)
	if err != nil {
		t.Fatalf("Second execution failed: %v", err)
	}

	if result1 != result2 {
		t.Errorf("Cached result differs: got %v, want %v", result2, result1)
	}

	// Different data - should not use cache
	data2 := map[string]string{"Value": "different"}
	result3, err := engine.Execute(template, data2)
	if err != nil {
		t.Fatalf("Third execution failed: %v", err)
	}

	if result3 == result1 {
		t.Errorf("Should not have used cache for different data")
	}
}

func BenchmarkEngine_Execute(b *testing.B) {
	engine := NewEngine()
	template := `{{ if eq .Status "Running" }}{{ color "green" .Status }}{{ else }}{{ color "red" .Status }}{{ end }}`
	data := map[string]string{"Status": "Running"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(template, data)
	}
}

func BenchmarkEngine_ExecuteWithCache(b *testing.B) {
	engine := NewEngine()
	template := `{{ if eq .Status "Running" }}{{ color "green" .Status }}{{ else }}{{ color "red" .Status }}{{ end }}`
	data := map[string]string{"Status": "Running"}

	// Warm up cache
	_, _ = engine.Execute(template, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(template, data)
	}
}

func BenchmarkEngine_ComplexTemplate(b *testing.B) {
	engine := NewEngine()
	template := `
{{- if eq .Status "Running" -}}
  {{- color "green" (icon "running") }} {{ color "green" .Status }} 
  CPU: {{ millicores .CPU }} ({{ percent .CPUUsed .CPURequest }})
  Memory: {{ humanizeBytes .Memory }}
{{- else -}}
  {{- color "red" (icon "error") }} {{ color "red" .Status -}}
{{- end -}}`

	data := map[string]interface{}{
		"Status":     "Running",
		"CPU":        0.25,
		"CPUUsed":    250,
		"CPURequest": 500,
		"Memory":     536870912,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(template, data)
	}
}
