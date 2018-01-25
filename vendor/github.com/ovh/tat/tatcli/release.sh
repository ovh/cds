#!/bin/bash

#
# Tag release and push to origin remotes.
#

if [ $# -ne 1 ]
then
    echo "Tag and push master to 'origin' remotes"
    echo "Usage: `basename $0` version"
    exit 1
fi

version="$1"
current_branch=`git rev-parse --abbrev-ref HEAD`

# Check we are on master branch
if [ "$current_branch" != "master" ]
then
    echo -e "$0 only works from *master* branch. When ready, Please run \n\tgit checkout master\n\t$0"
    exit  1
fi

# Check there is no pending changes --> branch is clean
if [ -n "`git status --porcelain`" ]
then
    echo "There are pending/uncommited changes in current branch."
    echo "Please commit or stash them."
    exit 1
fi

# Ensure commit is tagged and annotated
current_tag=$(git describe --exact-match 2>/dev/null)
if [ -n "$current_tag" ]
then
    if [ "$current_tag" != "v$version" ]
    then
        echo "Error: version mismatch '$current_tag' != 'v$version'"
    fi
else
    sed -i.bak "s/const VERSION =.*/const VERSION = \"$version\"/g" internal/version.go
    rm -f internal/version.go.bak
    git commit -am "[auto] bump version to v$version"
    git tag -s "v$version"
fi

echo "Pushing master and 'v$version'"
git push origin master
git push origin "v$version"
