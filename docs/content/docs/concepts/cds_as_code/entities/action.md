---
title: "Action"
weight: 3
---

# Description

An action is a sequence of steps that can be used within a job

# As Code directory

An action is described directly on your repository inside the directory `.cds/actions`

# Permission

To be able to manage actions you will need the permission `manage-action` on your project

# Fields

```yaml
name: test-parent-action2
description: simple parent action as code
inputs:
  name:
    description: my description
    required: true
    default: Steven
  workflow:
    description: event receive
    default: ${{ cds.workflow }}
runs:
  steps:
  - run: |
      echo "Welcome ${{ inputs.name }}
```



* <span style="color:red">*</span>`name`: The name of your workflow
* `descrption`: information about the action
* `inputs`: action inputs
    * `inputs.description`
    * `required`: indicates if the inputs is mandatory
    * `default`: default value
* `runs.steps`: the list of [steps](./../workflow/#step) executed by the action 

