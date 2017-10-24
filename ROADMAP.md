# Roadmap

1. [Workflow file format](#fileformat)
1. [Application Workflow / Workflow As Code](#wasc)
1. [Workflow Template](#template)
1. [Workflow Hooks](#hooks)
1. [Code coverage report](#coverage)
1. [Static analysis report](#static)
1. [Security analysis report](#security)
1. [Platforms & Providers](#plaforms)
1. [Mesos Hatchery](#mesos)
1. [Kubernetes support](#k8s)


## CDS Workflow file format  <a name="fileformat"></a>

Export / Import

## CDS Workflow As Code <a name="wasc"></a>

### Definition

Beyond pipeline configuration CDS must support **Application Workfow** configuration file located in **Git repositories**. We talk here about application workflow because this kind of workflow will be dedicated to only one application.

- Application have to be linked to a repository managed.
- An **application workflow** cannot be updated in CDS, it is only maintained in the yaml file.
- An **application workflow** can embed a pipeline or use a project-defined pipeline
- An **application workflow embedded pipeline** is only define ine the pipeline, is cannot be used in another workflow. It can not be edited by itself
- An **application workflow** has always a webhook or a github poller on the root pipeline
- On each branch, the  **application workflow** is redefined for the branch
- The webhook or the poller will delete the **application workflow** redefined for the branch, on branch deletion

### Use case #1

As a developper, considering :

- the working directory is in a git repository (.git directory is available is the directory or in parent directory)
- a yml file is defined in working directory
- `PROJECT_KEY` is a project in CDS

I create the **application workflow** with `$ cds push [--git-remote origin] <PROJECT_KEY> .`:

- if the project is not linked to the repository manager defined by the `git remote origin`.
  - ask to link the project
- create the application linked to the repository
- create the workflow with embedded pipelines
- create the webhook or the poller

### Use case #2

As a developper, considering :

- the working directory is in a git repository (.git directory is available is the directory or in parent directory)
- a yml file is defined in working directory and the workflow has been pushed to CDS

I commit and push on `origin`:

- the workflow hook detects the change on the remote git repository
- it analyzes the yaml definition of the workflow
- it triggers the workflow as defined

On my command line:

- `$ cds track`: print the execution status of the workflow for my current branch and my current commit 
- `$ cds logs <pipeline>`: print the logs of the pipeline nammed `<pipeline>`
- `$ cds restart <pipeline>`: rerun the workflow node one the first pipeline `<pipeline>` found
- `$ cds status`: print :
  - if the workflow has been push of not,
  - last execution of the workflow for this branch,
  - the last branch executed,

### Example

comming soon


## Workflow template <a name="template"></a>

Template as binary file we be removed from CDS. We consider that the main use case of template is the bootstrap on an **Application Workflow**. So Workflow Template must be just generic configuration file.

In CDS API, multiple template repositories will be registered. A template repositories (such as 'public' or 'my-great-company') is a kind of namespace. In each template repostory we register template workflow with a `name`, a `description` and an `url`. The `URL` will be used to download the `yaml` file.

The goal of templates is to instanciate an **Application Workfow** configuration file in a Git repository. Basically a template is a **Golang tesx/template** formatted file.

When a template will be used from command line in a Git repository:

- The use choose a project in wich he is able to create a workflow
- If the project is not linked to any repository-manager, it ask for it
- Each variable os the template is prompted on command line.
- The **Application Workfow** configuration file is generated in the current directory.
- The current repostory is added in the CDS project as an application linked to the repository.
- The workflow is imported in the project

### Use case #1

As a developper, considering :

- the working directory is in a git repository (.git directory is available is the directory or in parent directory)
- `PROJECT_KEY` is a project in CDS
- I can see all the templates with: `$ cds template list`
- I choose my template and run
- `$ cds new --template <template>`
  - for each variable prompt the use and replace in the template
  - if the variable is a secret, prompt the user, and encrypt the secret with the project GPG key
  - write the file in the current directory.
- `$ cds push [PROJECT_KEY]`

## Workflow Hooks <a name="hooks"></a>

Provides workflows hooks SDK to give to users ability to develop their own hooks to trigger their workflow on every events. For instances:

- Kafka messages
- Openstack nova server status change
- Openstack swift container events
- Marathon events (application failure): Marathon Apps self healing
- SNMP Traps: SNTMP traps will trigger some alerts in Nagios/Shinker, and you may trigger some workflow in CDS
- HTTP/TCP/UDP Dial Error: For advanced health check a self healing workflow
- ...

## Code coverage report <a name="coverage"></a>

As Units tests report, CDS will support code coverage report. It should also provides a way to measure improvements/regression of code coverage.

## Security analysis report <a name="security"></a>

As Units tests report, CDS will support security analysis report per workflow. It should support [owasp dependency checks](https://www.owasp.org/index.php/OWASP_Dependency_Check) for code dependencies vulnerability, [coreos clair](https://github.com/coreos/clair) for container images vulnerability.

## Static analysis report <a name="static"></a>

As Units tests report, CDS will support static analysis/linter report. With or without sonarcube. It should also provides a way to measure improvements/regression of code quality.


## Platforms & Providers <a name="platforms"></a>

Platform Openstack, providers: 
 - File Storage File: artifact
 - Block Storage : persistent workspace on workspace 
 - Deploy on Environment
 - CDS Compute: let users to use their own platform to run CDS Workers

Platform Marathon, providers:
 - Deploy on Environment
 - CDS Compute

## Mesos Hatchery <a name="mesos"></a>

For the moment, CDS provides a Mesos/Marathon hatchery wich to spawn worker on a mesos cluster. As worker are not long-running application, their is a lot of workaround to solves this equation. So the goal is develop a mesos hatchery as a mesos framework.

## Kubernetes(K8S) Support <a name="k8s"></a>

Support Kubernetes as a CDS infra structure to run worker as Kubernetes pods, and also support Kuberneted as a Platform.



