---
title: "Migrate 0.49"
weight: 1
---

## CDN service

The release 0.49 introduced a new parameter in CDN configuration to disable and remove a CDN unit.
In the previous version 0.48, you migrate log from CDS to CDN, by adding a temporary CDN Unit [here]({{< relref "/hosting/upgrade/migrate_0.48.md" >}}). Time is coming to disabled it.
In the next release (0.50), all logs data in CDS will be deleted, and CDN will become the only way to manage logs.


# Disable CDS Unit

Add in your cdn configuration, the property "disableSync = true" for the CDS Unit

```toml
[cdn.storageUnits.storages.cds-backend]
    syncParallel = 6
    disableSync = true
    [cdn.storageUnits.storages.cds-backend.cds]
        host = "https://<your.cds.api>"
        token = "<your.token>"
```

# Remove CDS Unit item from CDN Database

To remove CDS Unit items, follow these steps:

* Disable CDS Unit in CDN
* Retrieve CDS Unit identifier usings cdsctl 

```sh
cdsctl -c prod admin cdn unit list
```
   * Mark CDS Unit item as delete using cdsctl
```sh
cdsctl admin cdn unit delete-items <unit_id>
```

# Remove CDS Unit from CDN Database

To remove CDS Unit from CDN, follow these steps:

* Remove all CDS Unit items from CDN database
* Remove the unit using cdsctl

```sh
cdsctl admin cdn unit delete <unit_id>
```
