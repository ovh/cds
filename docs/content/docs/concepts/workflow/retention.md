---
title: "Retention"
weight: 10
---

You can configure two options in Workflow advanced section on the CDS UI:
* Workflow run retention policy. A lua rule to check if a run should be kept or not.
* Maximum number of Workflow Runs. The maximum number of run to keep for the Workflow. 

![retention.png](../images/workflow_retention.png)

The dry run button allows you to test your lua expression. The result is a table filled with all runs that would be kept

## Workflow run retention policy

{{% notice note %}}
This feature is not currently enabled by default. However, you can try this feature on a CDS project using the feature flipping.
To activate the feature you can create a file like the following:
```sh
cat <<EOF > workflow-retention-policy.yml
name: workflow-retention-policy
rule: return project_key == "KEY_FOR_PROJECT_THAT_YOU_WANT_TO_ACTIVATE"
EOF
cdsctl admin feature import workflow-retention-policy.yml
```
{{% /notice %}}

Retention policy is defined through a lua condition. This condition should be evaluated as **true** to keep a Workflow Run.

You can define a custom condition on a Workflow, if not set it will fallback to the default one from CDS API configuration file (key: api.workflow.defaultRetentionPolicy).

You will be able to use these variables in conditions:
  * **run_days_before** (number): count of days between Workflow creation date and now.
  * **has_git_branch** (string: true|false): True if a *git.branch* variable is set **(added in 0.48.1)**.
  * **git_branch_exist** (string: true|false): True if a *git.branch* variable is set and branch still exists on the git repository.
  * **run_status** (string: Success|Fail|...): the Workflow Run status.
  * **gerrit_change_merged** (string: true|false): to identify if the gerrit change has been merged.
  * **gerrit_change_abandoned** (string: true|false): to identify if the gerrit change has been abandoned.
  * **gerrit_change_days_before** (number): to identify gerrit change older than x days.
  * All other variables from the Workflow Run payload (ex: cds_triggered_by_username, git_branch...).

Examples:
```lua
  -- Keep Run for 365 days
  return run_days_before < 365
````
```lua
  -- Keep Run for 365 days if git_branch is set and exists in VCS or only 2 days for removed branches
  -- Else keep Run for 365 days if no git_branch info is set
  if(has_git_branch == "true") then
    if(git_branch_exist == "true") then
      return run_days_before < 365
    else
      return run_days_before < 2
    end
  else 
    return run_days_before < 365
  end
```
```lua
  -- Keep Run for ever
  return true
```

## Maximum number of Workflow Runs

{{% notice note %}}
This feature is not currently enabled by default. However, you can try this feature on a CDS project using the feature flipping.
To activate the feature you can create a file like the following:
```sh
cat <<EOF > workflow-retention-maxruns.yml
name: workflow-retention-maxruns
rule: return project_key == "KEY_FOR_PROJECT_THAT_YOU_WANT_TO_ACTIVATE"
EOF
cdsctl admin feature import workflow-retention-maxruns.yml
```
{{% /notice %}}

This value can be set only by a CDS administrator. In some case it prevent a Workflow to keep a lot of runs.
When this feature is active, you'll not be able to start new Runs on a Workflow if the maximum count was reached.
