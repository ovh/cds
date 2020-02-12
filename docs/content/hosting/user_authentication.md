---
title: "User Authentication"
weight: 10
card: 
  name: operate
---

CDS can use several authentication backends:

 - local
 - [LDAP]({{< relref "/docs/integrations/ldap.md" >}})
 - [GitHub]({{< relref "/docs/integrations/github/github_authentication.md" >}})
 - [GitLab]({{< relref "/docs/integrations/gitlab/gitlab_authentication.md" >}})

All backends can be enabled at the same time, ie. a user can authenticate both with GitHub, GitLab, Ldap or with local authentication at the same time.

## Local Authentication

Edit the [toml configuration file]({{<relref "/hosting/configuration.md" >}}):

- section `[api.auth.local]`
  - enable the signin with `enabled = true`
  - if you want to let user signup with GitHub, set `signupDisabled = true`
  - you can authorize only some domains with the key `signupAllowedDomains`
  
```toml
    [api.auth.local]
      enabled = true

      # Allow signup from selected domains only - comma separated. Example: your-domain.com,another-domain.com
      # signupAllowedDomains = ""
      signupDisabled = false
```

# User Token

See [Token Documentation]({{<relref "/development/sdk/token.md" >}})