+++
title = "GitTag"
chapter = true

+++

**GitTag** is a builtin action, you can't modify it.

This action creates a tag. You can use a pgp key to sign it.

## Parameters

* url - mandatory - URL must contain information about the transport protocol, the address of the remote server, and the path to the repository.
* authPrivateKey - optional - the private key to be able to git tag from ssh
* user - optional - the user to be able to git tag from https with authentication
* password - optional - the password to be able to git tag from https with authentication
* tagName - optional - Name of the tag you want to create. If empty, it will make a patch version from your last tag.
* tagMessage - optional - Message for the tag
* path - optional - path to your git repository
* signKey - optional - pgp key to sign the tag

