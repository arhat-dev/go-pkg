#!/bin/sh

# Copyright 2020 The arhat.dev Authors.
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

set -e

GOPATH=$(go env GOPATH)
GOOS=$(go env GOHOSTOS)
GOARCH=$(go env GOHOSTARCH)

export GOPATH
export GOOS
export GOARCH

_download_go_pakage() {
  GO111MODULE=off go get -u -v "$1"
}

install() {
  temp_dir="$(mktemp -d)"
  cd "${temp_dir}" || exit 1

  _download_go_pakage github.com/dvyukov/go-fuzz/go-fuzz
  _download_go_pakage github.com/dvyukov/go-fuzz/go-fuzz-build

  cd - || exit 1

  rmdir "${temp_dir}"
}

run() {
  fuzz_build="go-fuzz-build"
  fuzz="go-fuzz"

  if ! command -v "${fuzz_build}"; then
    fuzz_build="${GOPATH}/bin/go-fuzz-build"

    if [ ! -f "${fuzz_build}" ]; then
      echo "Please install fuzz tools by running make install.fuzz"
      exit 1
    fi
  fi

  if ! command -v "${fuzz}"; then
    fuzz="${GOPATH}/bin/go-fuzz"

    if [ ! -f "${fuzz}" ]; then
      echo "Please install fuzz tools by running make install.fuzz"
      exit 1
    fi
  fi

  mkdir -p build/fuzz-result

  ${fuzz_build} -o build/fuzz.zip

  ${fuzz} -bin build/fuzz.zip -workdir build/fuzz-result &
  pid=$!

  trap 'kill ${pid} >/dev/null 2>&1 || true' EXIT

  echo "running fuzz test for 10m, pid: ${pid}"
  sleep 600
}

# shellcheck disable=SC2068
$@
