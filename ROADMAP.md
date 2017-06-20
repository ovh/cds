# Roadmap

## CDS Workflow

Issues #538, #539, #540, #742.

A CDS Worflow is set of pipelines and allow pipelines reusability with multiples execution contexts. It orchestrates pipelines executions by triggering the right pipelines at the right time with the right input.

Workflows embeds following concepts:

- **Pipeline**: It's the key component of CDS. A pipeline can have paremeters.
- **Payload**: The payload is the input for running the pipeline.
- **Context**: A context defines it the pipeline has to be run with or without an **application** and **environment**. It also defines a default **Payload** and default **pipeline parameters**.
- **Hook**: A hook is the way to trigger a workflow. You can attach one or more hooks on each pipelines of the workflow.
- **Root**: A workflow has one and only one root. You will often hooks on this very fist pipelines
- **Trigger**: Triggers are links between two pipelines. You can define conditions on each triggers to define when the destination pipeline have to be ran.
- **Join**: Joins allows to to wait for multiple pipelines execution before triggering pipelines through a **trigger**
- **Run**: A Run is a workflow execution. It is identified by a **Number**. In a Run, every pipelines can be re-run, it introduce the **Subnumber**. So a workflow run with number 12, can have pipeline runs identified by 12.0, 12.1, 12.2 etc.

### CDS Workflow As Code

#### Definition

Beyond pipeline configuration CDS must support **Application Workfow** configuration file located in **Git repositories**. We talk here about application workflow because this kind of workflow will be dedicated to only one application.

- Application have to be linked to a repository managed.
- An **application workflow** cannot be updated in CDS, it is only maintained in the yaml file.
- An **application workflow** can embed a pipeline or use a project-defined pipeline
- An **application workflow embedded pipeline** is only define ine the pipeline, is cannot be used in another workflow. It can not be edited by itself
- An **application workflow** has always a webhook or a github poller on the root pipeline
- On each branch, the  **application workflow** is redefined for the branch
- The webhook or the poller will delete the **application workflow** redefined for the branch, on branch deletion

#### Use case #1

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

#### Use case #2

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

#### Example

comming soon


### Workflow template

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

#### Use case #1

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

### CDS Workflow file format

#### Export

comming soon

#### Import

comming soon 
