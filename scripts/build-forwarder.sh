#!/bin/bash

git_root=$(git rev-parse --show-toplevel)

pushd $git_root
    go get -u github.com/apoydence/cf-space-security/cmd/proxy

    pushd cmd/group-manager
        go build
    popd
    pushd cmd/syslog-forwarder
        go build
    popd

    zip -j forwarder.zip cmd/group-manager/group-manager cmd/syslog-forwarder/syslog-forwarder $(which proxy) scripts/run.sh
popd
