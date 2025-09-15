{{/*
Common labels
*/}}
{{- define "tekton-assist.labels" -}}
helm.sh/chart: {{ include "tekton-assist.chart" . }}
app.kubernetes.io/name: {{ include "tekton-assist.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/* Expand the name of the chart. */}}
{{- define "tekton-assist.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Common selector labels */}}
{{- define "tekton-assist.selector" -}}
app.kubernetes.io/name: {{ include "tekton-assist.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: tekton-assist
app.kubernetes.io/part-of: tekton-assist
{{- end -}}

{{/* Common annotations */}}
{{- define "tekton-assist.annotations" -}}
prometheus.io/scrape: "true"
prometheus.io/port: "9090"
prometheus.io/path: "/metrics"
{{- end -}}

{{/* Create chart name and version as used by the chart label. */}}
{{- define "tekton-assist.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}
