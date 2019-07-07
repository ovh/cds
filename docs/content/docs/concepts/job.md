---
title: "Job"
weight: 1
card:
  name: concept_pipeline
  weight: 3
---


A job is composed of steps, which will be run sequentially. A job is executed in a dedicated workspace (i.e. filesystem). A new workspace is assigned for each new run of a job.

![Job](../images/job_steps.png?height=300px)

A standard build job looks like this:


![Job](../images/job.png?height=500px)

You can use « built-in » actions, such as checkoutApplication, script, jUnit, artifact upload/download.

- The [checkoutApplication]({{< relref "/docs/actions/builtin-checkoutapplication.md" >}}) action clones your Git repository
- The [Script]({{< relref "/docs/actions/builtin-script.md" >}}) action executes your build command as “make build”
- The [artifactUpload]({{< relref "/docs/actions/builtin-artifact-upload.md" >}}) action uploads previously-built binaries
- The [jUnit]({{< relref "/docs/actions/builtin-junit.md" >}}) action parses a given Junit-formatted XML file to extract its test results


**Notice**: you cannot share a workspace between jobs or between two runs of the same job. Actions [Artifact Upload]({{< relref "/docs/actions/builtin-artifact-upload.md" >}}) and [Artifact Download]({{< relref "/docs/actions/builtin-artifact-download.md" >}}) can be used to transfert artifacts between jobs.

A Job is executed by a **worker**. CDS will select or provision a worker for the job dependending on the [Requirements]({{< relref "/docs/concepts/requirement/_index.md" >}}) the job's requirements.

## Steps

The steps of a job is the list of the different operations performed by the CDS worker. Each step is based on an **Action** pre-defined by CDS. The list of all actions is defined on `*<your cds url ui>/#/action*`. When a step fails, its parent job is stopped and marked as `failed`.

You can define a Step as final. It mean that even if the job fails before reaching it, the step will be executed anyway. The *final* steps are executed after all other steps.

You can find below an example of steps creation in CDS.
You have 2 configuration flags:

- Optional: The failure of the step does not cause the failure of the whole job.
- Always executed: with this flag checked, this step will be executed even if previous steps fail. This can be helpful, for example, if you run tests in a step and you would like to upload the tests report even if the tests fail.

![Steps Examples](/images/concepts_step_example.png)
