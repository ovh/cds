+++
title = "delete"
+++
## cdsctl worker model delete

`Delete a CDS worker model`

### Synopsis

`Delete a CDS worker model`

```
cdsctl worker model delete NAME ... [flags]
```

### Examples

```
cdsctl worker model delete myModelA myModelB
```

### Options

```
      --force   Force delete without confirmation and exit 0 if resource does not exist
  -h, --help    help for delete
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl worker model](/manual/components/cdsctl/worker/model/)	 - `Manage Worker Model`

