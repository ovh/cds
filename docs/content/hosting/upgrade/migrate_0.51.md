---
title: "Migrate 0.51"
weight: 1
---

## Migrate an existing instance

Before upgrading your CDS Instance:
- You have to backup your databases: cds and cdn databases.
- You have to install the version 0.50.0 if you use the <= 0.49 version.

## CDN Service

All artifacts upload / download will be done through the CDN service, this is now enabled by default.

The feature flipping `cdn-artifact` to enable cdn artifact is now obsolete, you can remove it.

```sh
cdsctl admin feature delete cdn-artifact
```

## Hatchery Marathon

This hatchery is now deprecated, it will be removed in the next release.

