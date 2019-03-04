+++
title = "list"
+++
## cdsctl project platform list

`List platforms available on a project`

### Synopsis

`List platforms available on a project`

```
cdsctl project platform list [ PROJECT-KEY ] [flags]
```

### Options

```
      --fields string   Only display specified object fields. 'empty' will display all fields, 'all' will display all object fields, 'field1,field2' to select multiple fields
      --filter string   Filter output based on conditions provided
      --format string   Output format: table|json|yaml (default "table")
  -h, --help            help for list
  -q, --quiet           Only display object's key
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl project platform](/cli/cdsctl/project/platform/)	 - `Manage CDS project platforms`

