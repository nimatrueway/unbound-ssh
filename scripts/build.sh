#!/usr/bin/env bash
set -e
flags="-X github.com/nimatrueway/unbound-ssh/internal/config.Version=$(git describe --tags)"
app="unbound-ssh"
output_dir="output"
build_types="darwin_amd64 linux_amd64 darwin_arm64 linux_arm64"

rm -f ${output_dir}/* || true
pushd cmd
for build_type in $build_types
do
    GOOS=${build_type%_*} GOARCH=${build_type#*_} go build -ldflags "${flags}" -o ../${output_dir}/${app}_${build_type}
done
popd