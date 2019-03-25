---
title: "upload"
notitle: true
notoc: true
---
# worker upload

`worker upload --tag=tagValue {{.cds.workspace}}/fileToUpload`

## Synopsis


Inside a job, there are two ways to upload an artifact:

* with a step using action Upload Artifacts
* with a step script (https://ovh.github.io/cds/docs/actions/script/), using the worker command: `worker upload <path>`

`worker upload --tag={{.cds.version}} {{.cds.workspace}}/files*.yml`

You can use you storage integration: 
	worker upload --destination="yourStorageIntegrationName"
		

```
worker upload [flags]
```

## Options

```
      --destination string   optional. Your storage integration name
      --tag string           Tag for artifact Upload - Tag is mandatory
```

## SEE ALSO

* [worker](/docs/components/worker/worker/)	 - CDS Worker

