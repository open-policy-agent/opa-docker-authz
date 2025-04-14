#!/usr/bin/env bash

set -e


OPA_VERSION=$(go list -m -f '{{.Version}}' github.com/open-policy-agent/opa)

echo "Building opa-docker-authz version: $VERSION (OPA version: $OPA_VERSION)"


platforms=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
for platform in "${platforms[@]}"
do
	platform_split=(${platform//\// })
	GOOS=${platform_split[0]}
	GOARCH=${platform_split[1]}

	echo -e "\nBuilding opa-docker-authz for $platform ..."
	CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -ldflags \
	    "-X github.com/open-policy-agent/opa-docker-authz/version.Version=$VERSION -X github.com/open-policy-agent/opa-docker-authz/version.OPAVersion=$OPA_VERSION" \
	    -buildvcs=false \
	    -o opa-docker-authz-$GOOS-$GOARCH
	if [ $? -ne 0 ]; then
   		echo 'An error has occurred! Aborting the script execution...'
		exit 1
	fi
done

echo -e "\n... done!"
