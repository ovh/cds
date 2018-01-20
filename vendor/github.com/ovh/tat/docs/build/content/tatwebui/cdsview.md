---
title: "CDS View"
weight: 20
toc: true
prev: "/tatwebui/releaseview"
next: "/tatwebui/development"

---

## Screenshot

![CDS View](/imgs/devops-lifecycle-build-cds.png?width=80%)

## Details of messages sent by CDS

Example: message and replies created by CDS by pipeline building Tat Engine:

```javascript
{
    "_id": "58030253dc55630001c69635",
    "text": "#cds #type:pipelineBuild #project:TAT #app:tat-engine #pipeline:build-go-api-tat #environment:NoEnv #build:74 #idp:297065 #branch:master",
    "topic": "/Internal/CDS/Notifs",
    "inReplyOfID": "",
    "inReplyOfIDRoot": "",
    "nbLikes": 0,
    "labels": [
        {
            "text": "Success",
            "color": "#3c763d"
        }
    ],
    "nbVotesUP": 0,
    "nbVotesDown": 0,
    "tags": [
        "cds",
        "type:pipelineBuild",
        "project:TAT",
        "app:tat-engine",
        "pipeline:build-go-api-tat",
        "environment:NoEnv",
        "build:74",
        "idp:297065",
        "branch:master"
    ],
    "dateCreation": 1476592211.0365698,
    "dateUpdate": 1476592269.024161,
    "author": {
        "username": "tat.system.cds",
        "fullname": "CDS System"
    },
    "replies": [
        {
            "_id": "58030280a41e6200014bf98a",
            "text": "#cds #type:actionBuild #project:TAT #app:tat-engine #pipeline:build-go-api-tat #build:74 #action: #actionId:400588 #idp:297065",
            "topic": "/Internal/CDS/Notifs",
            "inReplyOfID": "58030253dc55630001c69635",
            "inReplyOfIDRoot": "58030253dc55630001c69635",
            "nbLikes": 0,
            "labels": [
                {
                    "text": "Success",
                    "color": "#3c763d"
                }
            ],
            "nbVotesUP": 0,
            "nbVotesDown": 0,
            "tags": [
                "cds",
                "type:actionBuild",
                "project:TAT",
                "app:tat-engine",
                "pipeline:build-go-api-tat",
                "build:74",
                "action:",
                "actionId:400588",
                "idp:297065"
            ],
            "dateCreation": 1476592256.450992,
            "dateUpdate": 1476592272.0423489,
            "author": {
                "username": "tat.system.cds",
                "fullname": "CDS System"
            },
            "nbReplies": 0
        },
        {
            "_id": "5803027f403f9d0001b1af55",
            "text": "#cds #type:actionBuild #project:TAT #app:tat-engine #pipeline:build-go-api-tat #build:74 #action:Packaging #actionId:400588 #idp:297065",
            "topic": "/Internal/CDS/Notifs",
            "inReplyOfID": "58030253dc55630001c69635",
            "inReplyOfIDRoot": "58030253dc55630001c69635",
            "nbLikes": 0,
            "labels": [
                {
                    "text": "Waiting",
                    "color": "#8a6d3b"
                }
            ],
            "nbVotesUP": 0,
            "nbVotesDown": 0,
            "tags": [
                "cds",
                "type:actionBuild",
                "project:TAT",
                "app:tat-engine",
                "pipeline:build-go-api-tat",
                "build:74",
                "action:Packaging",
                "actionId:400588",
                "idp:297065"
            ],
            "dateCreation": 1476592255.230196,
            "dateUpdate": 1476592255.230196,
            "author": {
                "username": "tat.system.cds",
                "fullname": "CDS System"
            },
            "nbReplies": 0
        },
        {
            "_id": "580302566f60b7000148d63d",
            "text": "#cds #type:actionBuild #project:TAT #app:tat-engine #pipeline:build-go-api-tat #build:74 #action:Commit #actionId:400584 #idp:297065",
            "topic": "/Internal/CDS/Notifs",
            "inReplyOfID": "58030253dc55630001c69635",
            "inReplyOfIDRoot": "58030253dc55630001c69635",
            "nbLikes": 0,
            "labels": [
                {
                    "text": "Success",
                    "color": "#3c763d"
                }
            ],
            "nbVotesUP": 0,
            "nbVotesDown": 0,
            "tags": [
                "cds",
                "type:actionBuild",
                "project:TAT",
                "app:tat-engine",
                "pipeline:build-go-api-tat",
                "build:74",
                "action:Commit",
                "actionId:400584",
                "idp:297065"
            ],
            "dateCreation": 1476592214.5933328,
            "dateUpdate": 1476592255.251998,
            "author": {
                "username": "tat.system.cds",
                "fullname": "CDS System"
            },
            "nbReplies": 0
        }
    ],
    "nbReplies": 3
}
```

## Configuration
In plugin.tpl.json file, add this line :

```
"tatwebui-plugin-cdsview": "git+https://github.com/ovh/tatwebui-plugin-cdsview.git"
```

## Source
[https://github.com/ovh/tatwebui-plugin-cdsview](https://github.com/ovh/tatwebui-plugin-cdsview)
