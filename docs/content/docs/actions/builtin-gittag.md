
---
title: "GitTag"
card:
  name: builtin
---

**GitTag** is a builtin action, you can't modify it.

Tag the current branch and push it. Use vcs config from your application.
Semver used if fully compatible with https://semver.org.


## Parameters

* **path**: (optional) The path to your git directory.
* **prefix**: (optional) Add a prefix for tag name.
* **tagLevel**: Set the level of the tag. Must be 'major' or 'minor' or 'patch'.
* **tagMessage**: (optional) Set a message for the tag.
* **tagMetadata**: (optional) Metadata of the tag. Example: cds.42 on a tag 1.0.0 will return 1.0.0+cds.42.
* **tagPrerelease**: (optional) Prerelease version of the tag. Example: alpha on a tag 1.0.0 will return 1.0.0-alpha.


## Requirements

* **git**: type: binary Value: git
* **gpg**: type: binary Value: gpg


## YAML example

Example of a pipeline using GitTag action:
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

```

## Example

Tutorial that use this action: [Build, tag and release an application]({{< relref "/docs/tutorials/step_by_step_build_tag_release.md" >}}).
