---
title: Bitbucket Server
main_menu: true
card: 
  name: repository-manager
---

The Bitbucket Server Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to link a Git Repository hosted by your Bitbucket Server
to a CDS Application.

This integration enables some features:

 - [Git Repository Webhook]({{<relref "/docs/concepts/workflow/hooks/git-repo-webhook.md" >}})
 - Easy to use action [CheckoutApplication]({{<relref "/docs/actions/builtin-checkoutapplication.md" >}}) and [GitClone]({{<relref "/docs/actions/builtin-gitclone.md">}}) for advanced usage
 - Send [build notifications](https://developer.atlassian.com/server/bitbucket/how-tos/updating-build-status-for-commits/) on your Pull-Requests and Commits on Bitbucket. [More information]({{<relref "/docs/concepts/workflow/notifications.md#vcs-notifications" >}})
 - [Send comments on your Pull-Requests when a workflow is failed]({{<relref "/docs/concepts/workflow/notifications.md#vcs-notifications" >}})

### Create the Personal Access Token on Bitbucket Datacenter

Generate a new token on https://your-bitbucket-datacenter/plugins/servlet/access-tokens/manage with the following scopes:
 - `PROJECT READ`
 - `REPOSITORY READ`

### Import configuration

Create a yml file:

```yaml
version: v1.0
name: bitbucket
type: bitbucketserver
description: "My Bitbucket Datacenter"
url: "http://localhost:7990/bitbucket"
auth:
    user: username-on-bitbucket
    token: the-long-token-here
options:
    disableStatus: false    # Set to true if you don't want CDS to push statuses on the VCS server - optional
    disableStatusDetails: false # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server - optional
    disablePolling: false   # Does polling is supported by VCS Server - optional
    disableWebHooks: false  # Does webhooks are supported by VCS Server - optional
```

```sh
cdsctl project vcs import YOUR_CDS_PROJECT_KEY vcs-bitbucket.yml
```

## Vcs events

For now, CDS supports push events. CDS uses this push event to remove existing runs for deleted branches (24h after branch deletion).