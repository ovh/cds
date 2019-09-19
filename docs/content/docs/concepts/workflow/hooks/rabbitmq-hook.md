+++
title = "RabbitMQ hook"
weight = 6

+++

Do you want to run a workflow from a [RabbitMQ](http://www.rabbitmq.com/) message? This kind of hook is for you.

This kind of hook will connect to a RabbitMQ queue and consume messages. For each message, it will trigger your workflow.

The RabbitMQ message have to be in JSON format. It will be used as a payload for your workflow. [See payload documentation]({{< relref "/docs/concepts/workflow/payload.md" >}}).

## Link your project to a RabbitMQ platform

On your CDS Project, select the platforms section then add a RabbitMQ platform.

![Integration](/images/workflows.design.hooks.rabbitmq-hook.platform.png)

## Add a RabbitMQ hook on the root pipeline of your workflow

Click on the pipeline root of a workflow, then choose 'Add a Hook' on the sidebar

![Select Pipeline](/images/workflows.design.hooks.rabbitmq-hook.add.png)

Select the RabbitMQ Hook and complete the information:

- The binding key (AMQP binding key)
- The consumer tag (AMQP consumer tag (should not be blank))
- The exchange name (Durable, non-auto-deleted AMQP exchange name)
- The exchange type (Exchange type - direct|fanout|topic|x-custom)
- The RabbitMQ platform previously configured
- The queue to listen

![Add Hook](/images/workflows.design.hooks.rabbitmq-hook.add.modal.png)

## Add run condition

The workflow will be triggered for all messages received in RabbitMQ queue.

If you don't want to launch the root pipeline for each message, you can add a [run condition]({{< relref "/docs/concepts/workflow/run-conditions.md" >}}).
