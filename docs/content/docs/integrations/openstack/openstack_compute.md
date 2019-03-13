---
title: Openstack Compute
main_menu: true
---


![Hatchery OpenStack](/images/hatchery.openstack.png)

CDS build using OpenStack infrastructure to spawn each CDS Workers inside dedicated virtual machine.

## Start OpenStack hatchery

Generate a token for group:

```bash
$ cdsctl token generate shared.infra persistent
expiration  persistent
created     2019-03-13 18:47:56.715104 +0100 CET
group_name  shared.infra
token       xxxxxxxxxe7x4af2d408e5xxxxxxxff2adb333fab7d05c7752xxxxxxx
```

Edit the CDS [configuration]({{< relref "/hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated and the OpenStack variables.

Then start hatchery:

```bash
engine start hatchery:openstack --config config.toml
```

This hatchery will now start worker of model 'openstack' on OpenStack infrastructure.

## Setup a worker model

See [Tutorial]({{< relref "/docs/tutorials/worker_model-openstack.md" >}})
