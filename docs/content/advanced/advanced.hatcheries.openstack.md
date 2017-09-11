+++
title = "Hatchery Openstack"
weight = 3

[menu.main]
parent = "hatcheries"
identifier = "hatchery_openstack"

+++

CDS build using Openstack infrastructure to spawn each CDS Workers inside dedicated VM.

## Start Opentack hatchery

Generate a token for group:

```bash
$ cds generate  token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Then start hatchery:

```bash
OPENSTACK_USER=<user> OPENSTACK_TENANT=<tenant> OPENSTACK_AUTH_ENDPOINT=https://auth.cloud.ovh.net OPENSTACK_PASSWORD=<password> OPENSTACK_REGION=SBG1 hatchery cloud \
        --api=https://api.cds.domain \
        --max-worker=10 \
        --provision=1 \
        --token=fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

This hatchery will now start worker of model 'openstack' on Openstack infrastructure.

## Setup a worker model

See [Tutorial]({{< relref "tutorials.worker-model-openstack.md" >}})
