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

package backoff

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoff(t *testing.T) {
	const (
		key       = "foo"
		threshold = 10
	)
	b := NewStrategy(time.Second, 1024*time.Second, 2, threshold)

	for i := 0; i < threshold; i++ {
		assert.Equal(t, time.Duration(0), b.Next(key))
	}

	for i := 0; i <= 10; i++ {
		assert.Equal(t, time.Duration(math.Pow(2, float64(i))*float64(time.Second)), b.Next(key))
	}

	b.Reset(key)
	assert.Equal(t, time.Duration(0), b.Next(key))
}
