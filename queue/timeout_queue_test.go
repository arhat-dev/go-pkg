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
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkTimeoutQueue(b *testing.B) {
	stopCh := make(chan struct{})
	q := NewTimeoutQueue()
	q.Start(stopCh)

	wg := new(sync.WaitGroup)

	b.N = 10000000

	//wg.Add(1)
	go func() {
		//defer wg.Done()
		dataCh := q.TakeCh()
		for i := 0; i < b.N; i++ {
			<-dataCh
			wg.Done()
		}
	}()

	wg.Add(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.OfferWithDelay(i, i, time.Millisecond)
	}

	wg.Wait()
}

func TestTimeoutQueue(t *testing.T) {
	cmd := make(map[string]string)
	ctx, cancel := context.WithCancel(context.TODO())
	q := NewTimeoutQueue()
	q.Start(ctx.Done())

	defer func() {
		cancel()

		_ = q.OfferWithDelay(0, cmd, time.Second)
		assert.Equal(t, 0, len(q.data))
	}()

	_ = q.OfferWithDelay(0, cmd, time.Second)
	start := time.Now()
	<-q.TakeCh()
	dur := time.Since(start)
	if dur < time.Second/2 {
		assert.FailNow(t, "wait time not match", "dur %s", dur)
	}
	assert.Equal(t, 0, len(q.data))
	assert.Equal(t, 0, len(q.timeoutCh))

	const N = 100
	for i := 0; i < N; i++ {
		_ = q.OfferWithDelay(uint64(i), cmd, time.Duration(i+1)*10*time.Millisecond)
	}

	start = time.Now()
	for i := 0; i < N; i++ {
		<-q.TakeCh()
		// t.Logf("[%d] wait: %v", i, time.Since(start))
	}

	for i := 0; i < N; i++ {
		_ = q.OfferWithDelay(uint64(i), cmd, time.Hour)
	}

	deleted := 0
	for i := 0; i < N*2; i += 2 {
		_, ok := q.Remove(uint64(i))
		if ok {
			deleted++
		}
	}
	assert.Equal(t, N-deleted, len(q.index))
	assert.Equal(t, N-deleted, len(q.data))
	q.Clear()

	for i := 0; i < N; i++ {
		_ = q.OfferWithDelay(uint64(i), cmd, time.Duration(i)*10*time.Millisecond)
	}

	time.Sleep(time.Second)
	start = time.Now()
	for i := 0; i < N; i++ {
		<-q.TakeCh()

		dur := time.Since(start)
		if dur > 20*time.Millisecond {
			assert.FailNow(t, "wait time exceeded")
		}
		// t.Logf("[%d] wait: %v", i, dur)
	}

	for i := 0; i < N; i++ {
		_ = q.OfferWithDelay(uint64(i), cmd, time.Hour)
	}

	assert.Equal(t, N, len(q.data))
	q.Clear()
	assert.Equal(t, 0, len(q.data))

	_ = q.OfferWithDelay(0, cmd, time.Second)
	_ = q.OfferWithDelay(0, cmd, time.Millisecond)
	start = time.Now()
	<-q.TakeCh()
	dur = time.Since(start)
	if dur > 10*time.Millisecond {
		assert.FailNow(t, "wait time exceed", "dur: %s", dur)
	}
	<-q.TakeCh()
	dur = time.Since(start)
	if dur > 2*time.Second || dur < time.Second/2 {
		assert.FailNow(t, "wait time not match", "dur: %s", dur)
	}

	for i := 0; i < N; i++ {
		_ = q.OfferWithDelay(uint64(1), time.Second, time.Microsecond)

		data := <-q.TakeCh()

		key, ok := data.Key.(uint64)
		assert.True(t, ok)
		assert.Equal(t, uint64(1), key)

		val, ok := data.Data.(time.Duration)
		assert.True(t, ok)
		assert.Equal(t, time.Second, val)
	}
}
