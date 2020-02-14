---
title: "Gerrit Hook"
weight: 7
---

Do you want to trigger a workflow from a gerrit event? This kind of hook is for you.

You have to:

* link your project to a Gerrit Server, on Advanced Section
* link an application to a Gerrit repository
* add a Gerrit Hook on the root pipeline, this pipeline have the application linked in the [context]({{< relref "/docs/concepts/workflow/pipeline-context.md" >}})

With this hook, you will have access to specific variables:

* gerrit.change.id: ID of the change
* gerrit.change.url: URL of the change
* gerrit.change.status: Status of the change
* gerrit.change.branch: Destination branch of the change
* gerrit.ref.name: Full reference name within project
* gerrit.change.ref: Git reference of the change
