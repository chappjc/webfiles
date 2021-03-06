#!/bin/sh

# This will stop a running instance of webfiles, pull updates from
# git, build, and launch,

echo 'Stopping webfiles...'
killall -s INT webfiles
sleep 1
killall -w -s TERM webfiles

echo 'Rebuilding...'
cd $GOPATH/src/github.com/chappjc/webfiles

git diff --no-ext-diff --quiet --exit-code
if [ $? -ne 0 ]; then
  echo "Dirty git workspace. Bailing!"
  exit 1
fi

git checkout master
git pull --ff-only origin master
#SHORTREV=$(git rev-parse --short HEAD)
$GOPATH/bin/dep ensure
cd cmd/webfiles
go build

echo 'Launching!'
./webfiles &
