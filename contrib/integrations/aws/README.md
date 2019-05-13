# How to import a Amazon AWS configuration

An AWS Integration use the builtin integration model named **AWS**.

You can set an AWS configuration on a CDS Project.

If you want to set a global AWS configuration, available on all CDS Projects, you 
have just to set the attribute **public** to `true` in the aws.yml file.

```bash

# 1 - edit the aws.yml file

# 2 - run 
$ cdsctl project integration import aws.yml

```