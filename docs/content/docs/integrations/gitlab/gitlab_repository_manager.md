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


## How to configure GitLab integration

What you need to perform the following steps:

 - GitLab admin privileges

### Create a CDS application on GitLab

Notice: if you have already a CDS Application in GitLab for CDS Authentication, you can't reuse it for Repository Manager.

In GitLab go to *Settings* / *Application* section. Create a new application with:

 - Name: **CDS VCS**
 - Redirect URI: **https://your-cds-api/repositories_manager/oauth2/callback**

Example for a local configuration:
- with API through /cdsapi proxy on ui, Redirect URI will be `http://localhost:8080/cdsapi/repositories_manager/oauth2/callback`

Scopes:

 - API
 - read_user
 - read_registry

### Complete CDS Configuration File

Set value to `appId` and `secret`


```yaml
    [vcs.servers.Gitlab]

      # URL of this VCS Server
      url = "https://gitlab.com"

      [vcs.servers.Gitlab.gitlab]

        #######
        # CDS <-> GitLab. Documentation on https://ovh.github.io/cds/hosting/repositories-manager/gitlab/
        ########
        appId = "xxxx"

        # Does polling is supported by VCS Server
        disablePolling = false

        # Does webhooks are supported by VCS Server
        disableWebHooks = false

        # If you want to have a reverse proxy URL for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK
        # proxyWebhook = ""
        secret = "xxxx"

        [vcs.servers.Gitlab.gitlab.Status]

          # Set to true if you don't want CDS to push statuses on the VCS server
          # disable = false

          # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server
          # showDetail = false
```


## Start the vcs µService

```bash
$ engine start vcs

# you can also start CDS api and vcs in the same process:
$ engine start api vcs
```
