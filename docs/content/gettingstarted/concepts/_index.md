+++
title = "Concepts"
weight = 20

+++

## What is CDS?

Enterprise-Grade Continuous Delivery & DevOps Automation Open Source Platform

 - Easy to deploy
 - Cloud native
 - Scalable & Extensible
 - Infrastructure & Architecture agnostic
 - Event-Driven Pipeline

Designed for scalability, CDS tasks can run either on cloud infrastructure or on your own machines, should you start some workers using a [hatchery]({{< relref "hatchery/_index.md" >}}).

CDS exposes an API available to [workers]({{< relref "worker/_index.md" >}}) and humans through CLI or WebUI.

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

  * Declaration of worker models (specific hosts, docker image, OpenStack recipe)
  * Conditional build path depending on build parameters (ie: git branch)

### Deployment

  * Completely cross platform workers (built in Go) without any dependency
  * Support for deployment environments (different sets of variables for the same deployment pipeline)


## Basic principles

- An action will start on any worker **matching all its requirements**
- **Every action runs in a different worker**, all build data needed for the next step should be **uploaded as artifact**
- It is possible to run some of your pipelines on-premise, some on CDS workers


