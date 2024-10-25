---
title: "Region roles"
weight: 2
---

These roles allow users/groups to manage a region or start job on a region

* `list`: Allow users/groups to list/get the given region
* `manage`: Allow users/groups to manage the given region
* `execute`: Allow users/groups to start jobs on the given region

Yaml example:
```yaml
name: my-permission-name
regions:
  - role: execute
    region: nyc-infra
    all_users: true
    organization: US

```

List of fields:

* `role`: <b>[mandatory]</b> role to applied
* `region`: <b>[mandatory]</b> the region name
* `all_users`: applied the permission for all users
* `organizations`: [organization](/docs/concepts/organization/) allowed
* `users`: list of usernames
* `groups`: list of groups
