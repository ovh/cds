---
title: "download"
notitle: true
notoc: true
---
# worker download

`worker download [--workflow=<workflow-name>] [--number=<run-number>] [--tag=<tag>] [--pattern=<pattern>]`

## Synopsis


Inside a job, there are two ways to download an artifact:

* with a step using action Download Artifacts
* with a step script (https://ovh.github.io/cds/docs/actions/builtin-script/), using the worker command.

Worker Command:

	worker download --tag=<tag> <path>

Example:

	worker download --pattern="files.*.yml"

Theses two commands have the same result:

	worker download
	worker download --workflow={{.cds.workflow}} --number={{.cds.run.number}}

		

```
worker download [flags]
```

## Options

```
      --number string     Workflow Number to download from. Optional, default: if workflow is the current workflow: current run, else latest run
      --pattern string    Pattern matching files to download. Optional, default: *
      --tag string        Tag matching files to download. Optional
      --workflow string   Workflow name to download from. Optional, default: current workflow
```

## SEE ALSO

* [worker](/docs/components/worker/worker/)	 - CDS Worker

