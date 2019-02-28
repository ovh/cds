+++
title = "apply"
+++
## cdsctl template apply

`Apply CDS workflow template`

### Synopsis

`Apply CDS workflow template`

```
cdsctl template apply [ PROJECT-KEY WORKFLOW-NAME ] [TEMPLATE-PATH] [flags]
```

### Examples

```
cdsctl template apply project-key workflow-name group-name/template-slug
```

### Options

```
      --force               Force, may override files
  -h, --help                help for apply
      --import-as-code      If true, will import the generated workflow as code on given project
      --import-push         If true, will push the generated workflow on given project
  -n, --no-interactive      Set to not ask interactively for params
  -d, --output-dir string   Output directory (default ".cds")
  -p, --params strings      Specify params for template
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

* [cdsctl template](/cli/cdsctl/template/)	 - `Manage CDS workflow template`

