+++
title = "Artifact Upload"
chapter = true

[menu.main]
parent = "actions-builtin"
identifier = "builtin-artifact-upload"

+++

# Artifact Upload Action

**Artifact Upload Action** is a builtin action, you can't modify it.

This action can be used to upload artifact in CDS. This is the good way to share files between pipelines or stages.

## Action Parameter
* path: Path of file to upload
* tag: Tag to apply to your file.

### Example of Job Configuration

* With a tag to indicate the build version

![img](/img/actions/artifact_upload_version.png)

* With a latest tag

![img](/img/actions/artifact_upload_latest.png)
