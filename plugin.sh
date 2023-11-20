#!/usr/bin/env bash

set -ex

[ -d ./rootfs ] && rm -rf ./rootfs
mkdir ./rootfs

echo "Creating root filesystem for plugin ..."
docker image build -t rootfsimage .
id=`docker container create rootfsimage true`
docker container export "$id" | tar -x -C ./rootfs

echo "Creating plugin "${REPO}:${VERSION}" ..."
docker plugin create "${REPO}:${VERSION}" .

echo "Cleanup..."
docker container rm -f "$id" > /dev/null
docker image rm -f rootfsimage > /dev/null
rm -rf ./rootfs
