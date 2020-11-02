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

GOTEST := GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) CGO_ENABLED=1 go test -mod=readonly -v -race -failfast -covermode=atomic

test.unit:
	${GOTEST} -coverprofile=coverage.txt -coverpkg=./... -run '!(TestJQCompability)' ./...

view.coverage:
	go tool cover -html=coverage.txt

test.unit.backoff:
	${GOTEST} -coverprofile=coverage.backoff.txt -coverpkg=./backoff ./backoff

view.coverage.backoff:
	go tool cover -html=coverage.backoff.txt

test.unit.confhelper:
	go run ./test/kube_incluster_env.go \
		-ca-data-from ./test/testdata/ca.crt \
		-token-data-from ./test/testdata/token \
		-- ${GOTEST} -coverprofile=coverage.confhelper.txt -coverpkg=./confhelper ./confhelper

view.coverage.confhelper:
	go tool cover -html=coverage.confhelper.txt

install.fuzz:
	sh scripts/fuzz.sh install

test.fuzz:
	sh scripts/fuzz.sh run
