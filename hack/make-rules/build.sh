#!/usr/bin/env bash

# Copyright 2020 The OpenYurt Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -x

YURT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
YURT_OUTPUT_DIR=${YURT_ROOT}/_output
YURT_LOCAL_BIN_DIR=${YURT_OUTPUT_DIR}/local/bin
YURT_MOD="$(head -1 $YURT_ROOT/go.mod | awk '{print $2}')"

GIT_VERSION=${GIT_VERSION:-$(git describe --abbrev=0 --tags)}
GIT_COMMIT=$(git rev-parse HEAD)
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')

# project_info generates the project information and the corresponding value
# for 'ldflags -X' option
project_info() {
    PROJECT_INFO_PKG=${YURT_MOD}/pkg/projectinfo
    echo "-X ${PROJECT_INFO_PKG}.gitVersion=${GIT_VERSION}"
    echo "-X ${PROJECT_INFO_PKG}.gitCommit=${GIT_COMMIT}"
    echo "-X ${PROJECT_INFO_PKG}.buildDate=${BUILD_DATE}"
}

# get_binary_dir_with_arch generated the binary's directory with GOOS and GOARCH.
# eg: ./_output/bin/darwin/arm64/
get_binary_dir_with_arch(){
    echo $1/$(go env GOOS)/$(go env GOARCH)
}

build_binaries() {
    local goflags goldflags gcflags
    goldflags="${GOLDFLAGS:--s -w $(project_info)}"
    gcflags="${GOGCFLAGS:-}"
    goflags=${GOFLAGS:-}

    local target_bin_dir=$(get_binary_dir_with_arch ${YURT_LOCAL_BIN_DIR})
    rm -rf ${target_bin_dir}
    mkdir -p ${target_bin_dir}
    cd ${target_bin_dir}
    echo "Building ${binary}"
   	go build -o edge-proxy \
          -ldflags "${goldflags:-}" \
          -gcflags "${gcflags:-}" ${goflags} $YURT_ROOT/cmd/edge-proxy
   	go build -o benchmark \
          -ldflags "${goldflags:-}" \
          -gcflags "${gcflags:-}" ${goflags} $YURT_ROOT/cmd/benchmark

}

build_binaries
