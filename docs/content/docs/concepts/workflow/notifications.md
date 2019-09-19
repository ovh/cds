---
title: "Notifications"
weight: 9
---

On a workflow you can have 2 kinds of notifications:

+ User notifications: they are useful to notify users by email or with a message of an event on your workflow (success, fail, change, etc...).
+ Events: linked to event integrations to let you write microservices which can interact with these events plugged on your event integrations.

## User notifications

You can configure user notifications to send email or a message on jabber with different parameters. Inside the body of the notification you can customise the message thanks to the CDS variable templating with syntax like `{{.cds.myvar}}`. You can also use `HTML` to customise the message, then in order to let CDS interpret your message as an `HTML` one you just need to wrap all your message inside html tag like this `<html>MyContentHere</html>`.

## Events

If you need to trigger some specific actions on the technical side, like for example use a microservice which listens to all events in your workflow (updates, launch, stop, etc.), you can add an event integration like, for example, [Kafka]({{< relref "/docs/integrations/kafka/kafka_events.md">}}) and listen to the kafka topic to trigger some actions on your side. Events are more like sending notifications to machines instead of user notifications which are made for users. The see structure of sent events, you can look [here](https://github.com/ovh/cds/blob/master/sdk/event.go) and [here](https://github.com/ovh/cds/blob/master/sdk/event_workflow.go).
