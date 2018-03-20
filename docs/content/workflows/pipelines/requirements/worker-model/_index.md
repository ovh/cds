+++
title = "Worker Model"
weight = 1

+++

### Purpose

The goal of CDS is to start a worker when you need it, which should match all your requirements exactly.
In order to scale automatically on demand, it is possible to register a worker model.

The goal of a worker model is to describe the capabilities of a given docker/iso image in terms of architecture, pre-installed binaries or libraries.

### Types

There are 2 types of worker models:

 * Docker images, see [how to create a worker model docker]({{< relref "workflows/pipelines/requirements/worker-model/docker-simple.md" >}})
 * Openstack images, see [how to create a worker model openstack]({{< relref "workflows/pipelines/requirements/worker-model/openstack.md" >}})

### Behavior

All registered CDS [hatcheries]({{< relref "hatchery/_index.md" >}}) start/kill workers as needed.
