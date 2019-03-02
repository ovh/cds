+++
title = "Hooks"
weight = 3

+++

If you want to trigger the run of your workflow you need some hooks on your root pipeline inside the workflow.

On the root pipeline only, you can add hooks:

* [webhook]({{< relref "/manual/concepts/workflow/hooks/webhook.md" >}})
* [scheduler]({{< relref "/manual/concepts/workflow/hooks/scheduler.md" >}})
* [repository webhooks]({{< relref "/manual/concepts/workflow/hooks/git-repo-webhook.md" >}})
* [git poller]({{< relref "/manual/concepts/workflow/hooks/git-poller.md" >}})
* [kafka hook] ({{< relref "/manual/concepts/workflow/hooks/kafka-hook.md" >}})
* [RabbitMQ hook] ({{< relref "/manual/concepts/workflow/hooks/rabbitmq-hook.md" >}})

There are two hooks on this pipeline, a repository webhook (GitHub here) and a webhook:

![Hooks](/images/workflows.design.hooks.png)
