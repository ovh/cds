---
layout: "install"
page_title: "Upgrade Vault"
sidebar_current: "docs-install-upgrade"
description: |-
  Learn how to upgrade Vault.
---

# Upgrading Vault

These are general upgrade instructions for Vault for both non-HA and HA setups.
Please ensure that you also read the version-specific upgrade notes for the
version you are upgrading to.

## Non-HA Installations

Upgrading non-HA installations of Vault is as simple as replacing the Vault
binary with the new version and restarting Vault. Any upgrade tasks that can be
performed for you will be taken care of when Vault is unsealed.

Always use `SIGINT` or `SIGTERM` to properly shut down Vault.

Be sure to also read and follow any instructions in the version-specific
upgrade notes.

## HA Installations

This is our recommended upgrade procedure, and the procedure we use internally
at HashiCorp. However, you should consider how to apply these steps to your
particular setup since HA setups can differ on whether a load balancer is in
use, what addresses clients are being given to connect to Vault (standby +
leader, leader-only, or discovered via service discovery), etc.

Please note that Vault does not support true zero-downtime upgrades, but with
proper upgrade procedure the downtime should be very short (a few hundred
milliseconds to a second depending on how the speed of access to the storage
backend).

Perform these steps on each standby:

1. Properly shut down Vault on the standby node via `SIGINT` or `SIGTERM`
2. Replace the Vault binary with the new version
3. Start the standby node
4. Unseal the standby node

At this point all standby nodes will be upgraded and ready to take over. The
upgrade will not be complete until one of the upgraded standby nodes takes over
active duty. To do this:

1. Properly shut down the remaining (active) node. Note: it is _**very
   important**_ that you shut the node down properly. This causes the HA lock to
   be released, allowing a standby node to take over with a very short delay.
   If you kill Vault without letting it release the lock, a standby node will
   not be able to take over until the lock's timeout period has expired. This
   is backend-specific but could be ten seconds or more.
2. Replace the Vault binary with the new version
3. Start the node
4. Unseal the node (it will now be a standby)

Internal upgrade tasks will happen after one of the upgraded standby nodes takes over active duty.

Be sure to also read and follow any instructions in the version-specific
upgrade notes.
