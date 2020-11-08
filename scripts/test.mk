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

GOTEST := GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) CGO_ENABLED=1 \
	go test -mod=readonly -v -race -failfast -covermode=atomic

test.unit.kubeclientconf:
	go run ./test/kube_incluster_env.go \
		-ca-data-from ./test/testdata/ca.crt \
		-token-data-from ./test/testdata/token \
		-- ${GOTEST} -tags 'kubeclientconf' \
			-coverprofile=coverage.confhelper.txt -coverpkg=./confhelper ./confhelper

test.unit.%:
	sh scripts/test.sh $@

.PHONY: test.unit
test.unit: \
	test.unit.backoff \
	test.unit.decodehelper \
	test.unit.encodehelper \
	test.unit.envhelper \
	test.unit.exechelper \
	test.unit.hashhelper \
	test.unit.iohelper \
	test.unit.kubehelper \
	test.unit.log \
	test.unit.patchhelper \
	test.unit.perfhelper \
	test.unit.pipenet \
	test.unit.queue \
	test.unit.reconcile \
	test.unit.textquery \
	test.unit.tlshelper

test.benchmark.%:
	sh scripts/test.sh $@

.PHONY: test.benchmark
test.benchmark: \
	test.benchmark.backoff \
	test.benchmark.decodehelper \
	test.benchmark.encodehelper \
	test.benchmark.envhelper \
	test.benchmark.exechelper \
	test.benchmark.hashhelper \
	test.benchmark.iohelper \
	test.benchmark.log \
	test.benchmark.patchhelper \
	test.benchmark.perfhelper \
	test.benchmark.pipenet \
	test.benchmark.queue \
	test.benchmark.reconcile \
	test.benchmark.textquery \
	test.benchmark.tlshelper

view.coverage.%:
	sh scripts/test.sh $@

install.fuzz:
	sh scripts/fuzz.sh install

test.fuzz:
	sh scripts/fuzz.sh run
