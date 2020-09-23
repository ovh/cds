---
title: "Migrate logs into CDN"
weight: 1
---

Here the steps to follow to migrate all your logs from CDS database into CDN

* Generate a token

```bash
$ cdsctl consumer new me \
--scopes=Project,Run \
--name="cdn" \
--description="Consumer token for cds storage unit " \
--groups="shared.infra" \
--no-interactive

Builtin consumer successfully created, use the following token to sign in:
xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx
```
* Update your CDN configuration to add CDS as a CDN Storage Unit

```
# Must be true to activate CDN
enableLogProcessing = true 


# Configuration of a CDS backend
[[cdn.storageUnits.storages]]
    cron = "* * * * * ?"
    name = "cds-backend"

    [cdn.storageUnits.storages.cds]
        host = "http://cdsapi:8081"
        insecureSkipVerifyTLS = false
        token = "<token.generated.previously>"

```

* Start CDN: it will begin to migrate all your logs from all your projects

* You can follow the migration from CDN logs or with the CDS command line:

![CDN_STATUS](/images/cdn_status.png)
