# CDS API

This is the core component of CDS.
To start CDS api, the only mandatory dependency is a PostgreSQL database and a path to the directory containing other CDS binaries.

There are two ways to set up CDS:

- with [toml](https://github.com/toml-lang/toml) configuration
- with environment variables.

## CDS API Third-parties

At the minimum, CDS needs a PostgreSQL Database >= 9.4 and a Redis Server. But for serious usage your may need:

- An LDAP Server for authentication
- An SMTP Server for mails
- A [Kafka](https://kafka.apache.org/) Broker to manage CDS events
- An [Openstack Swift](https://docs.openstack.org/developer/swift/) Tenant to store builds artifacts
- A [Vault](https://www.vaultproject.io/) or [Consul](https://www.consul.io/) server for configuration management

See Configuration template for more details.

## Database management

CDS provides all needed tools scripts to perform Schema creation and auto-migration. Those tools are embedded inside the `engine` binary.

The migration files are available to download on [Github Releases](https://github.com/ovh/cds/releases) and the archive is named `sql.tar.gz`. You have to download it and untar (`tar xvzf sql.tar.gz`).

### Creation

On a brand new database, run the following command:

```bash
$ $PATH_TO_CDS/engine database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --migrate-dir <pathToSQLMigrationDir> --limit 0
```

### Upgrade

On an existing database, run the following command on each CDS update:

```bash
$ $PATH_TO_CDS/engine database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --migrate-dir <pathToSQLMigrationDir>
```

### More details

[Read more about CDS Database Management](https://github.com/ovh/cds/tree/master/engine/sql)

## Configuration
### Start CDS with local configuration file

You can also generate a configuration file template with the following command:

```bash
$ $PATH_TO_CDS/engine config new my_conf_file.toml
```

Edit this file, then run CDS:

```bash
$ $PATH_TO_CDS/engine start api --config my_conf_file.toml
Reading configuration file my_new_file.toml
2017/04/04 16:33:17 [NOTICE]   Starting CDS server...
...
```

### Start CDS with Consul

Upload your `toml` configuration to consul:

```bash
$ consul kv put cds/config.api.toml -
<PASTE YOUR CONFIGURATION>
<ENDS WITH CRTL-D>
Success! Data written to: cds/config.api.toml
```

Run CDS:

```bash
$ $PATH_TO_CDS/engine --remote-config localhost:8500 --remote-config-key cds/config.api.toml
Reading configuration from localhost:8500
2017/04/04 16:11:25 [NOTICE]   Starting CDS server...
...
```