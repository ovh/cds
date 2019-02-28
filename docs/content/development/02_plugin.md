+++
title = "Develop a plugin"
weight = 2

+++

A CDS worker executes job, and job is composed of steps.

A step is:

* a builtin action, as GitClone, etc... [read more]({{< relref "workflows/pipelines/actions/builtin/_index.md" >}})
* a user action [read more]({{< relref "workflows/pipelines/actions/user-actions/_index.md" >}})
* a plugin action

A Plugin is simply an executable which expose a GRPC server corresponding to the right [proto file](https://github.com/ovh/cds/tree/master/sdk/grpcplugin/actionplugin/actionplugin.proto). You can use the programming language of your choice. The CDS worker will simply query the GRPC server of the plugin.

In order to communicate with a CDS worker, a plugin MUST fill the following requirements:

+ Expose a GRPC server
+ Implement methods and messages coming from this [proto file](https://github.com/ovh/cds/tree/master/sdk/grpcplugin/actionplugin/actionplugin.proto)
+ Display this message at the launch of your plugin XXX is ready to accept new connection where XXX is your ip address with port or your Unix socket (example: `127.0.0.1:55939 is ready to accept new connection` or for a Unix socket `XXX.sock is ready to accept new connection`). Note that your plugin can use any Unix socket or tcp port as long as it informs the worker using the log line above.

More resources that may help you in developing a CDS plugin are available: [SDK in this directory](https://github.com/ovh/cds/tree/master/sdk/grpcplugin/actionplugin) with some examples [here](https://github.com/ovh/cds/tree/master/contrib/grpcplugins/action/examples).

Contribute on https://github.com/ovh/cds/tree/master/contrib/grpcplugin/action
