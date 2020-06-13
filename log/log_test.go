/*
Copyright 2019 The arhat.dev Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	Log.WithName("test")
	Log.D("no output")

	err := SetDefaultLogger([]Config{{Level: "debug", Format: "json", Destination: Destination{File: "stderr"}}})
	assert.NoError(t, err)

	l := Log.WithName("test")
	l.D("test debug")

	l = l.WithName("test")
	l.D("test test debug")

	l = l.WithFields(String("test", "test"))
	l.D("initial fields only")
	l.D("initial and extra fields", String("extra", "extra"))
}
