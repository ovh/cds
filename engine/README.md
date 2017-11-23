# CDS Engine

This is the core component of CDS.

This component is responsible and is the entry point to several µServices such as:

* API
* VCS
* Hooks
* [Hatcheries](https://ovh.github.io/cds/advanced/advanced.hatcheries/)

The API component is the core component and is mandatory in every setups.

To start CDS api, the mandatory dependencies are a [PostgreSQL database](https://www.postgresql.org/), a [Redis Server](https://redis.io/) and a path to the directory containing other CDS binaries, for serious usages you may need other third parties [Read more](https://ovh.github.io/cds/installation/installation.requirements/)

## Configuration
There are two ways to set up CDS:

- with [toml](https://github.com/toml-lang/toml) configuration
- with environment variables.

[Read more](https://ovh.github.io/cds/installation/installation.configuration/)
 
## Startup

A docker-compose file is provided for light deployment and dev environments, [follow this guide](https://ovh.github.io/cds/installation/installation.ready-to-run.docker-compose/).

For larger deployments you have to go deeper and read this [advanced startup page](https://ovh.github.io/cds/installation/installation.advanced/).

## Database management

CDS provides all needed tools scripts to perform Schema creation and auto-migration. Those tools are embedded inside the `engine` binary.

The migration files are available to download on [Github Releases](https://github.com/ovh/cds/releases) and the archive is named `sql.tar.gz`. You have to download it and untar (`tar xvzf sql.tar.gz`).

[Read more](https://ovh.github.io/cds/installation/installation.database/)