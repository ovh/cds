+++
title = "Kafka hook"
weight = 3

+++

You want to run a workflow from a kafka message ? This kind of hook is for you.

This kind of hook will connect to a kafka topic and consume message. For each message it will trigger your workflow.

You have to:

* link your project to a Kafka platform, on Advanced Section.
* add a Kafka hook on the root pipeline of your workflow
