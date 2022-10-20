---
title: "Application configuration file"
weight: 3
notoc: true
card: 
  name: concept_workflow
  weight: 3
---

An **Application** allows you to enable some features of CDS within a workflow:

* Variables
* Keys
* VCS
* Deployment

The application configuration format is as following:

```yaml
version: v1.0
name: my-application

variables:
  artifact.name:
    type: string
    value: my-application
  docker.image:
    value: my-org/my-application
  docker.registry: 
    value: hub.docker.com

keys:
  app-mySSHKey:
    type: ssh

repo: my-org/my-application
vcs_server: github
vcs_connection_type: ssh
vcs_ssh_key: app-mySSHKey

deployments:
  my-kubernetes-cluster:
    timeout:
      value: 180
    namespace:
      my-namespace
    helm_chart:
      value: deploy/helm/
    helm_values:
      type: deploy/helm/values.yaml
    helm_version:
      type: 2.12.2
```

## Variables
Define application scoped variables as a dictionary. Each Variable must have a `type` and a `value`. You will be able to play with those variables using `{{.cds.app.my-variable}}` and `CDS_APP_MY_VARIABLE`. The recognized types are `string`, `password`, `text`, `boolean` or `number`. By default the type is `string`.

To be able to write secret in the CDS yaml files, you have to encrypt data within your CDS project.
```bash
  $ cdsctl encrypt MYPROJECT my-data my-super-secret-value
  my-data: 01234567890987654321
```
The command returns the value: `01234567890987654321`. You can use this value in a configuration file.

## Keys
Keys managed in CDS in are from two types: `SSH` and `PGP`. Let's import an application with a SSH Key. Those key could be important to manage your Git repositories.
```bash
➜  ~ cat app.yml
name: myapp
keys:
  app-mysshkey:
    type: ssh

➜  ~ cdsctl application import MYPROJ app.yml
Application myapp successfully created
SSH key app-mysshkey created on application myapp
```

CDS has just created a new SSH keypair on its own. To be able to keep this keypair for the future, you can gather an identifier by running an export.
```bash
➜  ~ cdsctl application export FSAMIN myapp
version: v1.0
name: myapp
keys:
  app-mysshkey:
    type: ssh
    value: 19a2ca3271854c3183aabb4af9df05e1
```

Note that each time you want to import the application and *keep* the keypair as it, you *must* provide the exported value.

If you want to keep your application in your git repository and let CDS configure and reconfigure the application automatically, we suggest to use the `regen` option. With this option CDS will generate the SSH keypair if it doesn't exist, and won't touch it on each import.
```yaml
name: myapp
keys:
  app-mysshkey:
    type: ssh
    regen: false
```

## VCS

To be able to link an application to a VCS, you must have at least one [repository manager]({{< relref "../../integrations" >}}) properly configured on your CDS instance.
Each application in CDS can be linked to one repository on a repository manager. 

Defining your VCS setup on an application will allow to benefit for the deep integration of CDS and the Repository Manager (such as GitHub). So you would be able to setup webhooks, browse through commits and publish your releases easily.

| Setting               | Definition                                                                                   |
| -------------         |----------------------------------------------------------------------------------------------|
| vcs_server            | Set the name of the repository manager on which on repository is hosted                      |
| repo                  | The fullname of the repositiry i.e `myorg/myrepo`                                            |
| vcs_connection_type   | Define the way you would like to checkout the code. Allowed values are `ssh` or `https`      |
| vcs_ssh_key           | If you set `vcs_connection_type = ssh`, choose the ssh key you want to use to git clone      |
| vcs_user              | If you set `vcs_connection_type = http`, set the HTTP Username                               |
| vcs_password          | If you set `vcs_connection_type = http`, set the HTTP Password                               |
| vcs_pgp_key           | If you want to commit and sign, you can choose here a PGP Key                                |

Please note that you can use key at `project` or `application` level. Default `vcs_connection_type` is `https`. If your repository is public, you can omit `vcs_connection_type`, `vcs_user` and `vcs_password`.

### Example

```yaml
version: v1.0
name: myapp
repo: myorg/myapp
vcs_server: github
vcs_connection_type: ssh
vcs_ssh_key: proj-ssh-key
vcs_pgp_key: proj-pgp-key
```

Now with this setup you will be able to use the actions [CheckoutApplication]({{< relref "../../actions/builtin-checkoutapplication/" >}}) and [Release]({{< relref "../../actions/builtin-releasevcs/" >}}) in your pipelines.

## Deployment

In this section, you can define the setup to deploy your application on a platform. To be able to setup it, you must have at least one [integration supporting deployment]({{< relref "../../integrations" >}}) properly configured on your CDS instance.

The `deployments` section is the list of the settings you can use to deploy to several platforms. For instance, if you want to be able to be deploy the same application, from the same helm chart with subtle changes in variables, depending on the cluster, you can set the following configuration.

```yaml
version: v1.0
name: myapp

deployments:

  my-kubernetes-cluster-A:
    namespace:
      my-namespace-A
    helm_chart:
      value: deploy/helm/
    helm_values:
      type: deploy/helm/values-cluster-A.yaml
    helm_version:
      type: 2.12.2

  my-kubernetes-cluster-B:
    namespace:
      my-namespace-B
    helm_chart:
      value: deploy/helm/
    helm_values:
      type: deploy/helm/values-cluster-B.yaml
    helm_version:
      type: 2.12.2
```

The list of the available deployment platform is available from the Web UI on the `project / integration` section, or with the command `cdsctl project integration list`

```bash
➜  ~ cdsctl project integration list MYPROJ
+-----------------------------+
|            NAME             |
+-----------------------------+
| my-kubernetes-cluster-A     |
| my-kubernetes-cluster-B     |
+-----------------------------+
```

The settings depend on the integration. Please refer to the [integration documentation]({{< relref "../../integrations" >}}).

Now you are ready to use the [DeployApplication]({{< relref "../../actions/builtin-deployapplication/" >}}) action in your pipelines.
