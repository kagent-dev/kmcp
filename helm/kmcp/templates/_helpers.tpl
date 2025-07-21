{{/*
Expand the name of the chart.
*/}}
{{- define "kmcp.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "kmcp.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kmcp.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kmcp.labels" -}}
helm.sh/chart: {{ include "kmcp.chart" . }}
{{ include "kmcp.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kmcp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kmcp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kmcp.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (printf "%s-controller-manager" (include "kmcp.fullname" .)) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the namespace to use
*/}}
{{- define "kmcp.namespace" -}}
{{- .Release.Namespace }}
{{- end }}

{{/*
Create the image reference
*/}}
{{- define "kmcp.image" -}}
{{- $tag := .Values.image.tag | default .Chart.Version }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Create controller manager container args
*/}}
{{- define "kmcp.controllerArgs" -}}
{{- $args := list }}
{{- if .Values.controller.leaderElection.enabled }}
{{- $args = append $args "--leader-elect" }}
{{- end }}
{{- if .Values.controller.healthProbe.bindAddress }}
{{- $args = append $args (printf "--health-probe-bind-address=%s" .Values.controller.healthProbe.bindAddress) }}
{{- end }}
{{- if .Values.controller.metrics.enabled }}
{{- $args = append $args (printf "--metrics-bind-address=%s" .Values.controller.metrics.bindAddress) }}
{{- end }}
{{- toYaml $args }}
{{- end }} 