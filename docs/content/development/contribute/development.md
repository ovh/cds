---
title: "Development Environment"
weight: 3
card: 
  name: contribute
---

Before contributing to CDS, you'll need to install your
development environment. 

* PostgreSQL
* Redis
* Node.js
* Golang
* CDS

If you are familiar with these different tools, you probably will not need to read this page :-)

## PostgreSQL

Download PostgreSQL from https://www.postgresql.org/download/, version >= 9.6. Version 14.0 recommanded

You can easily use only PostgreSQL binaries, downloaded from https://www.enterprisedb.com/download-postgresql-binaries.

Initialize DB by running:

```bash
$ mkdir -p ~/data/postgres/data
$ initdb -D ~/data/postgres/data
```

Create user `cds` and database `cds`

```
$ psql -d postgres
postgres=# create user cds with password 'cds';
postgres=# create database cds owner cds;
postgres=# create database cdn owner cds;
postgres=\q
```

Then launch PostgreSQL with:

```bash
$ postgres -D ~/data/postgres/data
```

Note: this is not recommended for a production installation.

That's all for a local PostgreSQL installation

## Redis

Download the latest stable Redis from https://redis.io/download, version >= 3.2

Example with version 4.0.11:

```bash
$ wget http://download.redis.io/releases/redis-4.0.11.tar.gz
$ tar xzf redis-4.0.11.tar.gz
$ cd redis-4.0.11
$ make
# launch redis-server
$ src/redis-server
# you should add src/ to your PATH
```

That's all for a local Redis installation.


## Node.js

Download the latest stable Node.js from https://nodejs.org/en/download/current/, version >= 16.4.2

Example with version 16.4.2 on macOS:

```bash
$ curl -O https://nodejs.org/dist/v16.4.2/node-v16.4.2-darwin-x64.tar.gz
$ tar xzf node-v16.4.2-darwin-x64.tar.gz
# directory node-v16.4.2-darwin-x64 is created
# You should add node-v16.4.2-darwin-x64/bin to your PATH
```

## Golang

Download the latest Golang version from https://golang.org/dl/, version >= 1.18

Example with version 1.18 on macOS:

```bash
$ export GOROOT=~/go
$ export PATH=$PATH:$GOROOT/bin
$ cd ~
$ curl -O https://dl.google.com/go/go1.18.darwin-amd64.tar.gz
$ tar xzf go1.18.darwin-amd64.tar.gz
```

Check if Go installation is ok

```bash
$ go version
go version go1.18 darwin/amd64
```

## CDS

Compile CDS:

```bash
# Checkout code
$ mkdir -p $(go env GOPATH)/src/github.com/ovh
$ cd $(go env GOPATH)/src/github.com/ovh
$ git clone https://github.com/ovh/cds.git

# Compile everything
$ cd $(go env GOPATH)/src/github.com/ovh/cds
$ make clean # useful if you had already compile CDS before
$ make build

# if you want to build only one OS/ARCH, you can do for linux/amd64:
$ make build OS="linux" ARCH="amd64"
```

All binaries are stored in the `dist/` directory

Configure CDS:

```bash
# Generate default configuration file
$ engine config new > ~/.cds/dev.toml

# edit ~/.cds/dev.toml file 
## in section [api]
### --> set variable defaultOS to your OS, darwin if you are on macOS for example

## in section [hatchery.local.commonConfiguration]
### --> set name to "hatchery-local"

## in section [hatchery.local.commonConfiguration.api.http]
### --> uncomment url, should be set to url = "http://localhost:8081" 

## in section [hatchery.local]
### basedir = "/tmp/cds" 
# this directory will contains the cds workers workspace

## in section [api.directories]
### baseDirectory = "/your-gopath/src/github.com/ovh/cds/engine/worker/dist" 
# this directory should contains the workers binaries

## in section [api.artifact.local]
# baseDirectory = "/tmp/cds/artifacts"

```

Prepare database:

This command will create tables, indexes and initial data on CDS database.
you have to launch it each time you have to upgrade cds.

```bash
$ cd $(go env GOPATH)/src/github.com/ovh/cds
$ engine database upgrade --db-password cds --db-sslmode disable
```

If you don't have a local PostgreSQL, you should run `engine database upgrade --help`
and update `~/.cds/dev.toml` file.

Launch CDS engine API:

```bash
$ engine --config ~/.cds/dev.toml start api
```

Launch CDS UI:

```bash
$ cd $(go env GOPATH)/src/github.com/ovh/cds/ui
$ npm start
```

Register first user with cdsctl:

```bash
# INIT_TOKEN is used to create the first user as an administrator of CDS.
export INIT_TOKEN=`./engine config init-token --config ~/.cds/dev.toml`
$ cdsctl signup -H http://localhost:8081 --email your-username@localhost.local --fullname yourFullname --username your-username
# Check CDS API logs to get the validation code
```

Launch local hatchery:

```bash
$ engine --config ~/.cds/dev.toml start hatchery:local
```

Open a browser, go on http://localhost:8080 - Have fun.

## Notes

If you want to launch uService on different process:

```bash
# launch API only
$ engine --config ~/.cds/dev.toml start api

# launch local hatchery only
$ engine --config ~/.cds/dev.toml start hatchery:local
```

If you want to launch vcs & hooks µServices, you have to:

- set name in sections `[vcs]` and `[hooks]`
- uncomment API URL in sections `[vcs.api.http]` and `[hooks.api.http]`
- for vcs uService, please read tutorial on [GitHub]({{< relref "/docs/integrations/github/_index.md" >}}), [Bitbucket Server]({{< relref "/docs/integrations/bitbucket.md" >}}) or [GitLab]({{< relref "/docs/integrations/gitlab/_index.md" >}}).

Of course, you have to do the same thing with other µServices `repositories`, `elasticsearch`, `hatchery.swarm`, etc...

A remark / question / suggestion, feel free to join us on https://github.com/ovh/cds/discussions