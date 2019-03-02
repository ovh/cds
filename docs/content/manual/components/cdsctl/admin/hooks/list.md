+++
title = "list"
+++
## cdsctl admin hooks list

`List CDS Hooks Tasks`

### Synopsis

`List CDS Hooks Tasks`

```
cdsctl admin hooks list [flags]
```

### Options

```
      --fields string   Only display specified object fields. 'empty' will display all fields, 'all' will display all object fields, 'field1,field2' to select multiple fields
      --filter string   Filter output based on conditions provided
      --format string   Output format: table|json|yaml (default "table")
  -h, --help            help for list
  -q, --quiet           Only display object's key
      --sort string     Sort task by nb_executions_total,nb_executions_todo
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl admin hooks](/manual/components/cdsctl/admin/hooks/)	 - `Manage CDS Hooks tasks`

