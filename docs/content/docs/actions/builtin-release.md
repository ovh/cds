
---
title: "Release"
card:
  name: builtin
---

**Release** is a builtin action, you can't modify it.

This action creates a release on the git repository linked to the application, if repository manager implements it.

## Parameters

* **artifacts**: (optional) Set a list of artifacts, separate by ','. You can also use regexp.
* **releaseNote**: (optional) Set a release note for the release.
* **tag**: Tag attached to the release.
* **title**: Set a title for the release.


## Requirements

No Requirement

## YAML example

Example of a pipeline using Release action:
```yml
version: v1.0
name: Pipeline1
parameters:
  tagLevel:
    type: list
    default: major;minor;patch
    description: major, minor or patch
stages:
- Stage1
jobs:
- job: Job1
  stage: Stage1
  steps:
  - checkout: '{{.cds.workspace}}'
  - gitTag:
      path: '{{.cds.workspace}}'
      tagLevel: '{{.cds.pip.tagLevel}}'
      tagMessage: Release from CDS run {{.cds.version}}
  - script:
    - '#!/bin/sh'
    - TAG=`git describe --abbrev=0 --tags`
    - worker export tag $TAG
  - release:
      artifacts: '{{.cds.workspace}}/myFile'
      releaseNote: My release {{.cds.build.tag}}
      tag: '{{.cds.build.tag}}'
      title: '{{.cds.build.tag}}'

```

## Notes

This action is actually implemented for GitHub only.
