---
title: vSphere
main_menu: true
card: 
  name: compute
---


CDS build using VMware vSphere infrastructure to spawn each CDS Workers inside dedicated VM.

## Pre-requisites

This hatchery spawns VM which obtains IP from DHCP. So first you have to create a DHCP server on your host with NAT if you want to access to the internet. In order to create you have multiple possibilities like create your own VM with a DHCP server configured or if you are comfortable with the VMware tools you can use the [NSX system](https://www.vmware.com/products/nsx.html). This system will create DHCP gateway for you.

Also we recommend you to create a VM base that the hatchery will use to linked clone your new VM to execute your jobs. For example in our case we create different VM base with a minimal debian installed in different versions. In order to save your host resources we advice you to turn these VMs off.

## Start vSphere hatchery

Generate a token:

```bash
$ cdsctl consumer new me \
--scopes=Hatchery,RunExecution,Service,WorkerModel \
--name="hatchery.vsphere" \
--description="Consumer token for vsphere hatchery" \
--groups="" \
--no-interactive

Builtin consumer successfully created, use the following token to sign in:
xxxxxxxx.xxxxxxx.4Bd9XJMIWrfe8Lwb-Au68TKUqflPorY2Fmcuw5vIoUs5gQyCLuxxxxxxxxxxxxxx
```

Edit the section `hatchery.vsphere` in the [CDS Configuration]({{< relref "/hosting/configuration.md">}}) file.
The token have to be set on the key `hatchery.vsphere.commonConfiguration.api.http.token`.

This hatchery will now start worker of model 'vsphere' on vSphere infrastructure.

## Setup a worker model

See [Tutorial]({{< relref "/docs/tutorials/worker_model-vsphere.md" >}})
