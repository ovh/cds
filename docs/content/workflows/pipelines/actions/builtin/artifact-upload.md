+++
title = "Artifact Upload"
chapter = true

+++

**Artifact Upload Action** is a builtin action, you can't modify it.

This action can be used to upload artifacts in CDS. This is the recommended way to share files between pipelines or stages.

## Parameters
* path: Path of file to upload
* tag: Tag to apply to your file.

### Example

* Create a file `myFile` and upload it.

![img](/images/workflows.pipelines.actions.builtin.artifact-upload-job.png)


* Launch pipeline, check logs

![img](/images/workflows.pipelines.actions.builtin.artifact-upload-logs.png)

* View artifact

![img](/images/workflows.pipelines.actions.builtin.artifact-upload-view-artifact.png)
