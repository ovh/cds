+++
title = "Artifact Download"
chapter = true

[menu.main]
parent = "actions-builtin"
identifier = "builtin-artifact-download"

+++


**Artifact Download Action** is a builtin action, you can't modify it.

This action can be used to get artifact uploaded by the [Artifact Upload]({{< relref "building-pipelines.actions.builtin.artifact-upload.md" >}}) action

## Action Parameter
* application: Application from where artifacts will be downloaded
* pipeline: Pipeline from where artifacts will be downloaded
* tag: Tag set in the Artifact Upload action
* path: Path where artifacts will be downloaded

### Example of Job Configuration

* Download artifact from the parent pipeline

![img](/img/actions/artifact_download_parent.png)

* Download artifact from the previous stage

![img](/img/actions/artifact_download_stage.png)
