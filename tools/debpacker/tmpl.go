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

	preinstTmpl = `#!/bin/bash
set +e
{{if  (.NeedsService) -}}
if  [ -f "/lib/systemd/system/{{.PackageName}}.service" ];then
	systemctl stop {{.PackageName}}
	systemctl disable {{.PackageName}}
fi
systemctl daemon-reload
{{end -}}
`

	postinstTmpl = `#!/bin/bash
set -e
echo "Create the {{.SystemdServiceConfig.User}} User, Group and Directories"
{{if not (eq .SystemdServiceConfig.User "root") -}}
adduser --system --group {{.SystemdServiceConfig.User}} --home /home/{{.SystemdServiceConfig.User}} --shell /bin/bash
{{end -}}
mkdir -p /var/lib/{{.PackageName}}
chown -R {{.SystemdServiceConfig.User}}:{{.SystemdServiceConfig.User}} /var/lib/{{.PackageName}}
chmod 770 /var/lib/{{.PackageName}}
{{if .SystemdServiceConfig.PostInstallCmd -}}
echo "Service initialization"
{{.SystemdServiceConfig.PostInstallCmd}}
{{end -}}
{{if .SystemdServiceConfig.ExecStart -}}
chmod +x {{.SystemdServiceConfig.ExecStart}}
{{end -}}

set +e
{{if  (.NeedsService) -}}
echo "Service installed"
systemctl daemon-reload
systemctl enable {{.PackageName}}
{{if  (.SystemdServiceConfig.AutoStart) -}}
systemctl start {{.PackageName}}
{{end -}}
systemctl --no-pager status {{.PackageName}}
echo "run systemctl start {{.PackageName}} to start"
{{end -}}
`

	prermTmpl = `#!/bin/bash
set +e
{{if  (.NeedsService) -}}
systemctl stop {{.PackageName}}
systemctl disable {{.PackageName}}
systemctl daemon-reload
{{end -}}
`

	postrmTmpl = `#!/bin/bash
set +e
{{if  (.NeedsService) -}}
systemctl daemon-reload
{{end -}}
`
)
