+++
title = "Step"
weight = 4

[menu.main]
parent = "concepts"
identifier = "concepts-step"

+++

The steps of a job is the list of the different operation performed by the CDS worker. Each step is based on an **Action** which is defined by CDS. The list of all actions is defined on `*<your cds url ui>/#/action*`. On the very first step failed, the job is marked as Failed and execution is stopped.

You can define a Step as final. It mean that even if the job is failed, the step will be executed. The *final* steps are executed after all other steps.

Here is an example of steps creation in CDS.
You have 2 configuration flags:

- Optional : with this flag checked, even if this step fails, the stage execution will continue.
- Always executed : with this flag checked, this step will be executed even if previous steps fail. For example, if you make tests in your previous step and the tests fail you want to upload the report but not deploy, here is a use case.

![Steps Examples](/images/concepts_step_example.png)
