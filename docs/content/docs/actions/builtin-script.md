---
title: "Script"
card:
  name: builtin
---

**Script** is a builtin action, you can't modify it.

This action executes a given script with a given interpreter.

## Parameters

* **script**: Content of your script.
You can put #!/bin/bash, or #!/bin/perl at first line.
Make sure that the binary used is in
the pre-requisites of action.


## Requirements

No Requirement

## YAML example

Example of a pipeline using Script action:
```yml
version: v1.0
name: Pipeline1
stages:
- Stage1
jobs:
- job: Job1
  stage: Stage1
  steps:
  - script:
    - '#!/bin/sh'
    - echo "{{.cds.application}}"

```

#### Using CDS variables in a script

You can use [CDS Variables]({{< relref "/docs/concepts/variables.md" >}}) in a step script.

![img](/images/workflows.pipelines.actions.builtin.script-bash.png)

#### Using CDS keys in a script

You can use CDS SSH keys and PGP keys in a step script with the [worker key command]({{< relref "/docs/components/worker/key/_index.md" >}}). Just use `worker key install proj-mykey` and it will install the SSH/PGP environment and private SSH/PGP key of your key in your project named **mykey**.

The command `worker key install proj-mykey` will return the path where the private key is stored. In that way you can save this value in a variable and use it for a ssh command like this:

```bash
PKEY=`worker key install proj-mykey`
ssh -i $PKEY myuser@myhost echo "test" #PKEY only works with SSH key
```

Pay attention, to use a PGP key, please add in your pipeline requirements the binary named `gpg`.

#### Using worker CLI in a script

You can use worker CLI to make different actions

+ [worker artifacts]({{< relref "/docs/components/worker/artifacts.md" >}})
+ [worker download]({{< relref "/docs/components/worker/download.md" >}})
+ [worker export]({{< relref "/docs/components/worker/export.md" >}})
+ [worker tag]({{< relref "/docs/components/worker/tag.md" >}})
+ [worker cache]({{< relref "/docs/components/worker/cache/_index.md" >}})
+ [worker tmpl]({{< relref "/docs/components/worker/tmpl.md" >}})
+ [worker key]({{< relref "/docs/components/worker/key/_index.md" >}})

## Example

* Job Configuration, a step with perl, another with bash

![img](/images/workflows.pipelines.actions.builtin.script-job.png)

* Launch pipeline, check logs

![img](/images/workflows.pipelines.actions.builtin.script-logs.png)

## Notes

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
