---
title: Docker Swarm
main_menu: true
card: 
  name: compute
---

The Docker Swarm integration have to be configured by CDS administrator.

This integration allows you to run the Swarm [Hatchery]({{<relref "/docs/components/hatchery/_index.md">}}) to start CDS Workers.

As an end-users, this integration allows:

 - to use [Worker Models]({{<relref "/docs/concepts/worker-model/_index.md">}}) of type "Docker"
 - to use Service Prerequisite on your [CDS Jobs]({{<relref "/docs/concepts/job.md">}}).

## Start Swarm hatchery

Generate a token:

```bash
$ cdsctl consumer new me \
--scopes=Hatchery,RunExecution,Service,WorkerModel \
--name="hatchery.swarm" \
--description="Consumer token for swarm hatchery" \
--groups="" \
--no-interactive

Builtin consumer successfully created, use the following token to sign in:
xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx
```

Edit the section `hatchery.swarm` in the [CDS Configuration]({{< relref "/hosting/configuration.md">}}) file.
The token have to be set on the key `hatchery.swarm.commonConfiguration.api.http.token`.

This hatchery use the standard Docker environment variables to connect to a Docker host.

Then start hatchery:

```bash
export DOCKER_HOST=tcp://xx.xx.xx.xx:2375
engine start hatchery:swarm --config config.toml
```

This hatchery will now start worker of model 'docker' on you Docker installation.

## Setup a worker model

See [Tutorial]({{< relref "/docs/tutorials/worker_model-docker/_index.md" >}})
