---
title: "Introduction"
weight: 1
prev: "/overview"
next: "/overview/concepts"
toc: false

---

Tat, aka Text And Tags, is a communication tool - Human & Robot all together.

Some use cases:

- Viewing Pull Requests, Build And Deployment in one place
- Alerting & Monitoring Overview
- Agile view as simple as a whiteboard with post-it
- Team Communication & Reporting facilities
...

Tat Engine exposes only an HTTP REST API. You can manipulate this API with Tat Command Line Interface, aka tatcli, see https://github.com/ovh/tat/tatcli.

A WebUI is also available, see https://github.com/ovh/tatwebui.

Tat Engine:

- Uses MongoDB as backend
- Is fully stateless, scale as you want
- Is the central Hub of Tat microservices ecosystem

The initial goal of TAT was to make an overview on Continuous Delivery Pipeline, with some pre-requisites:

- Scalable, High Availability, Self-Hosted
- API, CLI
- Simple Usage

![DevOps LifeCycle](/imgs/tat-cd-overview.png?width=700px)
