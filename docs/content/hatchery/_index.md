+++
title = "Hatchery"
weight = 5

+++

![Hatchery](/images/hatchery.png)

Hatchery is a service dedicated to spawn and kill worker in accordance with build queue needs.

An hatchery is started with permissions to build all pipelines accessible from a given group, using token.

There are 5 modes for hatcheries:

 * Local (Start local workers on a single host)
 * Marathon (Start worker model instances on a mesos cluster with marathon framework)
 * Swarm (Start worker on a docker swarm cluster)
 * Openstack (Start virtual machines on an openstack cluster)
 * VSphere (Start virtual machines on an VSphere cluster)

### Local mode

Hatchery starts workers directly as local process.

### Marathon mode

Hatchery starts workers inside containers on a mesos cluster using Marathon API.

### Openstack mode

Hatchery starts workers on Openstack virtual machines using Openstack Nova.

### VSphere mode

Hatchery starts workers on VSphere datacenter using VMWare VSphere.

### Swarm mode

The hatchery connects to a Docker Swarm cluster and starts workers inside containers.

## Admin hatchery

As a CDS administrator, it is possible to generate an access token for all projects using the `shared.infra` group.

This group is builtin to CDS, and all CDS administrators are administrator of this group.

This means that by default, an hatchery using a token generated for this group will be able to spawn workers able to build all pipelines.
