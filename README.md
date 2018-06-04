# webfiles

[![GoDoc](https://godoc.org/github.com/chappjc/webfiles?status.svg)](https://godoc.org/github.com/chappjc/webfiles)
[![Go Report Card](https://goreportcard.com/badge/github.com/chappjc/webfiles)](https://goreportcard.com/report/github.com/chappjc/webfiles)
[![Build Status](https://travis-ci.org/chappjc/webfiles.svg?branch=master)](https://travis-ci.org/chappjc/webfiles)

webfiles is a simple file server with some fun features.

Dependencies are vendored with [`dep`](https://github.com/golang/dep).  Travis
CI configuration is included.

## Features

- Upload a file via POST, returning a JSON response including the file's unique
  identifier, and a JWT access token.
- Download a file via the file's unique identifier.
- User authentication is handled with JWT tokens.
- Tokens are generated automatically if one is not provided.
- Tokens may be provided any of: (1) URL query such as `?jwt={the token}`, (2)
  HTTP Authorization (Bearer) header, or (3) a cookie named "jwt".
- Includes a script, relaunch.sh, that works well with webhooks to pull changes
  from git, build, and restart webfiles.

### URL Paths

- `/` - A basic HTML page with a file selection dialog for uploading.
- `/token` - Show your current JWT token, which can be used to identifier yourself.
- `/upload` - The file upload path, POST only. Response is JSON including file's
  UID and the user's current JWT.
- `/user-files` - Shows all files associated with you in a JSON array of file
  UIDs. User authentication via JWT.
- `/file/{fileid}` - The file download path. Requires user authorization.


## Requirements

* [Go](http://golang.org/dl/) 1.9.x or 1.10.x.
* git, internet, the usual.

## Installation

### Build from Source

The following instructions assume a Unix-like shell (e.g. bash).

* [Install Go](http://golang.org/doc/install)

* Verify Go installation:

      go env GOROOT GOPATH

* Ensure `$GOPATH/bin` is on your `$PATH`.
* Install `dep`, the dependency management tool.

      go get -u -v github.com/golang/dep/cmd/dep

* Clone the repository. It **must** be cloned into the following directory.

      git clone https://github.com/chappjc/webfiles $GOPATH/src/github.com/chappjc/webfiles

* Fetch dependencies, and build the `webfiles` executable.

      cd $GOPATH/src/github.com/chappjc/webfiles
      dep ensure
      # install all executables
      go install ./cmd/...
      # or build the webfiles executable in the workspace
      cd cmd/webfiles
      go build
