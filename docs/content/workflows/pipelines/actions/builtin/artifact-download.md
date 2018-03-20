+++
title = "Artifact Download"
chapter = true

+++


**Artifact Download Action** is a builtin action, you can't modify it.

This action can be used to retrieve an artifact previously uploaded by an [Artifact Upload]({{< relref "workflows/pipelines/actions/builtin/artifact-upload.md" >}}) action.

## Parameters
* application: Application from where artifacts will be downloaded
* pipeline: Pipeline from where artifacts will be downloaded
* tag: Tag set in the Artifact Upload action
* path: Path where artifacts will be downloaded

### Example

* Workflow Configuration: a pipeline doing an `upload artifact` and another doing a `download artifact`.

![img](/images/workflows.pipelines.actions.builtin.artifact-download-workflow.png)

* Job Configuration: download artifact from pipeline `parent`

![img](/images/workflows.pipelines.actions.builtin.artifact-download-job.png)

* Run pipeline, check logs

![img](/images/workflows.pipelines.actions.builtin.artifact-download-logs.png)
