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

+ On project
+ On workflow
+ On workflow node


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

## Tokens

A group permission is also attached to [CLI]({{< relref "/docs/components/cdsctl/_index.md" >}}), [workers]({{< relref "/docs/components/worker/_index.md" >}}), [worker models]({{< relref "/docs/concepts/worker-model/_index.md" >}}), [hatchery]({{< relref "/docs/components/hatchery/_index.md" >}}) and all different services in CDS.

When using the CDS [CLI]({{< relref "/docs/components/cdsctl/_index.md" >}}) in a script, instead of providing your own passwords, you want to generate and use an [CLI]({{< relref "/docs/components/cdsctl/_index.md" >}}) **authentication token**. Similarly, when you start a [hatchery]({{< relref "/docs/components/hatchery/_index.md" >}}) you will need an authentication token to contact the CDS API.

The `cdsctl group token` command allows to list, generate and remove tokens linked to a group permission ([documentation available here]({{< relref "/docs/components/cdsctl/token/_index.md" >}})). Alternatively, you can do it via the user interface in the group administration view.

Click on the `Groups` entry in the menu
![Job](/images/groups_menu.png)

And then in the group edition view, you can manage all your generated tokens for this group and generate new ones.
![Job](/images/group_view.png)

If you want to list all the tokens that you can use go to your profile page and you will find a list of all the tokens associated to your groups.
