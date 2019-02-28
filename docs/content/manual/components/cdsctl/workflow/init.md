+++
title = "init"
+++
## cdsctl workflow init

`Init a workflow`

### Synopsis

[WARNING] THIS IS AN EXPERIMENTAL FEATURE
Initialize a workflow from your current repository, this will create yml files and push them to CDS.

Documentation: https://ovh.github.io/cds/gettingstarted/firstworkflow/



```
cdsctl workflow init [PROJECT-KEY] [flags]
```

### Options

```
  -r, --from-remote   Initialize a workflow from your git origin
  -h, --help          help for init
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl workflow](/cli/cdsctl/workflow/)	 - `Manage CDS workflow`

