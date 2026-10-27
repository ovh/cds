---
title: "Authentication"
weight: 1
tags: ["scope", "scopes", "consumer", "consumers", "session", "sessions", "builtin", "gitlab", "github", "sso", "local", "ldap"]
card: 
  name: concept_authentication
  weight: 4
---

# Consumer

Two type of consumer: 

- first level: GitLab, GitHub, CorporateSSO, LDAP, Local.
- n level: Builtin.

A builtin consumer can be created by a user. 
Every builtin consumer should have a parent consumer that can also be another builtin consumer.
Using a child consumer you can give permission for all or a part of what its parent can access.

## Groups

A consumer includes a list of groups. A first level consumer is always a wildcard.
A builtin consumer is either a wildcard (an empty list) or an explicit list of groups.
A wildcard resolves to the groups its user is currently a member of.

When a builtin consumer is created, the requested groups are checked against the
groups its parent consumer grants:

- The parent's granted groups are its own explicit list, or, when the parent is a
  wildcard, the groups its user is currently a member of.
- Every explicit group requested must be one of the parent's granted groups;
  a group outside that set is rejected. A CDS admin creating a consumer from a
  first level (non-builtin) consumer may add any group, including `shared.infra`.
- A request with no groups (a wildcard) inherits the parent: from a wildcard
  parent it stays a wildcard (all the user's groups); from a parent restricted to
  an explicit list it takes that list. A child is therefore never broader than its
  parent.

## Scopes

Scope are setup on api routes, this mecaniscm allows to let a consumer access only a part of CDS handlers.
A consumer includes a list of Scopes, first level consumer contains all scopes by default (wildcard). Second level consumer should at least include one scope.
Each scope added in a builtin consumer should be in its parent.

Hatchery: service, hatchery, run execution, worker model

Hook: service, hooks, project, run

Other: service

Scopes list:

- User: access to handlers for user profile and contact management.
- AccessToken: access to handlers for user authentication management, create new consumers, revoke sessions...
- Action: access to handlers for action management. 
- Admin: access to admin handlers.
- Group: access to handlers for group management.
- Template: access to handlers for workflow template management.
- Project: access to handlers for project management.
- Run: access to handlers for workflow run management.
- RunExecution.
- Hooks.
- Worker. 
- WorkerModel: access to handlers for worker model management.
- Hatchery.
- Service.

## Builtin consumer regen

This allow you to get a new consumer signin token for a builtin consumer.
Only consumers that are not disabled can be regen. If there are invalidated groups in the consumer, they will be removed.
When a consumer is regenerated, its issued date will be updated so all old signin token will be invalidated.

## Changing user's group

If a user is removed from a group, the group should be invalidated in all the consumers that contains it.
If it was the last group of the consumer we also want to disable the consumer.
If user is re-added in a given group we restore consumers where this group was invalidated. Also if the consumer was disabled we re-enable it.

## Deleting a group

Is a group was removed we removes all references to this group from all consumers.
If it was the last group for a consumer the consumer will be disabled.

## Changing user ring

A CDS admin can create builtin consumers that includes all groups including the shared.infra group.
A CDS maintainer or a simple user can only includes some of its groups.
When a user is downgraded from admin to another ring, we invalidates all the groups in its consumers where he is not part of.
If all the groups are invalid the consumer will be disabled.
When a user ring is set to admin, we check if there are consumers that contains invalid group that can be restored and re-enable consumers if needed.

