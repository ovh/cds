+++
title = "cds-go-build"

+++

Go Build: compile a go program

## Parameters

* **architecture**: GOARCH
* **artifactUpload**: Upload Binary as CDS Artifact
* **binary**: Binary name: Put foo for go build -o foo
* **cgoDisabled**: 
* **detectRaceCondition**: Enable data race detection. It's flag -race
* **flags**: flags for go build. Put -ldflags "-X main.xyz=abc" for go build -ldflags "-X main.xyz=abc"
* **os**: GOOS
* **package**: go package to compile. Put host.ext/foo/bar for go build host.ext/foo/bar
* **preRun**: Pre-command. Example: export CGO_ENABLED=0
* **runGoGet**: Run go get -u before go build


## Requirements

* **bash**: type: binary Value: bash
* **go**: type: binary Value: go


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-go-build.yml)


