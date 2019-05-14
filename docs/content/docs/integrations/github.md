---
title: GitHub
main_menu: true
card: 
  name: repository-manager
---

The Github Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to link a Git Repository hosted by Github to a CDS Application.

This integration enables some features:

 - [Git Repository Webhook]({{<relref "/docs/concepts/workflow/hooks/git-repo-webhook.md" >}})
 - [Git Repository Poller]({{<relref "/docs/concepts/workflow/hooks/git-repo-poller.md" >}})
 - Easy to use action [CheckoutApplication]({{<relref "/docs/actions/checkoutapplication.md" >}}) and [GitClone]({{<relref "/docs/actions/gitclone.md">}}) for advanced usage
 - Send build notifications on your Pull-Requests and Commits on GitHub
 - Send comments on your Pull-Requests when a workflow is failed

## Resume on what you have to do before using the GitHub Integration

1. As a CDS Administrator: 
  1. Create a CDS application on GitHub
  1. Complete CDS Configuration File
  1. Start the vcs µService
1. As a user, which is admin on a CDS Project: link Project to GitHub
1. As a user, with writing rights on a CDS project: 
  1. Link a CDS Application to a Git repository
  1. Add a repository webhook on a workflow (this will automatically create a webhook on GitHub)

## How to configure GitHub integration

### Create a CDS application on GitHub

*As a CDS Administrator* 

Go to https://github.com/settings/developers and **Register a new OAuth application**. Set :

- an `application name`, example: `CDS company name`
- the `Homepage URL`: `http(s)://<your-cds-webui>` (with a local webui, it will be: `http://localhost:4200`)
- the `Authorization callback URL`: `http(s)://<your-cds-api>/repositories_manager/oauth2/callback`

Example for a local configuration:

- with WebUI default port (4200), Homepage URL will be `http://localhost:4200`
- with API default port (8081), callback URL will be `http://localhost:8081/repositories_manager/oauth2/callback`

![Integration Github New OAuth App](../images/github-new-oauth-app.png?height=500px)

Click on **Register Application**, then on the next page, GitHub give you a **Client ID** and a **Client Secret**

### Complete CDS Configuration File

#### VCS µService Configuration

The file configuration for the VCS µService can be retreived with:

```bash
$ engine config new vcs > vcs-config.toml

# or with all other configuration parts:
$ engine config new > config.toml
```

Edit the toml file:

- section `[vcs]`
  - the URL will be used by CDS API to reach this µService
  - add a name, as `cds-vcs`. Without a name, the service VCS will not start
  
```toml
[vcs]
  URL = "http://localhost:8084"

  # Name of this CDS VCS Service
  # Enter a name to enable this service
  name = "cds-vcs"
```

- section `[vcs.UI.http]`
  - URL of CDS UI. This URL will be used by GitHub as a callback on Oauth2. This url must be accessible by users' browsers.
  
```toml
    [vcs.UI.http]
      url = "http://localhost:4200"
```

- section `[vcs.api]`
  - this section will be used to communicate with CDS API. Check the url and enter a shared.infra token.
  - Token can be generated with cdsctl: `cdsctl token generate shared.infra persistent`.

```toml
  [vcs.api]
    maxHeartbeatFailures = 10
    requestTimeout = 10
    token = "enter sharedInfraToken from section [api.auth] here"

    [vcs.api.grpc]
      # insecure = false
      url = "http://localhost:8082"

    [vcs.api.http]
      # insecure = false
      url = "http://localhost:8081"
```

- section `[vcs.servers.Github]`
  - Set a value to `clientId` and `clientSecret`

```toml
   [vcs.servers.Github]

      # URL of this VCS Server
      url = "https://github.com"

      [vcs.servers.Github.github]

        #######
        # CDS <-> Github. Documentation on https://ovh.github.io/cds/hosting/repositories-manager/github/
        ########
        # Github OAuth Application Client ID
        clientId = "xxxx"

        # Github OAuth Application Client Secret
        clientSecret = "xxxx"

        # Does polling is supported by VCS Server
        disablePolling = false

        # Does webhooks are supported by VCS Server
        disableWebHooks = false

        # If you want to have a reverse proxy url for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK
        # proxyWebhook = ""

        # optional, Github Token associated to username, used to add comment on Pull Request
        token = ""

        # optional. Github username, used to add comment on Pull Request on failed build.
        username = ""

        [vcs.servers.Github.github.Status]

          # Set to true if you don't want CDS to push statuses on the VCS server
          # disable = false

          # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server
          # showDetail = false
```

#### hooks µService Configuration

As the `vcs` µService, you have to configured the `hooks` µService

```bash
$ engine config new hooks > hooks-config.toml
```

In the `[hooks]` section

- check the URL, this will be used by CDS API to call CDS Hooks
- configure `urlPublic` if you want to use [simple Webhook]({{<relref "/docs/concepts/workflow/hooks/webhook.md">}})
- add a name, as `cds-hooks`

In the `[hooks.api]` section

- put the same token as the `[vcs.api]` section


### Start the vcs and hooks µService

*As a CDS Administrator* 

```bash
$ engine start vcs --config vcs-config.toml
$ engine start hooks --config hooks-config.toml

# you can also start CDS api and vcs in the same process:
$ engine start api vcs hooks --config config.toml
```

### Link Project to GitHub

*As a user, which is admin on a CDS Project*

Go on your **CDS Project -> Advanced -> Link a repository manager**

Select Github in the list, then click on **Connect**

![github-add-on-prj.png](../images/github-add-on-prj.png?height=500px)

A confirmation page is now displayed, click on the link **Click Here**

![github-modal-click-here.png](../images/github-modal-click-here.png?height=200px)

By clicking on **Click Here**, you call GitHub and you will be redirected on the same page.
The Repository is now added on the CDS Project with a small warning
 *Unused repository manager Github on ..* as you don't use the repository yet in your CDS Project. 

![github-added.png](../images/github-added.png?height=200px)


### Link a CDS Application to a Git repository

*As a user, with writing rights on a CDS project* 

Go on your **CDS Project -> You Application -> Advanced -> Link application to a repository**

Select GitHub, then select a Git Repository

![github-app-repo.png](../images/github-app-repo.png?height=500px)

The application is linked, you have now to choose a method to Git Clone your repository.

![github-app-freshly-added.png](../images/github-app-freshly-added.png?height=200px)

Example with `https` method, without authentication:

![github-app-repo-configured.png](../images/github-app-repo-configured.png?height=200px)

### Add a repository webhook on a workflow

*As a user, with writing rights on a CDS project* 

Select the first pipeline, then click on `Add a hook` in the sidebar.

![github-wf-select-pipeline.png](../images/github-wf-select-pipeline.png?height=500px)

Select the **RepositoryWebhook**, then click on **Save**.

![github-wf-add-repowebhook.png](../images/github-wf-add-repowebhook.png?height=200px)

The webhook is automatically created on GitHub. 

## What's next?

- Use [CheckoutApplication]({{<relref "/docs/actions/checkoutapplication.md">}}) in your pipeline
- `git push` on your Git Repository on GitHub
- See the build status on GitHub.

## FAQ

### **My CDS is not accessible since GitHub, how can I do?**

When someone git push on your Git Repository, GitHub have to call your CDS to run your CDS Workflow.
This is the behaviour of the [RepositoryWebhook]({{<relref "/docs/concepts/workflow/hooks/git-repo-webhook.md">}}). But if your CDS is not reacheable from GitHub, how can you do?

By chance, you have two choices :) 

- When you add a Hook on your workflow, select the **Git Repository Poller**. The µService Hooks
will call regularly GitHub. With this way, GitHub doesn't need to call your CDS.

[Git Repository Poller documentation]({{<relref "/docs/concepts/workflow/hooks/git-repo-poller.md">}})

- But if you prefer use the WebHook, you can configure a Reverse Proxy and set the URL in the `[vcs.servers.Github.github]` section

```toml
    # If you want to have a reverse proxy url for your repository webhook, for example if you put https://myproxy.com it will generate a webhook URL like this https://myproxy.com/UUID_OF_YOUR_WEBHOOK
    # proxyWebhook = ""
```

If you hesitate between the two: the `RepositoryWebhook` is more *reactive* than the `Git Repository Poller`.

### **I don't see the type Git Repository Poller nor RepositoryWebhook when I add a Hook**

Before adding a hook on your Workflow, you have to add the application in the Pipeline Context.
Select the first pipeline, then click on **Edit the pipeline context** from the [sidebar]({{<relref "/docs/concepts/workflow/sidebar.md">}}).

[Pipeline Context Documentation]({{<relref "/docs/concepts/workflow/pipeline-context.md">}})

## Vcs events

For now, CDS supports push events. CDS uses this push event to remove existing runs for deleted branches (24h after branch deletion).
