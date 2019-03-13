---
title: Docker Swarm
main_menu: true
---

The Docker Swarm integration have to be configured by CDS administrator.

This integration allows you to run the Swarm [Hatchery]({{<relref "/docs/components/hatchery/_index.md">}}) to start CDS Workers.

As an end-users, this integration allows:

 - to use [Worker Models]({{<relref "/docs/concepts/worker-model/_index.md">}}) of type "Docker"
 - to use Service Prerequisite on your [CDS Jobs]({{<relref "/docs/concepts/job.md">}}).

## Start Swarm hatchery

Generate a token for group:

```bash
$ cdsctl token generate shared.infra persistent
expiration  persistent
created     2019-03-13 18:47:56.715104 +0100 CET
group_name  shared.infra
token       xxxxxxxxxe7x4af2d408e5xxxxxxxff2adb333fab7d05c7752xxxxxxx
```

Edit the CDS [configuration]({{< relref "/hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated.

This hatchery use the standard Docker environment variables to connect to a Docker host.

Then start hatchery:

```bash
export DOCKER_HOST=tcp://xx.xx.xx.xx:2375
engine start hatchery:swarm --config config.toml
```

This hatchery will now start worker of model 'docker' on you Docker installation.

## Setup a worker model

See [Tutorial]({{< relref "/docs/tutorials/worker_model-docker/_index.md" >}})
