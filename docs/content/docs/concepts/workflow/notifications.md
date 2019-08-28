---
title: "Notifications"
weight: 9
---

On a workflow you can have 2 kinds of notifications:

+ User notifications: they are useful to notify users by email or with a message of an event on your workflow (success, fail, change, etc...).
+ Events: linked to event integrations to let you write microservices which can interact with these events plugged on your event integrations.

## User notifications

You can configure user notifications to send email or a message on jabber with different parameters. Inside the body of the notification you can customise the message thanks to the CDS variable templating with syntax like `{{.cds.myvar}}`. You can also use `HTML` to customise the message, then in order to let CDS interpret your message as an `HTML` one you just need to wrap all your message inside html tag like this `<html>MyContentHere</html>`.

## VCS Notifications

You can configure for which node in your workflow CDS have to send a status on your repository service provider (Github, Bitbucket, ...). You can configure if you want to have a comment on your pull-request when your workflow fails. By default you already have a default template for your pull-request comment but you can customize it with different kinds of templating. To have access about the `node run` data and write some loops and conditions you can use the standard syntax as the [go templating](https://golang.org/pkg/text/template/#hdr-Actions) but with `[[` `]]` delimitters. You can also use the CDS interpolation engine with the same syntax you already know and use inside pipelines, for example: `{{.cds.workflow}}` to get the name of the workflow.

For the go templating you have few variables you can use/iterate over.

- `.Stages`: an array of stages with `.RunJobs`inside which are the array of runned jobs with their `.Name`
    - `.RunJobs`: Inside a stage object which are the array of runned jobs
        - `.Job.Action.Name`: The name of the runned job
        - `.Job.Status`: The status of runned job
- `.Tests`: array of tests results
    - `.Total`: total number of tests
    - `.TotalOK`: total number of OK tests
    - `.TotalKO`: total number of KO tests
    - `.TotalSkipped`: total number of skipped tests

If you need to know about other variable you can check data structure [here](https://github.com/ovh/cds/blob/master/sdk/workflow_run.go#L40).

## Events

If you need to trigger some specific actions on the technical side, like for example use a microservice which listens to all events in your workflow (updates, launch, stop, etc.), you can add an event integration like, for example, [Kafka]({{< relref "/docs/integrations/kafka/kafka_events.md">}}) and listen to the kafka topic to trigger some actions on your side. Events are more like sending notifications to machines instead of user notifications which are made for users. The see structure of sent events, you can look [here](https://github.com/ovh/cds/blob/master/sdk/event.go) and [here](https://github.com/ovh/cds/blob/master/sdk/event_workflow.go).
