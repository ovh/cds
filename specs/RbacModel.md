# RBAC Model (Role-Based Access Control)

## Overview

The CDS RBAC model provides fine-grained control over authorizations granted to users and groups on the various platform resources. An RBAC permission is a named entity that groups together a set of access rules organized by scope.

## General Structure

An RBAC permission (`RBAC`) is identified by a unique name. It aggregates several types of rules, each corresponding to a different scope:

- **Global**: platform administration rights
- **Project**: rights on CDS projects
- **Workflow**: rights on workflows
- **Region**: rights on execution regions
- **Hatchery**: rights on hatcheries (worker managers)
- **VariableSet**: rights on a project's variable sets
- **RegionProject**: rights to execute a project on a given region

## Scopes and Their Roles

### Global

Available roles:

| Role | Usage |
|------|-------|
| `create-project` | Create new projects |
| `manage-permission` | Manage RBAC permissions |
| `manage-organization` | Manage organizations |
| `manage-region` | Manage regions |
| `manage-hatchery` | Manage hatcheries |
| `manage-user` | Manage users |
| `manage-group` | Manage groups |
| `manage-plugin` | Manage plugins |

A global rule must designate at least one user or group.

### Project

Available roles:

| Role | Usage |
|------|-------|
| `read` | Read project resources |
| `manage` | Administer the project |
| `manage-notification` | Manage notifications |
| `manage-worker-model` | Manage worker models |
| `manage-action` | Manage actions |
| `manage-workflow` | Manage workflows |
| `manage-workflow-template` | Manage workflow templates |
| `manage-variableset` | Manage variable sets |

A project rule applies to a list of project keys (`projects`). Recipients can be:
- a list of users or groups
- all users (`all_users: true`)
- VCS users (`vcs_users`)
- all VCS users (`all_vcs_users: true`)

`all_users` and an explicit list of users/groups are mutually exclusive.

### Workflow

Available roles:

| Role | Usage |
|------|-------|
| `trigger` | Trigger a workflow |

A workflow rule is scoped to a project (`project`) and to a list of workflows (`workflows`). The `all_workflows` flag targets all workflows in the project. Recipients follow the same rules as for projects.

### Region

Available roles:

| Role | Usage |
|------|-------|
| `list` | List executions on the region |
| `execute` | Execute on the region |
| `manage` | Administer the region |

A region rule applies to a region identified by its ID. It must reference at least one organization. Recipients can be users, groups, VCS users, or all VCS users (`all_vcs_users: true`). Consistency is verified: a group must belong to one of the organizations declared in the rule.

### Hatchery

Available roles:

| Role | Usage |
|------|-------|
| `start-worker` | Authorize a hatchery to start workers on a region |

A hatchery rule links a hatchery (`hatchery_id`) to a region (`region_id`).

### VariableSet

Available roles:

| Role | Usage |
|------|-------|
| `use` | Use the variable set in a workflow |
| `manage-item` | Manage items of the variable set |

A variable set rule is scoped to a project (`project`) and to a list of variable sets (`variablesets`). The `all_variablesets` flag targets all variable sets in the project.

### RegionProject

Available roles:

| Role | Usage |
|------|-------|
| `execute` | Authorize a project to run jobs on a region |

A region-project rule links a region to a list of projects (`projects`). The `all_projects` flag targets all projects.

## Data Integrity

All RBAC entities in the database are cryptographically signed (`SignedEntity`). During loading, the signature is verified for each record; a corrupted entry is ignored with a log error, without interrupting the loading of other rules.

## Validation

Before any insertion or update, the RBAC rule is validated:

- The name cannot be empty.
- Each sub-rule must have a valid role for its scope.
- Required fields (users/groups, region, organization, etc.) must be present.
- The combination of `all_users` + an explicit list is forbidden.
- For regions, organization/group consistency is verified against the database.
