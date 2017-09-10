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

Make sure that the binary used is in the pre-requisites of action.

If you use a shebang bash, sh, zsh or ksh, CDS will return a failure on your step if an executed command fails.

If you want to control command's exit code, you have to add
```bash
set +e
```

Example of step, CDS will exit at the first line, as `which a-unknown-binary will return an error` :

```bash
which a-unknown-binary # Step will fail here, lines below won't be executed
if [ $? -ne 0 ]; then
  echo "binary a-unknown-binary does not exists";
  exit 1
fi;
exit 0
```

Example of step, CDS will execute all lines:

```bash
set +e
which a-unknown-binary
if [ $? -ne 0 ]; then
  echo "binary a-unknown-binary does not exists"; # this will be displayed
  exit 1
fi;
exit 0
```


#### Variable

You can use [CDS Variables]({{< relref "building-pipelines.variables.md" >}}) in a step script.

![img](/images/building-pipelines.actions.builtin.script-bash.png)

### Example

* Job Configuration, a step with perl, another with bash

![img](/images/building-pipelines.actions.builtin.script-job.png)

* Launch pipeline, check logs

![img](/images/building-pipelines.actions.builtin.script-logs.png)
