+++
title = "import"
+++
## cdsctl workflow import

`Import a workflow`

### Synopsis


In case you want to import just your workflow.
		
If you want to update also dependencies likes pipelines, applications or environments at same time you have to use workflow push instead workflow import.

	

```
cdsctl workflow import [ PROJECT-KEY ] FILENAME [flags]
```

### Options

```
      --force   Override workflow if exists
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

* [cdsctl workflow](/cli/cdsctl/workflow/)	 - `Manage CDS workflow`

