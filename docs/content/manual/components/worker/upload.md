+++
title = "upload"
+++
## worker upload

`worker upload --tag=tagValue {{.cds.workspace}}/fileToUpload`

### Synopsis


Inside a job, there are two ways to upload an artifact:

* with a step using action Upload Artifacts
* with a step script (https://ovh.github.io/cds/workflows/pipelines/actions/builtin/script/), using the worker command: `worker upload <path>`

`worker upload --tag={{.cds.version}} {{.cds.workspace}}/files*.yml`

		

```
worker upload [flags]
```

### Options

```
  -h, --help         help for upload
      --tag string   Tag for artifact Upload - Tag is mandatory
```

### SEE ALSO

* [worker](/cli/worker/worker/)	 - CDS Worker

