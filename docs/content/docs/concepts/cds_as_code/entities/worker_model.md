---
title: "Worker Model"
weight: 3
---

# Description

Worker model can be defined ascode. That's mean the definition of the worker model will be in a git repository, and each time you will push something, CDS will update it.

# As Code directory

A workflow is described directly on your repository inside the directory `.cds/worker-models`

# Permission

To be able to manage a workflow you will need the permission `manage-worker-model` on your project

# Files 

<span style="color:red">*</span>: mandatory field

## Docker

```yaml
name: my-worker-model-name
description: my description
osarch: linux/amd64
type: docker
spec:
  image: myregistry.org/ns/myworkermodel:1.0
  username: foo
  password: bar
  envs:
    myvar: myvalue
```

Fields:

* <span style="color:red">*</span>`name`: Name of the worker model
* `description`: Description of the worker model
* <span style="color:red">*</span>`type`: Type of worker model
* <span style="color:red">*</span>`osarch`: OS and architecture of the model
* <span style="color:red">*</span>`spec.image`: Docker image name
* `spec.username`: Docker registry username
* `spec.password`: Docker registry password. <b>The field must be encrypted with [cdsctl]({{< relref "/docs/components/cdsctl/encrypt/_index.md" >}})</b>
* `spec.envs`: Additional environment variables

## Openstack

```yaml
name: my-worker-model-name
description: my description
type: openstack
osarch: linux/amd64
spec: 
  image: Ubuntu
```

Fields:

* <span style="color:red">*</span>`name`: Name of the worker model
* `description`: Description of the worker model
* <span style="color:red">*</span>`osarch`: OS and architecture of the model
* <span style="color:red">*</span>`type`: Type of worker model
* <span style="color:red">*</span>`spec.image`: Openstack image name


## vSphere

```yaml
name: my-worker-model-name
description: my description
osarch: linux/amd64
type: vsphere
spec:
  image: Ubuntu
  username: foo
  password: bar
```

Fields:

* <span style="color:red">*</span>`name`: Name of the worker model
* `description`: Description of the worker model
* <span style="color:red">*</span>`osarch`: OS and architecture of the model
* <span style="color:red">*</span>`type`: Type of worker model
* <span style="color:red">*</span>`spec.image`: vSphere template name
* <span style="color:red">*</span>`spec.username`: username to use to connect to the VM
* <span style="color:red">*</span>`spec.password`: password to use to connect to the VM. <b>The field must be encrypted with [cdsctl]({{< relref "/docs/components/cdsctl/encrypt/_index.md" >}})</b>
