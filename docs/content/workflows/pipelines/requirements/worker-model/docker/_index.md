+++
title = "Docker Worker Model"
weight = 1

+++

A worker model of type `docker` can be spawned by a Hatchery Docker Swarm or a Hatchery Marathon.

## Register a worker Model from an existing Docker Image

Docker Image *golang:1.10* have a "curl" in $PATH, so it can be used as it is.

* In the UI, click on the wheel on the hand right top corner and select *workers" (or go the the route *#/worker*)
* At the bottom of the page, fill the form
    * Name of your worker *Golang-1.10*
    * type *docker*
    * image *golang:1.10*
    * if you aren't an administrator select a [configuration pattern]({{< relref "workflows/TODO_PATTERNS" >}}) that an administrator have already created for this type of worker model.
    * if you are an administrator you can directly edit the `main shell command` (main shell command is the command which accept a command to execute, for example `sh -c "echo CDS"`, here `sh -c` is the main shell command) and the `main worker command` (represent the command to launch the CDS worker cf: [worker CLI]({{< relref "cli/worker/_index.md" >}}))
    * in order to launch your worker CDS allow you to use [a specific list of variables]({{< relref "workflows/TODO_VARIABLES" >}}) which is interpolate when your worker will be spawned by your hatchery.
* Click on *Add* button and that's it

![Add worker model](/images/workflows.pipelines.requirements.docker.worker-model.docker.add.png)

## Worker Model Docker on Hatchery Swarm

This hatchery offers some features on job pre-requisites, usable only on user's hatchery (ie. not a shared.infra hatchery).

* [Service Link]({{< relref "workflows/pipelines/requirements/service/_index.md" >}})
* options on worker model prerequisite
    * Port mapping: `--port=8080:8081/tcp --port=9080:9081/tcp`
    * Priviledge flag: `--privileged`
    * Add host flag: `--add-host=aaa:1.2.3.4 --add-host=bbb:5.6.7.8`
    * Use all: `--port=8080:8081/tcp --privileged --port=9080:9081/tcp --add-host=aaa:1.2.3.4 --add-host=bbb:5.6.7.8`
* options on volume prerequisite
    * Bind: `type=bind,source=/hostDir/sourceDir,destination=/dirInJob,readonly`

![Job Prerequisites](/images/workflows.pipelines.requirements.docker.worker-model.docker.png)
