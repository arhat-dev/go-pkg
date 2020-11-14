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

gen.testdata.cert:
	mkcert -ecdsa \
		-cert-file ./test/testdata/tls-cert.pem \
		-key-file ./test/testdata/tls-key.pem \
		'127.0.0.1' '::1' 'localhost'
	mkcert -client -ecdsa \
		-cert-file ./test/testdata/client-tls-cert.pem \
		-key-file ./test/testdata/client-tls-key.pem \
		'127.0.0.1' '::1' 'localhost'
