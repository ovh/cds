+++
title = "import"
+++
## cdsctl project platform import

`Import a platform configuration on a project from a yaml file`

### Synopsis

`Import a platform configuration on a project from a yaml file`

```
cdsctl project platform import [ PROJECT-KEY ] FILENAME [flags]
```

### Examples

```
cdsctl project platform import MY-PROJECT file.yml
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

* [cdsctl project platform](/cli/cdsctl/project/platform/)	 - `Manage CDS project platforms`

