---
title: "Run with binaries"
weight: 3
card: 
  name: ready-to-run
---

This article contains the steps to start CDS locally, with API, UI and a local Hatchery.

- Download CDS from GitHub
- Prepare Database
- Launch CDS API, CDS UI and a Local Hatchery

## Prerequisite

- a Redis
- a PostgreSQL 9.6 min

## Get the latest release from GitHub

```bash
mkdir $HOME/cds
cd cds

LAST_RELEASE=$(curl -s https://api.github.com/repos/ovh/cds/releases | grep tag_name | head -n 1 | cut -d '"' -f 4)
OS=linux # could be linux, darwin, windows, freebsd, openbsd
ARCH=amd64 # could be 386, arm, amd64, arm64, ppc64le

# GET Binaries from GitHub
curl -L https://github.com/ovh/cds/releases/download/$LAST_RELEASE/cds-engine-$OS-$ARCH -o cds-engine
curl -L https://github.com/ovh/cds/releases/download/$LAST_RELEASE/cdsctl-$OS-$ARCH -o cdsctl
# if you don't want to use the your keychain, you have to use this:
# curl -L https://github.com/ovh/cds/releases/download/$LAST_RELEASE/cdsctl-$OS-$ARCH-nokeychain -o cdsctl
chmod +x cds-engine cdsctl

```

## Generate configuration

Generate a **[Configuration File]({{<relref "/hosting/configuration.md" >}})**

```bash
cd $HOME/cds

./cds-engine config new > conf.toml
```

You will probably need to update some values in this file. If you need to automatize some update, you 
can use the `./cds-engine config edit` command.

Example:

```bash
mkdir -p $HOME/cds/download $HOME/cds/download $HOME/cds/hatchery-basedir
./cds-engine config edit conf.toml --output conf.toml \
  api.artifact.local.baseDirectory=$HOME/cds/artifacts \
  api.directories.download=$HOME/cds/download \
  hatchery.local.basedir=$HOME/cds/hatchery-basedir
```

## Prepare Cache

For this example, we consider that the redis is installed on `localhost`, port `6379` with no password.
You can edit the section `api.cache.redis` in `conf.toml` file if needed.

If it's just for test purpose, you can start a redis with docker, as:

```bash
docker run --name cds-cache -p 127.0.0.1:6379:6379 -d redis:5
```


## Prepare Database

For this example, we consider that the database is installed on `localhost`,
port `5432`, with an existing empty database and user named `cds` and a password 'cds'.

You can edit the section `api.database` in `conf.toml` file if needed.

If it's just for test purpose, you can start a postgreSQL database with docker, as:

```bash
docker run --name cds-db -e POSTGRES_PASSWORD=cds -e POSTGRES_USER=cds -e POSTGRES_DB=cds -p 127.0.0.1:5432:5432 -d postgres:9.6
```

```bash
cd $HOME/cds
./cds-engine download sql --config conf.toml
./cds-engine database upgrade --db-host localhost --db-user cds --db-password cds --db-name cds --dh-schema public --db-sslmode disable --db-port 5432 --migrate-dir sql/api
PGPASSWORD=cds psql -h localhost -U cds -d cds -c "CREATE SCHEMA IF NOT EXISTS cdn AUTHORIZATION cds;"
./cds-engine database upgrade --db-host localhost --db-user cds --db-password cds --db-name cds --dh-schema cdn --db-sslmode disable --db-port 5432 --migrate-dir sql/cdn
```

## Launch CDS API

Generate a **[Configuration File]({{<relref "/hosting/configuration.md" >}})**

```bash
cd $HOME/cds

./cds-engine download workers --config conf.toml
./cds-engine start api --config conf.toml
```

Check that CDS is up and running:

```bash
curl http://localhost:8081/mon/version
curl http://localhost:8081/mon/status
```

## Launch Signup & CDS UI

Signup with cdsctl. `INIT_TOKEN` is used to validate the user as an administrator.

```bash
export INIT_TOKEN=`./cds-engine config init-token --config conf.toml`
./cdsctl signup --api-url http://localhost:8081 --email admin@localhost.local --username admin --fullname admin
```

If you don't have email service configured you just have to check your CDS API logs to get the `cdsctl signup verify...` command to run.


```bash
cd $HOME/cds
./cdsctl signup verify --api-url ... # Get this command from the API Logs
./cds-engine download ui --config conf.toml
./cds-engine start ui --config conf.toml
```

Then, open a browser on http://localhost:8080/ .

## Launch CDS cdn

<b style="color: red">⚠ Do not activate CDN log processing in production yet. It's in active development.
Be sure that config flag 'enableLogProcessing' is set to false</b>

```bash
./cds-engine start cdn --config $HOME/cds/conf.toml
```

## Launch CDS Local Hatchery

Start the local hatchery:

```bash
./cds-engine start hatchery:local --config $HOME/cds/conf.toml

# notice that you can run api, ui, cdn and hatchery with one common only:
# ./cds-engine start api ui cdn hatchery:local --config $HOME/cds/conf.toml
```

## Note about CDS Engine

It is possible to start all services as a single process `$ ./cds-engine start api ui hooks hatchery:local --config config.toml`.

```bash
$ ./cds-engine start api hooks hatchery:local --config config.toml
Reading configuration file config.toml
Starting service api
...
Starting service ui
...
Starting service hooks
...
Starting service vcs
...
Starting service hatchery:local
...
```

For serious deployment, we strongly suggest to run each service as a dedicated process.

```bash

$ ./cds-engine start api --config config.toml

$ ./cds-engine start ui --config config.toml

$ ./cds-engine start hooks --config config.toml

$ ./cds-engine start vcs --config config.toml

$ ./cds-engine start hatchery:local --config config.toml
$ ./cds-engine start hatchery:docker --config config.toml
$ ./cds-engine start hatchery:swarm --config config.toml
$ ./cds-engine start hatchery:marathon --config config.toml
$ ./cds-engine start hatchery:openstack --config config.toml
$ ./cds-engine start hatchery:vsphere --config config.toml

```

You can scale as you want each of this component, you probably will have to create a configuration for each instance of each service expect the API.

```bash
$ ./cds-engine config new > config.api.toml # All API instance can share the same configuration.

$ cp config.api.toml config.hatchery.swarm-1.toml
$ cp config.api.toml config.hatchery.swarm-2.toml
$ cp config.api.toml config.hatchery.swarm-3.toml
$ cp config.api.toml config.hooks.toml
$ cp config.api.toml config.vcs.toml

$ vi config.hatchery.local.toml # Edit the file and keep only the [logs] and [hatchery]/[hatchery.local] sections
$ vi config.hatchery.docker.toml # Edit the file and keep only the [logs] and [hatchery]/[hatchery.docker] sections
$ vi config.hatchery.swarm-1.toml # Edit the file and keep only the [logs] and [hatchery]/[hatchery.swarm] sections
$ vi config.hatchery.swarm-2.toml # Edit the file and keep only the [logs] and [hatchery]/[hatchery.swarm] sections
$ vi config.hatchery.swarm-3.toml # Edit the file and keep only the [logs] and [hatchery]/[hatchery.swarm] sections
$ vi config.hooks.toml # Edit the file and keep only the [logs] and [hooks] sections
$ vi config.vcs.toml # Edit the file and keep only the [logs] and [vcs] sections

...
```

If you decide to use consul or vault to store your configuration, you will have to use different key/secrets to store each piece of the configuration

## Go further

- How to use OpenStack infrastructure to spawn CDS container [read more]({{< relref "/docs/integrations/openstack/openstack_compute.md" >}})
* Link CDS to a Repository Manager as [GitHub]({{< relref "/docs/integrations/github/_index.md" >}}), [Bitbucket Server]({{< relref "/docs/integrations/bitbucket.md" >}}) or [GitLab]({{< relref "/docs/integrations/gitlab/_index.md" >}}) setted up on your CDS Instance.
- Learn more about CDS variables [read more]({{< relref "/docs/concepts/variables.md" >}})
