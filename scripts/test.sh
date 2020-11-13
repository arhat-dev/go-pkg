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

unit() {
  pkg="$1"

  GOOS=$(go env GOHOSTOS) GOARCH=$(go env GOHOSTARCH) CGO_ENABLED=1 \
    go test -mod=readonly -v -race -failfast -covermode=atomic \
    -coverprofile "coverage.${pkg}.txt" -coverpkg "./..." "./${pkg}"
}

coverage() {
  pkg="$1"

  go tool cover -html "coverage.${pkg}.txt"
}

benchmark() {
  pkg="$1"

  GOOS=$(go env GOHOSTOS) GOARCH=$(go env GOHOSTARCH) CGO_ENABLED=1 \
    go test -timeout 30m -mod=readonly -bench '^Benchmark.*' \
    -benchmem -covermode=atomic -coverprofile "coverage.bench.${pkg}.txt" \
    -benchtime 5s -run '^Benchmark.*' -v -coverpkg "./..." "./${pkg}"
}

TARGET=$(printf "%s" "$@" | cut -d. -f2)
PKG=$(printf "%s" "$@" | cut -d. -f3)

${TARGET} "${PKG}"
