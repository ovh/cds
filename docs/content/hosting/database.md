+++
title = "Database Management"
weight = 6

[menu.main]
parent = "hosting"
identifier = "hosting.database"

+++


CDS provides all needed tools scripts to perform Schema creation and auto-migration. Those tools are embedded inside the `engine` binary.

The migration files are available to download on [Github Releases](https://github.com/ovh/cds/releases) and the archive is named `sql.tar.gz`. You have to download it and untar (`tar xvzf sql.tar.gz`).

### Creation

On a brand new database run the following command:

```bash
$ $PATH_TO_CDS/engine database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --migrate-dir $PATH_TO_CDS/engine/sql --limit 0
```

### Upgrade

On an existing database, run the following command on each CDS update:

```bash
$ $PATH_TO_CDS/engine database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --migrate-dir $PATH_TO_CDS/engine/sql
```

### More details

[Read more about CDS Database Management](https://github.com/ovh/cds/blob/master/engine/sql/README.md)
