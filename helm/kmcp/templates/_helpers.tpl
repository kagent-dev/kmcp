{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "kmcp.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- if not .Values.nameOverride }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
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
app.kubernetes.io/name: {{ default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
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
Expand the namespace of the release.
Allows overriding it for multi-namespace deployments in combined charts.
*/}}
{{- define "kmcp.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Create the image reference
*/}}
{{- define "kmcp.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion | default "latest" }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Guards on the rbac block
*/}}
{{- define "kmcp.rbac.validate" -}}
{{- if and .Values.rbac (hasKey .Values.rbac "clusterScoped") -}}
{{- fail "rbac.clusterScoped has been removed. Leave rbac.namespaces empty for cluster-scoped RBAC, or set rbac.namespaces=[<ns>, ...] for namespaced RBAC." -}}
{{- end -}}
{{- if and .Values.rbac .Values.rbac.namespaces -}}
{{- $installNs := include "kmcp.namespace" . -}}
{{- if not (has $installNs .Values.rbac.namespaces) -}}
{{- fail (printf "rbac.namespaces is set but does not include the install namespace %q" $installNs) -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Report whether a PodDisruptionBudget field is set.
Outputs "true" when the value is neither nil nor an empty string, otherwise the
empty string. The integer 0 counts as set, so it is not silently dropped.
Usage: {{ eq (include "kmcp.pdb.isSet" .Values.podDisruptionBudget.minAvailable) "true" }}
*/}}
{{- define "kmcp.pdb.isSet" -}}
{{- if and (not (kindIs "invalid" .)) (ne (toString .) "") -}}
true
{{- end -}}
{{- end -}}

{{/*
Guards on the podDisruptionBudget block
*/}}
{{- define "kmcp.pdb.validate" -}}
{{- $pdb := .Values.podDisruptionBudget -}}
{{- $hasMin := eq (include "kmcp.pdb.isSet" $pdb.minAvailable) "true" -}}
{{- $hasMax := eq (include "kmcp.pdb.isSet" $pdb.maxUnavailable) "true" -}}
{{- if and $hasMin $hasMax -}}
{{- fail "podDisruptionBudget.minAvailable and podDisruptionBudget.maxUnavailable are mutually exclusive. Set exactly one of them." -}}
{{- end -}}
{{- if and (not $hasMin) (not $hasMax) -}}
{{- fail "podDisruptionBudget.enabled is true but neither podDisruptionBudget.minAvailable nor podDisruptionBudget.maxUnavailable is set. Set exactly one of them." -}}
{{- end -}}
{{- if eq (include "kmcp.pdb.isSet" $pdb.unhealthyPodEvictionPolicy) "true" -}}
{{- if not (has $pdb.unhealthyPodEvictionPolicy (list "AlwaysAllow" "IfHealthyBudget")) -}}
{{- fail (printf "podDisruptionBudget.unhealthyPodEvictionPolicy must be either \"AlwaysAllow\" or \"IfHealthyBudget\", got %q" (toString $pdb.unhealthyPodEvictionPolicy)) -}}
{{- end -}}
{{- end -}}
{{- end -}}

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
{{- if and .Values.rbac .Values.rbac.namespaces }}
{{- $namespaces := .Values.rbac.namespaces | uniq }}
{{- $args = append $args (printf "--watch-namespaces=%s" (join "," $namespaces)) }}
{{- end }}
{{- toYaml $args }}
{{- end }} 