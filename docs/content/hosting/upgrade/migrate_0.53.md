---
title: "Migrate 0.53"
weight: 1
---

## Migrate an existing instance

Before upgrading your CDS Instance:
- You have to backup your databases: cds and cdn databases.
- You have to install the version 0.52.0
- You must follow the following step before upgrading to 0.53.0


## Before upgrading

### Organization

The version 0.52.0 introduces the notion of Organization in CDS for all authentication drivers. In 0.53.0, organizations are mandatories so you need to add them before upgrading to 0.53.0


* Upgrade you CDS API configuration to add organization on your different authentication drivers
* List all allowed organization in the field 'allowedOrganizations'

```toml
[api.auth]
    allowedOrganizations = ["my-organization"]

    [api.auth.local]
      enabled = true
      organization = "my-organization"
      signupDisabled = false


    [api.auth.github]
      organization = "my-organization"
      apiUrl = "https://api.github.com"
      clientId = "xxx"
      clientSecret = "xxx"
      enabled = true
      signupDisabled = false
      url = "https://github.com"

    [api.auth.gitlab]
      organization = "my-organization"
      applicationID = "xxx"
      enabled = true
      secret = "xxx"
      signupDisabled = false
      url = "https://gitlab.com"

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

As we are updating DB table around authentication consumer, you will need to completly shutdown your CDS instances and all µservices.

* Shutdown CDS and all µservices
* Run the database migration, documentation on https://ovh.github.io/cds/hosting/database/
* Start 1 (scale to 1 if you usually use multiple instances) CDS API, check if there is no error on migration, with `cdsctl admin migration list`
  * There are two migrations to check: 'OrganizationMigration' and 'ConsumerMigration'
  * Migration can take a few minutes (between 1 and 5) depending on the number of users you have.
* Scale up CDS API if you usually use multiple instances 
* Start other µservices 




