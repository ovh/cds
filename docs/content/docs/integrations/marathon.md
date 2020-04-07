---
title: Mesos/Marathon
main_menu: true
card: 
  name: compute
---


![Hatchery Marathon](/images/hatchery.marathon.png)

The Mesos/Marathon integration have to be configured by CDS administrator.

This integration allows you to run the Marathon [Hatchery]({{<relref "/docs/components/hatchery/_index.md">}}) to start CDS Workers.

As an end-users, this integration allows to use [Worker Models]({{<relref "/docs/concepts/worker-model/_index.md">}}) of type "Docker"
 
## Start Marathon hatchery

Generate a token:

```bash
$ cdsctl consumer new me \
--scopes=Hatchery,RunExecution,Service,WorkerModel \
--name="hatchery.marathon" \
--description="Consumer token for marathon hatchery" \
--groups="" \
--no-interactive

Builtin consumer successfully created, use the following token to sign in:
xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx
```

Edit the section `hatchery.marathon` in the [CDS Configuration]({{< relref "/hosting/configuration.md">}}) file.
The token have to be set on the key `hatchery.marathon.commonConfiguration.api.http.token`.

Then start hatchery:

```bash
engine start hatchery:marathon --config config.toml
```

This hatchery will spawn Application on Marathon. Each application is a CDS Worker, using the Worker Model of type 'docker'.
