+++
title = "list"
+++
## cdsctl workflow logs list

`List logs from a workflow run`

### Synopsis

List logs from a workflow run. There on log file for each step.

	# list all logs files from projet KEY, with workflow named WD on latest run
	$ cdsctl workflow logs list KEY WF

	# list all logs files from projet KEY, with workflow named WD on run 1
	$ cdsctl workflow logs list KEY WF 1



```
cdsctl workflow logs list [ PROJECT-KEY WORKFLOW-NAME ] [RUN-NUMBER] [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl workflow logs](/cli/cdsctl/workflow/logs/)	 - `Manage CDS Workflow Run Logs`

