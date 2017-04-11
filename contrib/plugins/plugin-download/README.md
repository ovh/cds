# Download

CDS plugin to download a file from an URL

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

- **url** : The url of your file.
- **filepath** : The destination of your file to be copied.
- **headers** : Headers to pass on the download request ("headerName"="value" newline separated list)
