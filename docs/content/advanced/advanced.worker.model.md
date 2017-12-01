+++
title = "Worker Model"
weight = 2

[menu.main]
parent = "advanced-worker"
identifier = "advanced-worker-model-simple"

+++

### Purpose

The goal of CDS is to start a worker when you need it, which should match all your requirements exactly.
In order to scale automatically on demand, it is possible to register a worker model.

The goal of a worker model is to describe the capabilities of a given docker/iso image in terms of architecture, pre-installed binaries or libraries.

### Types

There are 2 types of worker models:

 * Docker images, see [how to create a worker model docker]({{< relref "tutorials.worker-model-docker-simple.md" >}})
 * Openstack images, see [how to create a worker model openstack]({{< relref "tutorials.worker-model-openstack.md" >}})

### Capabilities

Capabilities have a name, a type and a value.

Existing capability types are:

 * Binary
 * Network access
 * Hostname
 * Memory
 * Service

### Behavior

All registered CDS [hatcheries]({{< relref "advanced.hatcheries.md" >}}) get the number of instances of each model needed. Then, they start/kill workers accordingly.    
