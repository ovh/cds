---
title: "Permission"
weight: 1
card:
  name: cds_as_code
  weight: 1
---

# Description
CDS permissions system uses a Role-based Access Control (RBAC).

You will be able to manage all CDS resources, included ascode resources. 
For example, if a user updates your workflow files on your repository but doesn't have permission to do so, his changes will be discarded.

You can manage permissions on all CDS resources through 5 sections:

* [`global`](./global/)
* [`hatcheries`](./hatchery/)
* [`regions`](./region/)
* [`workflows`](./workflow/)
* [`projects`](./project/)

# CLI

Permissions can be managed by [CDS cli]({{< relref "/docs/components/cdsctl/experimental/rbac" >}}).

# Permission

You need the permission `manage-permission` to be able to created/update/delete a permission

# Yaml Example

```yaml
name: my-full-permission
global:
  - role: manage-permission
    users: [foo]
hatcheries:
  - role: start-worker
    region: nyc-infra
    hatchery: my-swarm-hatchery
projects:
  - role: read
    all: true
    users: [foo]
    groups: [grpFoo]
  - role: manage-workflow
    users: [foo]
    projects: [PROJ_KEY1, PROJ_KEY2]
regions:
  - role: execute
    region: nyc-infra
    all_users: true
    organization: US
workflows:
  - role: trigger
    all_users: true
    project: PROJ_KEY1
    all_workflows: true
    users: [foo]
```