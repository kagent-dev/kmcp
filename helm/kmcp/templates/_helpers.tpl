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
{{- $tag := .Values.image.tag | default .Chart.AppVersion | default "latest" -}}
{{/* image.repository used to be one string carrying its registry
     (ghcr.io/kagent-dev/kmcp/controller). It is now the environment-invariant
     path only, joined onto image.registry -- the same registry/repository split
     every kagent-family chart uses, so one global.imageRegistry value redirects
     them all. A values file still carrying a host in repository would render a
     doubled path that fails only at pod start, as ImagePullBackOff, so it fails
     the render here instead and names the split. */}}
{{- $first := first (splitList "/" .Values.image.repository) -}}
{{- if or (contains "." $first) (contains ":" $first) -}}
{{- fail (printf "image.repository (%q) carries a registry host. It is now the image path only: move the host into image.registry (or global.imageRegistry) and keep repository as the path, e.g. registry: ghcr.io, repository: kagent-dev/kmcp/controller." .Values.image.repository) -}}
{{- end -}}
{{- include "kmcp.images.image" (dict "imageRoot" (dict "registry" .Values.image.registry "repository" .Values.image.repository "tag" $tag) "global" .Values.global) -}}
{{- end }}

{{/*
The resolved RBAC scope, as a JSON list so callers can range over it.
Precedence: rbac.namespaces > global.watchNamespaces > empty (cluster-scoped).
The global is a fallback, not an override: a values file that sets rbac.namespaces
renders exactly what it rendered before the global existed, and an explicit empty
list forces cluster-scoped RBAC (hasKey, not coalesce, so a present-but-empty key
wins). On the global path the install namespace is auto-appended: the global is a
shared signal a parent may aim at other charts, and failing this chart's render
over it would brick an install the value was never about.
*/}}
{{- define "kmcp.rbacNamespaces" -}}
{{- $scope := list -}}
{{- if and .Values.rbac (hasKey .Values.rbac "namespaces") -}}
{{- $scope = .Values.rbac.namespaces | default list -}}
{{- else if ((.Values.global).watchNamespaces) -}}
{{- $scope = concat (.Values.global).watchNamespaces (list (include "kmcp.namespace" .)) -}}
{{- end -}}
{{- $scope | uniq | sortAlpha | toJson -}}
{{- end -}}

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
{{- $watchNs := include "kmcp.rbacNamespaces" . | fromJsonArray }}
{{- if $watchNs }}
{{- $args = append $args (printf "--watch-namespaces=%s" (join "," $watchNs)) }}
{{- end }}
{{- toYaml $args }}
{{- end }} 
{{/*
Pull secrets for the pod: the chart's own list merged (union) with
global.imagePullSecrets. Renders nothing when both are empty.
*/}}
{{- define "kmcp.imagePullSecrets" -}}
{{- $merged := concat (.Values.imagePullSecrets | default list) (((.Values.global).imagePullSecrets) | default list) | uniq -}}
{{- if $merged -}}
imagePullSecrets:
{{- toYaml $merged | nindent 2 }}
{{- end -}}
{{- end -}}

{{/*
imagePullPolicy for a container: the component's own value, then
global.imagePullPolicy, then IfNotPresent. One definition so the fallback chain
cannot drift between pods.
*/}}
{{- define "kmcp.imagePullPolicy" -}}
{{- .local | default (((.root.Values.global)).imagePullPolicy) | default "IfNotPresent" -}}
{{- end -}}
