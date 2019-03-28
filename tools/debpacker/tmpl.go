package main

const (
	controlTmpl = `Package: {{.PackageName}}
Architecture: {{.Architecture}}
Maintainer: {{.Maintainer}}
Priority: optional
Version: {{.Version}}
Description: {{.Description}}
Depends: {{ StringsJoin .Dependencies ", "}}

`

	systemdServiceTmpl = `[Unit]
Description={{.Description}}
After={{.SystemdServiceConfig.After}}

[Service]
{{if not (eq .SystemdServiceConfig.User "root") -}}
User={{.SystemdServiceConfig.User}}
Group={{.SystemdServiceConfig.User}}
{{end -}}
ExecStart={{.SystemdServiceConfig.ExecStart}} {{ StringsJoin .SystemdServiceConfig.ExecStartArgs " "}}
ExecStop={{.SystemdServiceConfig.ExecStop}}
Restart={{.SystemdServiceConfig.Restart}}
{{if .SystemdServiceConfig.WorkingDirectory -}}
WorkingDirectory={{.SystemdServiceConfig.WorkingDirectory}}
{{end -}}
{{with .SystemdServiceConfig.Environments -}}
{{ range $key, $value := . -}}
Environment="{{$key}}={{$value}}"
{{end -}}
{{end}}
[Install]
WantedBy={{.SystemdServiceConfig.WantedBy}}
`

	postinstTmpl = `#!/bin/bash
set -e
echo "Create the {{.SystemdServiceConfig.User}} User, Group and Directories"
{{if not (eq .SystemdServiceConfig.User "root") -}}
adduser --system --group {{.SystemdServiceConfig.User}}
{{end -}}
mkdir -p /var/lib/{{.PackageName}}
chown -R {{.SystemdServiceConfig.User}}:{{.SystemdServiceConfig.User}} /var/lib/{{.PackageName}}
chmod 770 /var/lib/{{.PackageName}}
{{if .SystemdServiceConfig.PostInstallCmd -}}
echo "Service initialization"
{{.SystemdServiceConfig.PostInstallCmd}}
{{end -}}
chmod +x {{.SystemdServiceConfig.ExecStart}}

echo "Service installed"
systemctl enable {{.PackageName}}
systemctl status {{.PackageName}}
echo "run systemctl start {{.PackageName}} to start"
`
)
