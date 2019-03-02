+++
title = "pull"
+++
## cdsctl template pull

`Pull CDS workflow template`

### Synopsis

`Pull CDS workflow template`

```
cdsctl template pull [TEMPLATE-PATH] [flags]
```

### Examples

```
cdsctl template pull group-name/template-slug
```

### Options

```
      --force               Force, may override files
  -h, --help                help for pull
  -d, --output-dir string   Output directory (default ".cds")
      --quiet               If true, do not output filename created
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl template](/manual/components/cdsctl/template/)	 - `Manage CDS workflow template`

