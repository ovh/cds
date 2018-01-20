---
layout: "install"
page_title: "Upgrading to Vault 0.6"
sidebar_current: "docs-install-upgrade-to-0.6"
description: |-
  Learn how to upgrade to Vault 0.6
---

# Overview

This page contains the list of breaking changes for Vault 0.6. Please read it
carefully.

Please note that this includes the full list of breaking changes __since Vault
0.5__. Some of these changes were introduced in later releases in the Vault
0.5.x series.

## PKI Backend Disallows RSA Keys < 2048 Bits

The PKI backend now refuses to issue a certificate, sign a CSR, or save a role
that specifies an RSA key of less than 2048 bits. Smaller keys are considered
unsafe and are disallowed in the Internet PKI. Although this may require
updating your roles, we do not expect any other breaks from this; since its
inception the PKI backend has mandated SHA256 hashes for its signatures, and
software able to handle these certificates should be able to handle
certificates with >= 2048-bit RSA keys as well.

## PKI Backend Does Not Automatically Delete Expired Certificates

The PKI backend now does not automatically delete expired certificates,
including from the CRL. Doing so could lead to a situation where a time
mismatch between the Vault server and clients could result in a certificate
that would not be considered expired by a client being removed from the CRL.

Vault strives for determinism and putting the operator in control, so expunging
expired certificates has been moved to a new function at `pki/tidy`. You can
flexibly determine whether to tidy up from the revocation list, the general
certificate storage, or both. In addition, you can specify a safety buffer
(defaulting to 72 hours) to ensure that any time discrepancies between your
hosts is accounted for.

## PKI Backend Does Not Issue Leases for CA Certificates

When a token expires, it revokes all leases associated with it. This means that
long-lived CA certs need correspondingly long-lived tokens, something that is
easy to forget, resulting in an unintended revocation of the CA certificate
when the token expires. To prevent this, root and intermediate CA certs no
longer have associated leases. To revoke these certificates, use the
`pki/revoke` endpoint.

CA certificates that have already been issued and acquired leases will report
to the lease manager that revocation was successful, but will not actually be
revoked and placed onto the CRL.

## Cert Authentication Backend Performs Client Checking During Renewals

The `cert` backend now performs a variant of channel binding at renewal time
for increased security. In order to not overly burden clients, a notion of
identity is used, as follows:

* At both login and renewal time, the validity of the presented client
  certificate is checked
* At login time, the key ID of both the client certificate and its issuing
  certificate are stored
* At renewal time, the key ID of both the client certificate and its issuing
  certificate must match those stored at login time

Matching on the key ID rather than the serial number allows tokens to be
renewed even if the CA or the client certificate used are rotated; so long as
the same key was used to generate the certificate (via a CSR) and sign the
certificate, renewal is allowed. As Vault encourages short-lived secrets,
including client certificates (for instance, those issued by the `pki`
backend), this is a useful approach compared to strict issuer/serial number
checking.

You can use the new `cert/config` endpoint to disable this behavior.

## The `auth/token/revoke-prefix` Endpoint Has Been Removed

As part of addressing a minor security issue, this endpoint has been removed in
favor of using `sys/revoke-prefix` for prefix-based revocation of both tokens
and secrets leases.

## Go API Uses `json.Number` For Decoding

When using the Go API, it now calls `UseNumber()` on the decoder object. As a
result, rather than always decode as a `float64`, numbers are returned as a
`json.Number`, where they can be converted, with proper error checking, to
`int64`, `float64`, or simply used as a `string` value. This fixes some display
errors where numbers were being decoded as `float64` and printed in scientific
notation.

## List Operations Return `404` On No Keys Found

Previously, list operations on an endpoint with no keys found would return an
empty response object. Now, a `404` will be returned instead.

## Consul TTL Checks Automatically Registered

If using the Consul HA storage backend, Vault will now automatically register
itself as the `vault` service and perform its own health checks/lifecycle
status management. This behavior can be adjusted or turned off in Vault's
configuration; see the
[documentation](https://www.vaultproject.io/docs/config/index.html#check_timeout)
for details.
