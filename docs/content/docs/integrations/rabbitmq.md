---
title: RabbitMQ
main_menu: true
---

The RabbitMQ Integration is a Self-Service integration that can be configured on a CDS Project.

This integration enables the [RabbitMQ Hook feature]({{<relref "/docs/concepts/workflow/hooks/rabbitmq-hook.md">}}):

## Configure with WebUI

You can add a RabbitMQ Integration on your CDS Project.

![Platform](../images/rabbitmq-integration-webui.png)

## Configure with cdsctl

### Import a RabbitMQ Integration on your CDS Project

Create a file project-configuration.yaml:

```yml
project integration export DEMO your-rabbitmq-integration
name: my-rabbitmq-integration
model:
  name: RabbitMQ
  author: CDS
  identifier: github.com/ovh/cds/integration/builtin/rabbitmq
  hook: true
config:
  password:
    value: '**********'
    type: password
  uri:
    value: your-rabbit:5672
    type: string
  username:
    value: your-username
    type: string
```

Import the integration on your CDS Project with:

```bash
cdsctl project integration import PROJECT_KEY project-configuration.yaml
```

Then, as a standard user, you can add a [rabbitMQ Hook]({{<relref "/docs/concepts/workflow/hooks/rabbitmq-hook.md">}}) on your workflow.


### Create a Public RabbitMQ Integration for whole CDS Projects

You can also add a RabbitMQ Integration with cdsctl. As a CDS Administrator,
this allows you to propose a Public RabbitMQ Integration, available on all CDS Projects.

Create a file public-configuration.yaml:

```yml
name: your-rabbitmq-integration
hook: true
public: true
public_configurations:

incoming TODO YESNAULT
```

Import the integration with :

```bash
cdsctl admin integration-model import public-configuration.yaml
```

Then, as a standard user, you can add a [rabbitMQ Hook]({{<relref "/docs/concepts/workflow/hooks/rabbitmq-hook.md">}}) on your workflow.