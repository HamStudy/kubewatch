package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestResourceDefinition_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		check   func(t *testing.T, rd *ResourceDefinition)
	}{
		{
			name: "valid pod definition",
			yaml: `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: pod
  description: Kubernetes Pod resource
  icon: 🚀
spec:
  kubernetes:
    group: ""
    version: v1
    kind: Pod
    plural: pods
    namespaced: true
  columns:
    - name: NAME
      width: 30
      priority: 1
      template: "{{ .metadata.name }}"
      sortable: true
    - name: STATUS
      width: 20
      priority: 1
      template: "{{ .status.phase }}"
      sortable: true
      align: center
      condition: "showStatus"
      sortKey: "phase"
  operations:
    - name: describe
      key: "d"
      description: "Describe pod"
      command: "kubectl describe pod {{ .metadata.name }}"
      confirm: false
      requiresRunning: false
    - name: delete
      key: "x"
      description: "Delete pod"
      command: "kubectl delete pod {{ .metadata.name }}"
      confirm: true
      confirmMessage: "Delete pod {{ .metadata.name }}?"
      requiresRunning: false
      interactive: false
  grouping:
    enabled: true
    groupBy:
      - field: ".metadata.labels.app"
        name: "Application"
        icon: "📦"
    aggregations:
      - column: "CPU"
        function: "sum"
        format: "{{ . }} total"
  filters:
    - name: "Running Only"
      key: "r"
      condition: '.status.phase == "Running"'
`,
			wantErr: false,
			check: func(t *testing.T, rd *ResourceDefinition) {
				assert.Equal(t, "kubewatch.io/v1", rd.APIVersion)
				assert.Equal(t, "ResourceDefinition", rd.Kind)
				assert.Equal(t, "pod", rd.Metadata.Name)
				assert.Equal(t, "Kubernetes Pod resource", rd.Metadata.Description)
				assert.Equal(t, "🚀", rd.Metadata.Icon)

				// Check Kubernetes spec
				assert.Equal(t, "", rd.Spec.Kubernetes.Group)
				assert.Equal(t, "v1", rd.Spec.Kubernetes.Version)
				assert.Equal(t, "Pod", rd.Spec.Kubernetes.Kind)
				assert.Equal(t, "pods", rd.Spec.Kubernetes.Plural)
				assert.True(t, rd.Spec.Kubernetes.Namespaced)

				// Check columns
				require.Len(t, rd.Spec.Columns, 2)
				assert.Equal(t, "NAME", rd.Spec.Columns[0].Name)
				assert.Equal(t, 30, rd.Spec.Columns[0].Width)
				assert.Equal(t, 1, rd.Spec.Columns[0].Priority)
				assert.Equal(t, "{{ .metadata.name }}", rd.Spec.Columns[0].Template)
				assert.True(t, rd.Spec.Columns[0].Sortable)

				assert.Equal(t, "STATUS", rd.Spec.Columns[1].Name)
				assert.Equal(t, "center", rd.Spec.Columns[1].Align)
				assert.Equal(t, "showStatus", rd.Spec.Columns[1].Condition)
				assert.Equal(t, "phase", rd.Spec.Columns[1].SortKey)

				// Check operations
				require.Len(t, rd.Spec.Operations, 2)
				assert.Equal(t, "describe", rd.Spec.Operations[0].Name)
				assert.Equal(t, "d", rd.Spec.Operations[0].Key)
				assert.False(t, rd.Spec.Operations[0].Confirm)

				assert.Equal(t, "delete", rd.Spec.Operations[1].Name)
				assert.Equal(t, "x", rd.Spec.Operations[1].Key)
				assert.True(t, rd.Spec.Operations[1].Confirm)
				assert.Equal(t, "Delete pod {{ .metadata.name }}?", rd.Spec.Operations[1].ConfirmMessage)

				// Check grouping
				assert.True(t, rd.Spec.Grouping.Enabled)
				require.Len(t, rd.Spec.Grouping.GroupBy, 1)
				assert.Equal(t, ".metadata.labels.app", rd.Spec.Grouping.GroupBy[0].Field)
				assert.Equal(t, "Application", rd.Spec.Grouping.GroupBy[0].Name)
				assert.Equal(t, "📦", rd.Spec.Grouping.GroupBy[0].Icon)

				require.Len(t, rd.Spec.Grouping.Aggregations, 1)
				assert.Equal(t, "CPU", rd.Spec.Grouping.Aggregations[0].Column)
				assert.Equal(t, "sum", rd.Spec.Grouping.Aggregations[0].Function)

				// Check filters
				require.Len(t, rd.Spec.Filters, 1)
				assert.Equal(t, "Running Only", rd.Spec.Filters[0].Name)
				assert.Equal(t, "r", rd.Spec.Filters[0].Key)
				assert.Equal(t, `.status.phase == "Running"`, rd.Spec.Filters[0].Condition)
			},
		},
		{
			name: "deployment definition",
			yaml: `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: deployment
  description: Kubernetes Deployment resource
spec:
  kubernetes:
    group: apps
    version: v1
    kind: Deployment
    plural: deployments
    namespaced: true
  columns:
    - name: NAME
      width: 30
      priority: 1
      template: "{{ .metadata.name }}"
`,
			wantErr: false,
			check: func(t *testing.T, rd *ResourceDefinition) {
				assert.Equal(t, "deployment", rd.Metadata.Name)
				assert.Equal(t, "apps", rd.Spec.Kubernetes.Group)
				assert.Equal(t, "Deployment", rd.Spec.Kubernetes.Kind)
				assert.Equal(t, "deployments", rd.Spec.Kubernetes.Plural)
			},
		},
		{
			name: "missing required fields",
			yaml: `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: invalid
`,
			wantErr: false, // YAML will unmarshal, but validation should catch this
			check: func(t *testing.T, rd *ResourceDefinition) {
				// Spec.Kubernetes will be empty
				assert.Empty(t, rd.Spec.Kubernetes.Kind)
			},
		},
		{
			name: "invalid yaml",
			yaml: `
apiVersion: kubewatch.io/v1
kind: ResourceDefinition
metadata:
  name: [invalid
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rd ResourceDefinition
			err := yaml.Unmarshal([]byte(tt.yaml), &rd)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, &rd)
			}
		})
	}
}

func TestResourceDefinition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rd      ResourceDefinition
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid definition",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name:        "pod",
					Description: "Pod resource",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Group:      "",
						Version:    "v1",
						Kind:       "Pod",
						Plural:     "pods",
						Namespaced: true,
					},
					Columns: []Column{
						{
							Name:     "NAME",
							Width:    30,
							Priority: 1,
							Template: "{{ .metadata.name }}",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing api version",
			rd: ResourceDefinition{
				Kind: "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind: "Pod",
					},
				},
			},
			wantErr: true,
			errMsg:  "apiVersion must be kubewatch.io/v1",
		},
		{
			name: "wrong api version",
			rd: ResourceDefinition{
				APIVersion: "v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind: "Pod",
					},
				},
			},
			wantErr: true,
			errMsg:  "apiVersion must be kubewatch.io/v1",
		},
		{
			name: "missing kind",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind: "Pod",
					},
				},
			},
			wantErr: true,
			errMsg:  "kind must be ResourceDefinition",
		},
		{
			name: "missing metadata name",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata:   Metadata{},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind: "Pod",
					},
				},
			},
			wantErr: true,
			errMsg:  "metadata.name is required",
		},
		{
			name: "missing kubernetes kind",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Version: "v1",
						Plural:  "pods",
					},
				},
			},
			wantErr: true,
			errMsg:  "spec.kubernetes.kind is required",
		},
		{
			name: "missing kubernetes version",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind:   "Pod",
						Plural: "pods",
					},
				},
			},
			wantErr: true,
			errMsg:  "spec.kubernetes.version is required",
		},
		{
			name: "missing kubernetes plural",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind:    "Pod",
						Version: "v1",
					},
				},
			},
			wantErr: true,
			errMsg:  "spec.kubernetes.plural is required",
		},
		{
			name: "no columns defined",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind:    "Pod",
						Version: "v1",
						Plural:  "pods",
					},
					Columns: []Column{},
				},
			},
			wantErr: true,
			errMsg:  "at least one column must be defined",
		},
		{
			name: "column missing name",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind:    "Pod",
						Version: "v1",
						Plural:  "pods",
					},
					Columns: []Column{
						{
							Width:    30,
							Template: "{{ .metadata.name }}",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "column name is required",
		},
		{
			name: "column missing template",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind:    "Pod",
						Version: "v1",
						Plural:  "pods",
					},
					Columns: []Column{
						{
							Name:  "NAME",
							Width: 30,
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "column template is required",
		},
		{
			name: "column invalid width",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind:    "Pod",
						Version: "v1",
						Plural:  "pods",
					},
					Columns: []Column{
						{
							Name:     "NAME",
							Width:    0,
							Template: "{{ .metadata.name }}",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "column width must be positive",
		},
		{
			name: "invalid column align",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind:    "Pod",
						Version: "v1",
						Plural:  "pods",
					},
					Columns: []Column{
						{
							Name:     "NAME",
							Width:    30,
							Template: "{{ .metadata.name }}",
							Align:    "middle",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "column align must be one of: left, center, right",
		},
		{
			name: "operation missing name",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind:    "Pod",
						Version: "v1",
						Plural:  "pods",
					},
					Columns: []Column{
						{
							Name:     "NAME",
							Width:    30,
							Template: "{{ .metadata.name }}",
						},
					},
					Operations: []Operation{
						{
							Key:     "d",
							Command: "kubectl describe",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "operation name is required",
		},
		{
			name: "operation missing key",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind:    "Pod",
						Version: "v1",
						Plural:  "pods",
					},
					Columns: []Column{
						{
							Name:     "NAME",
							Width:    30,
							Template: "{{ .metadata.name }}",
						},
					},
					Operations: []Operation{
						{
							Name:    "describe",
							Command: "kubectl describe",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "operation key is required",
		},
		{
			name: "operation missing command",
			rd: ResourceDefinition{
				APIVersion: "kubewatch.io/v1",
				Kind:       "ResourceDefinition",
				Metadata: Metadata{
					Name: "pod",
				},
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Kind:    "Pod",
						Version: "v1",
						Plural:  "pods",
					},
					Columns: []Column{
						{
							Name:     "NAME",
							Width:    30,
							Template: "{{ .metadata.name }}",
						},
					},
					Operations: []Operation{
						{
							Name: "describe",
							Key:  "d",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "operation command is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rd.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResourceDefinition_GetGroupVersionKind(t *testing.T) {
	rd := ResourceDefinition{
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			},
		},
	}

	gvk := rd.GetGroupVersionKind()
	assert.Equal(t, "apps", gvk.Group)
	assert.Equal(t, "v1", gvk.Version)
	assert.Equal(t, "Deployment", gvk.Kind)
}

func TestResourceDefinition_GetGroupVersionResource(t *testing.T) {
	rd := ResourceDefinition{
		Spec: Spec{
			Kubernetes: KubernetesSpec{
				Group:   "apps",
				Version: "v1",
				Plural:  "deployments",
			},
		},
	}

	gvr := rd.GetGroupVersionResource()
	assert.Equal(t, "apps", gvr.Group)
	assert.Equal(t, "v1", gvr.Version)
	assert.Equal(t, "deployments", gvr.Resource)
}

func TestResourceDefinition_IsNamespaced(t *testing.T) {
	tests := []struct {
		name       string
		namespaced bool
		want       bool
	}{
		{
			name:       "namespaced resource",
			namespaced: true,
			want:       true,
		},
		{
			name:       "cluster-scoped resource",
			namespaced: false,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rd := ResourceDefinition{
				Spec: Spec{
					Kubernetes: KubernetesSpec{
						Namespaced: tt.namespaced,
					},
				},
			}
			assert.Equal(t, tt.want, rd.IsNamespaced())
		})
	}
}
