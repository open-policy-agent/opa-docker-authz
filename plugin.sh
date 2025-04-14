#!/usr/bin/env bash

set -ex

[ -d ./rootfs ] && rm -rf ./rootfs
mkdir ./rootfs

echo "Creating root filesystem for plugin ..."
docker image build --load -t rootfsimage .
id=`docker container create rootfsimage true`
docker container export "$id" | tar -x -C ./rootfs

echo "Creating plugin "${REPO}:${VERSION}" ..."
docker plugin create "${REPO}:${VERSION}" .

echo "Cleanup..."
docker container rm -f "$id" > /dev/null
docker image rm -f rootfsimage > /dev/null
rm -rf ./rootfs


platforms=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
for platform in "${platforms[@]}"
do
	platform_split=(${platform//\// })
	GOOS=${platform_split[0]}
	GOARCH=${platform_split[1]}

	[ -d ./rootfs ] && rm -rf ./rootfs
	mkdir ./rootfs

	echo "Creating root filesystem for plugin ..."
	docker buildx build --load --platform ${platform} -t rootfsimage-${GOOS}-${GOARCH} .
	#docker image build -t rootfsimage .
	id=`docker container create --platform ${platform} rootfsimage-${GOOS}-${GOARCH} true`
	docker container export "$id" | tar -x -C ./rootfs

	echo "Creating plugin "${REPO}:${VERSION}-${GOOS}-${GOARCH}" ..."
	docker plugin create "${REPO}:${VERSION}-${GOOS}-${GOARCH}" .

	echo "Cleanup..."
	docker container rm -f "$id" > /dev/null
	docker image rm -f rootfsimage-${GOOS}-${GOARCH} > /dev/null
	rm -rf ./rootfs
done
