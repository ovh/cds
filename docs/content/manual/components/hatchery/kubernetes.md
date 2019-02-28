+++
title = "Hatchery Kubernetes"
weight = 2

+++

![Hatchery Kubernetes](/images/hatchery.kubernetes.png)

## Start Kubernetes hatchery

Generate a token for group:

```bash
$ cds generate token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Edit the [CDS Configuration]({{< relref "hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated.

Then start hatchery:

```bash
engine start hatchery:kubernetes --config config.toml
```

This hatchery will spawn `Pods` on Kubernetes in the default namespace or the specified namespace in your `config.toml`. Each pods is a CDS Worker, using the Worker Model of type 'docker'.
