#!/bin/bash

git_root=$(git rev-parse --show-toplevel)

pushd $git_root
    GOOS=linux go get -u github.com/apoydence/cf-space-security/cmd/...

    pushd cmd/group-manager
        GOOS=linux go build
    popd
    pushd cmd/syslog-forwarder
        GOOS=linux go build
    popd

    zip -j forwarder.zip cmd/group-manager/group-manager cmd/syslog-forwarder/syslog-forwarder $GOPATH/bin/proxy scripts/run.sh
popd
