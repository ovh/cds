+++
title = "Hatchery Swarm"
weight = 4

+++

CDS build using Docker Swarm to spawn CDS Worker.

## Start Swarm hatchery

Generate a token for group:

```bash
$ cds generate  token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Edit the CDS [configuration]({{< relref "hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hactchery, just set the API HTTP and GRPC URL, the token freshly generated.

This hatchery use the standard docker environment variables to connect to a docker host.

Then start hatchery:

```bash
export DOCKER_HOST=tcp://xx.xx.xx.xx:2375
engine start hatchery:swarm --config config.toml
```

This hatchery will now start worker of model 'docker' on you docker installation.

## Setup a worker model

See [Tutorial]({{< relref "workflows/pipelines/requirements/worker-model/docker-simple.md" >}})
