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
- A persistent server-side store is used to keep sessions valid between restarts of webfiles.
- Tokens may be provided by any of: (1) URL query such as `?jwt={the token}`,
  (2) HTTP Authorization (Bearer) header, or (3) a cookie named "jwt".  They are
  valid only if they were signed by the server.
- User to file association is managed with a Bolt DB.
- Includes a script, relaunch.sh, that works well with webhooks to pull changes
  from git, build, and restart webfiles.

### Endpoints

- `/` - A basic HTML page with a file selection dialog for uploading.
- `/token` - Shows your current JWT token, which can be used to identify yourself.
- `/upload` - The file upload path, POST only. Response is JSON including file's
  UID and the user's current JWT.
- `/user-files` - Shows all files associated with you in a JSON array of file
  UIDs. User authentication via JWT.
- `/file/{fileid}` - The file download path. Requires user authorization.

### Example

Instead of uploading from your web browser, which has no progress indicator presently, you can use `curl` as follows:

```bash
curl https://deploy.site.you/upload -F "fileupload=@UX490UAR-AS.302"
```

The response:
```json
{
    "file": {
        "uid": "4b361b653e78c342",
        "file_name": "UX490UAR-AS.302",
        "file_size": 6488064
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTk2Mzc2NzIsImlhdCI6MTUyODEwMTY3MiwidXNlciI6IjU3NUhNMjVXSjZLWUVESDI1R0E1M0NGVk41SVJGVlNLUkU0MzJFMjRDUERVT05CR1NKR0EifQ.K6AjhrnT0sJay9NyrQvCKvjnhrk9Hanic-86EtknetA"
}
```

The `"uid"` field is used with the `/file/{fileid}` endpoint to download the
file. The `"token"` field contains the JWT associated to the user at time of
upload. This token is required to download the file.

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
