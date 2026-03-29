#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME="$(awk -F': ' '$1 == "local_dev_image_name" { print $2 }' deploy/docker/managed-user/image.lock)"

if [[ -z "${IMAGE_NAME}" ]]; then
  echo "failed to read local_dev_image_name from deploy/docker/managed-user/image.lock" >&2
  exit 1
fi

docker build -f deploy/docker/managed-user/Dockerfile -t "${IMAGE_NAME}" .
