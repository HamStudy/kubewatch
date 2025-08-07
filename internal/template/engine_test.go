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

func TestEngine_AddFunction(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "add two integers",
			template: `{{ add 5 3 }}`,
			data:     nil,
			want:     "8",
		},
		{
			name:     "add multiple values",
			template: `{{ add 1 2 3 4 }}`,
			data:     nil,
			want:     "10",
		},
		{
			name:     "add floats",
			template: `{{ add 1.5 2.5 }}`,
			data:     nil,
			want:     "4",
		},
		{
			name:     "add with variables",
			template: `{{ add .Value1 .Value2 }}`,
			data:     map[string]int{"Value1": 10, "Value2": 20},
			want:     "30",
		},
		{
			name:     "add zero values",
			template: `{{ add 0 0 }}`,
			data:     nil,
			want:     "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_DefaultFunction(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			name:     "default with empty string",
			template: `{{ default "N/A" "" }}`,
			data:     nil,
			want:     "N/A",
		},
		{
			name:     "default with nil",
			template: `{{ default "missing" .NotExists }}`,
			data:     map[string]string{},
			want:     "missing",
		},
		{
			name:     "default with value",
			template: `{{ default "N/A" "value" }}`,
			data:     nil,
			want:     "value",
		},
		{
			name:     "default with zero",
			template: `{{ default 10 0 }}`,
			data:     nil,
			want:     "10",
		},
		{
			name:     "default with non-zero",
			template: `{{ default 10 5 }}`,
			data:     nil,
			want:     "5",
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

func TestEngine_ListFunctions(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			name:     "create list",
			template: `{{ join "," (list "a" "b" "c") }}`,
			data:     nil,
			want:     "a,b,c",
		},
		{
			name:     "append to list",
			template: `{{ $l := list "a" "b" }}{{ join "," (append $l "c") }}`,
			data:     nil,
			want:     "a,b,c",
		},
		{
			name:     "slice list",
			template: `{{ join "," (slice (list "a" "b" "c" "d") 1 3) }}`,
			data:     nil,
			want:     "b,c",
		},
		{
			name:     "list length",
			template: `{{ len (list "a" "b" "c") }}`,
			data:     nil,
			want:     "3",
		},
		{
			name:     "empty list",
			template: `{{ len list }}`,
			data:     nil,
			want:     "0",
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

func TestEngine_StringFunctions(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			name:     "split string",
			template: `{{ split "a,b,c" "," | join "-" }}`,
			data:     nil,
			want:     "a-b-c",
		},
		{
			name:     "string length",
			template: `{{ len "hello" }}`,
			data:     nil,
			want:     "5",
		},
		{
			name:     "toString integer",
			template: `{{ toString 42 }}`,
			data:     nil,
			want:     "42",
		},
		{
			name:     "toString float",
			template: `{{ toString 3.14 }}`,
			data:     nil,
			want:     "3.14",
		},
		{
			name:     "toString bool",
			template: `{{ toString true }}`,
			data:     nil,
			want:     "true",
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

func TestEngine_MemoryConversionFunctions(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			name:     "toMB from bytes",
			template: `{{ toMB 1048576 }}`,
			data:     nil,
			want:     "1",
		},
		{
			name:     "toMB from kilobytes",
			template: `{{ toMB 2097152 }}`,
			data:     nil,
			want:     "2",
		},
		{
			name:     "humanizeBytes small",
			template: `{{ humanizeBytes 512 }}`,
			data:     nil,
			want:     "512",
		},
		{
			name:     "humanizeBytes KB",
			template: `{{ humanizeBytes 2048 }}`,
			data:     nil,
			want:     "2.0Ki",
		},
		{
			name:     "humanizeBytes MB",
			template: `{{ humanizeBytes 5242880 }}`,
			data:     nil,
			want:     "5.0Mi",
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

func TestEngine_CPUConversionFunctions(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
	}{
		{
			name:     "toMillicores from cores",
			template: `{{ toMillicores 0.5 }}`,
			data:     nil,
			want:     "500",
		},
		{
			name:     "toMillicores from string cores",
			template: `{{ toMillicores "2" }}`,
			data:     nil,
			want:     "2000",
		},
		{
			name:     "toMillicores from millicores string",
			template: `{{ toMillicores "250m" }}`,
			data:     nil,
			want:     "250",
		},
		{
			name:     "cores from millicores",
			template: `{{ cores 1500 }}`,
			data:     nil,
			want:     "1.50",
		},
		{
			name:     "cores from millicores string",
			template: `{{ cores "2500m" }}`,
			data:     nil,
			want:     "2.50",
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

func TestEngine_EdgeCases(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "nil data with default",
			template: `{{ default "N/A" .Missing }}`,
			data:     map[string]interface{}{},
			want:     "N/A",
		},
		{
			name:     "empty list operations",
			template: `{{ len (slice (list) 0 0) }}`,
			data:     nil,
			want:     "0",
		},
		{
			name:     "toString nil",
			template: `{{ toString .Missing }}`,
			data:     map[string]interface{}{},
			want:     "",
		},
		{
			name:     "add with strings",
			template: `{{ add "10" "20" }}`,
			data:     nil,
			want:     "30",
		},
		{
			name:     "len of nil",
			template: `{{ len .Missing }}`,
			data:     map[string]interface{}{},
			want:     "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_ComplexTemplateScenarios(t *testing.T) {
	engine := NewEngine()

	// Test with complex nested data similar to Kubernetes resources
	podData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":              "test-pod",
			"namespace":         "kube-system",
			"creationTimestamp": time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
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
					"restartCount": 1,
				},
			},
		},
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"name":  "app",
					"image": "nginx:latest",
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"cpu":    "100m",
							"memory": "128Mi",
						},
						"limits": map[string]interface{}{
							"cpu":    "500m",
							"memory": "256Mi",
						},
					},
				},
				map[string]interface{}{
					"name":  "sidecar",
					"image": "busybox:latest",
				},
			},
		},
	}

	tests := []struct {
		name     string
		template string
		want     string
		contains bool
	}{
		{
			name: "namespace coloring",
			template: `{{- if hasPrefix .metadata.namespace "kube-" -}}
				{{- color "blue" .metadata.namespace -}}
			{{- else -}}
				{{- .metadata.namespace -}}
			{{- end -}}`,
			want:     "kube-system",
			contains: true,
		},
		{
			name: "container ready count",
			template: `{{- $ready := 0 -}}
			{{- $total := len .status.containerStatuses -}}
			{{- range .status.containerStatuses -}}
				{{- if .ready -}}{{- $ready = add $ready 1 -}}{{- end -}}
			{{- end -}}
			{{- toString $ready -}}/{{- toString $total -}}`,
			want: "2/2",
		},
		{
			name:     "age formatting",
			template: `{{ ago .metadata.creationTimestamp }}`,
			want:     "2h",
			contains: true,
		},
		{
			name: "container list",
			template: `{{- $containers := list -}}
			{{- range .spec.containers -}}
				{{- $containers = append $containers .name -}}
			{{- end -}}
			{{- join $containers "," -}}`,
			want: "app,sidecar",
		},
		{
			name:     "CPU request processing",
			template: `{{ (index (index (index (index .spec.containers 0) "resources") "requests") "cpu") | toMillicores }}`,
			want:     "100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, podData)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
				return
			}
			if tt.contains {
				if !strings.Contains(got, tt.want) {
					t.Errorf("Execute() = %v, want to contain %v", got, tt.want)
				}
			} else {
				// Clean up whitespace for comparison
				got = strings.TrimSpace(got)
				if got != tt.want {
					t.Errorf("Execute() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestEngine_UnstructuredSupport(t *testing.T) {
	engine := NewEngine()

	// Test with map[string]interface{} (similar to unstructured.Unstructured)
	data := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "test-pod",
			"namespace": "default",
		},
		"status": map[string]interface{}{
			"containerStatuses": []interface{}{
				map[string]interface{}{
					"ready":        true,
					"restartCount": 2,
				},
				map[string]interface{}{
					"ready":        false,
					"restartCount": 1,
				},
			},
		},
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"name": "app",
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"cpu":    "100m",
							"memory": "128Mi",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "access nested field",
			template: `{{ .metadata.name }}`,
			want:     "test-pod",
		},
		{
			name:     "count containers",
			template: `{{ len .status.containerStatuses }}`,
			want:     "2",
		},
		{
			name:     "process CPU request",
			template: `{{ (index (index (index (index .spec.containers 0) "resources") "requests") "cpu") | toMillicores }}`,
			want:     "100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Execute(tt.template, data)
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
