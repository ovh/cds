+++
title = "export"
+++
## cdsctl project integration export

`Export a integration configuration from a project to stdout`

### Synopsis

`Export a integration configuration from a project to stdout`

```
cdsctl project integration export [ PROJECT-KEY ] NAME [flags]
```

### Examples

```
cdsctl integration export MY-PROJECT MY-INTEGRATION-NAME > file.yaml
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

* [cdsctl project integration](/manual/components/cdsctl/project/integration/)	 - `Manage CDS integration integrations`

