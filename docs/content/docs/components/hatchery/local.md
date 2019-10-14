---
title: "Hatchery Local"
weight: 1
---

## Use case

If you are a CDS user, you can use shared.infra Worker to run your job.

But, if you want to use your host for various good reasons as OS, Network, etc... you
can launch CDS Worker on your host.

Two prerequisites: 

* your host must reach your CDS API on HTTP port defined on your [CDS Configuration]({{< relref "/hosting/configuration.md">}})
* the basedir (default /tmp) must allow to execute binaries.

The worker binary is downloaded from CDS API at the start of the hatchery, it's placed into `basedir` directory.

## Start Local hatchery

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
engine start hatchery:local --config config.toml
```

This hatchery will now start worker binary on your host. You can manage settings, as `max workers` and `provision` in the hatchery configuration file.
