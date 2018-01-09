+++
title = "Hatchery Local"
weight = 2

[menu.main]
parent = "hatchery"
identifier = "hatchery.local"

+++



## Start Marathon hatchery

Generate a token for group:

```bash
$ cds generate token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Edit the [CDS Configuration]({{< relref "hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated.

Then start hatchery:

```bash
engine start hatchery:marathon --config config.toml
```

This hatchery will spawn Application on Marathon. Each application is a CDS Worker, using the Worker Model of type 'docker'.
