{{/*
Expand the name of the chart.
*/}}
{{- define "chaoskube.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "chaoskube.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "chaoskube.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "chaoskube.labels" -}}
helm.sh/chart: {{ include "chaoskube.chart" . }}
{{ include "chaoskube.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "chaoskube.selectorLabels" -}}
app.kubernetes.io/name: {{ include "chaoskube.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "chaoskube.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "chaoskube.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Returns the port of the metrics endpoint, defaulting to 8080 if not set in args.
We need to use some failure handling when default value '.Values.chaoskube.args' is used, otherwise we see
errors like: wrong type for value; expected map[string]interface {}; got interface {}
*/}}
{{- define "chaoskube.metricsPort" -}}
{{- $args := .Values.chaoskube.args -}}
{{- $metricsAddr := index $args "metrics-address" -}}
{{- $metricsPort := ($metricsAddr | toString | split ":")._1 -}}

{{ printf "%s" ($metricsPort | default "8080") -}}
{{- end -}}
