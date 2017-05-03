+++
title = "Write a Plugin"
chapter = true

[menu.main]
parent = "advanced"
identifier = "advanced-write_plugin"
weight = 3

+++

A CDS worker executes job, and job is composed of steps.

A step is :

* a builtin action, as GitClone, etc... [read more]({{< relref "builtin.md" >}})
* a user action [read more]({{< relref "user-actions.md" >}})
* a Plugin Action

A Plugin is a Golang Binary.

Take a look at https://github.com/ovh/cds/tree/master/sdk/plugin/dummy/dummy_plugin.go

Contribute on https://github.com/ovh/cds/tree/master/contrib/plugins
