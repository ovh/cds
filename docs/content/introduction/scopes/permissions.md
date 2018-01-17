+++
title = "Permissions"
weight = 3

+++

Permissions in CDS are managed by group and not on a single user.

You have 3 types of permissions:

+ Read
+ Read / Execute
+ Read / Write / Execute

You can attach a group permission at different scopes:

+ On project
+ On application
+ On environment
+ On pipeline
+ On workflow

Thanks to these different scopes you can for example have group to read, group to execute et group to write. If the permission group on the workflow, pipelines, environments and applications linked is `Read / Execute` then the users in this group can read and execute the workflow and pipeline inside but cannot edit the pipeline or workflow.

A good use case to identify the flexibility of the group permission at different scope is when you have a workflow with different pipelines and 2 pipelines used to deploy your application. One in the pre-production environment and another on the production environment. If you want to permit  all people to deploy on your `pre-production` environment but let specific users to deploy on your `production` environment because you want to control your deployment in `production`. It's possible thanks to these different scopes of group permission if you don't give `Read / Execute` permission on the `production` environment then this group won't be able to execute the pipeline linked to the environment `production`.

**Pay attention**, a common mistake when you add a new group permission on a workflow scope, make sure to give the permission on all linked scopes (project, environments, applications, pipelines).


# Tokens

A group permission is also attached to [cli]({{< relref "cli/_index.md" >}}), [workers]({{< relref "worker/_index.md" >}}), [worker models]({{< relref "workflows/pipelines/requirements/worker-model/_index.md" >}}), [hatchery]({{< relref "hatchery/_index.md" >}}) and all different services in CDS.

When you need to use the CDS CLI in a script and don't want to store password you can use the CLI with an **authentication token**. Or when you start an [hatchery]({{< relref "hatchery/_index.md" >}}) you need an authentication token to contact the CDS API.

In order to list, generate or remove tokens linked to a group permission you can do it with the `cdsctl group token` command ([documentation available here]({{< relref "cli/cdsctl/group/token/_index.md" >}})).

You can do the same via the user interface in the group administration view.

![Job](/images/groups_menu.png)

And then on a group edition view you can handle all your generated tokens and generate new ones.

![Job](/images/group_view.png)
