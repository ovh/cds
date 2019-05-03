---
title: "Artifact Download"
card:
  name: builtin
---

**Artifact Download** is a builtin action, you can't modify it.

This action can be used to retrieve an artifact previously uploaded by an Artifact Upload.

## Parameters

* **enabled**: (optional) Enable artifact download.
* **path**: Path where artifacts will be downloaded.
* **pattern**: (optional) Empty: download all files. Otherwise, enter regexp pattern to choose file: (fileA|fileB).
* **tag**: Artifact are uploaded with a tag, generally {{.cds.version}}.


## Requirements

No Requirement

## YAML example

Example of a pipeline using Artifact Download action:
```yml
version: v1.0
name: Pipeline1
stages:
- Stage1
jobs:
- job: Job1
  stage: Stage1
  steps:
  - artifactDownload:
      path: '{{.cds.workspace}}'
      pattern: '*.tag.gz'
      tag: '{{.cds.version}}'

```

## Example

* Workflow Configuration: a pipeline doing an `upload artifact` and another doing a `download artifact`.

![img](../images/artifact-download-workflow.png)

* Run pipeline, check logs

![img](../images/artifact-download-logs.png)

## Worker Download Command

You can download an artifact with the built-in action - or use the worker command.

Example of a step script using [worker download command]({{< relref "/docs/components/worker/download.md" >}})

![img](../images/artifact-worker-download.png)
