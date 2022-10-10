---
title: GitHub Authentication
main_menu: true
card: 
  name: authentication
---

The GitHub Authentication Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to authenticate user with the GitHub Authentication.

## Resume on what you have to do before using the GitHub Authentication Integration

1. As a CDS Administrator: 
  1. Create a CDS application on GitHub
  1. Complete CDS Configuration File

## How to configure GitHub Authentication integration

### Create a CDS application on GitHub

*As a CDS Administrator* 

Go to https://github.com/settings/developers and **Register a new OAuth application**. Set :

- an `application name`, example: `CDS company name`
- the `Homepage URL`: `http(s)://<your-cds-webui>` (with a local webui, it will be: `http://localhost:8080`)
- the `Authorization callback URL`: `http(s)://<your-cds-ui>/auth/callback/github#`

Example for a local configuration:

- with WebUI default port (8080)
 - Homepage URL will be `http://localhost:8080`
 - Callback URL will be `http://localhost:8080/auth/callback/github#`

![Integration GitHub New OAuth App](../../images/github-new-oauth-app.png?height=500px)

Click on **Register Application**, then on the next page, GitHub give you a **Client ID** and a **Client Secret**

### Complete CDS Configuration File

Edit the toml file:

- section `[api.auth.github]`
  - set a value to `clientId` and `clientSecret`
  - enable the signin with `enabled = true`
  - if you want to disable signup with GitHub, set `signupDisabled = true`
  
```toml
[api.auth.github]

      #######
      # GitHub API URL
      apiUrl = "https://api.github.com"

      #######
      # GitHub OAuth Client ID
      clientId = "xxxx"

      # GitHub OAuth Client Secret
      clientSecret = "xxxx"
      enabled = true
      signupDisabled = false

      #######
      # GitHub URL
      url = "https://github.com"
```