## What is CDS?

CDS is a Continuous Delivery solution with an architecture featuring:

 * A complete isolation between tenants
 * High availability oriented architecture
 * Automatic scaling
 * Automation oriented with iso-feature API, CLI and WebUI

Designed for scalability, CDS tasks can run either on cloud infrastructure or on your own machines, should you start some workers using a [hatchery](/doc/overview/hatchery.md).

CDS exposes an API available to [workers](/doc/overview/worker.md) and humans through cli or WebUI.


## What does CDS do?

Basically, CDS allows users to create building and delivery pipelines for all your applications.
An application is composed of one or multiple pipelines, that can be triggered:

  * manually
  * by VCS change
  * by another pipeline

### General

  * Creation of build and deployment pipelines
  * Simple organisation by projects and applications
  * Artifact storage available trough UI, API and CLI
  * Reusable build and deployment Actions

### Packaging

  * Declaration of worker models (specific hosts, docker image, openstack recipe)
  * Conditional build path depending of build parameters (ie: git branch)

### Deployment

  * Completely cross platform workers (built in Go) without any dependency
  * Support for deployment environments (different sets of variable for the same deployment pipeline)


## Basic principles

- It is not possible to enforce where an action will be run. A action will start on any worker **matching all its requirements**.
- **Every action runs in a different worker**, all build data needed for the next step should be **uploaded as artifact** or **run in a joined action**.
- It is possible to run some of your pipelines on-premise, some on CDS workers.

## General organisation

![organisation](/doc/img/project-app-pip-env-simple.png)

With CDS, you can create as many projects to organise your different applications. An application in CDS is a production unit meant to be deployed. To achieve this, you are able to creates pipelines and deployment environment to define how your applications must be build and deployed.

![complete-organisation](/doc/img/project-app-pip-env-complete.png)

On this view, you can see how an application attaches pipelines and environment together to creates a Continuous Delivery pipeline.


### Action requirements and worker capabilities

CDS is built on simples principles:

 * Any client operation is an Action and has requirements.
 * Every worker registered has capabilities and build if and only if all requirements are met.

![Action and Workers](/doc/img/action-worker.png)

Relation between workers and actions.

### Harness PaaS with worker models and [hatcheries](/doc/overview/hatchery.md)

It is possible to register worker models to automatically scale the number of available worker with specific capabilites.

CDS hatcheries start and kill worker model instances following engine orders.


![scaling](/doc/img/hatchery.png)


## Who should use CDS?

People who want to control their build environment for specific applications, while using PaaS infrastructure for basic operations.

CDS is for companies with people working in an ecosystem that need automatic operations on CD solution, capable of mutualising infrastructure and providing autonomy
and isolation between multiple teams inside the same company.

A CD solution where KPI could be extracted easily.

We wanted a CD ecosystem where workers are easy to setup anywhere.

## Next Steps

 * [Run with Docker-Compose](/doc/tutorials/run-with-docker-compose.md)
 * [Quick start](/doc/overview/quickstart.md)
