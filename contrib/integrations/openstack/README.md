# How to import a OpenStack configuration

An OpenStack Integration use the builtin integration model named **Openstack**.

You can set an OpenStack configuration on a CDS Project.

If you want to set a global OpenStack configuration, available on all CDS Projects, you 
have just to set the attribute **public** to `true` in the openstack.yml file.

```bash

# 1 - edit the openstack.yml file

# 2 - run 
$ cdsctl project integration import openstack.yml

```