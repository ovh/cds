# CDS: Continuous Delivery Service

[![Join the chat at https://gitter.im/ovh-cds/Lobby](https://badges.gitter.im/ovh-cds/Lobby.svg)](https://gitter.im/ovh-cds/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Go Report Card](https://goreportcard.com/badge/github.com/ovh/cds)](https://goreportcard.com/report/github.com/ovh/cds)
[![Coverage Status](https://coveralls.io/repos/github/ovh/cds/badge.svg?branch=master)](https://coveralls.io/github/ovh/cds?branch=master)
[![GoDoc](https://godoc.org/github.com/ovh/cds/sdk/cdsclient?status.svg)](https://godoc.org/github.com/ovh/cds/sdk/cdsclient)

<img align="right" src="https://raw.githubusercontent.com/ovh/cds/master/logo-background.png" width="25%">

CDS is an Enterprise-Grade Continuous Delivery & DevOps Automation Platform written in Go(lang).

**This project is under active development**

**[Documentation](https://ovh.github.io/cds/)**


## Super intuitive UI
CDS provides a super intuitive UI that allows you to build complex workflows, run them and dig into the logs when needed.

<img src="./docs/static/images/capture-start.gif" alt="CDS Demonstration" width="80%">


## The most powerful Command Line for a CI/CD Platform

cdsctl is the CDS Command Line - you can script everything with it, cdsctl also provide some cool commands as `cdsctl shell` to browse your projects and workflows without the need to open a browser.

[See all cdsctl commands](https://ovh.github.io/cds/cli/cdsctl/#see-also)


## Want a try?

Docker-Compose or Helm are your friends, see [Ready To Run Tutorials](https://ovh.github.io/cds/hosting/ready-to-run/)

## FAQ

### Why CDS? Discover the Origins

- [Self-Service](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#1-self-service)
- [Horizontal Scalability](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#2-horizontal-scalability)
- [High Availability](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#3-high-availability)
- [Pipeline Reutilisability](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#4-pipeline-reutilisability)
- [Rest API](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#5-rest-api)
- [Customizable](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#6-customizable)

### What is a CDS workflow?

Most of the CI/CD Tools play with jobs inside a pipeline. CDS introduce a new concept named `CDS Workflows`.
A [CDS Workflow](https://ovh.github.io/cds/gettingstarted/concepts/workflow/) allows you to chain pipelines with triggers.
A [pipeline](https://ovh.github.io/cds/gettingstarted/concepts/pipeline/) is structured in sequential [stages](https://ovh.github.io/cds/gettingstarted/concepts/stage/) containing one or multiple concurrent [jobs](https://ovh.github.io/cds/gettingstarted/concepts/job/).


### Can I use it in production?

Yes! CDS is used in production since 3y @OVH and launch more than 7M CDS workers per year. You can install the official release available on https://github.com/ovh/cds/releases

CDS provides everything needed to monitor and measure production activity (logs, metrics, monitoring)

### How to backup?

All data are stored in the database - nothing on filesystem. Just backup your database regularly and you will be safe.

### Need some help?

Core Team is available on [Gitter](https://gitter.im/ovh-cds/Lobby)

### Comparison Matrix

All the features of the table are detailed below.

| Feature | CDS | Bamboo | Buildbot | Gitlab CI | Jenkins | 
| --- | --- | --- | --- | --- | --- |
| [Built-in Pipeline](#built-in-pipeline) | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | 
| [Built-in Workflow - Workflow as an entity](#built-in-workflow) | :white_check_mark: | :x: | :white_check_mark: | :x: | :x: |
| [Graphical configuration with UI](#graphical-configuration-with-ui) | :white_check_mark: | :white_check_mark: | :x: | :x: | :x: *1 | 
| [Configuration on Git Repository](#configuration-on-git-repository) | :white_check_mark: | :x: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| [Configuration as code on UI](#configuration-as-code-on-ui) | :white_check_mark: | :x: | :x: | :x: | :x:*2 |
| [Native Git branching](#native-git-branching) | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark:*3 |
| [Job's Services](#job-s-services) | :white_check_mark: | :x: | :x: | :white_check_mark: | :x: |
| [Secure Remote Caching](#secure-remote-caching) | :white_check_mark: | :x: | :white_check_mark: | :white_check_mark: | :x: *4 | 
| [Enterprise Notification (bus)](#enterprise-notification-bus) & [Event Bus Built-in Hooks](#event-bus-built-in-hooks-rabbitmq-kafka-mqseries-etc) | :white_check_mark: | :x: | :x: | :x: | :x: |
| [Continuous Deployment / Built-in Environment Support](#continuous-deployment--built-in-environment-support) | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :x: *5 |
| [Easy Enterprise-grade permissions, Self-Service on Rights management](#easy-enterprise-grade-permissions) | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :x: *6 |
| [Build Artifacts Cloud](#build-artifacts-cloud) | :white_check_mark: | :x: | :white_check_mark: | :x: | :x: *7 |
| [Tests & Vulnerabilities Reports](#tests--vulnerabilities-reports) | :white_check_mark: | :white_check_mark: | :white_check_mark: | :x: *8 | :white_check_mark: |
| [Self-Service Project Creation - ability to create a tenant](#self-service-project-creation) | :white_check_mark: | :x: | :x:*9 | :white_check_mark: | :x: |
| [Self-Service Job's Flavor](#self-service-jobs-flavor) | :white_check_mark: | :x: | :x: | :white_check_mark: | :x: *10 |
| [Multi-Tenancy](#multi-tenancy) | :white_check_mark: | :x: | :x: | :white_check_mark: | :x: *11 | 
| [Command Line Interface (cdsctl): 100% features supported & User Friendly](#command-line-interface-cdsctl-100-features-supported) | :white_check_mark: | :x: | :white_check_mark: | :x: | :x: *12| 
| [REST API & SDK](#rest-api--sdk) | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| [Self-Hosting](#self-hosting) | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| [High Availability & Scalability & Upgrade without User Downtime](#high-availability--scalability--upgrade-without-user-downtime) | :white_check_mark: | :x: | :x: | :white_check_mark:*13 | :x: | 
| [Built-in Metrics](#built-in-metrics) | :white_check_mark: | :white_check_mark: | :white_check_mark: |  :white_check_mark: | :x: *14 |
| [Extensibility Plugins](#extensibility-plugins) | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| [OS/Arch Compatibility](#osarch-compatibility) | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| [Auto-Scale OnDemand multi-cloud](#auto-scale-ondemand-multi-cloud) | :white_check_mark: | :x: | :x: | :x:*15 | :x:*16 | 

Some explanations:
- *1 Impossible if you create your pipeline with Pipeline plugin, force usage of jenkinsfile
- *2 There is the Pipeline plugin, but not compatible with Graphical COnfiguration with ui, Git branching, and repository manager integration.
- *3 [Multi-branch pipeline plugin](https://wiki.jenkins.io/display/JENKINS/Pipeline+Multibranch+Plugin ) - but incompatible [Pipeline plugin](https://wiki.jenkins.io/display/JENKINS/Pipeline+Plugin)
- *4 Job Cacher plugin not compatible with Blue Ocean, MultiBranch PIpeline, Pipeline plugin, but not compatible with Swift Storage
- *5 "The current version of this plugin may not be safe to use. " https://wiki.jenkins.io/display/JENKINS/EnvInject+Plugin
- *6 https://jenkins.io/doc/book/managing/security/#authorization
- *7 it's not builtin, it's JCloud plugin
- *8 Vulnerability report not available on CE Edition
- *9 everything is in the same bucket
- *10 docker only
- *11 explained on https://www.cloudbees.com/blog/multi-tenancy-jenkins
- *12 https://jenkins.io/doc/book/managing/cli/
- *13 Upgrade Gitlab can be on several days https://docs.gitlab.com/ee/update/ 
- *14 it's a plugin
- *15 K8s, Docker machine & GKE only 
- *16 limitated to about 150 executors https://www.cloudbees.com/blog/multi-tenancy-jenkins(§Scale)


### CDS User features

#### Built-in Pipeline

A pipeline containing stages & jobs is a basic feature. This allows you to run multiple jobs simultaneously while keeping an isolation between jobs.

#### Built-in Workflow

A Workflow makes it possible to chain the pipelines. This is a key Feature of CDS. You can create workflows using one or more pipelines, pipelines that can be linked together with joins or forks.

You can imagine having only one builder workflow and deploying your entire microservice stack. The same pipeline can be used several times in a workflow, you can associate an application, an environment.
You will have only one deployment pipeline and only one build pipeline to maintain, even if you have hundreds of applications.

#### Workflow ready to use

A workflow ready to use is a Workflow Template. Eveybody can create a Workflow Template, maintains it as code or from UI, and bulk update a set of workflows in one click. 

As a company, you can offer a catalog of workflows for all users. Each team can create its own templates if needed.

If you have to maintain hundreds of workflows, no need to worry , update workflow at scale :)

#### Graphical configuration with UI

You can configure everything with the web UI. Even if you have complex use cases, it's usually easier to create your workflow graphically.

#### Configuration on Git Repository

Pipeline as code is a well-known concept of CI / CD tools. CDS, much more than pipeline as code, makes workflow as code. What is it? You can store with your source code the yml configuration files of your workflow (+ pipeline, + applications, + environment), and thus, you can upgrade your workflow on your dev branch, before you merge the changes on the master branch.


#### Configuration as code on UI

You can modify your workflow with the UI, you can also modify the configuration by editing the yml directly in the UI if you wish.

#### Native Git branching

What is a CI tool if it does not have the possibility to launch a build on each commit of all the developers? CDS takes into account the Git branches - with each push, whatever the master branch or not, CDS will be able to trigger your workflow. You can put launch conditions on your pipelines, to deploy your application automatically on your preprod if it is the master branch for example.

#### Native Github / Bitbucket Server / Gitlab / Gerrit integration

Do you have a workflow and want to trigger it on each commit? It's easy, add a RepositoryWebhook to your Workflow. CDS natively supports Github, Gitlab, Bitbucket Server and Gerrit.
The link between your repo git and CDS is via a CDS application: 1 repository Git = a CDS application.
Through this integration, you will have the opportunity to have on your commits a status of your workflow : Building, Success or Failed.

#### Multiple VCS Support in Pipeline/Job

CDS gives you the freedom to make clone git from several different git repositories within a single workflow. A workflow can involve several different applications - or none if you do not want to have a connection with a repo git, that will not be a problem either.

#### Job's Services

Need an ephemeral database, started only for the purpose of the job? This is not a problem, it's called a Service Prerequisite in CDS.

In a CDS job, you have the option to start services, any service from the moment it is a docker image.

Take a simple example: you have a pipeline that builds a docker image containing your application. Your application needs a redis and a postgreSQL to work. You can in a CDS job put three prerequisites service: a redis, a postgreSQL and your application. CDS will take care of making a private network between its services so that they can communicate with each other.
Your CDS job can thus perform integration tests on your application started without mock, but with a real database and a real cache.

Please read: https://ovh.github.io/cds/workflows/pipelines/requirements/service/

#### Secure Remote Caching

Do you find your (npm | mvn) install too slow? Use the "worker cache"! please read: https://ovh.github.io/cds/cli/worker/cache/

#### Enterprise Notification (bus)

As an Enterprise-Grade platform, CDS can send all events in an event bus. You will be able to easily feed other tools in continuous as big data tool.

#### Event Bus Built-in Hooks (RabbitMQ, Kafka, MQSeries etc..)

Ok, you can start your workflow with each commit, manually, via a scheduler or via a webhook. CDS also offers you an event bus to trigger your workflow each time you receive a kafka or RabbitMQ message.

#### Continuous Deployment / Built-in Environment Support

How to imagine a Continuous Delivery tool without having fully integrated environment concepts?

In a CDS project, you can have applications, environments, and workflows. Each workflow can use 1 or n pipelines, 0 or n applications, 0 or n environments. You can use a deployment pipeline on your preproduction environment and use that same deployment pipeline on your production environment. So, what is an environment? An environment is a set of variables that you can use within your workflows. Simple, effective and totally integrated in CDS.

#### Easy Enterprise-grade permissions

Users are free to create groups and manage users in their groups. A group can have the rights to read, write, execute on a project (s), a workflow (s). You can also restrict the execution of some pipelines to a few groups if you wish.

A workflow allows to build all the branches of an application, you can let all the developers execute this workflow. You can also restrict production deployment to another group of users.

#### Build Artifacts Cloud

If you use CDS as a CI / CD tool, you will probably have built artifacts. CDS jobs are isolated from each other, but you can pass artifacts from one job to another using the Artifact Upload and Artifact Download actions.
At the end of a CDS job, all the files are deleted. Where are the artifacts stored? The artifacts are stored in the cloud :) -> Swift Storage, Storage CDS Integration, or at worst, if you do not have storage cloud, on your filesystem of course.


#### Tests & Vulnerabilities Reports

This is basic, CDS clearly displays the results of unit tests and vulnerabilities detected during your builds.

#### Self-Service Project Creation

A CDS project is like a tenant. All users can create a CDS project, this project will bring together applications, environments, pipelines and of course workflows.

CDS projects are isolated from one another, but the same group may have access rights to multiple projects if you wish.

#### Self-Service Job's Flavor

What is a Job Flavor? For this, the term CDS for is "Worker Model".
A worker model is a worker execution context. Do you want a job that has a binary "go" in version 1.11.5? No problem, just create a Go worker model, containing a go in version 1.11.5.
A worker model can be a docker image, an openstack image, a VSphere image. In the case of our example Go, in version 1.11.5, the worker model is neither more nor less than the official golang docker image from https://hub.docker.com/_/golang.
Although CDS administrators can offer shared worker models, users can create their own template workers if they wish.

#### Self Service User’s Integrations

On a CDS project, you can add integrations like openstack, kubernetes, etc .... This will offer you features within your workflows. For example, with the Kubernetes integration, you can add your own cluster to your CDS project and thus be able to use the Deploy Application action to deploy your newly built application on your cluster, in helm format if you wish.
You can of course develop your own integrations.

#### Multi-Tenancy

After reading the previous points, you've understood: self-service everywhere. All users can do their project / workflow / worker models / workflow templates / actions ... And run Jobs in a totally isolated environment. CDS projects are builders, on which you can add integrations. All this allows you to have only one instance of CDS for your entire company.

#### Command Line Interface (cdsctl): 100% features supported

All you can do with the UI is available via the Command-Line Interface (CLI), named "cdsctl". cdsctl is available on all the OS: darwin, freebsd, linux, openbsd ... cdsctl will allow you to create, launch, export, import your workflows, monitor your CDS, navigate through your projects, workflows.
No need to go to the UI of CDS or your repository manager to check the status of your commit, `git push && cdsctl workflow --track` will display your workflow in your command line.

#### REST API & SDK

Do you have even more advanced automation needs, or the desire to develop an application that queries CDS? the [REST API](https://ovh.github.io/cds/cli/api/) and the [SDK](https://ovh.github.io/cds/cli/sdk/) will allow you to easily develop your software.

### CDS Administration features

#### Self-Hosting

CDS is open-source since October 2016. You can install it freely in your company or at home. Some tutorials are available to help you start a CDS, [docker-compose](https://ovh.github.io/cds/hosting/ready-to-run/docker-compose/), [Kubernetes with Helm](https://ovh.github.io/cds/hosting/ready-to-run/helm/), [Install with binaries](https://ovh.github.io/cds/hosting/ready-to-run/from-binaries/).

#### High Availability / Scalability / Upgrade without User Downtime

High availability is a very important point for a CI / CD tool. CDS is stateless, nothing is stored on the filesystem. This makes it possible to launch several CDS APIs behind a load balancer. Thus, you can scale the API of CDS to your needs. It also allows upgrades of CDS in full day without impact for users.
In production @OVH, CDS can be updated several times a day, without impacting users or stopping CDS workers.
Asking your users to stop working because updating the Continuous Delivery tool would be ironic, isn't it? ;-)


#### Built-in Metrics

CDS natively exposes monitoring data. You will be able to feed your instance [prometheus](https://prometheus.io/) or [warp10](https://www.warp10.io/) using [beamium](https://github.com/ovh/beamium).

#### Extensibility Plugins

A CDS job consists of steps. Each step is a built-in type action (script, checkoutApplication, Artifact upload / download ...). You can create your own actions, using existing actions - or develop your own action as a plugin. All languages are supported, as long as the language supports GRPC.

#### OS/Arch Compatibility

CDS is agnostic to languages and platforms. Users can launch Jobs on linux, windows, freebsd, osx, raspberry ... in Virtual Machine spawn on demand, in a docker container, on a dedicated host.

So, if your company uses multiple technologies, CDS will not be a blocker for building and deploying your internal software.

#### Auto-Scale OnDemand multi-cloud

One of the initial objectives of CDS at OVH: builder and deploy 150 applications as a container in less than 7 minutes. This has become reality since 2015. What is the secret key?
Auto-Scale on Demand!

Thus, you can have hundreds of workers model and when necessary, CDS will start the workers using the hatcheries.

A hatchery is like an incubator, it gives birth to the CDS Workers and the right of life and death over them.

Several types of hatchery are available: hatchery kubernetes (start workers in pods), **hatchery openstack** (start virtual machines), **hatchery swarm** (start docker containers), **hatchery marathon** (starts docker containers), **hatchery VShpere** (start virtual machines), **hatchery local** ( starts processes on a host). So yes, buzzwords or not, a multi-cloud Auto-scale OnDemand is a reality with CDS :-)




## License

[3-clause BSD](./LICENCE)
