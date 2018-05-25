+++
title = "Worker Model Variables"
weight = 3

+++

When you want to add a new worker model or a new worker model pattern you have to use some variables that CDS give you with interpolation. To use them for example for the `--api` flag, CDS provides a variable named API that you can use like this `--api={{.API}}`.


Here is the list of available variables :

+ API `string` --> URI of the CDS API set in the hatchery configuration
+ Token `string` --> token set in the hatchery configuration
+ Name `string` --> Name of the worker
+ BaseDir `string` --> basedir configuration set in the hatchery configuration
+ HTTPInsecure `bool` --> http insecure configuration set in the hatchery configuration
+ Model `int64` --> ID of the model that the hatchery want to spawn
+ Hatchery `int64` --> ID of the hatchery
+ HatcheryName `string`
+ PipelineBuildJobID `int64`
+ WorkflowJobID `int64` --> Useful to know which workflow job the hatchery will launch spawning this worker
+ TTL `int`
+ FromWorkerImage `bool`   
+ GraylogHost `string`
+ GraylogPort `int`    
+ GraylogExtraKey `string`
+ GraylogExtraValue `string`
+ GrpcAPI `string`
+ GrpcInsecure `bool`
