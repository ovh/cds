---
title: GitLab Authentication
main_menu: true
card: 
  name: authentication
---

The GitHub Authentication Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to authenticate user with the GitHub Authentication.

## Resume on what you have to do before using the GitHub Authentication Integration

1. As a CDS Administrator: 
  1. Create a CDS application on GitLab
  1. Complete CDS Configuration File

## How to configure GitLab integration

What you need to perform the following steps:

 - GitLab admin privileges

### Create a CDS application on GitLab

In GitLab go to *Settings* / *Application* section. Create a new application with:

 - Name: **CDS**
 - Redirect URI: **http(s)://<your-cds-ui>/auth/callback/gitlab#**

Scopes:

 - API
 - read_user
 - read_registry

### Complete CDS Configuration File

Edit the toml file:

- section `[api.auth.gitlab]`
  - set a value to `applicationID` and `secret`
  - enable the signin with `enabled = true`
  - if you want to let user signup with GitHub, set `signupDisabled = true`
  
```toml
[api.auth.gitlab]

      #######
      # Gitlab OAuth Application ID
      applicationID = ""
      enabled = false

      # Gitlab OAuth Application Secret
      secret = ""
      signupDisabled = false

      #######
      # Gitlab URL
      url = "https://gitlab.com"
```
