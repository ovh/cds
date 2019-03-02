+++
title = "Worker"
weight = 9

+++


A pipeline is structured in sequential stages containing one or multiple concurrent jobs. A Job will be executed by a worker.

## What is a worker

Basically, a worker is a binary. This binary can be launched inside a Docker Container, or on a Host (as an OpenStack VM). A worker executes a [CDS Job]({{< relref "/manual/concepts/job/_index.md" >}}).

If you want to auto-scale workers, have a look at the [CDS Hatchery]({{< relref "/hatchery/_index.md" >}})

## Worker life cycle

![Pipeline](/images/concepts_worker_flow.png)

If the worker is spawned by a Hatchery, the Docker Container or Host where the worker will be launched must contain a sane installation of "curl".

So, in the case of a worker model of type "docker", the Docker image must have `curl` available in the PATH.

### About worker model

Building your own worker model enables you to integrate your own tools, or to customize the tools you need to use. For instance, to build an AngularJS application, you shall need a worker capable of installing `npm` tools, importing `bower` packages (these are `Node.js` tools), building webfonts with `fontforge`...


[More about worker model]({{< relref "/workflows/pipelines/requirements/worker-model/_index.md" >}})
