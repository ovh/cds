---
title: Kubernetes Compute
main_menu: true
---

![Hatchery Kubernetes](/images/hatchery.kubernetes.png)

The Kubernetes integration have to be configured by CDS administrator.

This integration allows you to run the Kubernetes [Hatchery]({{<relref "/docs/components/hatchery/_index.md">}}) to start CDS Workers.

As an end-users, this integration allows:

 - to use [Worker Models]({{<relref "/docs/concepts/worker-model/_index.md">}}) of type "Docker"
 - to use Service Prerequisite on your [CDS Jobs]({{<relref "/docs/concepts/job.md">}}).

## Start Kubernetes hatchery

Generate a token for group:

```bash
$ cdsctl token generate shared.infra persistent
expiration  persistent
created     2019-03-13 18:47:56.715104 +0100 CET
group_name  shared.infra
token       xxxxxxxxxe7x4af2d408e5xxxxxxxff2adb333fab7d05c7752xxxxxxx
```

Edit the [CDS Configuration]({{< relref "/hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated.

Then start hatchery:

```bash
engine start hatchery:kubernetes --config config.toml
```

This hatchery will spawn `Pods` on Kubernetes in the default namespace or the specified namespace in your `config.toml`. Each pods is a CDS Worker, using the Worker Model of type 'docker'.
