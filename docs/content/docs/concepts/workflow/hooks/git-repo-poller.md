---
title: "Git Repository Poller"
weight: 4
---

Do you want to run a workflow after a git push on a repository **BUT your CDS instance isn't accessible from the internet**? This kind of hook is for you. (If your CDS instance is accessible from the internet please check the [Git Repository Webhook]({{< relref "/docs/concepts/workflow/hooks/git-repo-webhook.md" >}})).

This kind of hook will poll periodically the GitHub API to know the push and pull-request events on your repository.

You have to:

* link your project to a Repository Manager, on Advanced Section
* link an application to a git repository
* add a Git Poller on the root pipeline, this pipeline have the application linked in the [context]({{< relref "/docs/concepts/workflow/pipeline-context.md" >}})

For now, only GitHub are supported for git poller by CDS.
