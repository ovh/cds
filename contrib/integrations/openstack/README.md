# How to import a openstack configuration

A Openstack Integration use the builtin integration model named **Openstack**.

You can set an openstack configuration on a CDS Project.

If you want to set a global openstack configuration, available on all CDS Projects, you 
have just to set the attribute **public** to `true` in the openstack.yml file.

```bash

# 1 - edit the openstack.yml file

# 2 - run 
$ cdsctl project integration import openstack.yml

```