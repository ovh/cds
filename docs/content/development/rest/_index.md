---
title: "REST API"
main_menu: true
weight: 6
card: 
  name: rest-sdk
---

## How to request CDS API?

You need two HTTP Headers to request CDS API:

- `Authorization: Bearer cds-session-token`

```bash
# List CDS Project
curl -H "Authorization: Bearer cds-session-token" https://your-cds-api/project
```

{{< note >}}
To generate the CDS token please check [here]({{< relref "/development/sdk/token.md" >}})
{{< /note >}}

## CDS HTTP Routes

{{%children style="ul"%}}