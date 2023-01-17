---
title: "[Experimental] Ascode Worker Model"
weight: 3
---

# Description

Worker model can be defined ascode. That's mean the definition of the worker model will be in a git repository, and each time you will push something, CDS will update it.

# Prerequisite
* You must use the new CDS permission system [RBAC]({{< relref "/docs/concepts/rbac" >}})

# Files 

To be detected by CDS, your worker model files must be in this directory `.cds/worker-models/` 

<span style="color:red">*</span>: mandatory field

## Docker

```yaml
name: my-worker-model-name
description: my description
type: docker
spec:
  image: ns/myworkermodel:1.0
  registry: myregistry.org
  username: foo
  password: bar
  cmd: curl {{.API}}/download/worker/linux/$(uname -m) -o worker && chmod +x worker && exec ./worker
  shell: sh -c
  envs:
    myvar: myvalue
```

Fields:

* <span style="color:red">*</span>`name`: Name of the worker model
* `description`: Description of the worker model
* <span style="color:red">*</span>`type`: Type of worker model
* <span style="color:red">*</span>`spec.image`: Docker image name
* `spec.registry`: Docker registry
* `spec.username`: Docker registry username
* `spec.password`: Docker registry password. <b>The field must be encrypted with [cdsctl]({{< relref "/docs/components/cdsctl/encrypt/_index.md" >}})</b>
* <span style="color:red">*</span>`spec.cmd`: Command that start the worker
* <span style="color:red">*</span>`spec.shell`: Shell use to run the command
* `spec.envs`: Additional environment variables

## Openstack

```yaml
name: my-worker-model-name
description: my description
type: openstack
spec: 
  image: Ubuntu
  flavor: "b2-4"
  cmd: curl {{.API}}/download/worker/linux/$(uname -m) -o worker && chmod +x worker && exec ./worker
  pre_cmd:
  post_cmd: shutdown -h
```

Fields:

* <span style="color:red">*</span>`name`: Name of the worker model
* `description`: Description of the worker model
* <span style="color:red">*</span>`type`: Type of worker model
* <span style="color:red">*</span>`spec.image`: Openstack image name
* <span style="color:red">*</span>`spec.flavor`: Openstack flavor to use
* <span style="color:red">*</span>`spec.cmd`: Command that start the worker
* `spec.pre_cmd`: Command executed before running the worker
* <span style="color:red">*</span>`spec.post_cmd`:Command executed after your job to stop the VM


## vSphere

```yaml
name: my-worker-model-name
description: my description
type: vsphere
spec:
  image: Ubuntu
  username: foo
  password: bar
  cmd: curl {{.API}}/download/worker/linux/$(uname -m) -o worker && chmod +x worker && exec ./worker
  pre_cmd:
  post_cmd: shutdown -h
```

Fields:

* <span style="color:red">*</span>`name`: Name of the worker model
* `description`: Description of the worker model
* <span style="color:red">*</span>`type`: Type of worker model
* <span style="color:red">*</span>`spec.image`: vSphere template name
* <span style="color:red">*</span>`spec.username`: username to use to connect to the VM
* <span style="color:red">*</span>`spec.password`: password to use to connect to the VM. <b>The field must be encrypted with [cdsctl]({{< relref "/docs/components/cdsctl/encrypt/_index.md" >}})</b>
* <span style="color:red">*</span>`spec.cmd`: Command that start the worker
* `spec.pre_cmd`: Command executed before running the worker
* <span style="color:red">*</span>`spec.post_cmd`: Command executed after your job to stop the VM
