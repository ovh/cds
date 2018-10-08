+++
title = "Write a Plugin"
weight = 10

+++

A CDS worker executes job, and job is composed of steps.

A step is :

* a builtin action, as GitClone, etc... [read more]({{< relref "workflows/pipelines/actions/builtin/_index.md" >}})
* a user action [read more]({{< relref "workflows/pipelines/actions/user-actions/_index.md" >}})
* a plugin action

A Plugin is simply an executable which expose a GRPC server corresponding to the right [proto file](https://github.com/ovh/cds/tree/master/sdk/grpcplugin/actionplugin/actionplugin.proto). You can use any languages. The CDS worker will simply query in GRPC the plugin (which is the GRPC server).

So a CDS plugin have these requirements in order to communicate with worker:

+ Must expose a GRPC server
+ Must implement methods and messages coming from this [proto file](https://github.com/ovh/cds/tree/master/sdk/grpcplugin/actionplugin/actionplugin.proto)
+ And last but not least at the launch of your plugin you can use random unix socket or random tcp port but in order to inform worker what is your address you have to display this log at the launch of your plugin `XXX is ready to accept new connection` where `XXX` is your ip address with port or your unix socket (example: `127.0.0.1:55939 is ready to accept new connection`).

We give you some resources to help you develop a CDS plugin like [SDK in this directory](https://github.com/ovh/cds/tree/master/sdk/grpcplugin/actionplugin) and examples [here](https://github.com/ovh/cds/tree/master/contrib/grpcplugins/action/examples).

Contribute on https://github.com/ovh/cds/tree/master/contrib/grpcplugin/action
