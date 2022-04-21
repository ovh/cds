---
title: "Hooks"
weight: 3
card: 
  name: concept_workflow
  weight: 8
---

If you want to trigger the run of your workflow you need some hooks on your root pipeline inside the workflow.

On the root pipeline only, you can add hooks:

* [webhook]({{< relref "/docs/concepts/workflow/hooks/webhook.md" >}})
* [scheduler]({{< relref "/docs/concepts/workflow/hooks/scheduler.md" >}})
* [git repository webhooks]({{< relref "/docs/concepts/workflow/hooks/git-repo-webhook.md" >}})
* [git repository poller]({{< relref "/docs/concepts/workflow/hooks/git-repo-poller.md" >}})
* [kafka hook]({{< relref "/docs/concepts/workflow/hooks/kafka-hook.md" >}})
* [RabbitMQ hook]({{< relref "/docs/concepts/workflow/hooks/rabbitmq-hook.md" >}})

There are two hooks on this pipeline, a repository webhook (GitHub here) and a webhook:

![Hooks](/images/workflows.design.hooks.png)
