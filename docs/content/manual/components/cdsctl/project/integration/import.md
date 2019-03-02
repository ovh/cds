+++
title = "import"
+++
## cdsctl project integration import

`Import a integration configuration on a project from a yaml file`

### Synopsis

`Import a integration configuration on a project from a yaml file`

```
cdsctl project integration import [ PROJECT-KEY ] FILENAME [flags]
```

### Examples

```
cdsctl integration import MY-PROJECT file.yml
```

### Options

```
      --force   
  -h, --help    help for import
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

