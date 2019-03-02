+++
title = "logs"
+++
## cdsctl workflow logs

`Manage CDS Workflow Run Logs`

### Synopsis

Download logs from a workflow run.

	# list all logs files on latest run
	$ cdsctl workflow logs list KEY WF

	# list all logs files on run number 1
	$ cdsctl workflow logs list KEY WF 1

	# download all logs files on latest run
	$ cdsctl workflow logs download KEY WF

	# download only one file, for run number 1
	$ cdsctl workflow logs download KEY WF 1 --pattern="MyJob"
	# this will download file WF-1.0-pipeline.myPipeline-stage.MyStage-job.MyJob-status.Success-step.0.log



### Options

```
  -h, --help   help for logs
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
* [cdsctl workflow logs download](/manual/components/cdsctl/workflow/logs/download/)	 - `Download logs from a workflow run.`
* [cdsctl workflow logs list](/manual/components/cdsctl/workflow/logs/list/)	 - `List logs from a workflow run`

