#!/usr/bin/env sh

grep '^- package: github.com/open-policy-agent/opa$' glide.yaml  -A 1 | grep 'version: ' | awk '{print $2}'
