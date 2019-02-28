+++
title = "Job"
weight = 1

+++

A Job is an important concept in CDS. A job is composed of **steps** which will be run sequentially. A Job is executed in a dedicated workspace and each new run of a job will have a new dedicated workspace. This means that you cannot share a workspace between jobs or between two runs of the same job.

![Job](/images/concepts_job.png)

A Job is executed by a **worker**. CDS will select or provision a worker for the job dependending on the [Requirements]({{< relref "workflows/pipelines/actions/builtin/artifact-upload.md" >}}) the job's requirements.

If you want to share files or artifacts between jobs, stages or pipelines you have to use the *Artifact upload* and *Artifact download* steps. You can also share variables between stages, see [variables tutorial](variables.md) for more details.


# Job's Requirements

Type of requirements:

- Binary
- Model
- Hostname
- [Network access]({{< relref "/workflows/pipelines/requirements/network_access/_index.md" >}})
- [Service]({{< relref "/workflows/pipelines/requirements/service/_index.md" >}})
- Memory
- [OS & Architecture]({{< relref "/workflows/pipelines/requirements/os_architecture/_index.md" >}})

A [Job]({{< relref "gettingstarted/concepts/job.md" >}}) will be executed by a **worker**.

CDS will choose and provision a worker for dependending on the **requirements** you define on your job.

You can set as many requirements as you want, following these rules:

- Only one model can be set as requirement
- Only one hostname can be set as requirement
- Only one OS & Architecture requirement can be set as at a time
- Memory and Services requirements are available only on Docker models

## Note on Service Requirement

A Service in CDS is a Docker container which is linked with your base image. To summarize, if you add mysql as service requirement to your pipeline job, the required image will then be used to create a container that is linked to the build container.

### How to

When editing a pipeline job, choose your model as usual, then add a new **service** requirement, the name you set will be the service's hostname, set the Docker image for the service as the value.

When the pipeline will be triggered, a worker defined by the model will be spawned with a [docker link](https://docs.docker.com/engine/userguide/networking/default_network/dockerlinks/) to the service you defined as requirement.

#### Environment variables

You can defined environment variables of the service by setting requirement value as:
```bash
    registry.ovh.net/official/postgres:9.5.3 POSTGRES_USER=myuser POSTGRES_PASSWORD=mypassword
```

To define your job's requirements in the UI, you just have to go on the job's edition page and click on requirements:

![Job's requirement UI](/images/job_requirements_ui.png)

Then a modal will appear in order to select your requirements:

![Job's requirement modal](/images/requirements_ui.png)

### Tutorials

* [Tutorial - Service Link Requirement NGINX]({{< relref "workflows/pipelines/requirements/service/example-nginx.md" >}})
* [Tutorial - Service Link Requirement PostgreSQL]({{< relref "workflows/pipelines/requirements/service/example-pg.md" >}})
