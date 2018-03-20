+++
title = "Kafka hook"
weight = 3

+++

You want to run a workflow from a kafka message? This kind of hook is for you.

This kind of hook will connect to a kafka topic and consume message. For each message, it will trigger your workflow.

The kafka message have to be a JSON message, it in will be used as a payload for your workflow [See playload documentation]({{< relref "workflows/design/payload.md" >}}).

## How to use it

### Link your project to a Kafka platform

On your CDS Project, select the platforms section then add a Kafka platform.

![Platform](/images/workflows.design.hooks.kafka-hook.platform.png)

### Add a Kafka hook on the root pipeline of your workflow

Click on the pipeline root of a workflow, then choose 'Add a Hook' on the sidebar

![Select Pipeline](/images/workflows.design.hooks.kafka-hook.add.png)

Select the Kafka Hook and complete the information:

- The Consumer group
- Select the kafka platform
- The kafka topic to read

![Add Hook](/images/workflows.design.hooks.kafka-hook.add.modal.png)

### Add run condition

The workflow will be triggered for all message received in kafka queue.

If you don't want to launch the root pipeline for each message, you can add a [run condition]({{< relref "workflows/design/run-conditions.md" >}}).