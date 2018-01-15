+++
title = "Script"
chapter = true

+++

**Script** is a builtin action, you can't modify it.

This action executes a given script with a given interpreter.

## Parameters

* script: Content of your script. You can put

```bash
#!/bin/bash
```

 or

```bash
#!/bin/perl
```

 at first line. This will define the interpreter.

Make sure that the binary used is in the pre-requisites of the action.

If you use a shebang bash, sh, zsh or ksh, CDS will return a failure on your step if an executed command fails.

If you want to control command's exit code, you have to add
```bash
set +e
```

Below is an example of a step that will fail at the first line:

```bash
which a-unknown-binary # Step will fail here, lines below won't be executed
if [ $? -ne 0 ]; then
  echo "binary a-unknown-binary does not exists"; # this won't be displayed
  exit 1
fi;
exit 0
```

If you want to display an error message before exiting, you should rather use:

```bash
set +e
which a-unknown-binary
if [ $? -ne 0 ]; then
  echo "binary a-unknown-binary does not exists"; # this will be displayed
  exit 1
fi;
exit 0
```


#### Using CDS variables in a script

You can use [CDS Variables]({{< relref "workflows/pipelines/variables.md" >}}) in a step script.

![img](/images/workflows.pipelines.actions.builtin.script-bash.png)

### Example

* Job Configuration, a step with perl, another with bash

![img](/images/workflows.pipelines.actions.builtin.script-job.png)

* Launch pipeline, check logs

![img](/images/workflows.pipelines.actions.builtin.script-logs.png)
