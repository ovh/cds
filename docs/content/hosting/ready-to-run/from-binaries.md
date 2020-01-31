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
- a PostgreSQL 9.5 min

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
curl -L https://github.com/ovh/cds/releases/download/$LAST_RELEASE/ui.tar.gz -o ui.tar.gz
curl -L https://github.com/ovh/cds/releases/download/$LAST_RELEASE/sql.tar.gz -o sql.tar.gz
chmod +x cds-engine cdsctl

```

## Generate configuration

Generate a **[Configuration File]({{<relref "/hosting/configuration.md" >}})**

```bash
cd $HOME/cds

./cds-engine config new > conf.toml
./cds-engine download workers --config conf.toml
./cds-engine start api --config conf.toml
```

## Prepare Database

For this example, we consider that the database is installed on `localhost`,
port `5432`, with an existing empty database and user named `cds` and a password 'cds'.

If it's just for test purpose, you can start a postgreSQL database with docker, as:

```
docker run --name cds-db -e POSTGRES_PASSWORD=cds -e POSTGRES_USER=cds -e POSTGRES_DB=cds -p 127.0.0.1:5432:5432 -d postgres:9.5
```

```bash
cd $HOME/cds
./cds-engine download sql --config conf.toml
./cds-engine database upgrade --db-host localhost --db-user cds --db-password cds --db-name cds --db-sslmode disable --db-port 5432 --migrate-dir sql
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

## Launch CDS UI

```bash
cd $HOME/cds
./cds-engine download ui --config conf.toml
./cds-engine start ui --config conf.toml
```

Then, open a browser on http://localhost:8080/ . You have to signup your first CDS user. It will be an administrator on CDS. In order to do that, just go on UI and click on signup or use `cdsctl signup`. If you don't have email service configured you just have to check your CDS API logs to have the confirmation link.

## Launch CDS Local Hatchery

The previously generated configuration file contains all CDS configuration.

To be able to start a local hatchery, enter a hatchery name in the section `hatchery.local.commonConfiguration`

```toml

...
[hatchery.local]

    # BaseDir for worker workspace
    basedir = "/tmp"

    [hatchery.local.commonConfiguration]

      # Name of Hatchery
      name = "my-local-hatchery"
...

```

Then, start the local hatchery


```bash
./cds-engine-linux-amd64 start hatchery:local --config $HOME/cds/conf.toml

# notice that you can run api and hatchery with one common only:
# ./cds-engine-linux-amd64 start api hatchery:local --config $HOME/cds/conf.toml
```

## Note about CDS Engine

It is possible to start all services as a single process `$ engine start api ui hooks hatchery:local --config config.toml`.

```bash
$ engine start api hooks hatchery:local --config config.toml
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

$ engine start api --config config.toml

$ engine start ui --config config.toml

$ engine start hooks --config config.toml

$ engine start vcs --config config.toml

$ engine start hatchery:local --config config.toml
$ engine start hatchery:docker --config config.toml
$ engine start hatchery:swarm --config config.toml
$ engine start hatchery:marathon --config config.toml
$ engine start hatchery:openstack --config config.toml
$ engine start hatchery:vsphere --config config.toml

```

You can scale as you want each of this component, you probably will have to create a configuration for each instance of each service expect the API.

```bash
$ engine config new > config.api.toml # All API instance can share the same configuration.

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
* Link CDS to a Repository Manager as [GitHub]({{< relref "/docs/integrations/github.md" >}}), [Bitbucket Server]({{< relref "/docs/integrations/bitbucket.md" >}}) or [GitLab]({{< relref "/docs/integrations/gitlab.md" >}}) setted up on your CDS Instance.
- Learn more about CDS variables [read more]({{< relref "/docs/concepts/variables.md" >}})
