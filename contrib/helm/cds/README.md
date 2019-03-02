# CDS

[CDS](https://github.com/ovh/cds) is an Enterprise-Grade Continuous Delivery & DevOps Automation Platform.

Documentation is available at https://ovh.github.io/cds/

## TL;DR;

```console
cd contrib/helm/cds
# To let CDS spawn workers on your kubernetes cluster you need to copy your kubeconfig.yaml in the current directory
cp yourPathToKubeconfig.yaml kubeconfig.yaml
helm dependency update
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
# Inside of cds/contrib/helm/cds
# To let CDS spawn workers on your kubernetes cluster you need to copy your kubeconfig.yaml in the current directory
cp yourPathToKubeconfig.yaml kubeconfig.yaml
helm dependency update
helm install --name my-cds . 
```

The command deploys CDS on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

This will install all the CDS services:

```console
$ helm install .
NAME:   my-cds
LAST DEPLOYED: Thu Feb  7 12:07:24 2019
NAMESPACE: cds
STATUS: DEPLOYED

RESOURCES:
==> v1/PersistentVolumeClaim
NAME                     AGE
my-cds-postgresql        59s
my-cds-cds-api           59s
my-cds-cds-repositories  59s

==> v1/Service
my-cds-elasticsearch-client     59s
my-cds-elasticsearch-discovery  59s
my-cds-postgresql               59s
my-cds-redis-master             59s
my-cds-redis-slave              59s
my-cds-cds-api                  59s
my-cds-cds-elasticsearch        59s
my-cds-cds-hatchery-k8s         59s
my-cds-cds-hooks                58s
my-cds-cds-repositories         58s
my-cds-cds-ui                   58s
my-cds-cds-vcs                  58s

==> v1beta1/Deployment
my-cds-elasticsearch-client  58s
my-cds-postgresql            58s
my-cds-redis-slave           58s
my-cds-cds-api               58s
my-cds-cds-elasticsearch     58s
my-cds-cds-hatchery-k8s      58s
my-cds-cds-hooks             58s
my-cds-cds-repositories      58s
my-cds-cds-ui                58s
my-cds-cds-vcs               58s

==> v1beta1/StatefulSet
my-cds-elasticsearch-data    58s
my-cds-elasticsearch-master  58s

==> v1beta2/StatefulSet
my-cds-redis-master  58s

==> v1/Pod(related)

NAME                                          READY  STATUS             RESTARTS  AGE
my-cds-elasticsearch-client-5797cb88cd-bdtx4  0/1    Running            0         58s
my-cds-postgresql-554cff77b5-tb2c9            1/1    Running            0         58s
my-cds-redis-slave-544478d54c-m2mg8           0/1    Running            0         58s
my-cds-cds-api-799f8c7c55-lf4zt               0/1    CrashLoopBackOff   1         58s
my-cds-cds-elasticsearch-7d666db5bf-8gngf     1/1    Running            1         58s
my-cds-cds-hatchery-k8s-554bccb9d5-ftsrj      1/1    Running            1         58s
my-cds-cds-hooks-765d94b886-lpsss             1/1    Running            1         58s
my-cds-cds-repositories-689b5c755f-kk5t4      1/1    Running            1         58s
my-cds-cds-ui-74b78df797-76kzs                1/1    Running            0         58s
my-cds-cds-vcs-6b58d46766-4pj24               1/1    Running            1         58s
my-cds-elasticsearch-data-0                   0/1    PodInitializing    0         58s
my-cds-elasticsearch-master-0                 0/1    Running            0         58s
my-cds-redis-master-0                         0/1    ContainerCreating  0         57s

==> v1/Secret

NAME               AGE
my-cds-postgresql  59s
my-cds-cds         59s

==> v1/ConfigMap
my-cds-elasticsearch  59s
my-cds-postgresql     59s


NOTES:

************************************************************************
*** PLEASE BE PATIENT: CDS may take a few minutes to install         ***
************************************************************************

1. Get the CDS URL:

  NOTE: It may take a few minutes for the LoadBalancer IP to be available.
        Watch the status with: 'kubectl get svc --namespace default -w my-cds-cds-ui'

  export SERVICE_IP=$(kubectl get svc --namespace default my-cds-cds-ui -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
  echo http://$SERVICE_IP/

2. Create an account from the ui using your web browser

And check in the log of your api server to get registration URL :

  export CDS_API_POD_NAME=$(kubectl get pods --namespace default -l "app=my-cds-cds-api" -o jsonpath="{.items[0].metadata.name}")
  kubectl logs -f --namespace default $CDS_API_POD_NAME | grep 'account/verify'

```

+ Create the first CDS User

This log asks you to open `http://$SERVICE_IP/` in your browser to create the first user. 

After create the first account and as there is no SMTP configured, you have to check the CDS Logs to retrieve the URL to validate.

With the previous example log, the command to run is:

```console
export CDS_API_POD_NAME=$(kubectl get pods --namespace default -l "app=my-cds-cds-api" -o jsonpath="{.items[0].metadata.name}")
kubectl logs -f --namespace default $CDS_API_POD_NAME | grep 'account/verify'
```

After registration on UI, keep the password displayed, we will use it in next step. 

The first user created on CDS is a CDS Administrator.

In order to have all that you need to run your first job you need to add a first [worker model](https://ovh.github.io/cds/manual/concepts/worker-model/). It's the perfect use case to use the [CDS Command Line](https://ovh.github.io/cds/manual/components/cdsctl/) named `cdsctl`.

+ Dowload cdsctl

```console
# on a Linux workstation:
$ curl http://$SERVICE_IP/cdsapi/download/cdsctl/linux/amd64 -o cdsctl
# on a osX workstation, it's curl http://$SERVICE_IP/cdsapi/download/cdsctl/darwin/amd64 -o cdsctl
$ chmod +x cdsctl
```

*please note that the version linux/amd64, darwin/amd64 and windows/amd64 use libsecret / keychain to store the CDS Password.
If you don't want to use the keychain, you can select the version i386*


+ Login with cdsctl
```console
$ ./cdsctl login --api-url http://$SERVICE_IP/cdsapi -u yourusername
CDS API URL: http://$SERVICE_IP/cdsapi
Username: yourusername
Password:
          You didn't specify config file location; /Users/yourhome/.cdsrc will be used.
Login successful
```

+ Create a worker model

```console
./cdsctl worker model import https://raw.githubusercontent.com/ovh/cds/master/contrib/worker-models/go-official-1.11.4-stretch.yml
```

In this case, it's a worker model based on the official golang docker image coming from docker hub. 
The hatchery will register the worker model before it can be used. You can check the 
registration information on the ui: Settings -> Worker models -> go-official-1.11.4-stretch -> flag "Need registration".

+ Import a workflow template

```console
$ ./cdsctl template push https://raw.githubusercontent.com/ovh/cds/master/contrib/workflow-templates/demo-workflow-hello-world/demo-workflow-hello-world.yml
Workflow template shared.infra/demo-workflow-hello-world has been created
Template successfully pushed !
```

+ Create a project with reference key DEMO and name FirstProject, then create your first workflow with a template:

```console
$ ./cdsctl project create DEMO FirstProject
$ ./cdsctl workflow applyTemplate
? Found one CDS project DEMO - FirstProject. Is it correct? Yes
? Choose the CDS template to apply: Demo workflow hello world (shared.infra/demo-workflow-hello-world)
? Give a valid name for the new generated workflow MyFirstWorkflow
? Push the generated workflow to the DEMO project Yes
Application MyFirstWorkflow successfully created
Application variables for MyFirstWorkflow are successfully created
Permission applied to group FirstProject to application MyFirstWorkflow
Environment MyFirstWorkflow-prod successfully created
Environment MyFirstWorkflow-dev successfully created
Environment MyFirstWorkflow-preprod successfully created
Pipeline build-1 successfully created
Pipeline deploy-1 successfully created
Pipeline it-1 successfully created
Workflow MyFirstWorkflow has been created
Workflow successfully pushed !
.cds/MyFirstWorkflow.yml
.cds/build-1.pip.yml
.cds/deploy-1.pip.yml
.cds/it-1.pip.yml
.cds/MyFirstWorkflow.app.yml
.cds/MyFirstWorkflow-dev.env.yml
.cds/MyFirstWorkflow-preprod.env.yml
.cds/MyFirstWorkflow-prod.env.yml
```

+ On CDS all actions could be done with UI, CLI or API. So you can go on your CDS UI to check your new workflow and run it.

For any further informations about CDS please check [official documentation](https://ovh.github.io/cds/).

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm delete my-release --purge
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

Please refer to default values.yaml and source code
Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
# Inside of cds/contrib/helm/cds
# To let CDS spawn workers on your kubernetes cluster you need to copy your kubeconfig.yaml in the current directory
cp yourPathToKubeconfig.yaml kubeconfig.yaml
helm dependency update
helm install .
```

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
$ helm install --name my-release -f values.yaml .
```

> **Tip**: You can use the default [values.yaml](values.yaml)

+ If you use a Kubernetes provider without LoadBalancer ability you just have to set your `ui.serviceType` to `ClusterIP` and set `ingress.enabled` to `true` with the right `ingress.hostname` and `ingress.port` (for example: `helm install --kubeconfig kubeconfig.yml --name my-release -f values.yaml --set ui.serviceType=ClusterIP --set ingress.enabled=true --set ingress.hostname=cds.MY_NODES_URL --set ingress.port=32080 .`).

+ If you use a minikube you have to set `ui.serviceType` to `ClusterIP`.

+ If you use a Kubernetes as GKE, EKS or if your cloud provider provide you an available LoadBalancer you just have to set `ui.serviceType` to `LoadBalancer`.

+ Your `kubeconfig.yaml` must be located in this directory.

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
