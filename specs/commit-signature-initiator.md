# Specification: Commit Signature and Workflow Initiator

## Overview

A workflow run started by a repository push runs with the permissions of its initiator. For a push,
the initiator is the CDS user identified from the signature of the pushed commit (or tag). The
signature is the only trusted source of identity: the author or committer written in the commit can be
set to anything by whoever creates it.

This document is the reference for how the initiator is established, what happens when it cannot be,
and what the user is told.

---

## 1 — Trust in a signature

### 1.1 The key ID is not proof

A signature names the key that made it, but that name is metadata written by the signer. A key ID is
trusted only once the signature has been cryptographically verified against that key. An unverified
key ID is never used to identify a user, and is never recorded or forwarded as the commit signing key.

### 1.2 Who verifies

| VCS | Verification |
|---|---|
| GitHub, Gitea, Forgejo, Bitbucket Server | The verdict reported by the VCS for the commit or tag. |
| Bitbucket Cloud, GitLab | CDS checks out the commit and verifies the signature itself, with a public key known to CDS. |

When the VCS reports a reason with its verdict (GitHub, Gitea, Forgejo), the reason is shown to the
user. Bitbucket Server reports no signature at all for a commit signed with a key it does not know, so
such a commit cannot be told apart from an unsigned one.

### 1.3 From a verified key to a user

| Key | Initiator |
|---|---|
| Registered on a CDS user profile | That user. |
| Key of the VCS itself (commits the VCS creates, such as a merge or a rebase done from its web interface) | The committer reported by the VCS, which is the authenticated user who performed the operation. On GitHub and Forgejo, the author is used. |
| Key of a VCS account configured on a project VCS | That VCS account, when it is the committer reported by the VCS. |

A reported committer is mapped to a CDS user as follows:

| VCS | Mapping |
|---|---|
| GitHub, Bitbucket Server, Forgejo | The VCS account must be linked to the CDS user. A matching username is not enough. |
| Gitea, GitLab | The committer username is matched to the CDS username. |
| Bitbucket Cloud | Not supported: only keys registered on a CDS user profile identify a user. |

---

## 2 — Outcomes for a push

An analysis runs for every project that declares the pushed repository. It establishes the initiator
before reading any file. A run triggered by a user (manual analysis) is attributed to that user and
skips the signature check.

| Case | Analysis | Message points to |
|---|---|---|
| Signature verified and mapped to a CDS user | Continues with that initiator | — |
| Commit not signed, or signature not recognized by the VCS | `Skipped` | The VCS account |
| Signature present but not verified by the VCS | `Error` | The VCS account, with the VCS reason |
| Signature verified but not mapped to a CDS user (key not registered in CDS, VCS account not linked, unknown committer) | `Error` | The CDS profile |

An unsigned commit is skipped rather than failed so the event still reaches the workflows of other
repositories allowed to run on it (section 3). A signature that cannot be trusted is an error: it is
typically an attempt to use the workflows of the pushed repository with an identity that is not the
pusher's, and it must be visible.

When an analysis ends in `Error`, the workflows defined in the pushed repository by that project are
skipped with `repository analysis failed: <message>`, and the hook event ends in `Error`.

---

## 3 — Distant workflows

A workflow can listen to pushes on a repository other than its own. Such a repository does not have to
be declared in a CDS project: it then has no analysis, and the hooks service asks the API to establish
the initiator from the commit signature, with the same rules as section 1.

`insecure-skip-signature-verify` lets a distant workflow run when no initiator can be established. It is
rejected on a workflow that listens to its own repository, so it only ever applies to distant
workflows. The status of the analysis of the pushed repository does not change whether a distant
workflow runs.

---

## 4 — Other events

| Event | Initiator |
|---|---|
| Push | From the commit signature, as above. |
| Manual run | The user who requested it. |
| Pull request, pull request comment, scheduler, workflow run, webhook | The owner of the workflow definition. |

---

## 5 — Messages

A skipped workflow or a failed analysis carries the reason, worded so the user knows which tool to fix.

| Case | Message |
|---|---|
| Not signed | `commit <sha> is not signed, or its signature is not recognized by <vcs>: sign it with a GPG key registered on your <vcs> account` |
| Not verified by the VCS | `commit <sha> signature with key <key> is not verified by <vcs> (reason: <reason>): check that the key and the committer email are registered on your <vcs> account` |
| Key unknown to CDS | `signing key <key> is not registered in CDS: add it to your CDS user profile` |
| VCS account not linked (GitHub) | `github user <name> is not linked to a CDS user: link your github account from your CDS profile` |
| VCS account not linked (Bitbucket Server, Forgejo) | `<vcs type> user <name> is not linked to a CDS user` |
| Committer unknown to CDS | `committer "<name>" does not match any CDS user` |

The key ID appears in messages for diagnosis only.
