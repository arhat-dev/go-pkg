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
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkSeqQueue_BestCase(b *testing.B) {
	n := uint64(b.N)
	q := NewSeqQueue(func(seq uint64, d interface{}) {})
	q.SetMaxSeq(n)

	b.ResetTimer()
	for i := uint64(0); i <= n; i++ {
		q.Offer(i, 0)
	}
}

func BenchmarkSeqQueue_WorstCase(b *testing.B) {
	n := uint64(b.N)
	q := NewSeqQueue(func(seq uint64, d interface{}) {})
	q.SetMaxSeq(n)
	q.Offer(0, 0)

	b.ResetTimer()
	for i := n; i > 0; i-- {
		q.Offer(i, 0)
	}
}

func TestSeqQueue(t *testing.T) {
	const MaxSeq = 100

	t.Run("Sequential One Out Per Offer", func(t *testing.T) {
		var sequentialSerial []uint64
		for i := 0; i <= MaxSeq; i++ {
			sequentialSerial = append(sequentialSerial, uint64(i))
		}

		var data []interface{}
		q := NewSeqQueue(func(seq uint64, d interface{}) {
			data = append(data, d)
		})
		q.SetMaxSeq(MaxSeq)

		for idx, i := range sequentialSerial {
			complete := q.Offer(i, i)

			assert.Len(t, data, idx+1)
			assert.Equal(t, i, data[idx].(uint64))

			if idx == len(sequentialSerial)-1 {
				assert.True(t, complete)
			} else {
				assert.False(t, complete)
			}
		}
	})

	t.Run("Reversed No Data Until Last", func(t *testing.T) {
		var reversedSerial []uint64
		for i := MaxSeq; i >= 0; i-- {
			reversedSerial = append(reversedSerial, uint64(i))
		}

		var data []interface{}
		q := NewSeqQueue(func(seq uint64, d interface{}) {
			data = append(data, d)
		})

		q.SetMaxSeq(MaxSeq)

		for idx, i := range reversedSerial {
			complete := q.Offer(i, i)
			if idx == len(reversedSerial)-1 {
				assert.Len(t, data, MaxSeq+1)
				assert.True(t, complete)
			} else {
				assert.Len(t, data, 0)
				assert.False(t, complete)
			}
		}
	})

	t.Run("Only One Seq Data", func(t *testing.T) {
		q := NewSeqQueue(func(seq uint64, d interface{}) {})

		complete := q.Offer(0, 0)
		assert.False(t, complete)
		assert.True(t, q.SetMaxSeq(0))

		q = NewSeqQueue(func(seq uint64, d interface{}) {})
		assert.False(t, q.SetMaxSeq(0))
		complete = q.Offer(0, 0)
		assert.True(t, complete)
	})

	t.Run("Duplicate Data", func(t *testing.T) {
		var data []interface{}
		q := NewSeqQueue(func(seq uint64, d interface{}) {
			data = append(data, d)
		})

		for i := 0; i < MaxSeq; i++ {
			complete := q.Offer(uint64(i), i)
			assert.Len(t, data, i+1)
			assert.Equal(t, i, data[i].(int))
			assert.False(t, complete)
		}

		size := len(data)
		for i := 0; i < MaxSeq; i++ {
			complete := q.Offer(0, 0)
			assert.Len(t, data, size)
			assert.False(t, complete)
		}

		assert.False(t, q.SetMaxSeq(MaxSeq))
		assert.True(t, q.SetMaxSeq(MaxSeq-1))
	})

	t.Run("Random Concurrent Duplicate Sequence", func(t *testing.T) {
		const (
			serialLen = 10000
			workers   = 100
		)
		var result []uint64

		finished := make(chan struct{})
		q := NewSeqQueue(func(seq uint64, d interface{}) {
			result = append(result, d.(uint64))

			if seq == serialLen {
				close(finished)
			}
		})

		start := make(chan struct{})
		for i := 0; i < workers; i++ {
			go func() {
				var serial []uint64
				for i := serialLen; i >= 0; i-- {
					serial = append(serial, uint64(i))
				}

				rand.Shuffle(len(serial), func(i, j int) {
					serial[i], serial[j] = serial[j], serial[i]
				})

				<-start

				for _, i := range serial {
					_ = q.Offer(i, i)
				}
			}()
		}

		q.SetMaxSeq(serialLen)
		close(start)

		var expected []uint64
		for i := 0; i <= serialLen; i++ {
			expected = append(expected, uint64(i))
		}

		<-finished

		assert.EqualValues(t, expected, result)
	})
}
