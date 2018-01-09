+++
title = "Hosting your own instance"
weight = 6

[menu.main]
parent = ""
identifier = "hosting"

+++


## Introduction

This section will help you to undestand how to start all the differents components of CDS: UI, API and the other µServices.

## Pre-requisites

You need :

- to download the latest release
- a properly formatted [configuration file]({{< relref "configuration.md">}}),
- a properly configured [database]({{< relref "database.md">}}),
- the desired [third-parties]({{< relref "requirements.md">}}) up and running.

## Engine startup

The CDS engine is made of following components:

- API
- [Hatcheries (local, docker, openstack, swarm, vshpere)]({{< relref "hatchery/_index.md">}})
- Hooks

To start a services you just have to run `$PATH_TO_CDS/engine start <service>`.

**Caution: The API must always start first.**

```bash
$ engine start -h
Start CDS Engine Services:
 * API:
 	This is the core component of CDS.
 * Hatcheries:
	They are the components responsible for spawning workers. Supported platforms/orchestrators are:
	 * Local machine
	 * Local Docker
	 * Docker Swarm
	 * Openstack
	 * Vsphere
 * Hooks:
 	This component operates CDS workflow hooks
 * VCS:
    This component operates CDS VCS connectivity

Start all of this with a single command:
	$ engine start [api] [hatchery:local] [hatchery:docker] [hatchery:marathon] [hatchery:openstack] [hatchery:swarm] [hatchery:vsphere] [hooks]  [vcs]
All the services are using the same configuration file format.
You have to specify where the toml configuration is. It can be a local file, provided by consul or vault.
You can also use or override toml file with environment variable.

See $ engine config command for more details.

Usage:
  engine start [flags]

Flags:
      --config string              config file
      --remote-config string       (optional) consul configuration store
      --remote-config-key string   (optional) consul configuration store key (default "cds/config.api.toml")
      --vault-addr string          (optional) Vault address to fetch secrets from vault (example: https://vault.mydomain.net:8200)
      --vault-token string         (optional) Vault token to fetch secrets from vault

```

So it is possible to start all services as a single process `$ engine start api hooks hatchery:local --config config.toml`.

```bash
$ engine start api hooks hatchery:local --config config.toml
Reading configuration file config.toml
Starting service api
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

$ vi config.hatchery.local.toml # Edit the file an keep only the [logs] and [hatchery]/[hatchery.local] sections
$ vi config.hatchery.docker.toml # Edit the file an keep only the [logs] and [hatchery]/[hatchery.docker] sections
$ vi config.hatchery.swarm-1.toml # Edit the file an keep only the [logs] and [hatchery]/[hatchery.swarm] sections
$ vi config.hatchery.swarm-2.toml # Edit the file an keep only the [logs] and [hatchery]/[hatchery.swarm] sections
$ vi config.hatchery.swarm-3.toml # Edit the file an keep only the [logs] and [hatchery]/[hatchery.swarm] sections
$ vi config.hooks.toml # Edit the file an keep only the [logs] and [hooks] sections
$ vi config.vcs.toml # Edit the file an keep only the [logs] and [vcs] sections

...
```

If you decide to use consul or vault to store your configuration, you will have to use different key/secrets to store each piece of the configuration

## Web UI Startup

From the directory where you downloaded the release. Unarchive `ui.tar.gz`, it extract a dist directory.
Download and install [Caddy](https://caddyserver.com/download).

```bash
$ export BACKEND_HOST=<your http(s) URL to CDS API>
$ cd dist
$ caddy
```
