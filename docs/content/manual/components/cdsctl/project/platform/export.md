+++
title = "export"
+++
## cdsctl project platform export

`Export a platform configuration from a project to stdout`

### Synopsis

`Export a platform configuration from a project to stdout`

```
cdsctl project platform export [ PROJECT-KEY ] NAME [flags]
```

### Examples

```
cdsctl project platform export MY-PROJECT MY-PLATFORM-NAME > file.yaml
```

### Options

```
  -h, --help   help for export
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

