---
title: OpenID-Connect Authentication
main_menu: true
card: 
  name: authentication
---

The OpenID-Connect Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to delegate users authetication to an OpenID-Connect third party like [Keycloak](https://www.keycloak.org/getting-started) or [Hydra](https://github.com/ory/hydra)

## How to configure OpenID-Connect Authentication integration

Edit the toml file:

- section `[api.auth.oidc]`
  - enable the signin with `enabled = true`
  - if you want to disable signup, set `signupDisabled = true`

```toml
[api.auth.oidc]
      clientId = "YOUR CLIENT ID"
      clientSecret = "YOUR CLIENT SECRET"
      enabled = true
      signupDisabled = false
      url = "http[s]://<OIDC HOST>:<PORT>/auth/realms/<YOUR REALM>"
```

For example :
```toml
[api.auth.oidc]
      clientId = "cds_client"
      clientSecret = "6ebf3c3f-6f0b-4326-bebd-05fd472a90ec"
      enabled = true
      signupDisabled = false
      url = "http://openid-connect.myorg.com:8080/auth/realms/cds"
```
