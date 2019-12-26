---
title: "REST API"
main_menu: true
weight: 6
card: 
  name: rest-sdk
---

## How to request CDS API?

You need two HTTP Headers to request CDS API:

- `Authorization: Bearer your-cds-token`

```bash
# List CDS Project
curl -H "Authorization: Bearer your-cds-token" https://your-cds-api/project
```

{{< note >}}
To generate the CDS token please check [here]({{< relref "/development/sdk/token.md" >}})
{{< /note >}}

## About CDS Token

If you want to play with CDS API, you probably need a CDS consumer token.

You can generate it with:

- [cdsctl consumer new]({{< relref "/docs/components/cdsctl/consumer/new.md" >}})

## CDS HTTP Routes

{{%children style="ul"%}}