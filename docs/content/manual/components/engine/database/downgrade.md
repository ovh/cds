+++
title = "downgrade"
+++
## engine database downgrade

`Downgrade schema`

### Synopsis

Undo a database migration.

```
engine database downgrade [flags]
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
      --dry-run                  Dry run downgrade
  -h, --help                     help for downgrade
      --limit int                Max number of migrations to apply (0 = unlimited) (default 1)
      --migrate-dir string       CDS SQL Migration directory (default "./engine/sql")
```

### SEE ALSO

* [engine database](/manual/components/engine/database/)	 - `Manage CDS database`

