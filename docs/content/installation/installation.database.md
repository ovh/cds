+++
title = "Database Management"
weight = 4

[menu.main]
parent = "installation"
identifier = "database"

+++


CDS provides all needed tools scripts to perform Schema creation and auto-migration. Those tools are embedded inside the `api` binary.

The migration files are available to download on [Github Releases](https://github.com/ovh/cds/releases) and the archive is named `sql.tar.gz`. You have to download it and untar (`tar xvzf sql.tar.gz`).

### Creation

On a brand new database run the following command:

```bash
$ $PATH_TO_CDS/api database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --migrate-dir <pathToSQLMigrationDir> --limit 0
```

### Upgrade

On an existing database, run the following command on each CDS update:

```bash
$ $PATH_TO_CDS/api database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --migrate-dir <pathToSQLMigrationDir>
```

### More details

[Read more about CDS Database Management](https://github.com/ovh/cds/blob/master/engine/sql/README.md)
