+++
title = "status"
+++
## cdsctl workflow status

`Check the status of the run`

### Synopsis

`Check the status of the run`

```
cdsctl workflow status [ PROJECT-KEY WORKFLOW-NAME ] [RUN-NUMBER] [flags]
```

### Options

```
      --format string   Output format: plain|json|yaml (default "plain")
  -h, --help            help for status
      --track           Wait the workflow to be over
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

