+++
title = "Hatchery Vsphere"
weight = 3

[menu.main]
parent = "hatcheries"
identifier = "hatchery_vsphere"

+++

CDS build using VMWare Vsphere infrastructure to spawn each CDS Workers inside dedicated VM.

## Pre-requisites

This hatchery spawns VM which obtains IP from DHCP. So first you have to create a DHCP server on your host with NAT if you want to access to the internet. In order to create you have multiple possibilities like create your own VM with a DHCP server configured or if you are comfortable with the VMWare tools you can use the [NSX system](https://www.vmware.com/products/nsx.html). This system will create DHCP gateway for you.

Also we recommend you to create a VM base that the hatchery will use to linked clone your new VM to execute your jobs. For example in our case we create different VM base with a minimal debian installed in different versions. In order to save your host resources we advice you to turn these VMs off.

//Example of network infrastructure

## Start Vsphere hatchery

Generate a token for group:

```bash
$ cds generate  token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Then start hatchery:

```bash
VSPHERE-USER=<user> VSPHERE-ENDPOINT="pcc-11-222-333-444.ovh.com" VSPHERE-PASSWORD=<password> VSPHERE-DATACENTER=<datacenter> VSPHERE-DATASTORE=<datastore> VSPHERE-NETWORK=<vmNetwork> VSPHERE-ETHERNET-CARD=<ethernet card> hatchery vsphere \
        --api=https://api.domain \
        --max-worker=10 \
        --provision=1 \
        --token=fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
# VSPHERE-ETHERNET-CARD aren't mandatory
```

This hatchery will now start worker of model 'vsphere' on Vsphere infrastructure.

## Setup a worker model

See [Tutorial]({{< relref "tutorials.worker-model-vsphere.md" >}})
