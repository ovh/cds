---
title: Openstack Compute
main_menu: true
card: 
  name: compute
---


![Hatchery OpenStack](/images/hatchery.openstack.png)

CDS build using OpenStack infrastructure to spawn each CDS Workers inside dedicated virtual machine.

## Start OpenStack hatchery

Generate a token:

```bash
$ cdsctl consumer new me \
--scopes=Hatchery,RunExecution,Service,WorkerModel \
--name="hatchery.openstack" \
--description="Consumer token for openstack hatchery" \
--groups="" \
--no-interactive

Builtin consumer successfully created, use the following token to sign in:
xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx
```

Edit the section `hatchery.openstack` in the [CDS Configuration]({{< relref "/hosting/configuration.md">}}) file.
The token have to be set on the key `hatchery.openstack.commonConfiguration.api.http.token`.

Then start hatchery:

```bash
engine start hatchery:openstack --config config.toml
```

This hatchery will now start worker of model 'openstack' on OpenStack infrastructure.

## Setup a worker model

See [Tutorial]({{< relref "/docs/tutorials/worker_model-openstack.md" >}})
