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
{{- if .Values.substrate.enabled }}
{{- $s := .Values.substrate }}
{{- if not $s.ateApiEndpoint }}
{{- fail "substrate.ateApiEndpoint is required when substrate.enabled is true" }}
{{- end }}
{{- $args = append $args (printf "--substrate-ate-api-endpoint=%s" $s.ateApiEndpoint) }}
{{- if $s.ateApiInsecure }}
{{- $args = append $args "--substrate-ate-api-insecure" }}
{{- end }}
{{- with $s.atenetRouterURL }}
{{- $args = append $args (printf "--substrate-atenet-router-url=%s" .) }}
{{- end }}
{{- with $s.actorHostSuffix }}
{{- $args = append $args (printf "--substrate-actor-host-suffix=%s" .) }}
{{- end }}
{{- with $s.defaultWorkerPool.namespace }}
{{- $args = append $args (printf "--substrate-default-workerpool-namespace=%s" .) }}
{{- end }}
{{- with $s.defaultWorkerPool.name }}
{{- $args = append $args (printf "--substrate-default-workerpool-name=%s" .) }}
{{- end }}
{{- with $s.snapshots.locationPrefix }}
{{- $args = append $args (printf "--substrate-snapshots-location-prefix=%s" .) }}
{{- end }}
{{- with $s.pauseImage }}
{{- $args = append $args (printf "--substrate-pause-image=%s" .) }}
{{- end }}
{{- with $s.runsc.amd64.url }}
{{- $args = append $args (printf "--substrate-runsc-amd64-url=%s" .) }}
{{- end }}
{{- with $s.runsc.amd64.sha256 }}
{{- $args = append $args (printf "--substrate-runsc-amd64-sha256=%s" .) }}
{{- end }}
{{- with $s.runsc.arm64.url }}
{{- $args = append $args (printf "--substrate-runsc-arm64-url=%s" .) }}
{{- end }}
{{- with $s.runsc.arm64.sha256 }}
{{- $args = append $args (printf "--substrate-runsc-arm64-sha256=%s" .) }}
{{- end }}
{{- with $s.adapterBinary.amd64.url }}
{{- $args = append $args (printf "--substrate-adapter-amd64-url=%s" .) }}
{{- end }}
{{- with $s.adapterBinary.amd64.sha256 }}
{{- $args = append $args (printf "--substrate-adapter-amd64-sha256=%s" .) }}
{{- end }}
{{- with $s.adapterBinary.arm64.url }}
{{- $args = append $args (printf "--substrate-adapter-arm64-url=%s" .) }}
{{- end }}
{{- with $s.adapterBinary.arm64.sha256 }}
{{- $args = append $args (printf "--substrate-adapter-arm64-sha256=%s" .) }}
{{- end }}
{{- include "kmcp.substrate.ingress.validate" . }}
{{- $args = append $args (printf "--substrate-ingress-mode=%s" $s.ingress.mode) }}
{{- if eq $s.ingress.mode "managed-proxy" }}
{{- $args = append $args (printf "--substrate-ingress-proxy-namespace=%s" (include "kmcp.namespace" .)) }}
{{- $args = append $args (printf "--substrate-ingress-proxy-configmap=%s" (include "kmcp.substrate.proxyConfigMapName" .)) }}
{{- $args = append $args (printf "--substrate-ingress-proxy-port=%d" (int $s.ingress.proxy.port)) }}
{{- end }}
{{- end }}
{{- toYaml $args }}
{{- end }}

{{/*
Guards on the substrate.ingress block
*/}}
{{- define "kmcp.substrate.ingress.validate" -}}
{{- $mode := .Values.substrate.ingress.mode -}}
{{- if eq $mode "gateway-api" -}}
{{- fail "substrate.ingress.mode=gateway-api is not implemented yet; use managed-proxy or none" -}}
{{- else if not (has $mode (list "managed-proxy" "none")) -}}
{{- fail (printf "invalid substrate.ingress.mode %q (supported: managed-proxy, none)" $mode) -}}
{{- end -}}
{{- end -}}

{{/*
Names of the shared substrate ingress proxy objects
*/}}
{{- define "kmcp.substrate.proxyName" -}}
{{- printf "%s-substrate-ingress-proxy" (include "kmcp.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kmcp.substrate.proxyConfigMapName" -}}
{{- include "kmcp.substrate.proxyName" . -}}
{{- end -}}

{{/*
Namespaces in which the substrate ate-api server may read Secrets/ConfigMaps
referenced by generated ActorTemplates. Comma-separated.
*/}}
{{- define "kmcp.substrate.envSourceNamespaces" -}}
{{- if .Values.substrate.envSourceNamespaces -}}
{{- join "," (.Values.substrate.envSourceNamespaces | uniq | sortAlpha) -}}
{{- else if and .Values.rbac .Values.rbac.namespaces -}}
{{- join "," (.Values.rbac.namespaces | uniq | sortAlpha) -}}
{{- else -}}
{{- include "kmcp.namespace" . -}}
{{- end -}}
{{- end }}