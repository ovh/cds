#!/bin/bash

set -e

echo "building tatcli locally"
go build

_tatcli () {
  binary="./tatcli --configFile=$HOME/.tatcli/config.tat.integration.tests.admin.json"
  if [ -z "$1" ]                           # Is parameter #1 zero length?
  then
    echo "Invalid use of _test"  # No parameter passed.
    exit 1;
  fi
  CMD="$binary $1";
  echo "Running ${CMD}";
  $CMD
  if [ $? != 0 ]; then
    echo "$CMD is failed";
    exit 1;
  fi;
  echo "";
  echo "Passed";
}

_title () {
  echo "---------"
  echo "$1";
  echo "---------"
}

# TODO
_title "Testing tatcli config..."
_tatcli "config show"

_title "Testing tatcli group..."
_tatcli "group list 0 2"

_title "Testing tatcli message..."
_tatcli "message add /Private/tat.integration.tests.admin new message `date`"
_tatcli "message list /Private/tat.integration.tests.admin 0 1"

_title "Testing tatcli presence..."
_tatcli "presence list /Private/tat.integration.tests.admin 0 2"

_title "Testing tatcli socket..."
# enable ws option in tat engine to run this test  _tatcli "socket dump"

_title "Testing tatcli stats..."
_tatcli "stats count"

_title "Testing tatcli topic..."
_tatcli "topic list 0 2"
_tatcli "msg list /Private/tat.integration.tests.admin --onlyCount=true"
_tatcli "topic truncate /Private/tat.integration.tests.admin --force"
_tatcli "msg list /Private/tat.integration.tests.admin --onlyCount=true"


_title "Testing tatcli user..."
_tatcli "user list 0 2"
_tatcli "user me"

_title "Testing tatcli version..."
_tatcli "version"

_title "All Tests OK"

exit 0;
