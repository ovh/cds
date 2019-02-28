+++
title = "purge"
+++
## cdsctl admin hooks purge

`Delete all executions for a task`

### Synopsis

`Delete all executions for a task`

```
cdsctl admin hooks purge UUID [flags]
```

### Examples

```
cdsctl admin hooks purge 5178ce1f-2f76-45c5-a203-58c10c3e2c73
```

### Options

```
  -h, --help   help for purge
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl admin hooks](/cli/cdsctl/admin/hooks/)	 - `Manage CDS Hooks tasks`

