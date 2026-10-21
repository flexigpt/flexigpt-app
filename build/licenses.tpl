{{ range . }}
Name: {{ .Name }}
Version: {{ .Version }}
License: [{{ .LicenseName }}]({{ .LicenseURL }})
License Copyright:
===

{{ .LicenseText }}

---
{{ end }}
