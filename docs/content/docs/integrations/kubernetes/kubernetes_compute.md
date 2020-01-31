---
title: Kubernetes Compute
main_menu: true
card: 
  name: compute
---

![Hatchery Kubernetes](/images/hatchery.kubernetes.png)

The Kubernetes integration have to be configured by CDS administrator.

This integration allows you to run the Kubernetes [Hatchery]({{<relref "/docs/components/hatchery/_index.md">}}) to start CDS Workers.

As an end-users, this integration allows:

 - to use [Worker Models]({{<relref "/docs/concepts/worker-model/_index.md">}}) of type "Docker"
 - to use Service Prerequisite on your [CDS Jobs]({{<relref "/docs/concepts/job.md">}}).

## Start Kubernetes hatchery

Generate a token:

```bash
$ cdsctl consumer new me \
--scopes=Hatchery,RunExecution,Service,WorkerModel \
--name="hatchery.kubernetes" \
--description="Consumer token for kubernetes hatchery" \
--groups="" \
--no-interactive

Builtin consumer successfully created, use the following token to sign in:
xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx
```

Edit the section `hatchery.kubernetes` in the [CDS Configuration]({{< relref "/hosting/configuration.md">}}) file.
The token have to be set on the key `hatchery.kubernetes.commonConfiguration.api.http.token`.

Then start hatchery:

```bash
engine start hatchery:kubernetes --config config.toml
```

This hatchery will spawn `Pods` on Kubernetes in the default namespace or the specified namespace in your `config.toml`. Each pods is a CDS Worker, using the Worker Model of type 'docker'.
