+++
title = "About CDS"
weight = 1

[menu.main]
parent = "getting-started"
identifier = "about-cds"

+++

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
  * by a hook

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

- A action will start on any worker **matching all its requirements**
- **Every action runs in a different worker**, all build data needed for the next step should be **uploaded as artifact**
- It is possible to run some of your pipelines on-premise, some on CDS workers
