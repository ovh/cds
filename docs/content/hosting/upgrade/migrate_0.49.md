---
title: "Migrate 0.48"
weight: 1
---

## CDN service

The release 0.49 introduced a new parameter in CDN configuration to disable and remove a CDN unit.
In the previous version 0.48, you created a CDS Unit for CDN. Time is coming to disabled it.
In the next release, all logs datas in CDS will be deleted, and CDN will become the only way to manage logs.


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

With this parameter, CDN will ignore the unit for:

* unit synchronization
* cleaning redis buffers

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
* Remove the unit using cdsclt

```sh
cdsctl admin cdn unit delete <unit_id>
```
