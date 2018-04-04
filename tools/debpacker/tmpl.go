package main

import (
	"fmt"
	"html/template"
	"os"
	"strings"
)

const (
	controlTmpl = `Package: {{.PackageName}}
Architecture: {{.Architecture}}
Maintainer: {{.Maintainer}}
Priority: optional
Version: {{.Version}}
Description: {{.Description}}
Depends: {{ StringsJoin .Dependencies ", "}}

`

	postinstTmpl = `#!/bin/bash
echo "Create the {{.SystemdServiceConfig.User}} User, Group and Directories"
adduser --system --group --no-create-home {{.SystemdServiceConfig.User}}
mkdir -p /var/lib/{{.PackageName}}
chown -R {{.SystemdServiceConfig.User}}:{{.SystemdServiceConfig.User}} /var/lib/{{.PackageName}}
chmod 770 /var/lib/{{.PackageName}}
chmod +x {{.SystemdServiceConfig.ExecStart}}

echo "Starting service"
systemctl start {{.PackageName}}
systemctl status {{.PackageName}}
`

	systemdServiceTmpl = `[Unit]
Description={{.Description}}
After={{.SystemdServiceConfig.After}}

[Service]
User={{.PackageName}}
Group={{.PackageName}}
ExecStart={{.SystemdServiceConfig.ExecStart}} {{ StringsJoin .SystemdServiceConfig.ExecStartArgs " "}}
ExecStop={{.SystemdServiceConfig.ExecStop}}
Restart={{.SystemdServiceConfig.Restart}}
{{with .SystemdServiceConfig.Environments -}}
{{ range $key, $value := . -}}
Environment="{{$key}}={{$value}}"
{{- end}}
{{- end}}

[Install]
WantedBy={{.SystemdServiceConfig.WantedBy}}
`
)

func render(tmpl string, filename string, d DebPacker, perm os.FileMode) error {
	t := template.New("tmpl")
	t = t.Funcs(template.FuncMap{"StringsJoin": strings.Join})
	var err error
	t, err = t.Parse(tmpl)
	if err != nil {
		return err
	}

	fi, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, perm)
	if err != nil {
		return err
	}
	defer fi.Close()

	fmt.Println("creating file", filename)
	if err := t.Execute(fi, d); err != nil {
		return err
	}

	return nil
}
