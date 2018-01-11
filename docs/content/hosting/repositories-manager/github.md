+++
title = "Github"
weight = 1

+++

## Authorize CDS on Github
### Create a CDS application on Github
Go to `https://github.com/settings/developers` and **Register a new application**: set an application name, the url and a description. `Authorization callback URL`: `http(s)://<your-cds-api>/repositories_manager/oauth2/callback`

On the next page Github give you a **Client ID** and a **Client Secret**

### Connect CDS To Github

Set env CDS_VCS_REPOSITORIES_GITHUB_CLIENTSECRET or update your configuration file with `<client-secret>`:

```toml
[vcs.repositories.github]
clientsecret = "<client-secret>"
```

**Then restart CDS**

With CDS CLI run :

```bash
$ cds admin reposmanager add GITHUB github http://github.com client-id=<your_client_id>
```

Now check everything is OK with :
```bash
$ cds admin reposmanager list
```
