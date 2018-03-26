+++
title = "REST API"
+++

## How to request CDS API?

You need two HTTP Headers to request CDS API:

- `Session-Token: your-cds-token`
- `X-Requested-With: X-CDS-SDK`

```bash
# List CDS Project
curl -H "Session-Token: your-cds-token" -H "X-Requested-With: X-CDS-SDK" https://your-cds-api/project
```

## About CDS Token

The CDS UI uses a non-persistent token. If you want to play with CDS API, you probably need a persistent token.

You can generate it with:

- [cdsctl login]({{< relref "/cli/cdsctl/login.md" >}})
- Code it with the [Go SDK]({{< relref "/cli/sdk/_index.md" >}})
- Call CDS API: POST `/login` with body `{"username":"your-username","password":"your-password"}`

## CDS HTTP Routes

{{%children style="ul"%}}