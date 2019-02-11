# CDS

[CDS](https://github.com/ovh/cds) is a pipeline based Continuous Delivery Service written in Go(lang).
Documentation is available at https://ovh.github.io/cds/

## TL;DR;

```console
$ cd contrib/helm/cds;
helm dependency update;
helm install .
```


## FUTURE

When CDS helm chart will be released you'll be able to install with
```console
$ helm install stable/cds
```

## Introduction

This chart bootstraps a [CDS](https://github.com/ovh/cds) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

It starts a PostgreSQL server, a Redis server and an Elasticsearch server using the helm built-in dependency system. It also starts all µServices that CDS needed to use all CDS features :

+ Hatchery over Kubernetes (Only over the same kubernetes cluster and same namespace)
+ VCS service
+ Hooks service
+ Elasticsearch service
+ Repositories service

## Prerequisites

- Kubernetes 1.4+ with Beta APIs enabled
- PV provisioner support in the underlying infrastructure
- Kubernetes config file (`kubeconfig.yaml`) located at this path. (For minikube it's often located `~/.kube/config`)

## Installing the Chart

To install the chart with the release name `my-release`:

```console
$ helm install --name my-release stable/cds
```

The command deploys CDS on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

Please refer to default values.yaml and source code
Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
$ helm install --name my-release .
```

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
$ helm install --name my-release -f values.yaml .
```

> **Tip**: You can use the default [values.yaml](values.yaml)

+ If you use a Kubernetes provider without LoadBalancer ability you just have to set your `ui.serviceType` to `ClusterIP` and set `ingress.enabled` to `true` with the right `ingress.hostname` and `ingress.port` (for example: `helm install --kubeconfig kubeconfig.yml --name my-release -f values.yaml --set ui.serviceType=ClusterIP --set ingress.enabled=true --set ingress.hostname=cds.MY_NODES_URL --set ingress.port=32080 .`).

+ If you use a minikube you have to set `ui.serviceType` to `ClusterIP`.

+ If you use a Kubernetes as GKE, EKS or if your cloud provider provide you an available LoadBalancer you just have to set `ui.serviceType` to `LoadBalancer`.

+ If your `kubeconfig.yaml` is not located in this directory you can set path in `values.yaml` or launch with `--set kubernetesConfigFile=myPathTo/kubeconfig.yaml`.

## Image

The `image` parameter allows specifying which image will be pulled for the chart.

## Persistence

By default, cds api artifact directory is created as default PersistentVolumeClaim
### Existing PersistentVolumeClaim

1. Create the PersistentVolume
1. Create the PersistentVolumeClaim
1. Install the chart

```bash
$ helm install --name my-release --set cds.existingClaim=PVC_NAME .
```
