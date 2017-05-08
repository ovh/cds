+++
title = "Hatchery Swarm"
weight = 2

[menu.main]
parent = "hatcheries"
identifier = "hatchery_swarm"

+++

CDS build using Docker Swarm to spawn CDS Worker.

## Start Swarm hatchery

Generate a token for group:

```bash
$ cds generate  token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Then start hatchery:

```bash
export CDS_LOG_LEVEL=notice
export CDS_RATIO_SERVICE=50
export CDS_TOKEN="fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92"
export DOCKER_HOST=tcp://xx.xx.xx.xx:2375
export CDS_API=http://your-cds-api
export CDS_NAME=$(hostname)
export CDS_MAX_WORKER=10
export CDS_MAX_CONTAINERS=5
export CDS_PROVISION=0
export CDS_REQUEST_API_TIMEOUT=120
./hatchery swarm

# You can also use the flags instead of environment variable if you want
```

This hatchery will now start worker of model 'docker' on you docker installation.

## Setup a worker model

See [Tutorial]({{< relref "tutorials.worker-model-docker-simple.md" >}})
