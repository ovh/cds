# CDS: Continuous Delivery Service

[![Go Report Card](https://goreportcard.com/badge/github.com/ovh/cds)](https://goreportcard.com/report/github.com/ovh/cds)
[![GoDoc](https://godoc.org/github.com/ovh/cds/sdk/cdsclient?status.svg)](https://godoc.org/github.com/ovh/cds/sdk/cdsclient)

<img align="right" src="https://raw.githubusercontent.com/ovh/cds/master/logo-background.png" width="25%">

CDS is an Enterprise-Grade Continuous Delivery & DevOps Automation Platform written in Go(lang).

**This project is under active development**

**[Documentation](https://ovh.github.io/cds/)**


## Intuitive UI
CDS provides an intuitive UI that allows you to build complex workflows, run them and dig into the logs when needed.

<p align="center">
  <kbd>
    <img src="./docs/static/images/capture-start.gif" alt="create and run workflow with CDS ui" title="create and run workflow with CDS ui">
  </kbd>
  <i>Create and run workflow with CDS ui.</i>
</p>

## The most powerful Command Line for a CI/CD Platform

cdsctl is the CDS Command Line - you can script everything with it, cdsctl also provides some cool commands such as `cdsctl shell` to browse your projects and workflows without the need to open a browser.

[See all cdsctl commands](https://ovh.github.io/cds/docs/components/cdsctl/)

<p align="center">
  <img src="./docs/static/images/init_template_as_code.gif" alt="create workflow as code with CDS command line" title="create workflow as code with CDS command line">
  <i>Create workflow as code with CDS command line.</i>
</p>

## Want a try?

Docker-Compose is your friend, see [Ready To Run Tutorials](https://ovh.github.io/cds/hosting/ready-to-run/)

## Blog posts and talks

- CDS Introduction: https://blog.ovhcloud.com/how-does-ovh-manage-the-ci-cd-at-scale/
- DataBuzzWord Podcast (French) : https://blog.ovhcloud.com/understanding-ci-cd-for-big-data-and-machine-learning/
- Continuous Delivery and Deployment Workflows with CDS: https://blog.ovhcloud.com/continuous-delivery-and-deployment-workflows-with-cds/
- Talk at conference [Breizhcamp](https://www.breizhcamp.org) to introduce CDS (French): https://www.youtube.com/watch?v=JUzEQuOehv4

## FAQ

### Why CDS? Discover the Origins

- [Self-Service](https://ovh.github.io/cds/about/why_cds/#1-self-service)
- [Horizontal Scalability](https://ovh.github.io/cds/about/why_cds/#2-horizontal-scalability)
- [High Availability](https://ovh.github.io/cds/about/why_cds/#3-high-availability)
- [Pipeline Reutilisability](https://ovh.github.io/cds/about/why_cds/#4-pipeline-reutilisability)
- [Rest API](https://ovh.github.io/cds/about/why_cds/#5-rest-api)
- [Customizable](https://ovh.github.io/cds/about/why_cds/#6-customizable)

### What is a CDS workflow?

Most of the CI/CD Tools play with jobs inside a pipeline. CDS introduces a new concept named `CDS Workflows`.
A [CDS Workflow](https://ovh.github.io/cds/docs/concepts/workflow/) allows you to chain pipelines with triggers.
A [pipeline](https://ovh.github.io/cds/docs/concepts/pipeline/) is structured in sequential [stages](https://ovh.github.io/cds/docs/concepts/pipeline/#stages) containing one or multiple concurrent [jobs](https://ovh.github.io/cds/docs/concepts/job/).


### Can I use it in production?

Yes! CDS is used in production since 2015 @OVH and it launches more than 7M CDS workers per year. You can install the official release available on https://github.com/ovh/cds/releases

CDS provides everything needed to monitor and measure production activity (logs, metrics, monitoring)

### How to backup?

All data are stored in the database - nothing on the filesystem. Just backup your database regularly and you will be safe.

### Need some help?

Core Team is available on [GitHub](https://github.com/ovh/cds/discussions)

### CDS User features

#### Pipeline

Ability to run multiple jobs simultaneously while keeping an isolation between them. [See doc about stages & jobs inside a pipeline](https://ovh.github.io/cds/docs/concepts/pipeline/). A pipeline is started with a context: 0 or 1 application, 0 or 1 environment.

#### Workflow

A Workflow makes it possible to chain the pipelines. This is a key feature of CDS. You can create workflows using one or more pipelines, pipelines that can be linked together with joins or forks.

You can imagine having only one workflow builder and deploying your entire microservice stack. The same pipeline can be used several times in a workflow, you can associate an application or an environment.
You will only have one deployment pipeline and one build pipeline to maintain, even if you have hundreds of applications.

#### Workflow templates

A workflow template allows you to share and reuse workflows across multiple teams. Any user can create a Workflow Template, maintain it as code or from UI, and bulk update a set of workflows with a single action.

As a company, you can offer a predefined catalog of Workflows allowing you to standardize test and deployment practices across all your teams.

This also reduces the maintenance efforts since templates allow a scalable centralized management.

#### Visual configuration with Web UI

You can configure everything with the web UI. Even if you have complex use cases, it's usually easier to create your workflows graphically.

#### Configuration on Git Repository

Pipeline as code is a well-known concept of CI / CD tools. CDS goes a step further and offers workflow as code. This is done by git-pushing using yaml configuration files of your workflow (+ pipeline, + applications, + environment). This is particularly useful as you can test your new workflow on a dev branch, before merging the changes on the master branch.


#### Configuration as code on UI

You can modify your workflow with the UI, you can also modify the configuration by editing the yaml file directly in the UI if you wish. This is an excellent way to learn how to use the workflow-as-code feature.

#### Native Git branching

Ability to launch builds based on a branch pattern. This allows, for example, to deploy dev/* branches to "staging" and deploy the master branch to "prod".

Note that CDS's default behavior is to launch the whole workflow on every git commit. This behavior can be altered using "run conditions".

#### Native GitHub / Bitbucket Server / GitLab / Gerrit integration

2-way integration with most popular git-based products.

1. Ability to get notified and start a build when a change is pushed.
2. Ability to notify the git-based tool of the success/failure of the build.

CDS natively supports GitHub, GitLab, Bitbucket Server, and Gerrit.
The link between your git repo and CDS is via a CDS application: 1 Git repository == a CDS application.
Through this integration, CDS will push the build status of your commits : Building, Success, or Failed.

#### Multiple VCS Support in Pipeline/Job

CDS gives you the possibility to clone from different git repositories within a single workflow. A CDS workflow can involve several different applications - or none if you do not want to have a connection with a git repo.

#### Job's Services

Ability to start ephemeral services (a database, a web server, etc.) to support your job. This is particularly handy while testing your code.

In CDS these services are called Service Prerequisites. You just need to specify the corresponding docker image and run params.

Take a simple example: you have a pipeline that builds a docker image containing your application. Your application needs a redis and a PostgreSQL to work. You can, in a CDS job, put three prerequisite services: a redis, a PostgreSQL, and your application. CDS will take care of making a private network between its services so that they can communicate with each other.
Your CDS job can thus perform integration tests on your application starting with a real database and a real cache.

Please read: https://ovh.github.io/cds/docs/concepts/requirement/requirement_service/

#### Secure Remote Caching

A remote cache is used by a team of developers and/or a continuous integration (CI) system to share build outputs. If your build is reproducible, the outputs from one machine can be safely reused on another machine, which can make builds significantly faster

Doc: https://ovh.github.io/cds/docs/components/worker/cache/

#### Enterprise Notification Bus

As an Enterprise-Grade platform, CDS can send a wide range of its internal events (e.g. build finished) in an event bus.
This event flow can then feed other services (reporting, notifications, etc., ).

#### Built-in Hooks

Ability to launch a workflow manually or with git pushes or via a scheduler or via a webhook.
In addition to the above, CDS can also be triggered using an event bus (kafka or RabbitMQ).

#### Continuous Deployment & Environment Support

Ability to manage multiple environments (e.g. dev/prod/staging) in a secure way with segregated access rights.
In practice, an environment is a set of variables that you can use within your workflows.

With CDS, You can use a deployment pipeline on your preproduction environment and use that same deployment pipeline on your production environment. The ability to deploy to production can be limited to a pre-established group of users.

#### Enterprise-grade permissions / Support of ACLs delegation

Users are free to create groups and manage users in their groups. A group can have the right to read, write and execute their projects and their workflows. You can also restrict the execution of some pipelines to some groups if you wish.

#### Build Artifacts Cloud

If you use CDS as a CI / CD tool, you will probably have built artifacts. CDS jobs are isolated from each other, but you can pass artifacts from one job to another using the Artifact Upload and Artifact Download actions.
At the end of a CDS job, all the files are deleted from the workers. To persist artifacts, CDS can use a Swift Storage or a given filesystem (not recommended though).


#### Tests & Vulnerabilities Reports

CDS clearly displays the results of unit tests and vulnerabilities detected during builds.

#### Self-Service Project Creation

A CDS project is like a tenant. All users can create a CDS project, this project will bring together applications, environments, pipelines and of course workflows.

CDS projects are isolated from one another, but the same group may have access rights to multiple projects if you wish.

#### Execution Environment Customization

A worker model is a worker execution context. Let's say, you need to run a job that requires GoLang v1.11.5. In CDS, you just need to create a Go worker model, containing a go in version 1.11.5.
A worker model can be a docker image, an OpenStack image or a VSphere image.
Although CDS administrators can offer shared worker models, users can create their own template workers if they wish.

#### Self-Service User’s Integrations

On a CDS project, you can add integrations like OpenStack, Kubernetes, etc ... This will offer you features within your workflows. For example, with the Kubernetes integration, you can add your own cluster to your CDS project and thus be able to use the Deploy Application action to deploy your newly built application on your cluster, in helm format if you wish.
You can of course develop your own integrations.

#### Multi-Tenancy

After reading the previous points, you've understood: self-service is everywhere. All users can create their project/workflow/ worker models/workflow templates/actions... And run Jobs in a totally isolated environment. CDS projects are builders, on which you can add integrations. All this allows you to have only one instance of CDS for your entire company.

#### Command Line Interface (cdsctl): 100% of features supported

All you can do with the UI is available via the Command-Line Interface (CLI), named "cdsctl". cdsctl is available on all the OS: Darwin, FreeBSD, Linux, OpenBSD... cdsctl will allow you to create, launch, export, import your workflows, monitor your CDS and navigate through your projects and workflows.
No need to go to the UI of CDS or your repository manager to check the status of your commit, `git push && cdsctl workflow --track` will display your workflow in your command line.

#### REST API & SDK

Do you have even more advanced automation needs, or the desire to develop an application that queries CDS? The [REST API](https://ovh.github.io/cds/development/rest/) and the [SDK](https://ovh.github.io/cds/development/sdk/golang/) will allow you to easily develop your software.

### CDS Administration features

#### Self-Hosting

CDS has been open-source since October 2016. You can install it freely in your company or at home. Some tutorials are available to help you start a CDS, [docker-compose](https://ovh.github.io/cds/hosting/ready-to-run/docker-compose/), [Install with binaries](https://ovh.github.io/cds/hosting/ready-to-run/from-binaries/).

#### High Availability / Scalability / Upgrade without User Downtime

High availability is a very important point for a CI / CD tool. CDS is stateless, nothing is stored on the filesystem. This makes it possible to launch several CDS APIs behind a load balancer. Thus, you can scale the API of CDS to your needs. It also allows upgrades of CDS in a full day without impact on users.
In production @OVH, CDS can be updated several times a day, without impacting users or stopping CDS workers.
Asking your users to stop working while updating the Continuous Delivery tool would be ironic, isn't it? ;-)


#### Built-in Metrics

CDS natively exposes monitoring data. You will be able to feed your instance [prometheus](https://prometheus.io/) or [warp10](https://www.warp10.io/) using [beamium](https://github.com/ovh/beamium).

#### Extensibility Plugins

A CDS job consists of steps. Each step is a built-in type action (script, checkoutApplication, Artifact upload/download...). You can create your actions, using existing actions - or develop your action as a plugin. All languages are supported, as long as the language supports GRPC.

#### OS/Arch Compatibility

CDS is agnostic to languages and platforms. Users can launch Jobs on Linux, Windows, FreeBSD, OS X, Raspberry ... in a Virtual Machine spawn on demand, in a docker container, on a dedicated host.

So, if your company uses multiple technologies, CDS will not be a blocker for building and deploying your internal software.

#### Auto-Scale OnDemand multi-cloud

One of the initial objectives of CDS at OVH: build and deploy 150 applications as a container in less than 7 minutes. This has become a reality since 2015. What is the secret key?
Auto-Scale on Demand!

Thus, you can have hundreds of worker models and when necessary, CDS will start the workers using the hatcheries.

A [hatchery](https://ovh.github.io/cds/docs/components/hatchery/) is like an incubator, it gives birth to the CDS Workers and the right to life and death over them.

Several types of hatchery are available:

 - **[hatchery kubernetes](https://ovh.github.io/cds/docs/integrations/kubernetes/kubernetes_compute/)** starts workers in pods
 - **[hatchery openstack](https://ovh.github.io/cds/docs/integrations/openstack/openstack_compute/)** starts virtual machines
 - **[hatchery swarm](https://ovh.github.io/cds/docs/integrations/swarm/)** starts docker containers
 - **[hatchery vSphere](https://ovh.github.io/cds/docs/integrations/vsphere/)** starts virtual machines
 - **[hatchery local](https://ovh.github.io/cds/docs/components/hatchery/local/)** starts processes on a host

 So yes, buzzwords or not, a multi-cloud Auto-scale OnDemand is a reality with CDS :-)


## License

[3-clause BSD](./LICENSE)
