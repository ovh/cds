+++
title = "Hatchery"
weight = 3

+++

![Hatchery](/images/hatchery.png)

Hatchery is a service dedicated to spawn and kill worker in accordance with build queue needs.

An hatchery is started with permissions to build all pipelines accessible from a given group, using token.

There are 6 modes for hatcheries:

 * [Local]({{< relref "local.md" >}}): Hatchery starts workers directly as local process.
 * [Marathon]({{< relref "marathon.md" >}}): Hatchery starts workers inside containers on a Mesos cluster using Marathon API.
 * [Swarm]({{< relref "swarm.md" >}}): The hatchery connects to a Docker Swarm cluster and starts workers inside containers.
 * [Kubernetes]({{< relref "kubernetes.md" >}}): The hatchery connects to a Kubernetes cluster and starts workers inside containers.
 * [OpenStack]({{< relref "openstack.md" >}}): Hatchery starts workers on OpenStack virtual machines using OpenStack Nova.
 * [vSphere]({{< relref "vsphere.md" >}}): Hatchery starts workers on vSphere datacenter using VMware vSphere.


## Admin hatchery

As a CDS administrator, it is possible to generate an access token for all projects using the `shared.infra` group.

This group is builtin to CDS, and all CDS administrators are administrator of this group.

This means that by default, an hatchery using a token generated for this group will be able to spawn workers able to build all pipelines.
