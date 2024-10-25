---
title: Kubernetes Deployment
main_menu: true
card: 
  name: deployment
---


The Kubernetes Deployment Integration is a Self-Service integration that can be configured on a CDS Project.

This integration enables the [DeployApplication]({{<relref "/docs/actions/builtin-deployapplication.md">}}) action

## Create Integration Model

Create a file `kubernetes-model-configuration.yml`:

```yml
name: Kubernetes
default_config:
  api_url:
    type: string
  ca_certificate:
    type: text
    description: Certificate Authority bundle (PEM format)
  token:
    type: password
deployment: true
deployment_default_config:
  timeout:
    type: string
    value: 180
    description: timeout in seconds for v2 or duration for v3 (ex: 3m)
  namespace:
    type: string
    value: default
    description: Kubernetes namespace in which you want to deploy your components (OPTIONAL)
  deployment_files:
    type: string
    description: Glob to yaml filepaths
  helm_chart:
    type: string
    description: "Keep empty if you don't use helm. Indicate a chart reference by chart reference himself (stable/mariadb), absolute URL (https://example.com/charts/nginx-1.2.3.tgz), path to a packaged chart (./nginx-1.2.3.tgz), path to an unpacked chart directory (./nginx) or even by your chart reference and repo URL (https://example.com/charts/ nginx)."
  helm_values:
    type: string
    description: specify helm values in a YAML file or a URL to configure/override your helm chart
  helm_version:
    type: string
    description: specify helm version to use (default: v2.12.2)
```

Import the integration with :

```bash
cdsctl admin integration-model import kubernetes-model-configuration.yml
```

## Import Kubernetes Plugin

The goal of this integration is to use the [DeployApplication]({{<relref "/docs/actions/builtin-deployapplication.md">}}) action in a Job.
This action use a dedicated plugin for each integration, you need to import the plugin-kubernetes.

You will find on CDS Release the plugin-kubernetes with associated yml file.

How to import the linux/amd64 version:

```bash
# download plugin-kubernetes-deployment.yml file
# download plugin-kubernetes-deployment-linux-amd64.yml file
# download plugin-kubernetes-deployment-linux-amd64 file
$ cdsctl admin plugins import plugin-kubernetes-deployment.yml
$ cdsctl admin plugins binary-add plugin-kubernetes-deployment plugin-kubernetes-deployment-linux-amd64.yml plugin-kubernetes-deployment-linux-amd64
```

If you can build and publish all os/arch:

```bash
$ cd $GOHOME/src/github.com/ovh
$ git clone git@github.com:ovh/cds.git
$ cd contrib/integrations/kubernetes/plugin-kubernetes
# make build will compile the plugin in all os and arch
# all binaries are under the dist/ directory
$ make build
# make publish create a yml file for each os/arch
# then call cdsctl to upload the plugin on your CDS Instance
$ make publish
```

## Configure with WebUI

You can add a Kubernetes Integration on your CDS Project.

![Integration](../images/kubernetes-integration-webui.png)

## Import a Kubernetes Integration on your CDS Project

Create a file `project-configuration.yml`:

```yml
name: myk8s
model:
  name: Kubernetes
  deployment: true
  default_config:
    api_url:
      value: ""
      type: string
    ca_certificate:
      value: ""
      type: text
      description: Certificate Authority bundle (PEM format)
    token:
      value: ""
      type: password
  deployment_default_config:
    deployment_files:
      value: ""
      type: string
      description: Glob to yaml filepaths
    helm_chart:
      value: ""
      type: string
      description: Keep empty if you don't use helm. Indicate a chart reference by
        chart reference himself (stable/mariadb), absolute URL (https://example.com/charts/nginx-1.2.3.tgz),
        path to a packaged chart (./nginx-1.2.3.tgz), path to an unpacked chart directory
        (./nginx) or even by your chart reference and repo URL (https://example.com/charts/
        nginx).
    helm_values:
      value: ""
      type: string
      description: specify helm values in a YAML file or a URL to configure/override
        your helm chart
    helm_version:
      value: ""
      type: string
      description: specify helm version to use (default: v2.12.2)
    namespace:
      value: default
      type: string
      description: Kubernetes namespace in which you want to deploy your components
        (OPTIONAL)
    timeout:
      value: "180"
      type: string
      description: timeout in seconds
config:
  api_url:
    value: https://your-k8s.localhost.local
    type: string
  ca_certificate:
    value: |-
      -----BEGIN CERTIFICATE-----
      XXX
      -----END CERTIFICATE-----
    type: text
    description: Certificate Authority bundle (PEM format)
  token:
    value: XXX
    type: string
```

Import the integration on your CDS Project with:

```bash
cdsctl project integration import PROJECT_KEY project-configuration.yml
```

### Create a Public Kubernetes Integration for whole CDS Projects

You can also add a Kubernetes Integration with cdsctl. As a CDS Administrator,
this allows you to propose a Public Kubernetes Integration, available on all CDS Projects.

Create a file `public-configuration.yml`:

```yml
name: Kubernetes-Public
hook: true
deployment: true
deployment_default_config:
  deployment_files:
    value: ""
    type: string
    description: Glob to yaml filepaths
  helm_chart:
    value: ""
    type: string
    description: Keep empty if you don't use helm. Indicate a chart reference by
      chart reference himself (stable/mariadb), absolute URL (https://example.com/charts/nginx-1.2.3.tgz),
      path to a packaged chart (./nginx-1.2.3.tgz), path to an unpacked chart directory
      (./nginx) or even by your chart reference and repo URL (https://example.com/charts/
      nginx).
  helm_values:
    value: ""
    type: string
    description: specify helm values in a YAML file or a URL to configure/override your helm chart
  helm_version:
    value: ""
    type: string
    description: specify helm version to use (default: v2.12.2)
  namespace:
    value: default
    type: string
    description: Kubernetes namespace in which you want to deploy your components (OPTIONAL)
  timeout:
    value: "180"
    type: string
    description: timeout in seconds
public_configurations:
  your-public-myk8s-integration:
    "api_url":
      value: https://your-k8s.localhost.local
      type: string
    "ca_certificate":
      value: |-
        -----BEGIN CERTIFICATE-----
        XXX
        -----END CERTIFICATE-----
      type: text
      description: Certificate Authority bundle (PEM format)
    "token":
      value: XXX
      type: string
```

Import the integration with :

```bash
cdsctl admin integration-model import public-configuration.yml
```

## Use DeployApplication Action

Add the deployment configuration on your application.

Parameters `deployment_files`, `helm_chart` and `helm_values` contain
path of the files in your CDS Job.

`contrib/helm/cds/` is the same as `{{.cds.workspace}}/contrib/helm/cds/`

![Add To Application](../images/link_kubernetes_to_application.png)

Then, as a standard user, you can use the [DeployApplication]({{<relref "/docs/actions/builtin-deployapplication.md">}}) action in a Job.
Before using this action, you probably want to use [CheckoutApplication]({{<relref "/docs/actions/builtin-checkoutapplication.md">}}) to git clone the kubernetes or helm files from your git repository.
