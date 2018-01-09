+++
title = "Hatchery Openstack"
weight = 3

[menu.main]
parent = "hatchery"
identifier = "hatchery.openstack"

+++

CDS build using Openstack infrastructure to spawn each CDS Workers inside dedicated virtual machine.

## Start Opentack hatchery

Generate a token for group:

```bash
$ cds generate  token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Edit the CDS [configuration]({{< relref "hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated and the openstack variables.

Then start hatchery:

```bash
engine start hatchery:openstack --config config.toml
```

This hatchery will now start worker of model 'openstack' on Openstack infrastructure.

## Setup a worker model

See [Tutorial]({{< relref "workflows/pipelines/requirements/worker-model/openstack.md" >}})
