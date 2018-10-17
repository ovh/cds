# Mesos/Marathon Deployment

This action helps you to deploy on Mesos/Marathon. Provide a `marathon.json` file to configure deployment.

Your `marathon.json` file can be templated with CDS variables `{{.cds.variables}}`.

Enable `waitForDeployment` option to ensure deployment is successful.

Enable `insecureSkipVerify` option if you want to use self-signed certificate.

## How to build

Make sure Go >=1.7 is installed and properly configured ($GOPATH must be set)

```shell
    $ mkdir -p $GOPATH/src/github/ovh/cds
    $ git clone $GOPATH/src/github/ovh/cds
    $ cd $GOPATH/src/github/ovh/cds/contrib/grpcplugins/action/marathon
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

- **configuration**: Marathon application configuration file (json format). It can contain variables "{{.cds.variables}}". Default is `marathon.json`
- **user**: Marathon User (please use project, application or environment variables). Default is `{{.cds.env.marathonUser}}`
- **password**: Marathon Password (please use project, application or environment variables). Default is `{{.cds.env.marathonPassword}}`
- **url**: Marathon URL like `http://127.0.0.1:8081,http://127.0.0.1:8082,http://127.0.0.1:8083`. Default is `{{.cds.env.marathonHost}}`
- **waitForDeployment**: Wait for instances deployment. If set, CDS will wait for all instances to be deployed until timeout is over. All instances deployment must be done to get a successful result. If not set, CDS will consider a successful result if Marathon accepts the provided.
- **timeout**: Marathon deployment timeout (seconds). Used only if "waitForDeployment" is true.
