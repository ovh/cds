# CDS Secret Backend Vault

The goal of this extension is to able CDS to retrieve its keys and secrets from Vault.

- [CDS](https://github.com/ovh/cds)
- [Vault](https://www.vaultproject.io)

## How to install

Make sure go >=1.7 is installed and properly configured ($GOPATH must be set)

```shell
    $ mkdir -p $GOPATH/src/github/ovh/cds
    $ git clone https://github.com/ovh/cds $GOPATH/src/github/ovh/cds
    $ cd $GOPATH/src/github/ovh/cds/contrib/secret-backends/secret-backend-vault
    $ go test ./...
    $ go install
```

## How to use

*/!\ Here we consider that you already know Vault concepts, and your Vault is properly configured and setup.*

### CDS Setup

Set `secret-backend-vault` in CDS startup options, with following `secret-backend` options :

- `vault_addr` : Your Vault address. If not set, environment variable `VAULT_ADDR` is used.
- `vault_token` : Your Vault token (don't use root token in production). If not set, environment variable `VAULT_TOKEN` is used.
- `vault_namespace` : The Vault path  used for CDS. It **must** have `cds` suffix (eg. `/secret/cds`).

Sample usage :

```shell
    $GOPATH/bin/cds-api [...] \
        --secret-backend $GOPATH/bin/secret-backend-vault \
        --secret-backend-option "vault_addr=https://vault.mydomain.net:8200 vault_token=09d1f099-3d41-666e-8337-492226789599 vault_namespace=/secret/cds"
```

### Vault Setup

- You Vault has to be unsealed before CDS startup.
- As we see earlier, all CDS secrets must be in path with `cds` as suffix.
- CDS on startup only need `read` policy. COnfigure your ACL in order to set a CDS token with this policy.
- See [CDS documentation](https://github.com/ovh/cds) to know the secrets your have to set in Vault.

#### Sample usage with a Vault dev Server

Start a server in `dev` mode

```shell
    vault server -dev
    ==> Vault server configuration:

                    Backend: inmem
                Listener 1: tcp (addr: "127.0.0.1:8200", cluster address: "", tls: "disabled")
                Log Level: info
                    Mlock: supported: true, enabled: false
                    Version: Vault v0.6.2

    ==> WARNING: Dev mode is enabled!

    In this mode, Vault is completely in-memory and unsealed.
    Vault is configured to only have a single unseal key. The root
    token has already been authenticated with the CLI, so you can
    immediately begin using the Vault CLI.

    The only step you need to take is to set the following
    environment variables:

        export VAULT_ADDR='http://127.0.0.1:8200'

    The unseal key and root token are reproduced below in case you
    want to seal/unseal the Vault or play with authentication.

    Unseal Key: 6VSHVWqZnkjElsMRmvSySdjjPAPJvWaFxPztjpKx/84=
    Root Token: 09d1f099-3d41-666e-8337-492226789599

    ==> Vault server started! Log data will stream in below:

    2016/10/20 16:20:52.685139 [INFO ] core: security barrier not initialized
    2016/10/20 16:20:52.687082 [INFO ] core: security barrier initialized: shares=1 threshold=1
    2016/10/20 16:20:52.687525 [INFO ] core: post-unseal setup starting
    2016/10/20 16:20:52.694753 [INFO ] core: successfully mounted backend: type=generic path=secret/
    2016/10/20 16:20:52.694783 [INFO ] core: successfully mounted backend: type=cubbyhole path=cubbyhole/
    2016/10/20 16:20:52.695136 [INFO ] core: successfully mounted backend: type=system path=sys/
    2016/10/20 16:20:52.695234 [INFO ] rollback: starting rollback manager
    2016/10/20 16:20:52.698508 [INFO ] core: post-unseal setup complete
    2016/10/20 16:20:52.699525 [INFO ] core: root token generated
    2016/10/20 16:20:52.699529 [INFO ] core: pre-seal teardown starting
    2016/10/20 16:20:52.699537 [INFO ] rollback: stopping rollback manager
    2016/10/20 16:20:52.699543 [INFO ] core: pre-seal teardown complete
    2016/10/20 16:20:52.699581 [INFO ] core: vault is unsealed
    2016/10/20 16:20:52.699592 [INFO ] core: post-unseal setup starting
    2016/10/20 16:20:52.699677 [INFO ] core: successfully mounted backend: type=generic path=secret/
    2016/10/20 16:20:52.699685 [INFO ] core: successfully mounted backend: type=cubbyhole path=cubbyhole/
    2016/10/20 16:20:52.699727 [INFO ] core: successfully mounted backend: type=system path=sys/
    2016/10/20 16:20:52.699795 [INFO ] rollback: starting rollback manager
    2016/10/20 16:20:52.700100 [INFO ] core: post-unseal setup complete

```

```shell
    export VAULT_ADDR='http://127.0.0.1:8200'
    export VAULT_TOKEN='09d1f099-3d41-666e-8337-492226789599'
    # Set the CDS AES Key
    vault write /secret/cds/aes-key aes-key=66eKVxCGLm6gwoH9LAQ66ZD1AOABo1XF
    # Set the CDS Github client-secret
    vault write /secret/repositoriesmanager-secrets-github-client-secret repositoriesmanager-secrets-github-client-secret=8ed279e27119a85f990e82c7f0b895dd193c6666
```
