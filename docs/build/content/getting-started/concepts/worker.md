+++
draft = false
title = "Worker"

weight = 5

[menu.main]
parent = "concepts"
identifier = "concepts-worker"

+++

A pipeline is structured in sequential stages containing one or multiple concurrent jobs. A Job will be executed by a worker.

Building your own worker model enable you to integrate your own tools, or to customize the tools you need to use. For instance, to build an AngularJs application, you shall need a worker capable of installing `npm` tools, importing `bower` packages (these are `nodeJs` tools), building webfonts with `fontforge`, ...

## What is a worker

Basically, a worker is a binary. This binary can be launched inside a Docker Containers, or on a Host (as VM Openstack).

## Worker cycle of life

![Pipeline](/images/concepts_worker_flow.png)

If the worker is spawned by a Hatchery, the Docker Container or Host where the worker will be lauched must contains a sane installation of "curl".

So, in the case of a worker model of type "docker", the docker image must contains `curl` in PATH.
