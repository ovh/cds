# TMPL

CDS plugin run venom https://github.com/runabove/venom

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

- **path** : Path containers yml venom files
- **exclude** : exclude some files, one file per line
- **parallel** : Launch Test Suites in parallel, default: 2
- **output** : Directory where output xunit result file

Add an extra step of type "junit" on your job to view results on CDS UI.
