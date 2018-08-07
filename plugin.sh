#!/usr/bin/env bash

set -e

static_url="https://download.docker.com/linux/static"

echo -e "\nPreparing to build plugin ..."

# Determine the channel (stable, edge nightly) required for the Docker client
if [[ "$DOCKER_VERSION" =~ ^0\.0\.0-[0-9]{14}-[0-9a-f]{7}$ ]]; then
    channel=nightly
elif [[ "$DOCKER_VERSION" =~ [0-9]{2}\.[0-9]{2}.[0-9]+-ce$ ]]; then
    month=$(echo "$DOCKER_VERSION" | cut -d '.' -f 2 | sed 's/^0*//')
    if [[ $(( month % 3 )) == 0 ]]; then
        channel=stable
    else
        channel=edge
    fi
else
    echo "Cannot determine the version of the Docker daemon. Aborting ..."
    exit 1
fi

# Download a version of the Docker client for the container, that matches the
# version of the Docker daemon
curl -s -L "${static_url}/${channel}/x86_64/docker-${DOCKER_VERSION}.tgz" | \
tar xzf - -C /usr/local/bin --strip-components=1 docker/docker

# Check that a plugin of the same name doesn't already exist
for plugin in $(docker plugin ls --format '{{.Name}}'); do
    if [ "$plugin" == "${REPO}-v2:${VERSION}" ]; then
        echo "The plugin "$plugin" already exists. Aborting ..."
        exit 1
    fi
done

# Remove any residual filesystem content from a previous build
[ -d ./rootfs ] && rm -rf ./rootfs
mkdir ./rootfs

echo -e "\nCreating root filesystem for plugin ..."

# Export filesystem content from a container derived from an image build from
# plugin Dockerfile
docker image build -t rootfsimage .
id=$(docker container create rootfsimage true)
docker container export "$id" | tar -x -C ./rootfs

echo -e "\nCreating plugin "${REPO}-v2:${VERSION}" ..."

docker plugin create "${REPO}-v2:${VERSION}" .

echo -e "\nCleaning up ..."
docker container rm -f "$id" > /dev/null
docker image rm -f rootfsimage > /dev/null
rm -rf ./rootfs

echo -e "\n... done!"
