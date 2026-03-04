{{- define "sdp-enterprise.tenantName" -}}
{{- printf "adapter-controller-%s" . | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "sdp-enterprise.tenantLabels" -}}
app.kubernetes.io/name: adapter-controller
app.kubernetes.io/part-of: sdp-enterprise
app.kubernetes.io/managed-by: Helm
{{- end -}}
