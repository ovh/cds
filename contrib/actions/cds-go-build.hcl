
name = "CDS_GoBuild"
description = "Go Build: compile a go program"

// Requirements
requirements = {
	"go" = {
		type = "binary"
		value = "go"
	}
	"bash" = {
		type = "binary"
		value = "bash"
	}
}

// Parameters
parameters = {
	 "package" = {
		type = "string"
		description = "go package to compile. Put host.ext/foo/bar for go build host.ext/foo/bar"
		value = ""
	}
	"binary" = {
		type = "string"
		description = "Binary name: Put foo for go build -o foo"
		value = "{{.cds.application}}"
	}
	"flags" = {
		type = "string"
		description = "flags for go build. Put -ldflags \"-X main.xyz=abc\" for go build -ldflags \"-X main.xyz=abc\""
		value = ""
	}
	"os" = {
		type = "list"
		description = "GOOS"
		value = "linux;darwin;windows"
	}
	"architecture" = {
		type = "list"
		description = "GOOS"
		value = "amd64;386;arm"
	}
	"artifactUpload" = {
		type = "boolean"
		description = "Upload Binary as CDS Artifact"
		value = "true"
	}
	"runGoGet" = {
		type = "boolean"
		description = "Run go get -u before go build"
		value = "false"
	}
	"detectRaceCondition" = {
		type = "boolean"
		description = "Enable data race detection. It's flag -race"
		value = "true"
	}
	"preRun" = {
		type = "text"
		description = "Pre-command. Example: export CGO_ENABLED=0"
		value = ""
	}
}

// Steps
steps = [{
  script = <<EAF
#!/bin/bash
set -e

export GOOS="{{.os}}"
export GOARCH="{{.architecture}}"

if [ ! -d "${GOPATH}/src/{{.package}}" ]; then
  echo "directory '${GOPATH}/src/{{.package}}' does not exist"
	echo "Please put your source under ${GOPATH}/src/{{.package}} before running this action"
	exit 1;
fi;

cd ${GOPATH}/src/{{.package}}

if [ "xtrue" == "x{{.runGoGet}}" ]; then
	go get -v
else
	echo "not running go get ({{.runGoGet}})";
fi;

GOARGS=""
if [ "x" != "x{{.binary}}" ]; then
  GOARGS=" -o {{.binary}}"
fi;

if [ "xtrue" == "x${{.detectRaceCondition}}" ]; then
  GOARGS="${{GOARGS}} -race"
fi;

if [ "x" != "x{{.preRun}}" ]; then
cat << EOF > preRun.sh
{{.preRun}}
EOF
chmod +x preRun.sh
./preRun.sh
fi;

go build -v {{.flags}} ${GOARGS}

if [ "xtrue" == "x{{.artifactUpload}}" ]; then
	worker upload --tag={{.cds.version}} ${GOPATH}/src/{{.package}}/{{.binary}}
else
	echo "artifact upload: {{.artifactUpload}}. So, artifact is not uploaded"
fi;

EAF
}]
