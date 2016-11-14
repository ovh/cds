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

## Generate token

Workers need to provide a token to API in order to register with correct permissions.
The hatchery will forward token to spawned workers.

You can generate a token for a given group using the CLI:

```shell
$ cds generate token --group shared.infra --expiration persistent
2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4
```
This token will allow a worker to build all applications group foo has access to.

*Note: You must be admin of the group in order to generate a token for this group*

Provide this key to the hatchery:

```shell
worker --api=<cds-api> --key=2706bda13748877c57029598b915d46236988c7c57ea0d3808524a1e1a3adef4
```

## Start hatchery

To start a new hatchery in docker mode, download hatchery binary on a host with docker and run:

```shell
$ hatchery docker --api=https://<api.domain> --token=<token>
```
