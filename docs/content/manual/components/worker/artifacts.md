+++
title = "artifacts"
+++
## worker artifacts

`worker artifacts [--workflow=<workflow-name>] [--number=<run-number>] [--tag=<tag>] [--pattern=<pattern>]`

### Synopsis


Inside a job, you can list artifacts of a workflow:

	worker artifacts --pattern="files.*.yml"

	#theses two commands have the same result:
	worker artifacts
	worker artifacts --workflow={{.cds.workflow}} --number={{.cds.run.number}}

		

```
worker artifacts [flags]
```

### Options

```
  -h, --help              help for artifacts
      --number string     Workflow Number. Optional, default: current workflow run
      --pattern string    Pattern matching files to list. Optional, default: *
      --tag string        Tag matching files to list. Optional
      --workflow string   Workflow name. Optional, default: current workflow
```

### SEE ALSO

* [worker](/cli/worker/worker/)	 - CDS Worker

