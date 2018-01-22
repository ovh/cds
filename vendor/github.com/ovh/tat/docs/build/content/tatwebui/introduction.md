---
title: "Introduction"
weight: 1
prev: "/tatwebui"
next: "/tatwebui/standardview"
toc: false
---


Tatwebui is a web application, with a nodejs HTTP Server for serving js/html/css files.
This application requests Tat Engine for all actions:

* Rights Controls, Users & Groups
* Topics management, parameters, ACLs
* Manipulate messages

The views are used to display messages in different ways. Views are plugins, see
[Standard View](https://github.com/ovh/tatwebui-plugin-standardview) for example.
Some OVH views are opensourced like:

* [Standard View](/tatwebui/standardview): standard view with all features on messages
* [Notifications View](/tatwebui/notificationsview): display user notifications in /Private/username/Notifications topic
* [Monitoring View](/tatwebui/monitoringview): quick look on many items, UP or Down
* [Release View](/tatwebui/releaseview): Plan, Changelog, Release... communicate with others teams
* [Dashing View](/tatwebui/dashingview): widgets, graph... one way to create dashing about everything
* [Pastat View](/tatwebui/pastatview): a Gist like
* [CDS View](/tatwebui/cdsview): Display [CDS](https://github.com/ovh/cds) Notifications
