---
title: "Workflow roles"
weight: 2
---

These roles allow users/groups to realize action on workflows

* `trigger`: Allow users/groups to trigger workflows

Yaml example:
```yaml
name: my-permission-name
workflows:
  - role: trigger
    all_users: true
    project: MYPROJECT
    workflows: [wkf1,wkf2]
    all_workflows: false
    users: [foo,bar]
    groups: [grpFoo]

```

List of fields:

* `role`: <b>[mandatory]</b> role to applied
* `all_users`: applied the permission for all users
* `project`: <b>[mandatory]</b> the key of the project that contains the workflows
* `workflows`: list of workflows inside the given project
* `all_workflows`: applied the permission on all workflow inside the given project
* `users`: list of usernames
* `groups`: list of groups