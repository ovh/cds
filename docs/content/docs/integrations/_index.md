---
title: Integrations
main_menu: true
weight: 5
---

What's an integration?

An integration enable some features on CDS.
It can concern the storage of the artifacts, the repositories manager, the hooks available to trigger workflows, the infrastructure used to spawn the workers.

Here are the extensions available:

## Storage

- [Openstack Swift]({{< relref "openstack/openstack_swift.md" >}})

## Infrastructure used by CDS Workers

- [Docker Swarm]({{< relref "swarm.md" >}})
- [Kubernetes]({{< relref "kubernetes/kubernetes_compute.md" >}})
- [Openstack]({{< relref "openstack/openstack_compute.md" >}})
- [Mesos/Marathon]({{< relref "marathon.md" >}})
- [vSphere]({{< relref "vsphere.md" >}})

## Hooks on CDS Workflows

- [Kafka]({{< relref "kafka.md" >}})
- [RabbitMQ]({{< relref "rabbitmq.md" >}})

## Application Deployement

- [Kubernetes]({{< relref "kubernetes/kubernetes_deployment.md" >}})

## Repositories Managers

- [Bitbucket Server]({{< relref "bitbucket.md" >}})
- [GitHub]({{<relref "github.md" >}})
- [Gitlab]({{<relref "gitlab.md" >}})