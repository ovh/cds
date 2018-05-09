+++
title = "CDS - The origins"
weight = 1

+++

To build CDS, the development team took their inspiration from the Continuous Delivery principles:

- The process for releasing/deploying software MUST be repeatable and reliable.
- Automate everything!
- If somethings difficult or painful, do it more often
- Keep everything in source control
- Dev Done means “released”
- Build quality in!
- Everybody has responsibility for the release process
- Improve continuously
- Build binaries only once
- Use precisely the same mechanism to deploy to every environment
- Smoke test your deployment
- If anything fails, stop the line

ref. http://devopsnet.com/2011/08/04/continuous-delivery/

The team triggered the project after several dissatisfying attempts to integrate and use other open-source and commercial build tools at large scale.
Even if most of the tested tools did not contradict the principles above, they were failing at the "real world" test. In fact, their limitations were quickly visible at scale. As the number of managed projects grew, these tools were showing signs of weakness of became hard or expensive to manage and maintain. Basic software updates of the tools themselves needed preparation and downtime. Basic configuration changes needed to be centralized on a single team thus creating an organizational bottleneck. etc.

This is why, at the start of the project, the team knew exactly the requirements that would make CDS a suitable tool for their daily operation.
These requirements later became the ***CDS Building Design Principles*** listed below.

## 1 - Self-Service

In massively distributed architectures, development teams work independently to improve and deploy applications and services.

Growth in the number of teams and projects create some interesting dynamics where a handful of projects are launched every week. Some of them die at the proof-of concept stage while some others do survive and bring value. Of course, such a turnover rate comes with an ever-growing whishlist of build, test and deployment environments.

Centralizing the creation and the configuration of these deployments on a single team is considered harmful. At best, it would create an organizational bottleneck where the Continuous Delivery is overwhelmed with requests and delays their execution. At worst, this multi-layered/multi-team process would look too heavy from the outside and push the developers to censor themselves putting a soft break to the innovation dynamics.

CDS is built around a strong culture of self-service: Whenever it is possible, the control is delegated to the development teams and to the ops teams. Creation, configuration and deletion of CDS projects is completely decentralized. Moreover, project creators can re-delegate parts of their permissions to their teams through powerful built-in group ACLs.

Users are also free to add their own workers to the system if the workers provided by the Continuous Delivery team do not suit their needs. This covers the specific cases where specific hardware or software is required to build or test a software. To do so, users just need to start the CDS worker binary from their own machines and give it the IP address and the credentials of the CDS API. A worker CDS is a simple binary, you do not need libraries or particular JVMs on your machine to run it. CDS workers are compatible with Linux, Darwin, OSX, FreeBSD as well as Windows, in architectures 386, amd64 or arm.

To implement this strong *self-service culture* the team files an issue everytime a user needs the help of a CDS administrator to achieve a simple day-to-day task.

## 2 - Horizontal Scalability

CDS is built to scale. And this capability is challenged everyday in a large-scale production environment. This ability to scale has been made possible thanks to a simple design principle: **statelessness**

CDS's API servers are completely stateless. They do not store anything on the fileSystem. With this "share-nothing" architecture, servers can be deployed as much times as reauired to support the load. Instances can be spawned and decommissioned dynamically to handle usage surges when required while keeping the cost at its lowest when the platform is underused. All you need to provide is a scalable and highly available database.


## 3 - High Availability

When working with a continuous delivery tool on a daily basis using an actively maintained tool like CDS, updates are frequent.
For example, it is frequent that OVH's main CDS instance gets updated and redeployed several times a day.

This fast delivery and reactivity couldn't have been possible without CDS's High Availability architecture. The **statelessnes** property, described above, is again the preperty that allows to update API servers idependently without interrupting any of the running jobs.

« The loss of a CDS API server is a non-event ».

## 4 - Pipeline Reutilisability

CDS's built-in don't-repeat-yourself features help you minimize your effort when you need to build, test and deploy hundreds (thousands?) of projects with a similar workflow. This is especially useful when you are managing a micro-services-based infrastructure where containers are usually built and deployed the same way.

A system of templating allows to customize the build runs depending on the apps to be built and on the deployment environments.

This feature is essential as it allows to quickly deploy new applications, provided that a similar one already exists in CDS.

## 5 - REST API

CDS can be fully operated through its REST API. The API is used by the CDS's UI but also by the workers. All components speak the same language: REST.

Both the UI and the CLI use exclusively the REST API to operate. Therefore, if you can do it through the UI or the CLI, then you can do it through the REST API.

« Everything must be scriptable, automatable ».


## 6 - Customizable

CDS is shipped with a lot of built-in steps and hatcheries that should most users' needs. But, power users will want to customize it to suit their own needs. This is why CDS has been designed to accept plug-ins for steps actions and the REST API-based hatchery operations allow and easy addition of customized hatcheries.