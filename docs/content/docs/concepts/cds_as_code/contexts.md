---
title: "Contexts"
weight: 3
card:
  name: cds_as_code
---


# Contexts

Contexts are a way to access information inside a workflow run. Data can be access like this
```yaml
${{ <contextName>.data }}
```

Contexts available:

* `cds`: contains all the information about the workflow run
* `git`: contains the git information
* `vars`: contains all the project variables
* `env`: contains environment variables
* `jobs`: contains all parent jobs results and outputs
* `needs`: contains all direct parents ( `job.needs` ) results and outputs
* `inputs`: contains all job inputs
* `steps`: contains all previous step status
* `matrix`: contains the curent value for each [matrix](../entities/workflow/#strategy) variable
* `integrations`: contains data of integration linked to the current job
* `gate`: contains all gate inputs
* `secrets`: contains the secrets


## Context CDS

* `event_name`: the event name that trigger the workflow
* `event`: the event payload received by CDS
* `project_key`: the project identifier of the workflow
* `run_id`: The identifier of the workflow run
* `run_number`: The current run number
* `run_attempt`: The current run attempt
* `workflow`: The name of the workflow
* `workflow_ref`: The git refs of the worklow definition used in the current workflow run
* `workflow_sha`: The git commit of the workflow definition used in the current workflow run
* `workflow_vcs_server`: The vcs server name of the workflow definition
* `workflow_repository`: The name of the workflow definition repository
* `triggering_actor`: Username that trigger the workflow run
* `job`: The current job
* `stage`: The current stage
* `workspace`: Path of the current workspace
* `integrations`: a map containing integration data linked to the current job. The key of the map is the integration name

## Context Git

* `server`: The vcs server name linked to the workflow
* `repository`: The repository linked to the workflow
* `repositoryUrl`: Url of the linked repositoryy
* `ref`: Current git refs
* `sha`: Current commit
* `connection`: Type of connection used: https/ssh
* `ssh_key`: SSH Key name used
* `username`: Username used to connect to the repository
* `semver_current`: Current semantic version computed by CDS
* `semver_next`: Next semantic version computed by CDS

## Context Jobs

* `jobs.<job_id>.result`: status of the given parent job. 
* `jobs.<job_id>.outputs`: map of all job run result of type variable
    * `jobs.<job_id>.outputs.<run_result_name>`  

## Context Needs

* `needs.<job_id>.result`: status of the given parent job. 
* `needs.<job_id>.outputs`: map of all job run results of type variable
    * `needs.<job_id>.outputs.<run_result_name>`        

## Context Steps 

* `steps.<step_id>.outcome`: result of the given state before 'continue-on-error'
* `steps.<step_id>.conclusion`: result of the given state after 'continue-on-error'
* `steps.<step_id>.outputs`: map of all job run results of type variable by step

