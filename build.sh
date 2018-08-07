#!/usr/bin/env bash

set -e

echo "Building opa-docker-authz version: $VERSION"

echo -e "\nInstalling glide ..."
curl -s https://glide.sh/get | sh

echo -e "\nInstalling all the dependencies ..."
glide install

echo -e "\nSetting OPA version to $OPA_VERSION ..."
sed -i "s/\(  version: v\)[0-9]\.[0-9]\.[0-9]/\1$OPA_VERSION/g" glide.yaml

echo -e "\nBuilding opa-docker-authz ..."
CGO_ENABLED=0 go build -ldflags \
    "-X github.com/open-policy-agent/opa-docker-authz/version.Version=$VERSION -X github.com/open-policy-agent/opa-docker-authz/version.OPAVersion=$OPA_VERSION" \
    -o opa-docker-authz
rm -rf ./vendor

echo -e "\n... done!"
