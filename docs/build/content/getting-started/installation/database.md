+++
title = "Database Management"

[menu.main]
parent = "installation"
identifier = "database"
weight = 4

+++


CDS provides all needed tools scripts to perform Schema creation and auto-migration. Those tools are embedded inside the `api` binary.

### Creation

On a brand new database run the following command:

```bash
$ $PATH_TO_CDS/api database upgrade --db-host <host> --db-host <port> --db-password <password> --db-name <database> --limit 0
```

### Upgrade

On an existing database, run the following command on each CDS update:

```bash
$ $PATH_TO_CDS/api database upgrade --db-host <host> --db-host <port> --db-password <password> --db-name <database>
```

### More details

[Read more about CDS Database Management](https://github.com/ovh/cds/tree/master/engine/sql)
