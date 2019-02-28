+++
title = "run"
+++
## cdsctl workflow run

`Run a CDS workflow`

### Synopsis

`Run a CDS workflow`

```
cdsctl workflow run [ PROJECT-KEY WORKFLOW-NAME ] [flags]
```

### Options

```
  -d, --data string         Run the workflow with payload data
  -h, --help                help for run
  -i, --interactive         Follow the workflow run in an interactive terminal user interface
      --node-name string    Node Name to relaunch; Flag run-number is mandatory
  -o, --open-web-browser    Open web browser on the workflow run
  -p, --parameter strings   Run the workflow with pipeline parameter
      --run-number string   Existing Workflow RUN Number
  -s, --sync                Synchronise your pipelines with your last editions. Must be used with flag run-number
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

