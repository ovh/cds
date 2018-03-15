+++
title = "Worker"
weight = 4

+++


A pipeline is structured in sequential stages containing one or multiple concurrent jobs. A Job will be executed by a worker.

Building your own worker model enables you to integrate your own tools, or to customize the tools you need to use. For instance, to build an AngularJs application, you shall need a worker capable of installing `npm` tools, importing `bower` packages (these are `nodeJs` tools), building webfonts with `fontforge`...

## What is a worker

Basically, a worker is a binary. This binary can be launched inside a Docker Container, or on a Host (as an Openstack VM). A worker executes a [CDS Job]({{< relref "/gettingstarted/concepts/job.md" >}}).

If you want to auto-scale workers, have a look at the [CDS Hatchery]({{< relref "/hatchery/_index.md" >}})

## Worker life cycle

![Pipeline](/images/concepts_worker_flow.png)

If the worker is spawned by a Hatchery, the Docker Container or Host where the worker will be lauched must contain a sane installation of "curl".

So, in the case of a worker model of type "docker", the docker image must have `curl` available in the PATH.

#### Why would you need to setup your own worker ?

There are several cases where one would need to setup his own worker:

 * Perform incremental build
 * Build on a specific architecture
 * Perform integration tests in a specific network

### How does this work ?

Workers authenticate on CDS with a token and have the same permissions as the user who generated it.

To generate a token, run the following command, replace yourgroup with your group

```bash
$ cdsctl token generate yourgroup persistent
```
