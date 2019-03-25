---
title: "Service Link"
weight: 5
---

The Service Link prerequisite allow you to use any Docker image as a dependency of a job.

This is pretty useful if you want to make some tests with a real database, or put your builded application as a job prerequisite for doing some tests.

## How to use service requirement

When editing a pipeline job, choose your model as usual, then add a new **service** requirement, the name you set will be the service's hostname, set the Docker image for the service as the value.

When the pipeline will be triggered, a worker defined by the model will be spawned with a [docker link](https://docs.docker.com/engine/userguide/networking/default_network/dockerlinks/) to the service you defined as requirement.

## Environment variables

You can defined environment variables of the service by setting requirement value as:
```bash
    registry.ovh.net/official/postgres:9.5.3 POSTGRES_USER=myuser POSTGRES_PASSWORD=mypassword
```

To define your job's requirements in the UI, you just have to go on the job's edition page and click on requirements:

![Job's requirement UI](/images/job_requirements_ui.png)

Then a modal will appear in order to select your requirements:

![Job's requirement modal](/images/requirements_ui.png)

## Tutorials

* [Tutorial - Service Link Requirement NGINX]({{< relref "/docs/tutorials/service-requirement-nginx.md" >}})
* [Tutorial - Service Link Requirement PostgreSQL]({{< relref "/docs/tutorials/service-requirement-pg.md" >}})