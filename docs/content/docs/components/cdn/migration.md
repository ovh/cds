---
title: "Migrate logs into CDN"
weight: 1
---

# Prepare CDN service configuration
* Init CDN configuration using "engine" binary.

Depending your needs you can change the default configuration to use a different storage unit, the default configuration is using a local encrypted one.
```sh
$ engine config new cdn > cdn.toml
```



* Generate a auth consumer for your new CDN service.
```sh
$ cdsctl consumer new me \
--scopes=Service,Worker,RunExecution \
--name="cdn" \
--description="Consumer for cdn service" \
--groups="shared.infra" \
--no-interactive

Builtin consumer successfully created, use the following token to sign in:
xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx

$ engine config edit --output cdn.toml cdn.toml cdn.api.token=xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx
```

* Generate a auth consumer for CDS unit for CDN

This unit will be useful to migrate logs from CDS database to CDN.
```sh
$ cdsctl consumer new me \
--scopes=Project,Run \
--name="cdn-storage-cds" \
--description="Consumer for cds storage unit" \
--groups="shared.infra" \
--no-interactive

Builtin consumer successfully created, use the following token to sign in:
xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx

$ engine config edit --output cdn.toml cdn.toml cdn.storageUnits.storages.cds.token=xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx
```

# Start CDN service
By default CDN log processing and migration is active for all projects and you'll have to start the migration process manually. 

Migration will be executed in two steps, the first one will populate the CDN database from known log items and log data will be accessible through the temporary CDS unit. Then CDN will start to sync your storage unit with CDS unit.

If your CDS instance manage a lot of workflow the migration can take a long time, thanks to feature flipping you will be able to manually set the list of CDS project that should use CDN to gradually migrate your projects to CDN.

* (Optional) Set feature flipping for CDN logs to migrate only some projects.
```sh

```

* Start CDN service and migration.

At this steps you will be able to start CDN service. It will start to process logs for activated projects. 

Start migration using the command line:
```sh
$ cdsctl admin cdn migrate
```
This will only migrate activated project, if your are using feature flipping to gradually migrate your projects, you will have to reexecute this command each time you change this projects list.

You can follow the migration from CDN logs or with the CDS command line:
```sh
$ cdsctl admin cdn status
```

![CDN_STATUS](/images/cdn_status.png)
