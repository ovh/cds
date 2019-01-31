# How to import a openstack configuration

This action can be done only by CDS Administrator.

```bash

# 1 - edit the openstack.yml file

# 2 - run 
$ cdsctl admin integration-model import openstack.yml

# you can see the import with the command:
$ cdsctl admin integration-model list
$ cdsctl admin integration-model show Openstack

# 3 - import plugin
$ cd plugin-openstack-storage
# notice that the Mafefile use the cdsctl command line
$ make build
$ make publish
```