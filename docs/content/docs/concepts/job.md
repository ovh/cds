---
title: "Job"
weight: 1
card: 
  name: pipeline
  weight: 10
---


A job is composed of steps, which will be run sequentially. A job is executed in a dedicated workspace (i.e. filesystem). A new workspace is assigned for each new run of a job.

![Job](../images/job_steps.png?height=300px)

A standard build job looks like this: 


![Job](../images/job.png?height=500px)

You can use « built-in » actions, such as checkoutApplication, script, jUnit, artifact upload/download.

- The [checkoutApplication]({{< relref "/docs/actions/checkoutapplication.md" >}}) action clones your Git repository
- The [Script]({{< relref "/docs/actions/script.md" >}}) action executes your build command as “make build”
- The [artifactUpload]({{< relref "/docs/actions/artifact-upload.md" >}}) action uploads previously-built binaries
- The [jUnit]({{< relref "/docs/actions/junit.md" >}}) action parses a given Junit-formatted XML file to extract its test results


**Notice**: you cannot share a workspace between jobs or between two runs of the same job. Actions [Artifact Upload]({{{< relref "/docs/actions/artifact-upload.md" >}}}) and [Artifact Download]({{{< relref "/docs/actions/artifact-download.md" >}}}) can be used to transfert artifacts between jobs.

A Job is executed by a **worker**. CDS will select or provision a worker for the job dependending on the [Requirements]({{< relref "/docs/concepts/requirement/_index.md" >}}) the job's requirements.
