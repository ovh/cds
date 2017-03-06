# TMPL

CDS plugin to create a file from a template file using `text/template` package

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

- **file** : Template file with golang text/template variables.
- **output** : Output file (optionnal, default to <file>.out or just trimming .tpl extension)
- **params** : Parameters to pass on the template file (key=value newline separated list)

## Extra

sources : https://golang.org/pkg/text/template/
