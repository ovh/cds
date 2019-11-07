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
- `X-Requested-With: X-CDS-SDK`

```bash
# List CDS Project
curl -H "Authorization: Bearer your-cds-token" -H "X-Requested-With: X-CDS-SDK" https://your-cds-api/project
```

## About CDS Token

The CDS UI uses a session token. If you want to play with CDS API, you probably need a persistent token.

You can generate it with:

- [cdsctl consumer new]({{< relref "/docs/components/cdsctl/consumer/new.md" >}})

## CDS HTTP Routes

{{%children style="ul"%}}