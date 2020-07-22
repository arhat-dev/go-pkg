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

package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeqQueue(t *testing.T) {
	const N = 100

	q := NewSeqQueue()
	for i := 0; i < N; i++ {
		data, completed := q.Offer(uint64(i), i)
		assert.Equal(t, 1, len(data))
		assert.Equal(t, i, data[0].(int))
		assert.False(t, completed)
	}

	q.Reset()
	for i := N - 1; i > 0; i-- {
		data, completed := q.Offer(uint64(i), i)
		assert.Nil(t, data)
		assert.False(t, completed)
	}

	{
		data, completed := q.Offer(uint64(0), 0)
		assert.NotNil(t, data)
		assert.Equal(t, N, len(data))
		assert.False(t, completed)
		for i, d := range data {
			assert.Equal(t, i, d.(int))
		}
	}

	q.Reset()
	for i := 0; i < N; i++ {
		data, completed := q.Offer(uint64(i), i)
		assert.Equal(t, 1, len(data))
		assert.Equal(t, i, data[0].(int))
		assert.False(t, completed)
	}
	for i := 0; i < N; i++ {
		data, completed := q.Offer(0, 0)
		assert.Nil(t, data)
		assert.False(t, completed)
	}

	q.Reset()
	for i := 1; i < N/2; i += 2 {
		data, completed := q.Offer(uint64(i), i)
		assert.Equal(t, 0, len(data))
		assert.False(t, completed)
	}

	completed := q.SetMaxSeq(1)
	assert.False(t, completed)

	completed = q.SetMaxSeq(N)
	assert.False(t, completed)

	completed = q.SetMaxSeq(N / 2)
	assert.False(t, completed)

	for i := 0; i < N/2; i += 2 {
		var data []interface{}
		data, completed = q.Offer(uint64(i), i)
		assert.Equal(t, 2, len(data))
		assert.Equal(t, i, data[0].(int))
		assert.Equal(t, i+1, data[1].(int))
		if i == N/2-2 {
			assert.True(t, completed)
		} else {
			assert.False(t, completed)
		}
	}

	completed = q.SetMaxSeq(1)
	assert.True(t, completed)

	completed = q.SetMaxSeq(N)
	assert.False(t, completed)

	completed = q.SetMaxSeq(N / 2)
	assert.True(t, completed)

	for i := N / 2; i < N; i++ {
		data, completed := q.Offer(uint64(i), i)
		assert.Equal(t, 0, len(data))
		assert.True(t, completed)
	}
}
