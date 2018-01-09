+++
title = "Payload"
weight = 2

[menu.main]
parent = "design"
identifier = "design.payload"

+++


A CDS Workflow can be launched :

* manually, user can enter a Payload
* by webhooks / repository webhooks, payload contains the value sent by initiator of the hook
* scheduler, the payload contains the value of the default payload. You can edit the default payload on the root pipeline
* (on roadmap) listener, as a kafka listener. The payload will contain the content of the kafka message

A payload is a JSON value. You can use it inside your CDS Jobs.

Example:

```json
{
  "akey": "valueOfKey",
  "subkey": {
    "akey": "value"
  }
}
```

Two variables are available inside your jobs:

```json
{{.akey}}
```

and

```json
{{.subkey.akey}}
```


![Payload](/images/workflows.design.payload.png)


## Choose a git branch in the payload

On a `git.branch` attribute, you can choose a git branch if you attach on the pipline context an application linked to a Git Repository.

![Payload](/images/workflows.design.payload.gif)