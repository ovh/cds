+++
title = "Hatchery OpenStack"
weight = 3

+++

![Hatchery OpenStack](/images/hatchery.openstack.png)

CDS build using OpenStack infrastructure to spawn each CDS Workers inside dedicated virtual machine.

## Start OpenStack hatchery

Generate a token for group:

```bash
$ cds generate  token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Edit the CDS [configuration]({{< relref "/manual/hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated and the OpenStack variables.

Then start hatchery:

```bash
engine start hatchery:openstack --config config.toml
```

This hatchery will now start worker of model 'openstack' on OpenStack infrastructure.

## Setup a worker model

See [Tutorial]({{< relref "/manual/gettingstarted/tutorials/worker_model-openstack.md" >}})
