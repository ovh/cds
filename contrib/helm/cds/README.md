# CDS

[CSD](https://github.com/ovh/cds) is a pipeline based Continuous Delivery Service written in Go(lang).
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

It also starts a PostgreSQL server and a Redis server using the helm built-in dependency system.

## Prerequisites

- Kubernetes 1.4+ with Beta APIs enabled
- PV provisioner support in the underlying infrastructure

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

## Configuration (TODO)

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

## Image

The `image` parameter allows specifying which image will be pulled for the chart.

### Private registry

If you configure the `image` value to one in a private registry, you will need to [specify an image pull secret](https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod).

1. Manually create image pull secret(s) in the namespace. See [this YAML example reference](https://kubernetes.io/docs/concepts/containers/images/#creating-a-secret-with-a-docker-config). Consult your image registry's documentation about getting the appropriate secret.
2. Note that the `imagePullSecrets` configuration value cannot currently be passed to helm using the `--set` parameter, so you must supply these using a `values.yaml` file, such as:

```yaml
imagePullSecrets:
  - name: SECRET_NAME
```

1. Install the chart

```console
cd contrib/helm/cds
helm install --name my-release -f values.yaml .
```

## Persistence

By default, cds api artifact directory is created as default PersistentVolumeClaim
### Existing PersistentVolumeClaim

1. Create the PersistentVolume
1. Create the PersistentVolumeClaim
1. Install the chart

```bash
$ helm install --name my-release --set cds.existingClaim=PVC_NAME .
```

### Host path

#### System compatibility

- The local filesystem accessibility to a container in a pod with `hostPath` has been tested on OSX/MacOS with xhyve, and Linux with VirtualBox.
- Windows has not been tested with the supported VM drivers. Minikube does however officially support [Mounting Host Folders](https://github.com/kubernetes/minikube/blob/master/docs/host_folder_mount.md) per pod. Or you may manually sync your container whenever host files are changed with tools like [docker-sync](https://github.com/EugenMayer/docker-sync) or [docker-bg-sync](https://github.com/cweagans/docker-bg-sync).

#### Mounting steps

1. The specified `hostPath` directory must already exist (create one if it does not).
1. Install the chart

    ```bash
    $ helm install --name my-release --set cds.hostPath=/PATH/TO/HOST/MOUNT cds
    ```