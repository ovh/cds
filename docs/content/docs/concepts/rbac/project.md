---
title: "Project role"
weight: 2
---

These roles allow users/groups to manage a project

* `read`: Allow users/groups to list all resources defined inside a project
* `manage`: Allow users/groups to manage VCS and repository on a project
* `manage-worker-model`: Allow users/groups to create/update/delete a worker model

Yaml example:
```yaml
name: my-permission-name
projects:
  - role: read
    all: true
    users: [foo,bar]
    groups: [grpFoo]
  - role: manage-worker-model
    users: [foo]
    projects: [PROJ_KEY1, PROJ_KEY2]

```

List of fields:

* `role`: <b>[mandatory]</b> role to applied
* `all`: applied the permission on for all projects
* `projects`: list of projects key if there is no field `all`
* `users`: list of usernames
* `groups`: list of groups
