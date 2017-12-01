+++
title = "Job"
weight = 3

[menu.main]
parent = "concepts"
identifier = "concepts-job"

+++

The Job is more important concept in CDS. It will be composed of **steps** which will be run sequencially. A Job will be executed is a dedicated workspace and each new run of a job will have a new dedicated workspace. It means that you cannot share a workspace between jobs or between two runs of a job.

![Job](/images/concepts_job.png)

A Job will be executed by a **worker**. CDS will choose and provision a worker for dependending of the [Requirements]({{< relref "building-pipelines.actions.builtin.artifact-upload.md" >}}) you define on your job.

If you want to share files or artifact between jobs, stages or pipelines you have to use *Artifact upload* and *Artifact download*. You can also share variable between stages, see [variables tutorial](variables.md) for more details.


![Job Examples](/images/concepts_job_example.png)
