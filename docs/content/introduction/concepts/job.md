+++
title = "Job"
weight = 3

[menu.main]
parent = "concepts"
identifier = "concepts.job"

+++

A Job is an important concept in CDS. A job is composed of **steps** which will be run sequentially. A Job is executed in a dedicated workspace and each new run of a job will have a new dedicated workspace. This means that you cannot share a workspace between jobs or between two runs of the same job.

![Job](/images/concepts_job.png)

A Job is executed by a **worker**. CDS will select or provision a worker for the job dependending on the [Requirements]({{< relref "workflows/pipelines/actions/builtin/artifact-upload.md" >}}) the job's requirements.

If you want to share files or artifacts between jobs, stages or pipelines you have to use the *Artifact upload* and *Artifact download* steps. You can also share variables between stages, see [variables tutorial](variables.md) for more details.


![Job Examples](/images/concepts_job_example.png)
