+++
title = "Permissions"
weight = 3

+++

In CDS, there are 3 types of permissions:

+ Read
+ Read / Execute
+ Read / Write / Execute

These permissions can be attached to different objects:

+ On project
+ On application
+ On environment
+ On pipeline
+ On workflow

In CDS, permissions cannot be attached directly to users. Permissions need to be attached to groups of users. Users inherit their permissions from the groups they are belonging to.

Example usage: Enforce a strict separation of duties by allowing a group of people to view/inspect a workflow, a second group will be able to push it to a `staging` environment and a third group will be allowed to push it to a `production` environment. You can have a fourth group responsible of editing the workflow if needed.

A more common scenario consists in giving `Read / Execute` permissions on the `staging` environment to everyone in your development team while restricting the `production` deployments and the pipeline edition to a smaller group of users.

**Warning:** when you add a new group permission on a workflow scope, make sure to give the permission on all linked scopes (project, environments, applications, pipelines).

# Tokens

A group permission is also attached to [CLI]({{< relref "cli/_index.md" >}}), [workers]({{< relref "worker/_index.md" >}}), [worker models]({{< relref "workflows/pipelines/requirements/worker-model/_index.md" >}}), [hatchery]({{< relref "hatchery/_index.md" >}}) and all different services in CDS.

When using the CDS [CLI]({{< relref "cli/_index.md" >}}) in a script, instead of providing your own passwords, you want to generate and use an [CLI]({{< relref "cli/_index.md" >}}) **authentication token**. Similarly, when you start a [hatchery]({{< relref "hatchery/_index.md" >}}) you will need an authentication token to contact the CDS API.

The `cdsctl group token` command allows to list, generate and remove tokens linked to a group permission ([documentation available here]({{< relref "cli/cdsctl/token/_index.md" >}})). Alternatively, you can do it via the user interface in the group administration view.

Click on the `Groups` entry in the menu
![Job](/images/groups_menu.png)

And then in the group edition view, you can manage all your generated tokens for this group and generate new ones.
![Job](/images/group_view.png)

If you want to list all the tokens that you can use go to your profile page and you will find a list of all the tokens associated to your groups.
