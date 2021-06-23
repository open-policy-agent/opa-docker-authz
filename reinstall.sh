#/usr/bin/env bash

# Disable and remove the currently running plugin and replace with a new one built by the script.
# NOTE that this is for development/testing purposes only - if there are more than one plugin installed
# the script will just delete the first one.

installed=$(docker plugin ls | sed -n 2p | cut -d " " -f 1)

docker plugin disable "${installed}"
docker plugin rm "${installed}"

make plugin

new=$(docker plugin ls | sed -n 2p | cut -d " " -f 1)

docker plugin enable "${new}"

# Restarting the docker daemon required