+++
title = "Github"
weight = 1

[menu.main]
parent = "repositories_manager"
identifier = "repositories_manager_github"

+++

## Authorize CDS on Github
### Create a CDS application on Github
Go to `https://github.com/settings/developers` and **Register a new application**: set an application name, the url and a description. Dont set up `Authorization callback URL`.

On the next page Github give you a **Client ID** and a **Client Secret**

### Connect CDS To Github
With CDS CLI run :

 ```
 $ cds admin reposmanager add GITHUB github http://github.com client-id=<your_client_id> client-secret=<client-secret>
 ```

Set env CDS_VCS_REPOSITORIES_GITHUB_CLIENTSECRET or update your configuration file with `<client-secret>`:

```toml
[vcs.repositories.github]
clientsecret = "<client-secret>"
```

Then restart CDS.

Now check everything is OK with :
 ```
 $ cds admin reposmanager list
 ```
