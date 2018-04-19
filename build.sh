#!/usr/bin/env bash

set -ex

echo "building version: $VERSION"

cd /go/src/github.com/open-policy-agent/opa-docker-authz

echo "install glide"
curl https://glide.sh/get | sh

echo "install all the dependencies"
glide install

OPA_VERSION=$(grep 'package: github.com/open-policy-agent/opa$' glide.yaml -A 1 | tail -n 1 | awk '{print $2}')

echo "build opa-docker-authz"
CGO_ENABLED=0 go build -ldflags "-X github.com/open-policy-agent/opa-docker-authz/version.Version=$VERSION -X github.com/open-policy-agent/opa-docker-authz/version.OPAVersion=$OPA_VERSION" -o opa-docker-authz
