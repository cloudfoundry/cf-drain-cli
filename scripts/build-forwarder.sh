#!/bin/bash

git_root=$(git rev-parse --show-toplevel)

pushd $git_root
    GOOS=linux go get -u -d github.com/apoydence/cf-space-security/cmd/...

    pushd $GOPATH/src/github.com/apoydence/cf-space-security/cmd/proxy
        GOOS=linux go get -u -d ./...
        GOOS=linux go build
    popd

    pushd cmd/syslog-forwarder
        GOOS=linux go get -u -d ./...
        GOOS=linux go build
    popd

    zip \
        -j forwarder.zip \
        cmd/syslog-forwarder/syslog-forwarder \
        $GOPATH/src/github.com/apoydence/cf-space-security/cmd/proxy/proxy \
        scripts/run.sh
popd
