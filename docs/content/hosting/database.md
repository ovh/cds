---
title: "Database Management"
weight: 6
card: 
  name: operate
---


CDS provides all needed tools scripts to perform Schema creation and auto-migration. Those tools are embedded inside the `engine` binary.

The migration files are available to download on [GitHub Releases](https://github.com/ovh/cds/releases) and the archive is named `sql.tar.gz`. You have to download it and untar (`tar xvzf sql.tar.gz`).

## Creation

On a brand new database run the following command:

```bash
$ $PATH_TO_CDS/engine database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --db-schema=public --migrate-dir $PATH_TO_CDS/engine/sql/api --limit 0
$ PGPASSWORD=<password> psql -h <host> -p <port> -U <user> -d <database> -c "CREATE SCHEMA IF NOT EXISTS cdn AUTHORIZATION <user>;"
$ $PATH_TO_CDS/engine database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --db-schema=cdn --migrate-dir $PATH_TO_CDS/engine/sql/cdn --limit 0
```

## Upgrade

On an existing database, run the following command on each CDS update:

```bash
$ $PATH_TO_CDS/engine database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --db-schema=public --migrate-dir $PATH_TO_CDS/engine/sql/api
$ $PATH_TO_CDS/engine database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --db-schema=cdn --migrate-dir $PATH_TO_CDS/engine/sql/cdn
```

## More details

[Read more about CDS Database Management](https://github.com/ovh/cds/blob/master/engine/sql/README.md)
