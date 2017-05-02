+++
draft = false
title = "Step"

weight = 4

[menu.main]
parent = "concepts"
identifier = "concepts-step"

+++

The steps of a job is the list of the different operation performed by the CDS worker. Each steps is based on an **Action** which is defined by CDS. The list of all actions is defined on `*<your cds url ui>/#/action*`. On the very first step failed, the job is marked as Failed and execution is stopped.

You can define a Step as final. It mean that even if the job is failed, the step will be executed. The *final* steps are executed after all other steps.
