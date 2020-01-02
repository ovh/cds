---
title: "Permissions - ACLs"
weight: 3
card: 
  name: concept_organization
---

There are 3 types of permissions:

+ Read (as code value: 4)
+ Read / Execute (as code value: 5)
+ Read / Write / Execute (as code value: 7)

These permissions can be attached to different objects:

+ Project
+ Workflow
+ Workflow node

|                                                                                       | Project | Workflow | Workflow node                               |
|---------------------------------------------------------------------------------------|---------|----------|---------------------------------------------|
| Create a workflow                                                                     |   RWX   |     -    |                      -                      |
| Edit a workflow  (change run conditions, add nodes, edit payload, notifications, ...) |    RO   |    RWX   |                      -                      |
| Create/edit an environment/pipeline/application                                       |   RWX   |     -    |                      -                      |
| Manage permissions on project                                                         |   RWX   |     -    |                      -                      |
| Manage permissions on a workflow                                                      |    RO   |    RWX   |                                             |
| Run a workflow                                                                        |    RO   |    RX    | / - OR RX (if there is some groups on node) |

Permissions cannot be attached directly to users, they need to be attached to groups of users. Users inherit their permissions from the groups they are belonging to.

Example usage: Enforce a strict separation of duties by allowing a group of people to view/inspect a workflow, a second group will be able to push it to a `deploy-to-staging` node and a third group will be allowed to push it to a `deploy-to-production` node. You can have a fourth group responsible of editing the workflow if needed.

A more common scenario consists in giving `Read / Execute` permissions on the node `deploy-to-staging` to everyone in your development team while restricting the `deploy-to-production` node and the project edition to a smaller group of users.

**Warning:** when you add a new group permission on a workflow node, **only the groups linked on the node will be taken in account**.
