---
title: "Migrate 0.53"
weight: 1
---

## Migrate an existing instance

Before upgrading your CDS Instance:
- You have to backup your databases: cds and cdn databases.
- You have to install the version 0.52.0.
- You must follow the following step before upgrading to 0.53.0.


## Before upgrading

### Organization

The version 0.52.0 introduced the notion of Organization in CDS for all authentication drivers. In 0.53.0, organizations are now mandatory so you need to add them before upgrading to 0.53.0.


* Upgrade you CDS API configuration to add the following fields on your different authentication drivers.
* List all allowed organizations in the field 'allowedOrganizations'

```toml
[api.auth]
    allowedOrganizations = ["my-organization"]
    [api.auth.local]
      organization = "my-organization"
    [api.auth.github]
      organization = "my-organization"
    [api.auth.gitlab]
      organization = "my-organization"
    [api.auth.oidc]
      organization = "my-organization"      
    [api.auth.corporateSSO] # There is no organization in SSO configuration, as it's provided by the SSO itself
    [api.auth.ldap] # There is no organization in ldap configuration as it's provided by the company ldap field 
```

* Create organization in CDS through the cli

```shell
cdsctl admin organization add my-organization
cdsctl admin organization list
+--------------------------------------+-----------------+
|                  ID                  |      NAME       |
+--------------------------------------+-----------------+
| 47cc19b8-918e-4bc3-b291-b1cf1ba233ef | my-organization |
+--------------------------------------+-----------------+

```

* Migrate all existing users in the organization
```shell
cdsctl admin organization user-migrate my-organization
```


## Upgrading to 0.53.0

This version contains changes on database table used to authenticate users, this will requires CDS to be stopped before the migration.

* Shutdown all CDS's services.
* Apply the following changes to your CDS API configuration:
```
# The field enabled was renamed by signinEnabled in auth api.auth
[api.auth]
    [api.auth.local]
      signinEnabled = true
    [api.auth.github]
      signinEnabled = true
    [api.auth.gitlab]
      signinEnabled = true
    [api.auth.oidc]
      signinEnabled = true
    [api.auth.corporateSSO]
      signinEnabled = true
    [api.auth.ldap]
      signinEnabled = true

# The common configuration for auth drivers were moved to a new config section called drivers
[api.drivers]    
    [api.drivers.github]
      url = ""
      apiUrl = ""
      clientId = ""
      clientSecret = ""
    [api.drivers.gitlab]
      url = ""
      applicationID = ""
      secret = ""
    [api.drivers.oidc]
      ...
    [api.drivers.corporateSSO]
      ...
    [api.drivers.ldap]
      ...
```
* Run the database migration, documentation on https://ovh.github.io/cds/hosting/database/
* Start CDS API service (scale to 1 instance if you usually use multiple instances).
* Login to CDS using the command line and check if there is no error on migration using `cdsctl admin migration list`.
  * There are two migrations to check: 'OrganizationMigration' and 'ConsumerMigration'.
  * Migration can take a few minutes depending on the number of users.
* Scale up CDS API if you usually use multiple instances then restart others services.