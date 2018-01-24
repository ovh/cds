# Changelog

This changelog tracks the releases of docker-credential-helpers.
This project includes different binaries per platform.
The platform released is identified after the tag name.

## v0.4.0 (Go client, Mac OS X, Windows, Linux)

- Full implementation for OSX ready
- Fix some windows issues
- Implement client.List, change list API
- mac: delete credentials before adding them to avoid already exist error (fixes #37)


## v0.3.0 (Go client)

- Add Go client library to talk with the native programs.

## v0.2.0 (Mac OS X, Windows, Linux)

- Initial release of docker-credential-secretservice for Linux.
- Use new secrets payload introduced in https://github.com/docker/docker/pull/20970.

## v0.1.0 (Mac OS X, Windows)

- Initial release of docker-credential-osxkeychain for Mac OS X.
- Initial release of docker-credential-wincred for Microsoft Windows.
