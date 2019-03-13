---
title: Kubernetes
main_menu: true
---



![Hatchery Kubernetes](/images/hatchery.kubernetes.png)

## Start Kubernetes hatchery

Generate a token for group:

```bash
$ cdsctl token generate shared.infra persistent
expiration  persistent
created     2019-03-13 18:47:56.715104 +0100 CET
group_name  shared.infra
token       xxxxxxxxxe7x4af2d408e5xxxxxxxff2adb333fab7d05c7752xxxxxxx
creator     yesnault
```

Edit the [CDS Configuration]({{< relref "/hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated.

Then start hatchery:

```bash
engine start hatchery:kubernetes --config config.toml
```

This hatchery will spawn `Pods` on Kubernetes in the default namespace or the specified namespace in your `config.toml`. Each pods is a CDS Worker, using the Worker Model of type 'docker'.
