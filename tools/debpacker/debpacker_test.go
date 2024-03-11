package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestNew(t *testing.T) {
	source := `package-name: test
binary-file: /bin/sh
`

	var config Config
	assert.Nil(t, yaml.Unmarshal([]byte(source), &config))

	p := New(nil, config, "")

	bs, err := yaml.Marshal(p.Config())
	assert.Nil(t, err)

	expected := `package-name: test
architecture: ""
binary-file: /bin/sh
version: ""
description: ""
maintainer: ""
systemd-configuration:
  after: network.target
  user: test
  args: []
  stop-command: /bin/kill $MAINPID
  restart: always
  wanted-by: multi-user.target
  environments: {}
`

	assert.Equal(t, expected, string(bs))
}

func TestPrepare(t *testing.T) {
	mw := &mockWriter{}

	p := New(mw, Config{
		PackageName:        "test",
		BinaryFile:         "/bin/sh",
		Architecture:       "arch",
		Maintainer:         "me",
		Version:            "0.0.0",
		Description:        "My test package",
		Dependencies:       []string{"one", "two"},
		ConfigurationFiles: []string{"./one.conf", "two.conf"},
		Mkdirs:             []string{"one", "two/three"},
		CopyFiles:          []string{"./one/*.conf", "two.conf:conf/files"},
		SystemdServiceConfig: SystemdServiceConfig{
			ExecStartArgs:    []string{"start", "server"},
			WorkingDirectory: "/var/lib/test",
			Environments:     map[string]string{"env1": "val1", "env2": "val2"},
		},
	}, "target")

	assert.Nil(t, p.Prepare())

	if !(assert.Equal(t, 8, len(mw.directories)) &&
		assert.Equal(t, 3, len(mw.copies)) &&
		assert.Equal(t, 6, len(mw.files))) {
		t.FailNow()
	}

	assert.Equal(t, directory{"target/test/DEBIAN", os.FileMode(0755)}, mw.directories[0])

	// buildControlFile
	assert.Equal(t, file{"target/test/DEBIAN/control", `Package: test
Architecture: arch
Maintainer: me
Priority: optional
Version: 0.0.0
Description: My test package
Depends: one, two

`, os.FileMode(0644)}, mw.files[0])

	// copyBinaryFile
	assert.Equal(t, directory{"target/test/usr/bin", os.FileMode(0755)}, mw.directories[1])
	assert.Equal(t, copy{targetPath: "target/test", path: "usr/bin", perm: os.FileMode(0744), sources: []string{"/bin/sh"}}, mw.copies[0])

	// copyConfigurationFiles
	assert.Equal(t, directory{"target/test/etc/test", os.FileMode(0755)}, mw.directories[2])
	assert.Equal(t, copy{targetPath: "target/test", path: "etc/test", perm: os.FileMode(0644), sources: []string{"./one.conf", "two.conf"}}, mw.copies[1])

	// mkDirs
	assert.Equal(t, directory{"target/test/var/lib/test", os.FileMode(0755)}, mw.directories[3])
	assert.Equal(t, directory{"target/test/var/lib/test/one", os.FileMode(0755)}, mw.directories[4])
	assert.Equal(t, directory{"target/test/var/lib/test/two/three", os.FileMode(0755)}, mw.directories[5])

	// copyOtherFiles
	assert.Equal(t, directory{"target/test/var/lib/test", os.FileMode(0755)}, mw.directories[6])
	assert.Equal(t, copy{targetPath: "target/test", path: "var/lib/test", perm: os.FileMode(0644), sources: []string{"./one/*.conf", "two.conf:conf/files"}}, mw.copies[2])

	// writeSystemdServiceFile
	assert.Equal(t, directory{"target/test/lib/systemd/system", os.FileMode(0755)}, mw.directories[7])
	assert.Equal(t, file{"target/test/lib/systemd/system/test.service", `[Unit]
Description=My test package
After=network.target

[Service]
User=test
Group=test
ExecStart=/usr/bin/sh start server
ExecStop=/bin/kill $MAINPID
Restart=always
WorkingDirectory=/var/lib/test
Environment="env1=val1"
Environment="env2=val2"

[Install]
WantedBy=multi-user.target
`, os.FileMode(0644)}, mw.files[1])

	assert.Equal(t, file{"target/test/DEBIAN/prerm", `#!/bin/bash
set +e
systemctl stop test
systemctl disable test
systemctl daemon-reload
`, os.FileMode(0755)}, mw.files[2])

	assert.Equal(t, file{"target/test/DEBIAN/postrm", `#!/bin/bash
set +e
systemctl daemon-reload
`, os.FileMode(0755)}, mw.files[3])

	assert.Equal(t, file{"target/test/DEBIAN/preinst", `#!/bin/bash
set +e
if  [ -f "/lib/systemd/system/test.service" ];then
	systemctl stop test
	systemctl disable test
fi
systemctl daemon-reload
`, os.FileMode(0755)}, mw.files[4])

	// writePostinstFile
	assert.Equal(t, file{"target/test/DEBIAN/postinst", `#!/bin/bash
set -e
echo "Create the test User, Group and Directories"
adduser --system --group test --home /home/test --shell /bin/bash
mkdir -p /var/lib/test
chown -R test:test /var/lib/test
chmod 770 /var/lib/test
chmod +x /usr/bin/sh
set +e
echo "Service installed"
systemctl daemon-reload
systemctl enable test
systemctl --no-pager status test
echo "run systemctl start test to start"
`, os.FileMode(0755)}, mw.files[5])
}
