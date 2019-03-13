---
title: Mesos/Marathon
main_menu: true
---


![Hatchery Marathon](/images/hatchery.marathon.png)

The Mesos/Marathon integration have to be configured by CDS administrator.

This integration allows you to run the Marathon [Hatchery]({{<relref "/docs/components/hatchery/_index.md">}}) to start CDS Workers.

As an end-users, this integration allows to use [Worker Models]({{<relref "/docs/concepts/worker-model/_index.md">}}) of type "Docker"
 
## Start Marathon hatchery

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
engine start hatchery:marathon --config config.toml
```

This hatchery will spawn Application on Marathon. Each application is a CDS Worker, using the Worker Model of type 'docker'.
