---
title: "Artifact Download"
card: 
  name: builtin
---


**Artifact Download Action** is a builtin action, you can't modify it.

This action can be used to retrieve an artifact previously uploaded by an [Artifact Upload]({{< relref "/docs/actions/artifact-upload.md" >}}) action.

## Parameters
* enabled: Enable artifact download
* path: Path where artifacts will be downloaded
* pattern: Empty: download all files. Otherwise, enter regexp pattern to choose file: `(fileA|fileB)`
* tag: Artifact are uploaded with a tag, generally `{{.cds.version}}`

## Example

* Workflow Configuration: a pipeline doing an `upload artifact` and another doing a `download artifact`.

![img](../images/artifact-download-workflow.png)

* Run pipeline, check logs

![img](../images/artifact-download-logs.png)

## YAML Format

Example of Pipeline using Artifact Download Action

```yml
version: v1.0
name: test-artifacts
stages:
- Stage 1
jobs:
- job: JobWithDownload
  stage: Stage 1
  steps:
  - artifactDownload:
      path: '{{.cds.workspace}}'
      tag: '{{.cds.version}}'
  - script:
    - ls
```

## Worker Download Command

You can download an artifact with the built-in action - or use the worker command.

Example of a step script using [worker download command]({{< relref "/docs/components/worker/download.md" >}})

![img](../images/artifact-worker-download.png)
