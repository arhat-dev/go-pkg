/*
Copyright 2020 The arhat.dev Authors.

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

package envhelper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpand(t *testing.T) {
	mappingFunc := func(s, o string) string {
		switch s {
		case "*":
			return "all the args"
		case "#":
			return "NARGS"
		case "$":
			return "PID"
		case "1":
			return "ARGUMENT1"
		case "HOME":
			return "/usr/gopher"
		case "H":
			return "(Value of H)"
		case "home_1":
			return "/usr/foo"
		case "_":
			return "underscore"
		}
		return o
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"${FOO}$9$@", "${FOO}$9$@"},
		{"", ""},
		{"$*", "all the args"},
		{"$$", "PID"},
		{"${*}", "all the args"},
		{"$1", "ARGUMENT1"},
		{"${1}", "ARGUMENT1"},
		{"now is the time", "now is the time"},
		{"$HOME", "/usr/gopher"},
		{"$home_1", "/usr/foo"},
		{"${HOME}", "/usr/gopher"},
		{"${H}OME", "(Value of H)OME"},
		{"A$$$#$1$H$home_1*B", "APIDNARGSARGUMENT1(Value of H)/usr/foo*B"},
		{"start$+middle$^end$", "start$+middle$^end$"},
		{"mixed$|bag$$$", "mixed$|bagPID$"},
		{"$", "$"},
		{"$}", "$}"},
		{"${", ""},  // invalid syntax; eat up the characters
		{"${}", ""}, // invalid syntax; eat up the characters

		{"$(FOO)$9$@", "$(FOO)$9$@"},
		{"$(*)", "all the args"},
		{"$(1)", "ARGUMENT1"},
		{"$(HOME)", "/usr/gopher"},
		{"$(H)OME", "(Value of H)OME"},
		{"$)", "$)"},
		{"$(", ""},  // invalid syntax; eat up the characters
		{"$()", ""}, // invalid syntax; eat up the characters
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("testExpand[%d]", i), func(t *testing.T) {
			assert.Equal(t, test.expected, Expand(test.input, mappingFunc))
		})
	}
}
