---
title: "Artifact Upload"
card: 
  name: builtin
---

**Artifact Upload Action** is a builtin action, you can't modify it.

This action can be used to upload artifacts in CDS. This is the recommended way to share files between pipelines or stages.

## Parameters
* path: Path of file to upload, example: `./src/yourFile.json`
* tag: Artifact will be uploaded with a tag, generally `{{.cds.version}}`.
* enabled: Enable artifact upload, `true` or `false`
* destination: optional. Destination of this artifact. Use the name of integration attached on your project. Default is empty.

## Example

* Create a file `myfile` and upload it.

![img](../images/artifact-upload-job.png)


* Launch pipeline, check logs

![img](../images/artifact-upload-logs.png?width=500px)

* View artifact

![img](../images/artifact-upload-view-artifact.png)

## YML Format

Example of Pipeline using Artifact Upload Action

```yml
version: v1.0
name: test-artifacts
stages:
- Stage 1
jobs:
- job: JobWithUpload
  stage: Stage 1
  steps:
  - script:
    - echo "content file" > myfile
  - artifactUpload:
      path: myfile
      tag: '{{.cds.version}}'
```

## Worker Upload Command

You can upload an artifact with the built-in action - or use the worker command.

Example of a step script using [worker upload command]({{< relref "/docs/components/worker/upload.md" >}})

![img](../images/artifact-worker-upload.png)