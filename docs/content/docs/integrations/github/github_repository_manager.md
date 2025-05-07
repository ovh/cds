---
title: GitHub Repository Manager
main_menu: true
card: 
  name: repository-manager
---

The GitHub Repository Manager Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to link a Git Repository hosted by GitHub to a CDS Application.

This integration enables some features:

 - [Git Repository Webhook]({{<relref "/docs/concepts/workflow/hooks/git-repo-webhook.md" >}})
 - [Git Repository Poller]({{<relref "/docs/concepts/workflow/hooks/git-repo-poller.md" >}})
 - Easy to use action [CheckoutApplication]({{<relref "/docs/actions/builtin-checkoutapplication.md" >}}) and [GitClone]({{<relref "/docs/actions/builtin-gitclone.md">}}) for advanced usage
 - Send build notifications on your Pull-Requests and Commits on GitHub. [More informations]({{<relref "/docs/concepts/workflow/notifications.md#vcs-notifications" >}})
 - Send comments on your Pull-Requests when a workflow is failed. [More informations]({{<relref "/docs/concepts/workflow/notifications.md#vcs-notifications" >}})

## How to configure GitHub integration

### Create the Personal Access Token on GitHub

Generate a new token on https://github.com/settings/tokens with the following scopes:
 - repo:status
 - public_repo

### Import configuration

Create a yml file:

```yaml
version: v1.0
name: github
type: github
description: "my github"
auth:
    username: your-username
    token: ghp_your-token-here
options:
    urlApi: "" # optional, default is https://api.github.com
    disableStatus: false    # Set to true if you don't want CDS to push statuses on the VCS server - optional
    disableStatusDetails: false # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server - optional
    disablePolling: false   # Does polling is supported by VCS Server - optional
    disableWebHooks: false  # Does webhooks are supported by VCS Server - optional
```

```sh
cdsctl project vcs import YOUR_CDS_PROJECT_KEY vcs-github.yml
```

### Add a repository webhook on a workflow

*As a user, with writing rights on a CDS project* 

Select the first pipeline, then click on `Add a hook`.

![github-wf-select-pipeline.png](../../images/github-wf-select-pipeline.png?height=500px)

Select the **RepositoryWebhook**, then click on **Save**.

![github-wf-add-repowebhook.png](../../images/github-wf-add-repowebhook.png?height=200px)

The webhook is automatically created on GitHub. 

## What's next?

- Use [CheckoutApplication]({{<relref "/docs/actions/builtin-checkoutapplication.md">}}) in your pipeline
- `git push` on your Git Repository on GitHub
- See the build status on GitHub.

## FAQ

### **My CDS is not accessible from GitHub, how can I do?**

When someone git push on your Git Repository, GitHub have to call your CDS to run your CDS Workflow.
This is the behaviour of the [RepositoryWebhook]({{<relref "/docs/concepts/workflow/hooks/git-repo-webhook.md">}}). But if your CDS is not reachable from GitHub, how can you do?

By chance, you have two choices :) 

- When you add a Hook on your workflow, select the **Git Repository Poller**. The µService Hooks
will call regularly GitHub. With this way, GitHub doesn't need to call your CDS.

[Git Repository Poller documentation]({{<relref "/docs/concepts/workflow/hooks/git-repo-poller.md">}})

If you hesitate between the two: the `RepositoryWebhook` is more *reactive* than the `Git Repository Poller`.

### **I don't see the type Git Repository Poller nor RepositoryWebhook when I add a Hook**

Before adding a hook on your Workflow, you have to add the application in the Pipeline Context.
Select the first pipeline, then click on **Edit the pipeline context** from the [sidebar]({{<relref "/docs/concepts/workflow/sidebar.md">}}).

[Pipeline Context Documentation]({{<relref "/docs/concepts/workflow/pipeline-context.md">}})

## VCS events

For now, CDS supports push events. CDS uses this push event to remove existing runs for deleted branches (24h after branch deletion).