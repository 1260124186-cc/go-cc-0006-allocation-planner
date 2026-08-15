#!/usr/bin/env sh
set -eu

platform="${1:-linux/arm64}"
image="go-cc-0006-allocation-planner:${platform#linux/}"

docker buildx build --load --platform "$platform" -f benzhi.Dockerfile -t "$image" .
docker run --rm --platform "$platform" --entrypoint sh "$image" -c 'go build ./...'
docker run --rm --platform "$platform" "$image"
