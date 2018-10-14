# Roadmap

1. [Platforms & Providers](#platforms)
1. [Workflow Template](#template)
1. [Managed Pipelines](#managed-pipelines)
1. [Workflow Hooks](#hooks)
1. [Code coverage report](#coverage)
1. [Security analysis report](#security)
1. [Static analysis report](#static)

## Platforms & Providers <a name="platforms"></a>

Platform OpenStack, providers: 
 - File Storage File: artifact
 - Block Storage: persistent workspace on workspace 
 - Deploy on Environment
 - CDS Compute: let users to use their own platform to run CDS Workers

Platform Marathon, providers:
 - Deploy on Environment
 - CDS Compute


## Workflow template <a name="template"></a>

Template as binary file we be removed from CDS. We consider that the main use case of template is the bootstrap on an **Application Workflow**. So Workflow Template must be just generic configuration file.

In CDS API, multiple template repositories will be registered. A template repositories (such as 'public' or 'my-great-company') is a kind of namespace. In each template repository we register template workflow with a `name`, a `description` and an `url`. The `URL` will be used to download the `yaml` file.

The goal of templates is to instantiate an **Application Workflow** configuration file in a Git repository. Basically a template is a **Golang text/template** formatted file.

When a template will be used from command line in a Git repository:

- The use choose a project in which he is able to create a workflow
- If the project is not linked to any repository-manager, it ask for it
- Each variable os the template is prompted on command line
- The **Application Workflow** configuration file is generated in the current directory
- The current repository is added in the CDS project as an application linked to the repository
- The workflow is imported in the project

### Use case #1

As a developer, considering:

- the working directory is in a Git repository (.git directory is available is the directory or in parent directory)
- `PROJECT_KEY` is a project in CDS
- I can see all the templates with: `$ cds template list`
- I choose my template and run
- `$ cds new --template <template>`
  - for each variable prompt the use and replace in the template
  - if the variable is a secret, prompt the user, and encrypt the secret with the project GPG key
  - write the file in the current directory
- `$ cds push [PROJECT_KEY]`

## Managed Pipelines <a name="managed-pipeline"></a>

## Workflow Hooks <a name="hooks"></a>

Provides workflows hooks SDK to give to users ability to develop their own hooks to trigger their workflow on every events. For instances:

- Kafka messages
- OpenStack Nova server status change
- OpenStack Swift container events
- Marathon events (application failure): Marathon Apps self healing
- SNMP Traps: SNMP traps will trigger some alerts in Nagios/Shinken, and you may trigger some workflow in CDS
- HTTP/TCP/UDP Dial Error: For advanced health check a self healing workflow
- ...

## Code coverage report <a name="coverage"></a>

As Units tests report, CDS will support code coverage report. It should also provides a way to measure improvements/regression of code coverage.

## Security analysis report <a name="security"></a>

As Units tests report, CDS will support security analysis report per workflow. It should support [OWASP Dependency-Check](https://www.owasp.org/index.php/OWASP_Dependency_Check) for code dependencies vulnerability, [CoreOS Clair](https://github.com/coreos/clair) for container images vulnerability.

## Static analysis report <a name="static"></a>

As Units tests report, CDS will support static analysis/linter report. With or without SonarQube. It should also provides a way to measure improvements/regression of code quality.

