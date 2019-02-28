+++
title = "disable"
+++
## cdsctl worker disable

`Disable CDS workers`

### Synopsis

Disable one on more CDS worker by their names.

For example if your want to disable all CDS workers you can run:

$ cdsctl worker disable $(cdsctl worker list)

```
cdsctl worker disable NAME ... [flags]
```

### Options

```
  -h, --help   help for disable
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl worker](/cli/cdsctl/worker/)	 - `Manage CDS worker`

