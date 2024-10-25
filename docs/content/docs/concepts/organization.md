---
title: "Organization"
weight: 5
card:
  name: concept_organization
---

Organization represents an organizational group that contains a set of people.

* a project belongs to only one organization, its value is computed based on the groups that have R/W/E permissions on the project.
  * you'll be able to add a group from another organization only with R permission.
* a group belongs to only one organization. You cannot have users from different organizations in the same group.

# Configuration

* You must defined a list of allowed organization in your CDS configuration file
````toml
  ##############################
  # CDS Authentication Settings#
  ##############################
  [api.auth]

    allowedOrganizations = ["default"]
````

* For each authentication method, you must defined an organization that will be attached to the users that use it
  * github
  ```
    [api.auth.github]
      organization = "default"
  ```
  * gitlob
  ```
   [api.auth.gitlab]
      organization = "default"
  ```
  * local
  ```
  [api.auth.local]
      organization = "default"
  ```
  * openID connect
  ```
   [api.auth.oidc]
      organization = "default"
  ```
  * ldap: map to company field
