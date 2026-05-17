#!/usr/bin/env bash
set -euo pipefail

release_tag="${GITHUB_REF_NAME:-dev}"
version="${release_tag#v}"
output_dir="${DIST_DIR:-dist}"
platforms="${RELEASE_PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}"

mkdir -p "${output_dir}"
rm -rf "${output_dir:?}"/*

for platform in ${platforms}; do
  os="${platform%/*}"
  arch="${platform#*/}"
  name="go-httpbin_${version}_${os}_${arch}"
  build_dir="${output_dir}/${name}"
  binary="go-httpbin"

  if [[ "${os}" == "windows" ]]; then
    binary="go-httpbin.exe"
  fi

  mkdir -p "${build_dir}"
  CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" go build -trimpath -ldflags="-s -w" -o "${build_dir}/${binary}" ./cmd/server
  archive="${output_dir}/${name}.tar.gz"
  tar -C "${build_dir}" -czf "${archive}" "${binary}"
  sha256sum "${archive}" >"${archive}.sha256"
  rm -rf "${build_dir}"
done

sha256sum "${output_dir}"/*.tar.gz >"${output_dir}/checksums.txt"
