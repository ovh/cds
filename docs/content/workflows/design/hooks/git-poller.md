+++
title = "Git Poller hook"
weight = 3

+++

You want to run a workflow after a git push on a repository **BUT your CDS instance isn't accessible from the internet** ? This kind of hook is for you. (If your CDS instance is accessible from the internet please check the [Git Repository Webhook]({{< relref "workflows/design/hooks/git-repo-webhook.md" >}})).

This kind of hook will poll periodically the Github API to know the push and pull-request events on your repository.

You have to:

* link your project to a Repository Manager, on Advanced Section. [See how to setup repository manager on your CDS instance]({{< relref "hosting/repositories-manager/_index.md" >}}).
* link an application to a git repository
* add a Git Poller on the root pipeline, this pipeline have the application linked in the [context]({{< relref "workflows/design/pipeline-context.md" >}})

For now, only Github are supported for git poller by CDS.
