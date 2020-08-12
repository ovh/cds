---
title: "Kafka hook"
weight: 5
---

Do you want to run a workflow from a Kafka message? This kind of hook is for you.

This kind of hook will connect to a Kafka topic and consume messages. For each message, it will trigger your workflow.

The Kafka message have to be in JSON format. It will be used as a payload for your workflow. [See payload documentation]({{< relref "/docs/concepts/workflow/payload.md" >}}).

Notice that Kafka communication is done using SASL and TLS enable only.

## Link your project to a Kafka platform

On your CDS Project, select the platforms section then add a Kafka platform.

![Integration](/images/workflows.design.hooks.kafka-hook.platform.png)

## Add a Kafka hook on the root pipeline of your workflow

Click on the pipeline root of a workflow, then choose 'Add a Hook' on the sidebar

![Select Pipeline](/images/workflows.design.hooks.kafka-hook.add.png)

Select the Kafka Hook and complete the information:

- The Consumer group
- Select the Kafka platform
- The Kafka topic to read

![Add Hook](/images/workflows.design.hooks.kafka-hook.add.modal.png)

## Add run condition

The workflow will be triggered for all messages received in Kafka queue.

If you don't want to launch the root pipeline for each message, you can add a [run condition]({{< relref "/docs/concepts/workflow/run-conditions.md" >}}).
