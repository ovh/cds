# go-stash

go bindings for a minimal set of Stash REST APIs

Initially to provide Stash support to [drone.io](https://github.com/drone/drone) and heavily borrowed from [go-bitbucket](https://github.com/drone/go-bitbucket)


## Testing

### Setup Stash

One way to get that up and running is to make use of the atlassian/stash docker
[atlassian/stash](https://registry.hub.docker.com/u/atlassian/stash/)

If you use the above Stash instance, then access the Stash instance at `http://localhost:7990/` and following the setup instructions.

#### Setup Test Project and Repo

Setup a test project and repo, and add a `README.md` file in a branch labeled `master`.

#### Install Add On

In Administration Add-On section install the `HTTP Request Post-Receive Hook for Stash` add on.

#### Setup Application Link

In Administration Settings section, configure an Application Link.

#### Step 1.

    Application Name: go-stash
    Application Type: Generic Application
    Create Incoming Link: checked

#### Step 2.

    Consumer Key: <consumer_key>
    Consumer Name: <consumer_name>
    Public Key: <public_key>


#### Setup Enviroment Variables

If you do not have the `STASH_ACCESS_TOKEN` and `STASH_ACCESS_SECRET`, you can set those with the `make auth` command.

Follow the instructions provided and you will be provided with `STASH_ACCESS_TOKEN` and `STASH_ACCESS_SECRET` values that you can then use.


    export STASH_URL="<stash_url>"
    export STASH_PROJECT="<stash_project>"
    export STASH_REPO="<stash_repo>"
    export STASH_USER="<stash_user>"
    export STASH_PASSWORD="<stash_password>"
    export STASH_CONSUMER_KEY="<consumer_key>"
    export STASH_HOOK="de.aeffle.stash.plugin.stash-http-get-post-receive-hook%3Ahttp-get-post-receive-hook"
    export STASH_PRIVATE_KEY="<stash_private_key>"
    export STASH_ACCESS_TOKEN="<stash_access_token>"
    export STASH_ACCESS_SECRET="<stash_access_secret>"
