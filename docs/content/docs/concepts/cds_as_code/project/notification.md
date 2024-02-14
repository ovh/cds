---
title: "Notification"
weight: 5
---

# Description

Each action on CDS triggers an event. It's possible at the project level to setup notification through a webhook, filtering by event.

# Permission

To be able to manage notification you will need the permission `manage` on your project

# Add a notification using CLI

```
cdsctl experimental project notification import <PROJECT-KEY> <notification.yaml>
```
* `PROJECT-KEY`: The project key
* `notification.yaml`: The path to a yaml file name containing the webhook configuration.

## Example of webhook configuration

```
name: my-notif
webhook_url: https://myserver/notif
filters:
  workflowRun: 
    events: [Run.*]
  analysis: 
    events: [AnalysisStart, AnalysisDone]
auth:
  headers:
    Authorization: Bearer ey......
```

* `name`: The name of your notification
* `webhook_url`: URL that CDS will call to POST the notification
* `filters`: A map of named filters
  * `filters.<filter_name>.events`: a list of event which you want to have a notification. You can use regular expression
* `auth.headers`: a map of headers to send