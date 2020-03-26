#!/usr/bin/env bash

set -e


OPA_VERSION=$(go list -m -f '{{.Version}}' github.com/open-policy-agent/opa)

echo "Building opa-docker-authz version: $VERSION (OPA version: $OPA_VERSION)"

echo -e "\nBuilding opa-docker-authz ..."
CGO_ENABLED=0 go build -ldflags \
    "-X github.com/open-policy-agent/opa-docker-authz/version.Version=$VERSION -X github.com/open-policy-agent/opa-docker-authz/version.OPAVersion=$OPA_VERSION" \
    -o opa-docker-authz

echo -e "\n... done!"
