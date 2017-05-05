+++
title = "Script"
chapter = true

[menu.main]
parent = "actions-builtin"
identifier = "script"

+++

**Script** is a builtin action, you can't modify it.

This action execute a script, written in script attribute

## Parameters

* script: Content of your script. You can put

```bash
#!/bin/bash
```

 or

```bash
#!/bin/perl
```

 at first line.

Make sure that the binary used is in the pre-requisites of action

#### Variable

You can use [CDS Variables]({{< relref "building-pipelines.variables.md" >}}) in a step script.

![img](/images/building-pipelines.actions.builtin.script-bash.png)

### Example

* Job Configuration, a step with perl, another with bash

![img](/images/building-pipelines.actions.builtin.script-job.png)

* Launch pipeline, check logs

![img](/images/building-pipelines.actions.builtin.script-logs.png)
