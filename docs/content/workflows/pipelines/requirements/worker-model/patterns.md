+++
title = "Worker Model patterns"
weight = 3

+++

On CDS you can use worker model on any platform, distribution and arch you want. To permit that you need to indicate what will be executed on the worker vm or image before the execution of the worker binary itself. For example before launching the worker binary you need to download that binary with curl or wget ore anything else, depends on which OS you are.


To not copy/paste each time the same script for each worker model on the same OS and also to give the ability for a no CDS administrator to create their own worker model you can create worker model patterns. A pattern is only created by administrator and is linked to a worker model type (openstack, docker, vsphere, ...). For example you can have different patterns for type vsphere and for different OS like windows, linux, mac os, ...

CDS give you some variables to use in your patterns: [click here]({{< relref "workflows/pipelines/requirements/worker-model/variables.md" >}}).


To create a pattern on the UI (only for CDS administrator) go to the navbar admin menu:

![Worker Model Patterns menu](/images/worker_model_patterns_menu.png)


And then you can add/create/delete patterns with this kind of view:

![Worker Model Patterns menu](/images/worker_model_patterns_add.png)
