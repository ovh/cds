+++
title = "upgrade"
+++
## engine database upgrade

`Upgrade schema`

### Synopsis

Migrates the database to the most recent version available.

```
engine database upgrade [flags]
```

### Examples

```
engine database upgrade --db-password=your-password --db-sslmode=disable --db-name=cds --migrate-dir=./sql

# If the directory --migrate-dir is not up to date with the current version, this
# directory will be automatically updated with the release from https://github.com/ovh/cds/releases
	
```

### Options

```
      --db-connect-timeout int   Maximum wait for connection, in seconds (default 10)
      --db-host string           DB Host (default "localhost")
      --db-maxconn int           DB Max connection (default 20)
      --db-name string           DB Name (default "cds")
      --db-password string       DB Password
      --db-port int              DB Port (default 5432)
      --db-role string           DB Role
      --db-sslmode string        DB SSL Mode: require (default), verify-full, or disable (default "require")
      --db-timeout int           Statement timeout value in milliseconds (default 3000)
      --db-user string           DB User (default "cds")
      --dry-run                  Dry run upgrade
  -h, --help                     help for upgrade
      --limit int                Max number of migrations to apply (0 = unlimited)
      --migrate-dir string       CDS SQL Migration directory (default "./engine/sql")
```

### SEE ALSO

* [engine database](/cli/engine/database/)	 - `Manage CDS database`

