+++
title = "worker"
weight = 4
+++
## worker

CDS Worker

### Synopsis

CDS Worker

```
worker [flags]
```

### Options

```
      --api string                   URL of CDS API
      --auto-update                  Auto update worker binary from CDS API
      --basedir string               This directory (default TMPDIR os environment var) will contains worker working directory and temporary files
      --booked-job-id int            Booked job id
      --booked-workflow-job-id int   Booked Workflow job id
      --force-exit                   If single_use=true, force exit. This is useful if it's spawned by an Hatchery (default: worker wait 30min for being killed by hatchery)
      --from-github                  Update binary from latest github release
      --graylog-extra-key string     Ex: --graylog-extra-key=xxxx-yyyy
      --graylog-extra-value string   Ex: --graylog-extra-value=xxxx-yyyy
      --graylog-host string          Ex: --graylog-host=xxxx-yyyy
      --graylog-port string          Ex: --graylog-port=12202
      --graylog-protocol string      Ex: --graylog-protocol=xxxx-yyyy
      --grpc-api string              CDS GRPC tcp address
      --grpc-insecure                Disable GRPC TLS encryption
      --hatchery-name string         Hatchery Name spawing worker
  -h, --help                         help for worker
      --insecure                     (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
      --log-level string             Log Level: debug, info, notice, warning, critical (default "notice")
      --model int                    Model of worker
      --name string                  Name of worker
      --single-use                   Exit after executing an action
      --token string                 CDS Token
      --ttl int                      Worker time to live (minutes) (default 30)
```

### SEE ALSO

* [worker artifacts](/cli/worker/artifacts/)	 - `worker artifacts [--workflow=<workflow-name>] [--number=<run-number>] [--tag=<tag>] [--pattern=<pattern>]`
* [worker cache](/cli/worker/cache/)	 - 
* [worker check-secret](/cli/worker/check-secret/)	 - `worker check-secret fileA fileB`
* [worker download](/cli/worker/download/)	 - `worker download [--workflow=<workflow-name>] [--number=<run-number>] [--tag=<tag>] [--pattern=<pattern>]`
* [worker exit](/cli/worker/exit/)	 - `worker exit`
* [worker export](/cli/worker/export/)	 - `worker export <varname> <value>`
* [worker key](/cli/worker/key/)	 - 
* [worker tag](/cli/worker/tag/)	 - `worker tag key=value key=value`
* [worker tmpl](/cli/worker/tmpl/)	 - `worker tmpl inputFile outputFile`
* [worker update](/cli/worker/update/)	 - `worker update [flags]`
* [worker upload](/cli/worker/upload/)	 - `worker upload --tag=tagValue {{.cds.workspace}}/fileToUpload`
* [worker version](/cli/worker/version/)	 - `Print the version of the worker binary`

