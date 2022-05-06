---
title: Bitbucket Cloud
main_menu: true
card: 
  name: repository-manager
---

The Bitbucket Cloud Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to link a Git Repository hosted by your Bitbucket Cloud
to a CDS Application.

This integration enables some features:

 - [Git Repository Webhook]({{<relref "/docs/concepts/workflow/hooks/git-repo-webhook.md" >}})
 - Easy to use action [CheckoutApplication]({{<relref "/docs/actions/builtin-checkoutapplication.md" >}}) and [GitClone]({{<relref "/docs/actions/builtin-gitclone.md">}}) for advanced usage
 - Send [build notifications](https://confluence.atlassian.com/bitbucket/check-build-status-in-a-pull-request-945541505.html) on your Pull-Requests and Commits on Bitbucket Cloud. [More informations]({{<relref "/docs/concepts/workflow/notifications.md#vcs-notifications" >}})

## How to configure Bitbucket Cloud integration

- Follow the section **Create a consumer** on documentation https://support.atlassian.com/bitbucket-cloud/docs/use-oauth-on-bitbucket-cloud/  
- Bitbucket requests some informations:
 - **name** you can simply write CDS
 - **description** is optional
 - **callback url** must be the URL of your CDS -> `{CDS_UI_URL}/cdsapi/repositories_manager/oauth2/callback` (if you are in development mode you have to omit /cdsapi and replace {CDS_UI_URL} with your API URL)
 - **URL** is optional.
 - **Permissions** : select `Account Read`, `Workspace membership Read`, `Repositories Read`, `Pull requests Read`, `Webhooks Read and Write`
- Click on Save and toggle the consumer name to see the generated `Key` and `Secret`. It correspond to `clientId` and `clientSecret` in the CDS config.toml file.

### Create the Personal Access Token on Bitbucket Datacenter

On https://bitbucket.org/account/settings/app-passwords/ create a new app password with the following scopes:
 - Account `Email` and `Read`
 - Workspace membership `Read`
 - Projects `Read`
 - Repositories `Read` and `Write`
 - Pull requests `Read` and `Write`
 - Webhooks `Read and write`

### Import configuration

Create a yml file:

```yaml
version: v1.0
name: bitbucket-cloud
type: bitbucketcloud
description: "My Bitbucket Cloud"
auth:
    user: my-user-on-bitbucket-cloud
    token: the-long-token-here
options:
    disableStatus: false    # Set to true if you don't want CDS to push statuses on the VCS server - optional
    showStatusDetail: false # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server - optional
    disablePolling: false   # Does polling is supported by VCS Server - optional
    disableWebHooks: false  # Does webhooks are supported by VCS Server - optional
```

```sh
cdsctl experimental project vcs import YOUR_CDS_PROJECT_KEY vcs-bitbucketcloud.yml
```

## Vcs events

For now, CDS supports push events. CDS uses this push event to remove existing runs for deleted branches (24h after branch deletion).
