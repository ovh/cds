---
title: "Worker Model Docker"
weight: 5
card: 
  name: tutorial_worker-model
  weight: 1
---

A worker model of type `docker` can be spawned by a Hatchery Docker Swarm or a Hatchery Marathon.

## Register a worker Model from an existing Docker Image

Docker Image *golang:1.10* have a "curl" in $PATH, so it can be used as it is.

* In the UI, click on the wheel on the hand right top corner and select *workers" (or go the the route *#/worker*)
* At the bottom of the page, fill the form
    * **Name** of your worker *go-official-1.10*
    * **type** *docker*
    * **image** *golang:1.10*
    * **pattern**: if you aren't an administrator select a [configuration pattern]({{< relref "/docs/concepts/worker-model/patterns.md" >}}) that an administrator have already created for this type of worker model.
    * **shell command**: if you are an administrator you can directly edit the `main shell command` (main shell command is the command which accept a command to execute, for example `sh -c "echo CDS"`, here `sh -c` is the main shell command)
    * **the command**: represent the command to launch the CDS worker cf: [worker CLI]({{< relref "/docs/components/worker/_index.md" >}})
    * in order to launch your worker CDS allow you to use [a specific list of variables]({{< relref "/docs/concepts/worker-model/variables.md" >}}) which is interpolate when your worker will be spawned by your hatchery.
* Click on *Add* button and that's it

![Add worker model](/images/workflows.pipelines.requirements.docker.worker-model.docker.add.png)

{{< note >}}
If you want to specify an image using a private registry or a private image. You need to check the private checkbox and fill credentials in username and password to access to your image. And if your image is not on docker hub but from a private registry you need to fill the registry info (the registry api url, for example for docker hub it's https://index.docker.io/v1/ but we fill it by default).
{{< /note >}}

## Worker Model Docker on Hatchery Swarm

This hatchery offers some features on job pre-requisites, usable only on user's hatchery (ie. not a shared.infra hatchery).

* [Service Link]({{< relref "/docs/concepts/requirement/_index.md" >}})
* options on worker model prerequisite
    * Port mapping: `--port=8080:8081/tcp --port=9080:9081/tcp`
    * Privileged flag: `--privileged`
    * Add host flag: `--add-host=aaa:1.2.3.4 --add-host=bbb:5.6.7.8`
    * Use all: `--port=8080:8081/tcp --privileged --port=9080:9081/tcp --add-host=aaa:1.2.3.4 --add-host=bbb:5.6.7.8`

![Job Prerequisites](/images/workflows.pipelines.requirements.docker.worker-model.docker.png)
