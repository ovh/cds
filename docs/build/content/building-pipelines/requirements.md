+++
title = "Requirements"
weight = 6

[menu.main]
parent = "building-pipelines"
identifier = "requirements"

+++



A [Job]({{< relref "artifact-upload.md" >}}) will be executed by a **worker**.

CDS will choose and provision a worker for dependending of the **requirements** you define on your job.

You can set as many requirements as you want, following those rules :

- Only one model can be set as requirement
- Only one hostname can be set as requirement
- Memory and Services requirements are availabe only on Docker models


## Note on Service Requirement

A Service in CDS is a docker container which is linked with your base image. To summarize, if you add mysql as service requirement to your pipeline job, the required image will then be used to create a container that is linked to the build container.

### How to

When editing a pipeline job, choose your model as usual. Then add a new  **service** requirement, the name you set will be the service's hostname, set the docker image for the service as the value.

When the pipeline will be triggered, a worker defined by the model will be spawned with a link (https://docs.docker.com/engine/userguide/networking/default_network/dockerlinks/) to the service you defined as requirement.

#### Environment variables

You can defined environment variables of the service by setting requirement value as :
```bash
    registry.ovh.net/official/postgres:9.5.3 POSTGRES_USER=myuser POSTGRES_PASSWORD=mypassword
```

### Tutorials

* [Tutorial - Service Link Requirement Nginx]({{< relref "service-link-requirement-nginx.md" >}})
* [Tutorial - Service Link Requirement PostgreSQL]({{< relref "service-link-requirement-pg.md" >}})
