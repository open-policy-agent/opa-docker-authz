#!/usr/bin/env bash

set -ex

cd /go/src/github.com/open-policy-agent/opa-docker-authz
go get ./...
go build -o opa-docker-authz
