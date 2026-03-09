Package: {{.Package}}
Version: {{.Version}}
Architecture: {{.Architecture}}
Maintainer: {{.Maintainer}}
Section: {{.Section}}
Priority: {{.Priority}}
{{- if .Depends}}
Depends: {{.Depends}}
{{- end}}
{{- if .Homepage}}
Homepage: {{.Homepage}}
{{- end}}
Description: {{.Description}}
