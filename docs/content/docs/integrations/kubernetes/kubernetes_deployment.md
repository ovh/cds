---
title: Kubernetes Deployment
main_menu: true
---


The Kubernetes Deployment Integration is a Self-Service integration that can be configured on a CDS Project.

This integration enables the [DeployApplication]({{<relref "/docs/actions/deployapplication.md">}}) action

## Import Kubernetes Plugin


## Configure with WebUI

You can add a Kubernetes Integration on your CDS Project.

![Integration](../images/kubernetes-integration-webui.png)

## Import a Kubernetes Integration on your CDS Project

Create a file project-configuration.yml:

```yml
TODO
```

Import the integration on your CDS Project with:

```bash
cdsctl project integration import PROJECT_KEY project-configuration.yml
```

### Create a Public Kubernetes Integration for whole CDS Projects

You can also add a Kubernetes Integration with cdsctl. As a CDS Administrator,
this allows you to propose a Public Kubernetes Integration, available on all CDS Projects.

Create a file public-configuration.yml:

```yml
TODO
```

Import the integration with :

```bash
cdsctl admin integration-model import public-configuration.yml
```

## Use DeployApplication Action

Then, as a standard user, you can use the [DeployApplication]({{<relref "/docs/actions/deployapplication.md">}}) action in a Job.
