{{- define "dipole.name" -}}dipole{{- end -}}

{{- define "dipole.namespace" -}}dipole-system{{- end -}}

{{- define "dipole.labels" -}}
app.kubernetes.io/name: {{ include "dipole.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "dipole.controllerName" -}}dipole-agent-controller{{- end -}}
{{- define "dipole.injectorName" -}}dipole-agent-injector{{- end -}}
