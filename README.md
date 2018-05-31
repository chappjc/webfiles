# webfiles

webfiles is a demo file server with some fun features.

## Requirements

* [Go](http://golang.org/dl/) 1.9.x or 1.10.x.

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
      # install all packages and executables
      go install . ./cmd/...
      # or build the webfiles executable in the workspace
      cd cmd/webfiles
      go build
      