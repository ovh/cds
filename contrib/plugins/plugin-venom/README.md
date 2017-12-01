# Plugin Venom

CDS plugin run venom https://github.com/ovh/venom

## How to build

Make sure go is installed and properly configured ($GOPATH must be set)

```shell
$ go test ./...
$ go build
```

## How to install

### Install as CDS plugin

As CDS admin:

- login to Web Interface,
- go to Action admin page,
- choose the plugin binary file freshly built
- click on "Add plugin"

## How to use

### Parameters

- **path** : Path containers yml venom files. Format: adirectory/, ./*aTest.yml, ./foo/b*/**/z*.yml. Default: "."
- **exclude** : Exclude some files, one file per line. Default: empty
- **parallel** : Launch Test Suites in parallel. Enter here number of routines. Default: 2
- **output** : Directory where output xunit result file. Default: "."
- **details** : Output Details Level: low, medium, high. Default: low
- **loglevel** : Log Level: debug, info, warn or error. Default: error
- **vars** : Empty: all {{.cds...}} vars will be rewrited. Otherwise, you can limit rewrite to some variables. Example, enter cds.app.yourvar,cds.build.foo,myvar=foo to rewrite {{.cds.app.yourvar}}, {{.cds.build.foo}} and {{.foo}}. Default: Empty

Add an extra step of type "junit" on your job to view results on CDS UI.
