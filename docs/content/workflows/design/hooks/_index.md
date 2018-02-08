+++
title = "Hooks"
weight = 4

+++

If you want to trigger the run of your workflow you need some hooks on your root pipeline inside the workflow.

On the root pipeline only, you can add hooks:

* [webhook]({{< relref "workflows/design/hooks/webhook.md" >}})
* [scheduler]({{< relref "workflows/design/hooks/scheduler.md" >}})
* [repository webhooks]({{< relref "workflows/design/hooks/git-repo-webhook.md" >}})

There are two hooks on this pipeline, a repository webhook (Github here) and a webhook:

![Hooks](/images/workflows.design.hooks.png)
