## Setup a docker hatchery

Run all your builds inside docker containers to preserve isolation.

## Add docker worker models

Hatchery in docker mode looks for [worker models](/doc/overview/model.md) of type 'docker' to start.

We will add a worker model to build Go applications:
```shell
$ cds worker model add golang docker --image=golang:latest
```

Add Go binary capability to model:

```shell
$ cds worker model capability add golang go binary go
```

## Start hatchery

To start a new hatchery in docker mode, download hatchery binary on a host with docker and run:

```shell
$ hatchery --mode=docker --api=https://<api.domain> --cds-user=<user> --cds-password=<user.domain>
```

