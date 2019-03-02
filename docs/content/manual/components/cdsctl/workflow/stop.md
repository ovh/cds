+++
title = "stop"
+++
## cdsctl workflow stop

`Stop a CDS workflow or a specific node name`

### Synopsis

Stop a CDS workflow or a specific node name

```
cdsctl workflow stop [ PROJECT-KEY WORKFLOW-NAME ] [RUN-NUMBER] [NODE-NAME] [flags]
```

### Examples

```
cdsctl workflow stop # Stop the workflow run for the current repo and the current hash
cdsctl workflow stop MYPROJECT myworkflow 5 # To stop a workflow run on number 5
cdsctl workflow stop MYPROJECT myworkflow 5 compile # To stop a workflow node run on workflow run 5
	
```

### Options

```
  -h, --help   help for stop
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl workflow](/manual/components/cdsctl/workflow/)	 - `Manage CDS workflow`

