+++
title = "list"
+++
## cdsctl admin services list

`List CDS services`

### Synopsis

`List CDS services`

```
cdsctl admin services list [flags]
```

### Options

```
      --fields string   Only display specified object fields. 'empty' will display all fields, 'all' will display all object fields, 'field1,field2' to select multiple fields
      --filter string   Filter output based on conditions provided
      --format string   Output format: table|json|yaml (default "table")
  -h, --help            help for list
  -q, --quiet           Only display object's key
  -t, --type string     Filter service by type: api, hatchery, hook, repository, vcs
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl admin services](/manual/components/cdsctl/admin/services/)	 - `Manage CDS services`

