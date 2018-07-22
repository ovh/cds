+++
title = "RabbitMQ hook"
weight = 3

+++

You want to run a workflow from a [RabbitMQ](http://www.rabbitmq.com/) message? This kind of hook is for you.

This kind of hook will connect to a RabbitMQ queue and consume message. For each message, it will trigger your workflow.

The RabbitMQ message have to be a JSON message, it in will be used as a payload for your workflow [See playload documentation]({{< relref "workflows/design/payload.md" >}}).

## How to use it

### Link your project to a RabbitMQ platform

On your CDS Project, select the platforms section then add a RabbitMQ platform.

![Platform](/images/workflows.design.hooks.rabbitmq-hook.platform.png)

### Add a RabbitMQ hook on the root pipeline of your workflow

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

### Add run condition

The workflow will be triggered for all message received in RabbitMQ queue.

If you don't want to launch the root pipeline for each message, you can add a [run condition]({{< relref "workflows/design/run-conditions.md" >}}).
