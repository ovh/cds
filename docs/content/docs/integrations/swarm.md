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

## Docker engine maintenance

The hatchery keeps working when one of its Docker engines does not answer: that engine is skipped for new workers, while the other engines keep being provisioned and cleaned up. An engine that does not answer when the hatchery starts is used as soon as it answers.

To take an engine out of service without interrupting running jobs, drain it with the `cds.hatchery.drain=true` engine label, for instance in `/etc/docker/daemon.json`:

```json
{
  "labels": ["cds.hatchery.drain=true"]
}
```

Then reload the Docker daemon (`systemctl reload docker`); no restart is needed. Within a few seconds the hatchery stops starting containers on that engine, but keeps managing the existing ones: running jobs end normally, and their containers and services are removed as usual. The engine is empty once no container with the `hatchery` label remains on it. Remove the label and reload the daemon again to put the engine back in service.

The hatchery status reports the number of drained engines (`DockerEngines.Drained`).

## Setup a worker model

See [Tutorial]({{< relref "/docs/tutorials/worker_model-docker/_index.md" >}})
