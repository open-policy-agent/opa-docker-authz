#!/usr/bin/env bash

set -ex

echo "building version: $VERSION"

cd /go/src/github.com/open-policy-agent/opa-docker-authz

echo "install glide"
curl https://glide.sh/get | sh

echo "install all the dependencies"
glide install

echo "build opa-docker-authz"
CGO_ENABLED=0 go build -ldflags "-X github.com/open-policy-agent/opa-docker-authz.Version=$VERSION" -o opa-docker-authz
