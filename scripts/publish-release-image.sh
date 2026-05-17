#!/usr/bin/env bash
set -euo pipefail

release_tag="${GITHUB_REF_NAME:-}"
repository="${GITHUB_REPOSITORY:-}"
image_repository="${IMAGE_REPOSITORY:-ghcr.io/${repository}}"
platforms="${PLATFORMS:-linux/amd64,linux/arm64}"

if [[ -z "${release_tag}" ]]; then
  echo "GITHUB_REF_NAME or release tag is required" >&2
  exit 1
fi

if [[ -z "${repository}" && -z "${IMAGE_REPOSITORY:-}" ]]; then
  echo "GITHUB_REPOSITORY or IMAGE_REPOSITORY is required" >&2
  exit 1
fi

if [[ ! "${release_tag}" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
  echo "release tag must use SemVer, for example v1.2.3 or v1.2.3-rc.1" >&2
  exit 1
fi

version="${release_tag#v}"
major="${BASH_REMATCH[1]}"
minor="${BASH_REMATCH[2]}"
docker_release_tag="${release_tag//+/_}"
docker_version="${version//+/_}"
tags=(
  --tag "${image_repository}:${docker_release_tag}"
  --tag "${image_repository}:${docker_version}"
)

if [[ -n "${GITHUB_SHA:-}" ]]; then
  tags+=(--tag "${image_repository}:${GITHUB_SHA}")
fi

if [[ "${version}" != *-* ]]; then
  tags+=(
    --tag "${image_repository}:v${major}.${minor}"
    --tag "${image_repository}:v${major}"
    --tag "${image_repository}:latest"
  )
fi

if [[ "${DRY_RUN:-false}" == "true" ]]; then
  printf '%s\n' "${tags[@]}"
  exit 0
fi

docker buildx build \
  --platform "${platforms}" \
  "${tags[@]}" \
  --push \
  .
