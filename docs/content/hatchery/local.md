+++
title = "Hatchery Local"
weight = 1

+++

## Use case

If you are a CDS user, you can use shared.infra Worker to run your job.

But, if you want to use your host for various good reasons as OS, Network, etc... you
can launch CDS Worker on your host.

Two prerequisites: 

* your host must reach your CDS API on HTTP port or GPRC Port defined on your [CDS Configuration]({{< relref "hosting/configuration.md">}})
* you need the CDS Worker binary on your host. 

You can download it from [latest release on Github](https://github.com/ovh/cds/releases) or from download page on your CDS Instance (Navbar -> Settings -> Download)

## Start Local hatchery

Generate a token for group:

```bash
$ cds generate token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Edit the [CDS Configuration]({{< relref "hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated.

This hatchery use the CDS worker binary existing on the PATH on your host.

Then start hatchery:

```bash
engine start hatchery:local --config config.toml
```

This hatchery will now start worker binary on your host. You can manage settings, as `max workers` and `provision` in the hatchery configuration file.
