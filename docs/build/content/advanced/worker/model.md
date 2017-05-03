+++
title = "Worker Model"

[menu.main]
parent = "advanced-worker"
identifier = "advanced-worker-model"
weight = 2

+++

### Purpose

The goal of CDS is to start a worker when you need it and matching all your requirements exactly.
In order to scale automatically on demand, it is possible to register a worker model.

The goal of worker model is to describe the capabities of a given docker/iso image in terms of architecture, pre-installed binaries or libraries.

### Types

There is 2 types of worker models:

 * Docker images, see [how to create a worker model docker]({{< relref "worker-model-docker.md" >}})
 * Openstack images, see [how to create a worker model openstack]({{< relref "worker-model-openstack.md" >}})

### Capabilities

Capabilities have a name, a type and a value.

Existing capabilities type are:

 * Binary
 * Network access
 * Hostname
 * Memory
 * Service

### Behavior

All registered CDS [hatcheries]({{< relref "hatcheries.md" >}}) get the number of instances of each model needed. They then start/kill worker accordingly.    
