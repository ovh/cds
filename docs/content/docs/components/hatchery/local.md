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
* the basedir (default is `/var/lib/cds-engine`) must allow to execute binaries.

The worker binary is downloaded from CDS API at the start of the hatchery, it's placed into `basedir` directory.

## Start Local hatchery

Generate a token:

```bash
$ cdsctl consumer new me \
--scopes=Hatchery,RunExecution,Service,WorkerModel \
--name="hatchery.local" \
--description="Consumer token for local hatchery" \
--groups="" \
--no-interactive

Builtin consumer successfully created, use the following token to sign in:
xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx
```

Edit the section `hatchery.local` in the [CDS Configuration]({{< relref "/hosting/configuration.md">}}) file.
The token have to be set on the key `hatchery.local.commonConfiguration.api.http.token`.

Then start hatchery:

```bash
engine start hatchery:local --config config.toml
```

This hatchery will now start worker binary on your host. You can manage settings, as `max workers` in the hatchery configuration file.
