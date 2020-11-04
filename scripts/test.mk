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

test.unit.confhelper:
	go run ./test/kube_incluster_env.go \
		-ca-data-from ./test/testdata/ca.crt \
		-token-data-from ./test/testdata/token \
		-- ${GOTEST} -coverprofile=coverage.confhelper.txt -coverpkg=./confhelper ./confhelper

test.unit.queue:
	sh scripts/test.sh $@

test.unit.textquery:
	sh scripts/test.sh $@

test.unit.backoff:
	sh scripts/test.sh $@

test.unit.pipenet:
	sh scripts/test.sh $@

test.unit.log:
	sh scripts/test.sh $@

test.unit.iohelper:
	sh scripts/test.sh $@

test.unit.kubehelper:
	sh scripts/test.sh $@

test.unit.envhelper:
	sh scripts/test.sh $@

test.unit: \
	test.unit.pipenet \
	test.unit.backoff \
	test.unit.textquery \
	test.unit.queue \
	test.unit.log \
	test.unit.iohelper \
	test.unit.kubehelper \
	test.unit.envhelper

test.benchmark.queue:
	sh scripts/test.sh $@

test.benchmark.textquery:
	sh scripts/test.sh $@

test.benchmark.backoff:
	sh scripts/test.sh $@

test.benchmark.pipenet:
	sh scripts/test.sh $@

test.benchmark.log:
	sh scripts/test.sh $@

test.benchmark.iohelper:
	sh scripts/test.sh $@

test.benchmark.kubehelper:
	sh scripts/test.sh $@

test.benchmark.envhelper:
	sh scripts/test.sh $@

test.benchmark: \
	test.benchmark.pipenet \
	test.benchmark.backoff \
	test.benchmark.textquery \
	test.benchmark.queue \
	test.benchmark.log \
	test.benchmark.iohelper \
	test.benchmark.kubehelper \
	test.benchmark.envhelper

view.coverage.queue:
	sh scripts/test.sh $@

view.coverage.textquery:
	sh scripts/test.sh $@

view.coverage.backoff:
	sh scripts/test.sh $@

view.coverage.pipenet:
	sh scripts/test.sh $@

view.coverage.log:
	sh scripts/test.sh $@

view.coverage.iohelper:
	sh scripts/test.sh $@

view.coverage.kubehelper:
	sh scripts/test.sh $@

view.coverage.envhelper:
	sh scripts/test.sh $@

install.fuzz:
	sh scripts/fuzz.sh install

test.fuzz:
	sh scripts/fuzz.sh run
