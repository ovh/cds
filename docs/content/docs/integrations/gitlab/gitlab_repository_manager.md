---
title: GitLab Repository Manager
main_menu: true
card: 
  name: repository-manager
---

The GitLab Repository Manager Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to link a Git Repository hosted by GitLab
to a CDS Application.

This integration enables some features:

 - [Git Repository Webhook]({{<relref "/docs/concepts/workflow/hooks/git-repo-webhook.md" >}})
 - Easy to use action [CheckoutApplication]({{<relref "/docs/actions/builtin-checkoutapplication.md" >}}) and [GitClone]({{<relref "/docs/actions/builtin-gitclone.md">}}) for advanced usage
 - Send build notifications on your Pull-Requests and Commits on GitLab. [More informations]({{<relref "/docs/concepts/workflow/notifications.md#vcs-notifications" >}})


### Create the Personal Access Token on GitLab

Generate a new token on https://gitlab.com/-/profile/personal_access_tokens with the following scopes:
 - api
 - read_api
 - read_user
 - read_repository
 - write_repository

### Import configuration

Create a yml file:

```yaml
version: v1.0
name: gitlab
type: gitlab
description: "my gitlab"
auth:
    username: your-username
    token: glpat_your-token-here
options:
    disableStatus: false    # Set to true if you don't want CDS to push statuses on the VCS server - optional
    disableStatusDetails: false # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server - optional
    disablePolling: false   # Does polling is supported by VCS Server - optional
    disableWebHooks: false  # Does webhooks are supported by VCS Server - optional
```

```sh
cdsctl experimental project vcs import YOUR_CDS_PROJECT_KEY vcs-gitlab.yml
```