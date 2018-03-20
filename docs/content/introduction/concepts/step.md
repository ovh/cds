+++
title = "Step"
weight = 4

+++

The steps of a job is the list of the different operations performed by the CDS worker. Each step is based on an **Action** pre-defined by CDS. The list of all actions is defined on `*<your cds url ui>/#/action*`. When a step fails, its parent job is stopped and marked as `failed`.

You can define a Step as final. It mean that even if the job fails before reaching it, the step will be executed anyway. The *final* steps are executed after all other steps.

You can find below an example of steps creation in CDS.
You have 2 configuration flags:

- Optional : The failure of the step does not cause the failure of the whole job.
- Always executed : with this flag checked, this step will be executed even if previous steps fail. This can be helpful, for example, if you run tests in a step and you would like to upload the tests report even if the tests fail.

![Steps Examples](/images/concepts_step_example.png)
