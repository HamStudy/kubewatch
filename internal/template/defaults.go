package template

// DefaultTemplates contains all built-in formatting templates
var DefaultTemplates = map[string]string{
	// Row templates for table rendering
	"pod_row":                `{{ .Name }}	{{ template "pod-status" .Pod }}	{{ template "pod-ready" .Pod }}	{{ .Restarts }}	{{ ago .Age }}`,
	"pod_row_with_namespace": `{{ .Namespace }}	{{ .Name }}	{{ template "pod-status" .Pod }}	{{ template "pod-ready" .Pod }}	{{ .Restarts }}	{{ ago .Age }}`,

	"deployment_row":                `{{ .Name }}	{{ .Ready }}	{{ .UpToDate }}	{{ .Available }}	{{ ago .Age }}`,
	"deployment_row_with_namespace": `{{ .Namespace }}	{{ .Name }}	{{ .Ready }}	{{ .UpToDate }}	{{ .Available }}	{{ ago .Age }}`,

	"service_row":                `{{ .Name }}	{{ .Type }}	{{ .ClusterIP }}	{{ .ExternalIP }}	{{ .Ports }}	{{ ago .Age }}`,
	"service_row_with_namespace": `{{ .Namespace }}	{{ .Name }}	{{ .Type }}	{{ .ClusterIP }}	{{ .ExternalIP }}	{{ .Ports }}	{{ ago .Age }}`,

	"statefulset_row":                `{{ .Name }}	{{ .Ready }}	{{ ago .Age }}`,
	"statefulset_row_with_namespace": `{{ .Namespace }}	{{ .Name }}	{{ .Ready }}	{{ ago .Age }}`,

	"ingress_row":                `{{ .Name }}	{{ .Class }}	{{ .Hosts }}	{{ .Address }}	{{ .Ports }}	{{ ago .Age }}`,
	"ingress_row_with_namespace": `{{ .Namespace }}	{{ .Name }}	{{ .Class }}	{{ .Hosts }}	{{ .Address }}	{{ .Ports }}	{{ ago .Age }}`,

	"configmap_row":                `{{ .Name }}	{{ .Data }}	{{ ago .Age }}`,
	"configmap_row_with_namespace": `{{ .Namespace }}	{{ .Name }}	{{ .Data }}	{{ ago .Age }}`,

	"secret_row":                `{{ .Name }}	{{ .Type }}	{{ .Data }}	{{ ago .Age }}`,
	"secret_row_with_namespace": `{{ .Namespace }}	{{ .Name }}	{{ .Type }}	{{ .Data }}	{{ ago .Age }}`,
	"pod-status": `{{- /* Pod Status Formatter - Shows status with appropriate icon and color */ -}}
{{- $status := .Status.Phase -}}
{{- $reason := "" -}}

{{- /* Check for more specific status from conditions */ -}}
{{- range .Status.Conditions -}}
  {{- if and (eq .Type "Ready") (ne .Status "True") .Reason -}}
    {{- $reason = .Reason -}}
  {{- end -}}
{{- end -}}

{{- /* Check container statuses for waiting/terminated states */ -}}
{{- range .Status.ContainerStatuses -}}
  {{- if .State.Waiting -}}
    {{- $status = .State.Waiting.Reason -}}
  {{- else if .State.Terminated -}}
    {{- $status = .State.Terminated.Reason -}}
  {{- end -}}
{{- end -}}

{{- /* Apply appropriate styling based on status */ -}}
{{- if eq $status "Running" -}}
  {{- color "green" "●" }} {{ color "green" "Running" -}}
{{- else if eq $status "Succeeded" -}}
  {{- color "green" "✓" }} {{ color "green" "Succeeded" -}}
{{- else if eq $status "Pending" -}}
  {{- color "yellow" "◐" }} {{ color "yellow" "Pending" -}}
{{- else if eq $status "ContainerCreating" -}}
  {{- color "yellow" "◑" }} {{ color "yellow" "Creating" -}}
{{- else if eq $status "Terminating" -}}
  {{- color "magenta" "◉" }} {{ color "magenta" "Terminating" -}}
{{- else if or (eq $status "Failed") (eq $status "Error") -}}
  {{- color "red" "✗" }} {{ color "red" $status -}}
{{- else if eq $status "CrashLoopBackOff" -}}
  {{- color "red" "↻" }} {{ color "red" "CrashLoop" -}}
{{- else if eq $status "ImagePullBackOff" -}}
  {{- color "red" "⬇" }} {{ color "red" "ImagePull" -}}
{{- else if eq $status "ErrImagePull" -}}
  {{- color "red" "⬇" }} {{ color "red" "ImageErr" -}}
{{- else if eq $status "Completed" -}}
  {{- color "blue" "☐" }} {{ color "blue" "Completed" -}}
{{- else if eq $status "Evicted" -}}
  {{- color "yellow" "⚠" }} {{ color "yellow" "Evicted" -}}
{{- else -}}
  {{- color "gray" "○" }} {{ color "gray" $status -}}
{{- end -}}`,

	"cpu": `{{- /* CPU Formatter - Shows CPU usage with appropriate coloring */ -}}
{{- $cpu := .Metrics.CPU | default 0 -}}
{{- $cpuMilli := $cpu | toMillicores -}}
{{- $requested := 0 -}}
{{- $limit := 0 -}}

{{- /* Get request and limit if available */ -}}
{{- if .Spec.Containers -}}
  {{- range .Spec.Containers -}}
    {{- $requested = add $requested (.Resources.Requests.cpu | toMillicores | default 0) -}}
    {{- $limit = add $limit (.Resources.Limits.cpu | toMillicores | default 0) -}}
  {{- end -}}
{{- end -}}

{{- /* Calculate percentage if request is set */ -}}
{{- $percent := 0 -}}
{{- if gt $requested 0 -}}
  {{- $percent = div (mul $cpuMilli 100) $requested -}}
{{- end -}}

{{- /* Format based on value */ -}}
{{- if eq $cpuMilli 0 -}}
  {{- color "gray" "-" -}}
{{- else if lt $cpuMilli 1000 -}}
  {{- /* Show millicores for small values */ -}}
  {{- if and (gt $percent 0) (gt $percent 90) -}}
    {{- color "red" (printf "%dm" $cpuMilli) -}}
  {{- else if and (gt $percent 0) (gt $percent 70) -}}
    {{- color "yellow" (printf "%dm" $cpuMilli) -}}
  {{- else -}}
    {{- color "green" (printf "%dm" $cpuMilli) -}}
  {{- end -}}
{{- else -}}
  {{- /* Show cores for large values */ -}}
  {{- $cores := div $cpuMilli 1000.0 -}}
  {{- if and (gt $percent 0) (gt $percent 90) -}}
    {{- color "red" (printf "%.2f" $cores) -}}
  {{- else if and (gt $percent 0) (gt $percent 70) -}}
    {{- color "yellow" (printf "%.2f" $cores) -}}
  {{- else -}}
    {{- color "green" (printf "%.2f" $cores) -}}
  {{- end -}}
{{- end -}}`,

	"memory": `{{- /* Memory Formatter - Shows memory with appropriate units and coloring */ -}}
{{- $memory := .Metrics.Memory | default 0 -}}
{{- $memoryMB := $memory | toMB -}}
{{- $requested := 0 -}}
{{- $limit := 0 -}}

{{- /* Get request and limit if available */ -}}
{{- if .Spec.Containers -}}
  {{- range .Spec.Containers -}}
    {{- $requested = add $requested (.Resources.Requests.memory | toMB | default 0) -}}
    {{- $limit = add $limit (.Resources.Limits.memory | toMB | default 0) -}}
  {{- end -}}
{{- end -}}

{{- /* Calculate percentage if request is set */ -}}
{{- $percent := 0 -}}
{{- if gt $requested 0 -}}
  {{- $percent = div (mul $memoryMB 100) $requested -}}
{{- end -}}

{{- /* Format with appropriate units */ -}}
{{- if eq $memoryMB 0 -}}
  {{- color "gray" "-" -}}
{{- else -}}
  {{- $formatted := $memory | humanizeBytes -}}
  {{- if and (gt $percent 0) (gt $percent 90) -}}
    {{- color "red" $formatted -}}
  {{- else if and (gt $percent 0) (gt $percent 70) -}}
    {{- color "yellow" $formatted -}}
  {{- else if lt $memoryMB 128 -}}
    {{- color "green" $formatted -}}
  {{- else if lt $memoryMB 512 -}}
    {{- color "yellow" $formatted -}}
  {{- else -}}
    {{- color "red" $formatted -}}
  {{- end -}}
{{- end -}}`,

	"ready": `{{- /* Ready Formatter - Shows ready/total with coloring */ -}}
{{- $ready := 0 -}}
{{- $total := 0 -}}

{{- /* Handle different resource types */ -}}
{{- if .Status.ContainerStatuses -}}
  {{- /* Pod */ -}}
  {{- $total = len .Status.ContainerStatuses -}}
  {{- range .Status.ContainerStatuses -}}
    {{- if .Ready -}}{{- $ready = add $ready 1 -}}{{- end -}}
  {{- end -}}
{{- else if .Status.ReadyReplicas -}}
  {{- /* Deployment/StatefulSet */ -}}
  {{- $ready = .Status.ReadyReplicas | default 0 -}}
  {{- $total = .Spec.Replicas | default 1 -}}
{{- end -}}

{{- /* Format with color based on readiness */ -}}
{{- $text := printf "%d/%d" $ready $total -}}
{{- if eq $ready $total -}}
  {{- color "green" $text -}}
{{- else if eq $ready 0 -}}
  {{- color "red" $text -}}
{{- else -}}
  {{- color "yellow" $text -}}
{{- end -}}`,

	"restarts": `{{- /* Restart Formatter - Shows restart count with last restart time */ -}}
{{- $restarts := 0 -}}
{{- $lastRestart := "" -}}

{{- /* Sum up all container restarts */ -}}
{{- range .Status.ContainerStatuses -}}
  {{- $restarts = add $restarts .RestartCount -}}
  {{- if .LastTerminationState.Terminated -}}
    {{- $lastRestart = .LastTerminationState.Terminated.FinishedAt | ago -}}
  {{- end -}}
{{- end -}}

{{- /* Format based on restart count */ -}}
{{- if eq $restarts 0 -}}
  {{- color "gray" "0" -}}
{{- else -}}
  {{- $text := toString $restarts -}}
  {{- if $lastRestart -}}
    {{- $text = printf "%d (%s)" $restarts $lastRestart -}}
  {{- end -}}
  
  {{- if lt $restarts 3 -}}
    {{- color "yellow" $text -}}
  {{- else if lt $restarts 10 -}}
    {{- color "orange" (printf "⚠ %s" $text) -}}
  {{- else -}}
    {{- color "red" (printf "‼ %s" $text) -}}
  {{- end -}}
{{- end -}}`,

	"age": `{{- /* Age Formatter - Shows age with optional coloring */ -}}
{{- $age := .Metadata.CreationTimestamp | ago -}}
{{- $ageSeconds := .Metadata.CreationTimestamp | ageInSeconds -}}

{{- /* Color based on age (optional) */ -}}
{{- if lt $ageSeconds 300 -}}
  {{- /* Less than 5 minutes - new */ -}}
  {{- color "cyan" (printf "✨ %s" $age) -}}
{{- else if lt $ageSeconds 3600 -}}
  {{- /* Less than 1 hour */ -}}
  {{- color "green" $age -}}
{{- else if lt $ageSeconds 86400 -}}
  {{- /* Less than 1 day */ -}}
  {{- color "white" $age -}}
{{- else if lt $ageSeconds 604800 -}}
  {{- /* Less than 1 week */ -}}
  {{- color "gray" $age -}}
{{- else -}}
  {{- /* Older than 1 week */ -}}
  {{- color "darkgray" $age -}}
{{- end -}}`,

	"service-type": `{{- /* Service Type Formatter */ -}}
{{- if eq .Spec.Type "LoadBalancer" -}}
  {{- color "blue" "🌐" }} {{ .Spec.Type -}}
{{- else if eq .Spec.Type "NodePort" -}}
  {{- color "cyan" "📡" }} {{ .Spec.Type -}}
{{- else if eq .Spec.Type "ClusterIP" -}}
  {{- color "green" "🔒" }} {{ .Spec.Type -}}
{{- else if eq .Spec.Type "ExternalName" -}}
  {{- color "magenta" "🔗" }} {{ .Spec.Type -}}
{{- else -}}
  {{- color "gray" .Spec.Type -}}
{{- end -}}`,

	"namespace": `{{- /* Namespace Formatter */ -}}
{{- if hasPrefix .Namespace "kube-" -}}
  {{- color "blue" (printf "⚙ %s" .Namespace) -}}
{{- else if eq .Namespace "default" -}}
  {{- color "gray" .Namespace -}}
{{- else if contains .Namespace "prod" -}}
  {{- color "red" (printf "🔴 %s" .Namespace) -}}
{{- else if contains .Namespace "staging" -}}
  {{- color "yellow" (printf "🟡 %s" .Namespace) -}}
{{- else if contains .Namespace "dev" -}}
  {{- color "green" (printf "🟢 %s" .Namespace) -}}
{{- else -}}
  {{- .Namespace -}}
{{- end -}}`,

	"deployment-status": `{{- /* Deployment Status Formatter */ -}}
{{- $desired := .Spec.Replicas | default 1 -}}
{{- $ready := .Status.ReadyReplicas | default 0 -}}
{{- $available := .Status.AvailableReplicas | default 0 -}}
{{- $updated := .Status.UpdatedReplicas | default 0 -}}

{{- if eq $ready $desired -}}
  {{- color "green" "●" }} {{ color "green" "Ready" -}}
{{- else if eq $ready 0 -}}
  {{- color "red" "✗" }} {{ color "red" "NotReady" -}}
{{- else if lt $updated $desired -}}
  {{- color "yellow" "◐" }} {{ color "yellow" "Updating" -}}
{{- else -}}
  {{- color "yellow" "◑" }} {{ color "yellow" "Progressing" -}}
{{- end -}}`,

	"ingress-status": `{{- /* Ingress Status Formatter */ -}}
{{- $hasAddress := false -}}
{{- range .Status.LoadBalancer.Ingress -}}
  {{- if or .IP .Hostname -}}
    {{- $hasAddress = true -}}
  {{- end -}}
{{- end -}}

{{- if $hasAddress -}}
  {{- color "green" "●" }} {{ color "green" "Ready" -}}
{{- else -}}
  {{- color "yellow" "◐" }} {{ color "yellow" "Pending" -}}
{{- end -}}`,

	"configmap-data": `{{- /* ConfigMap Data Count Formatter */ -}}
{{- $dataCount := add (len .Data) (len .BinaryData) -}}
{{- if eq $dataCount 0 -}}
  {{- color "gray" "0" -}}
{{- else if eq $dataCount 1 -}}
  {{- color "green" "1" -}}
{{- else if lt $dataCount 10 -}}
  {{- color "blue" (toString $dataCount) -}}
{{- else -}}
  {{- color "yellow" (toString $dataCount) -}}
{{- end -}}`,

	"secret-type": `{{- /* Secret Type Formatter */ -}}
{{- $type := .Type | toString -}}
{{- if eq $type "Opaque" -}}
  {{- color "blue" "📄" }} {{ color "blue" "Opaque" -}}
{{- else if eq $type "kubernetes.io/tls" -}}
  {{- color "green" "🔐" }} {{ color "green" "TLS" -}}
{{- else if contains $type "dockercfg" -}}
  {{- color "cyan" "🐳" }} {{ color "cyan" "Docker" -}}
{{- else if contains $type "service-account" -}}
  {{- color "purple" "👤" }} {{ color "purple" "ServiceAccount" -}}
{{- else -}}
  {{- color "gray" $type -}}
{{- end -}}`,

	"node-status": `{{- /* Node Status Formatter */ -}}
{{- $ready := false -}}
{{- range .Status.Conditions -}}
  {{- if and (eq .Type "Ready") (eq .Status "True") -}}
    {{- $ready = true -}}
  {{- end -}}
{{- end -}}

{{- if $ready -}}
  {{- color "green" "●" }} {{ color "green" "Ready" -}}
{{- else -}}
  {{- color "red" "✗" }} {{ color "red" "NotReady" -}}
{{- end -}}`,

	"pv-status": `{{- /* PersistentVolume Status Formatter */ -}}
{{- if eq .Status.Phase "Available" -}}
  {{- color "green" "●" }} {{ color "green" "Available" -}}
{{- else if eq .Status.Phase "Bound" -}}
  {{- color "blue" "◉" }} {{ color "blue" "Bound" -}}
{{- else if eq .Status.Phase "Released" -}}
  {{- color "yellow" "◐" }} {{ color "yellow" "Released" -}}
{{- else if eq .Status.Phase "Failed" -}}
  {{- color "red" "✗" }} {{ color "red" "Failed" -}}
{{- else -}}
  {{- color "gray" "○" }} {{ color "gray" .Status.Phase -}}
{{- end -}}`,

	"pvc-status": `{{- /* PersistentVolumeClaim Status Formatter */ -}}
{{- if eq .Status.Phase "Bound" -}}
  {{- color "green" "●" }} {{ color "green" "Bound" -}}
{{- else if eq .Status.Phase "Pending" -}}
  {{- color "yellow" "◐" }} {{ color "yellow" "Pending" -}}
{{- else if eq .Status.Phase "Lost" -}}
  {{- color "red" "✗" }} {{ color "red" "Lost" -}}
{{- else -}}
  {{- color "gray" "○" }} {{ color "gray" .Status.Phase -}}
{{- end -}}`,

	"job-status": `{{- /* Job Status Formatter */ -}}
{{- $succeeded := .Status.Succeeded | default 0 -}}
{{- $failed := .Status.Failed | default 0 -}}
{{- $active := .Status.Active | default 0 -}}

{{- if gt $succeeded 0 -}}
  {{- color "green" "✓" }} {{ color "green" "Completed" -}}
{{- else if gt $failed 0 -}}
  {{- color "red" "✗" }} {{ color "red" "Failed" -}}
{{- else if gt $active 0 -}}
  {{- color "yellow" "◐" }} {{ color "yellow" "Running" -}}
{{- else -}}
  {{- color "gray" "○" }} {{ color "gray" "Pending" -}}
{{- end -}}`,

	"cronjob-status": `{{- /* CronJob Status Formatter */ -}}
{{- $suspended := .Spec.Suspend | default false -}}
{{- $lastSchedule := .Status.LastScheduleTime -}}

{{- if $suspended -}}
  {{- color "gray" "⏸" }} {{ color "gray" "Suspended" -}}
{{- else if $lastSchedule -}}
  {{- color "green" "●" }} {{ color "green" "Active" -}}
{{- else -}}
  {{- color "yellow" "◐" }} {{ color "yellow" "Waiting" -}}
{{- end -}}`,

	"event-type": `{{- /* Event Type Formatter */ -}}
{{- if eq .Type "Normal" -}}
  {{- color "green" "ℹ" }} {{ color "green" .Reason -}}
{{- else if eq .Type "Warning" -}}
  {{- color "yellow" "⚠" }} {{ color "yellow" .Reason -}}
{{- else -}}
  {{- color "red" "✗" }} {{ color "red" .Reason -}}
{{- end -}}`,

	"resource-version": `{{- /* Resource Version Formatter */ -}}
{{- $version := .Metadata.ResourceVersion -}}
{{- if $version -}}
  {{- color "blue" $version -}}
{{- else -}}
  {{- color "gray" "-" -}}
{{- end -}}`,

	"labels": `{{- /* Labels Formatter - Shows important labels */ -}}
{{- $important := list "app" "version" "env" "tier" "component" -}}
{{- $labels := list -}}
{{- range $key, $value := .Metadata.Labels -}}
  {{- if has $key $important -}}
    {{- $labels = append $labels (printf "%s=%s" $key $value) -}}
  {{- end -}}
{{- end -}}
{{- if gt (len $labels) 3 -}}
  {{- join (slice $labels 0 3) ", " }}...
{{- else -}}
  {{- join $labels ", " -}}
{{- end -}}`,

	"annotations": `{{- /* Annotations Formatter - Shows count */ -}}
{{- $count := len .Metadata.Annotations -}}
{{- if eq $count 0 -}}
  {{- color "gray" "0" -}}
{{- else if lt $count 5 -}}
  {{- color "green" (toString $count) -}}
{{- else if lt $count 15 -}}
  {{- color "yellow" (toString $count) -}}
{{- else -}}
  {{- color "red" (toString $count) -}}
{{- end -}}`,
}

// GetDefaultTemplate returns a default template by name
func GetDefaultTemplate(name string) (string, bool) {
	template, exists := DefaultTemplates[name]
	return template, exists
}

// GetAllDefaultTemplates returns all default template names
func GetAllDefaultTemplates() []string {
	var names []string
	for name := range DefaultTemplates {
		names = append(names, name)
	}
	return names
}

// IsDefaultTemplate checks if a template name is a built-in default
func IsDefaultTemplate(name string) bool {
	_, exists := DefaultTemplates[name]
	return exists
}
