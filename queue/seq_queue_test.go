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

func BenchmarkSeqQueue_BestCase(b *testing.B) {
	n := uint64(b.N)
	q := NewSeqQueue()
	q.SetMaxSeq(n)

	b.ResetTimer()
	for i := uint64(0); i <= n; i++ {
		q.Offer(i, 0)
	}
}

// func BenchmarkSeqQueue_WorstCase(b *testing.B) {
// 	n := uint64(b.N)
// 	q := NewSeqQueue()
// 	q.SetMaxSeq(n)
// 	q.Offer(0, 0)

// 	b.ResetTimer()
// 	for i := n; i > 0; i-- {
// 		q.Offer(i, 0)
// 	}
// }

func TestSeqQueue(t *testing.T) {
	const MaxSeq = 100

	var (
		sequentialSerial []uint64
		reversedSerial   []uint64
	)
	for i := 0; i <= MaxSeq; i++ {
		sequentialSerial = append(sequentialSerial, uint64(i))
	}
	for i := MaxSeq; i >= 0; i-- {
		reversedSerial = append(reversedSerial, uint64(i))
	}

	t.Run("Sequential One Out Per Offer", func(t *testing.T) {
		q := NewSeqQueue()
		q.SetMaxSeq(MaxSeq)

		for idx, i := range sequentialSerial {
			data, complete := q.Offer(i, i)

			assert.Len(t, data, 1)
			assert.Equal(t, i, data[0].(uint64))

			if idx == len(sequentialSerial)-1 {
				assert.True(t, complete)
			} else {
				assert.False(t, complete)
			}
		}
	})

	t.Run("Reversed No Data Until Last", func(t *testing.T) {
		q := NewSeqQueue()
		q.SetMaxSeq(MaxSeq)

		for idx, i := range reversedSerial {
			data, complete := q.Offer(i, i)

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
		q := NewSeqQueue()

		_, complete := q.Offer(0, 0)
		assert.False(t, complete)
		assert.True(t, q.SetMaxSeq(0))

		q = NewSeqQueue()
		assert.False(t, q.SetMaxSeq(0))
		_, complete = q.Offer(0, 0)
		assert.True(t, complete)
	})

	t.Run("Duplicated Data", func(t *testing.T) {
		q := NewSeqQueue()
		for i := 0; i < MaxSeq; i++ {
			data, complete := q.Offer(uint64(i), i)
			assert.Equal(t, 1, len(data))
			assert.Equal(t, i, data[0].(int))
			assert.False(t, complete)
		}

		for i := 0; i < MaxSeq; i++ {
			data, complete := q.Offer(0, 0)
			assert.Len(t, data, 0)
			assert.False(t, complete)
		}

		assert.False(t, q.SetMaxSeq(MaxSeq))
		assert.True(t, q.SetMaxSeq(MaxSeq-1))
	})
}
