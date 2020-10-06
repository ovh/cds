---
title: Kafka CDS Events
main_menu: true
card:
  name: events
---

The Kafka Integration is a Self-Service integration that can be configured on a CDS Project.
If you are a CDS Administrator, you can configue this integration to be available on all CDS Projects.

An example of use case could be that you want to generate some svg badge. Then you can just add an event
integration for your workflow and plug the
[badge microservice](https://github.com/ovh/cds/tree/master/contrib/uservices/badge) on this kafka topic.

Notice that Kafka communication is done using SASL and TLS enable only.

## Configure with WebUI

You can add a Kafka Integration on your CDS Project.

![Integration](../images/kafka-integration-webui.png)

## Configure with cdsctl

### Import a Kafka Integration on your CDS Project

Create a file `project-configuration.yml`:

```yml
name: your-kafka-integration
model:
  name: Kafka
  identifier: github.com/ovh/cds/integration/builtin/kafka
  event: true
config:
  broker url:
    value: n1.o1.your-broker:9093,n2.o1.n1.o1.your-broker:9093,n3.o1.n1.o1.your-broker:9093
    type: string
  password:
    value: "**********"
    type: password
  username:
    value: kafka-username
    type: string
  topic:
    value: your-topic
    type: string
  version:
    value: "2.1.1"
    type: string
```

Import the integration on your CDS Project with:

```bash
cdsctl project integration import PROJECT_KEY project-configuration.yml
```

Then, as a standard user, you can add a [Kafka Hook]({{<relref "/docs/concepts/workflow/hooks/kafka-hook.md">}}) on your workflow.

### Create a Public Kafka Integration for whole CDS Projects

You can also add a Kafka Integration with cdsctl. As a CDS Administrator,
this allows you to propose a Public Kafka Integration, available on all CDS Projects.

Create a file `public-configuration.yml`:

```yml
name: your-kafka-integration
event: true
public: true
public_configurations:
  name-of-integration:
    "broker url":
      type: string
      value: "n1.o1.your-broker:9093,n2.o1.n1.o1.your-broker:9093,n3.o1.n1.o1.your-broker:9093"
    "topic":
      type: string
      value: "your-topic.events"
    "username":
      type: string
      value: "your-topic.cds-reader"
    "password":
      type: password
      value: xxxxxxxx
```

Import the integration with :

```bash
cdsctl admin integration-model import public-configuration.yml
```

Then, as a standard user, you can add a [Kafka Hook]({{<relref "/docs/concepts/workflow/hooks/kafka-hook.md">}}) on your workflow.

### One Integration, two use case

You can use an integration kafka for two use cases: [Event]({{< relref "/docs/integrations/kafka/kafka_events.md">}}) and [Hooks]({{< relref "/docs/integrations/kafka/kafka_hooks.md">}}). Example of file `public-configuration.yml`:

```yml
name: your-kafka-integration
event: true
hook: true
public: true
```

### SASL / TLS

For testing purpose only, you can disable TLS / SASL with the `disableTLS` and `disableSASL` attributes.

Example of Public Integration with TLS and SASL disabled:

```yml
name: myPublicKafkaIntegration
event: true
public: true
public_configurations:
  myPublicKafkaIntegration:
    "broker url":
      type: string
      value: "localhost:9092"
    "topic":
      type: string
      value: "test.eventspublic"
    "disableTLS":
      type: string
      value: "true"
    "disableSASL":
      type: string
      value: "true"
    version:
      value: "2.1.1"
      type: string
```

### Kafka Version

If the attribute version could be not defined, default value is `0.10.0.1`
