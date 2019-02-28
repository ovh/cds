+++
title = "GitTag"
chapter = true

+++

**GitTag** is a builtin action, you can't modify it.

This action creates a tag. You can use a pgp key to sign it.

## Parameters

* url - mandatory - URL must contain information about the transport protocol, the address of the remote server, and the path to the repository.
* authPrivateKey - optional - the private key to be able to git tag from ssh
* user - optional - the user to be able to git tag from https with authentication
* password - optional - the password to be able to git tag from https with authentication
* tagName - optional - Name of the tag you want to create. If empty, it will make a patch version from your last tag.
* tagMessage - optional - Message for the tag
* path - optional - path to your git repository
* signKey - optional - pgp key to sign the tag
* Advanced parameter: prefix - add a prefix in tag name created

## Example of usage

Here, a pipeline as code, containing two actions:

* CheckoutApplication
* GitTag

```yml
version: v1.0
name: create-tag
parameters:
  tagLevel:
    type: list
    default: major;minor;patch
    description: major, minor or patch
jobs:
- job: CreateTag
  steps:
  - checkout: '{{.cds.workspace}}'
  - gitTag:
      path: '{{.cds.workspace}}'
      tagLevel: '{{.cds.pip.tagLevel}}'
      tagMessage: Release from CDS run {{.cds.version}}
```

This pipeline could be used in a workflow, with a Run Condition on cds.manual = true.

Tutorial: [Build, tag and release an application]({{< relref "step_by_step_build_tag_release.md" >}})