---
title: LDAP Authentication
main_menu: true
card: 
  name: authentication
---

The LDAP Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to authenticate users.

## How to configure LDAP Authentication integration

Edit the toml file:

- section `[api.auth.ldap]`
  - enable the signin with `enabled = true`
  - if you want to disable user signup, set `signupDisabled = true`

```toml
[api.auth.ldap]
      enabled = false
      host = ""

      # Define it if ldapsearch need to be authenticated
      managerDN = "cn=admin,dc=myorganization,dc=com"

      # Define it if ldapsearch need to be authenticated
      managerPassword = "SECRET_PASSWORD_MANAGER"
      port = 636
      rootDN = "dc=myorganization,dc=com"
      signupDisabled = false
      ssl = true
      userFullname = "{{.givenName}} {{.sn}}"
      userSearch = "uid={0}"
      userSearchBase = "ou=people"
```
