---
title: "Migrate 0.48"
weight: 1
---

## CDN service

The release 0.48 introduced a new CDS service called CDN. This service is dedicated to receive and store CDS’s job logs.

We created this service to be able to move out job's logs from CDS database to an object storage provider (more information about this list of providers [here]({{< relref "/docs/components/cdn.md">}})).

In this release, logs are stored both in CDN storage units and CDS database to facilitate migration. Old log data and database table will be removed in a future release.

# Prepare CDN service configuration
* Init CDN configuration using `engine` binary.
```sh
$ engine config new cdn > cdn.toml
```
Depending your needs you can change the default configuration to use a different storage unit (see: cdn.storageUnits.storages), the default configuration is using a local unit with encryption.

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

The CDS unit will be useful to migrate logs from CDS database to CDN.

# Init CDN database
CDN service will require a Postgres database, you can use the same database for CDS API and CDN with different schema or use two distinct databases.
To init the database please follow the database management guide [here]({{< relref "/hosting/database.md" >}}).

# Start CDN service
By default CDN log processing and migration is active for all projects and you'll have to start the migration process manually. 

Migration will be executed in two steps, the first one will populate the CDN database from known log items. Then log content will be accessible through the temporary CDS unit and CDN will start to sync your storage unit with CDS unit.

If your CDS instance handles a lot of workflows, the migration may take a long time, thanks to the feature flip, you will be able to manually define the list of CDS projects that should use CDN to gradually migrate your projects to CDN.

* (Optional) Set feature flipping for CDN logs to migrate only some projects.
```sh
cat <<EOF > feature.yaml
name: cdn-job-logs
rule: return project_key == "PROJECT1" or project_key == "PROJECT2"
EOF
cdsctl admin feature import feature.yaml
```

* Start CDN service and migration.

At this step, you will be able to start the CDN service. It will start processing logs for activated projects. 

Start migration using the command line:
```sh
$ cdsctl admin cdn migrate
```
This will only migrate activated projects. If you are using feature flipping to gradually migrate your projects, you will need to rerun this command each time you change this list of projects.

You can follow the migration from CDN logs or with the CDS command line:
```sh
$ cdsctl admin cdn status
```

![CDN_STATUS](/images/cdn_status.png)

## Other changes
- The CDS migrate service is now able to manage SQL migrations on both API and CDN databases. Its configuration file format changed to add CDN database config.
Please use the following command to generate a config example and update your existing one to this new format.
```sh
$ engine config new migrate
```
- Workflow Run will now contains a snapshot of required secrets when starting. This allows changing a secret on CDS entities without breaking existing runs. A migration script will generate this snapshot for recent Workflow Runs (< 3 Months), others will be set to read only mode.
- CDS UI service will now proxy both API and CDN calls, please add the following lines in your existing config.
```sh
# CDN µService URL
cdnURL = "http://localhost:8089" # change this URL if needed to match CDN address
```
