---
title: "Artifact Upload"
card:
  name: builtin
---

**Artifact Upload** is a builtin action, you can't modify it.

This action can be used to upload artifacts in CDS. This is the recommended way to share files between pipelines or stages.

## Parameters

* **destination**: (optional) Destination of this artifact. Use the name of integration attached on your project.
* **enabled**: (optional) Enable artifact upload, "true" or "false".
* **path**: Path of file to upload, example: ./src/yourFile.json.
* **tag**: Artifact will be uploaded with a tag, generally {{.cds.version}}.


## Requirements

No Requirement

## YAML example

Example of a pipeline using Artifact Upload action:
```yml
version: v1.0
name: Pipeline1
stages:
- Stage1
jobs:
- job: Job1
  stage: Stage1
  steps:
  - artifactUpload:
      path: '{{.cds.workspace}}/myFile'
      tag: '{{.cds.version}}'

```


## Example

* Create a file `myfile` and upload it.

![img](../images/artifact-upload-job.png)

* Launch pipeline, check logs

![img](../images/artifact-upload-logs.png?width=500px)

* View artifact

![img](../images/artifact-upload-view-artifact.png)

## Worker Upload Command

You can upload an artifact with the built-in action - or use the worker command.

Example of a step script using [worker upload command]({{< relref "/docs/components/worker/upload.md" >}})

![img](../images/artifact-worker-upload.png)
