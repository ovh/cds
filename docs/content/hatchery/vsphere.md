+++
title = "Hatchery Vsphere"
weight = 3

[menu.main]
parent = "hatcheries"
identifier = "hatchery.vsphere"

+++

CDS build using VMWare Vsphere infrastructure to spawn each CDS Workers inside dedicated VM.

## Pre-requisites

This hatchery spawns VM which obtains IP from DHCP. So first you have to create a DHCP server on your host with NAT if you want to access to the internet. In order to create you have multiple possibilities like create your own VM with a DHCP server configured or if you are comfortable with the VMWare tools you can use the [NSX system](https://www.vmware.com/products/nsx.html). This system will create DHCP gateway for you.

Also we recommend you to create a VM base that the hatchery will use to linked clone your new VM to execute your jobs. For example in our case we create different VM base with a minimal debian installed in different versions. In order to save your host resources we advice you to turn these VMs off.

## Start Vsphere hatchery

Generate a token for group:

```bash
$ cds generate  token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Edit the CDS [configuration]({{< relref "hosting.configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated and the VSphere variables.

This hatchery will now start worker of model 'vsphere' on Vsphere infrastructure.

## Setup a worker model

See [Tutorial]({{< relref "tutorials.worker-model-vsphere.md" >}})
