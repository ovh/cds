---
title: GitHub
main_menu: true
---


The Github Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to link a Git Repository hosted by Github
to a CDS Application.

This integration enables some features:

 - [Git Repository Webhook]({{<relref "/docs/concepts/workflow/hooks/git-repo-webhook.md" >}})
 - [Git Repository Poller]({{<relref "/docs/concepts/workflow/hooks/git-repo-poller.md" >}})
 - Easy to use action [CheckoutApplication]({{<relref "/docs/actions/checkoutapplication.md" >}}) and [GitClone]({{<relref "/docs/actions/gitclone.md">}}) for advanced usage
 - Send build notifications on your Pull-Requests and Commits on GitHub
 - Send comments on your Pull-Requests when a workflow is failed



## How to configure GitHub integration

### Create a CDS application on GitHub
Go to https://github.com/settings/developers and **Register a new application**: set an application name, the URL and a description. `Authorization callback URL`: `http(s)://<your-cds-api>/repositories_manager/oauth2/callback`

Example: for a local configuration with API default port (8081), callback URL will be `http://localhost:8081/repositories_manager/oauth2/callback`

On the next page GitHub give you a **Client ID** and a **Client Secret**

### Complete CDS Configuration File

Set value to `clientId` and `clientSecret`

```yaml
    [vcs.servers.Github]

      # URL of this VCS Server
      url = "https://github.com"

      [vcs.servers.Github.github]

        #######
        # CDS <-> GitHub. Documentation on https://ovh.github.io/cds/hosting/repositories-manager/github/
        ########
        # GitHub OAuth Application Client ID
        clientId = "xxxx"

        # GitHub OAuth Application Client Secret
        clientSecret = "xxxx"

        # Does polling is supported by VCS Server
        disablePolling = false

        # Does webhooks are supported by VCS Server
        disableWebHooks = false

        # If you want to have a reverse proxy URL for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK
        # proxyWebhook = ""

        # optional, GitHub Token associated to username, used to add comment on Pull Request
        token = ""

        # optional. GitHub username, used to add comment on Pull Request on failed build.
        username = ""

        [vcs.servers.Github.github.Status]

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

