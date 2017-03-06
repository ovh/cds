
name = "CDS_GitClone"
description = "Clone git repository"

// Requirements
requirements = {
	"git" = {
		type = "binary"
		value = "git"
	}
	"bash" = {
		type = "binary"
		value = "bash"
	}
}

// Parameters
parameters = {
	 "branch" = {
		type = "string"
		value = "{{.git.branch}}"
	}
	"commit" = {
		type = "string"
		value = "{{.git.hash}}"
	}
	"directory" = {
		type = "string"
		description = "target directory"
	}
	"gitOptions" = {
		type = "string"
		description = "git clone options"
	}
	"url" = {
		type = "string"
		description = "git URL"
		value = "{{.cds.app.repo}}"
	}
}

// Steps
steps = [{
	script = <<EOF
#!/bin/bash
set -e

echo "action git from directory"
pwd
echo "running git clone {{.gitOptions}} {{.url}} -b {{.branch}} {{.directory}}"

git clone {{.gitOptions}} {{.url}} -b {{.branch}} {{.directory}}

if [ "x{{.commit}}" != "x" ] && [ "x{{.commit}}" != "x{{.git.hash}}" ];  then
	cd {{.directory}}
	git reset --hard {{.commit}} || true
fi
EOF
	}]
