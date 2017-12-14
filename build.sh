#!/usr/bin/env bash

set -ex

cd /go/src/github.com/open-policy-agent/opa-docker-authz

echo "install glide"
curl https://glide.sh/get | sh

echo "install all the dependencies"
glide install

echo "build opa-docker-authz"
go build -o opa-docker-authz
