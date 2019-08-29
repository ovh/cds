---
title: "Payload"
weight: 1
card: 
  name: concept_workflow
---


A CDS Workflow can be launched:

* manually, user can enter a Payload
* by webhooks / repository webhooks, payload contains the value sent by initiator of the hook
* scheduler, the payload contains the value of the default payload. You can edit the default payload on the root pipeline
* (on roadmap) listener, as a Kafka listener. The payload will contain the content of the Kafka message

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

**If an application attached to the pipeline context is linked to a Git repository**, you can set `git.branch` attribute to a branch of your choice.

![Payload](/images/workflows.design.payload.gif)
