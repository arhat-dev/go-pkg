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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkQueue_delete(t *testing.T) {
	const (
		workCount = 100
	)

	q := NewJobQueue()
	for i := 0; i < workCount; i++ {
		assert.NoError(t, q.Offer(Job{ActionAdd, strconv.Itoa(i)}))
	}

	for i := 0; i < workCount/2; i++ {
		// delete nothing
		q.delete(ActionDelete, strconv.Itoa(i))
		assert.Equal(t, workCount, len(q.queue))
		assert.Equal(t, workCount, len(q.index))
	}

	j := 0
	for i := 0; i < workCount; i += 2 {
		podUID := strconv.Itoa(i)
		nextPodUID := strconv.Itoa(i + 1)

		q.delete(ActionAdd, podUID)

		assert.Equal(t, workCount-i/2-1, len(q.queue))
		assert.False(t, q.has(ActionAdd, podUID))

		idxInWorkQueue, ok := q.index[Job{Action: ActionAdd, Key: nextPodUID}]
		assert.True(t, ok)
		assert.Equal(t, j, idxInWorkQueue)
		nextWork := q.queue[idxInWorkQueue]
		assert.Equal(t, ActionAdd, nextWork.Action)
		assert.Equal(t, nextPodUID, nextWork.Key)
		j++
	}

}

func TestWorkQueueLogic(t *testing.T) {
	var (
		foo = "foo"
	)
	q := NewJobQueue()
	assert.True(t, q.isPaused())
	for i := 0; i < 10000; i++ {
		// work should be invalid since work queue has been closed
		work, more := q.Acquire()
		assert.False(t, more)
		assert.Equal(t, ActionInvalid, work.Action)
	}

	q.Resume()
	assert.False(t, q.isPaused())

	assert.NoError(t, q.Offer(Job{ActionUpdate, foo}))
	assert.Equal(t, ErrJobDuplicated, q.Offer(Job{ActionUpdate, foo}))

	assert.Equal(t, 1, len(q.queue))
	assert.Equal(t, 1, len(q.index))

	work, more := q.Acquire()
	assert.True(t, more)
	assert.Equal(t, ActionUpdate, work.Action)
	assert.Equal(t, foo, work.Key)
	assert.Equal(t, 0, len(q.queue))
	assert.Equal(t, 0, len(q.index))

	assert.NoError(t, q.Offer(Job{ActionAdd, foo}))
	assert.Equal(t, ErrJobDuplicated, q.Offer(Job{ActionAdd, foo}))
	assert.Equal(t, 1, len(q.queue))
	assert.Equal(t, 1, len(q.index))

	assert.Equal(t, ErrJobCounteract, q.Offer(Job{ActionDelete, foo}))
	assert.Equal(t, 0, len(q.queue))
	assert.Equal(t, 0, len(q.index))

	assert.NoError(t, q.Offer(Job{ActionUpdate, foo}))
	assert.NoError(t, q.Offer(Job{ActionDelete, foo}))
	assert.Equal(t, 1, len(q.queue))
	assert.Equal(t, 1, len(q.index))

	work, more = q.Acquire()
	assert.True(t, more)
	assert.Equal(t, ActionDelete, work.Action)
	assert.Equal(t, foo, work.Key)
	assert.Equal(t, 0, len(q.queue))
	assert.Equal(t, 0, len(q.index))
}

func TestWorkQueueAction(t *testing.T) {
	const (
		WorkCount    = 100
		TargetAction = ActionAdd
		WaitTime     = 10 * time.Millisecond
	)

	q := NewJobQueue()

	sigCh := make(chan struct{})
	finished := func() bool {
		select {
		case <-sigCh:
			return true
		default:
			return false
		}
	}

	startTime := time.Now()
	go func() {
		defer func() {
			time.Sleep(time.Second)
			q.Pause()

			close(sigCh)
		}()

		for i := 0; i < WorkCount; i++ {
			// some random pause and resume
			if i == WorkCount/4 {
				q.Resume()
			}

			if i == WorkCount/2 {
				q.Pause()
			}

			if i == WorkCount*3/4 {
				q.Resume()
			}

			time.Sleep(WaitTime)

			_ = q.Offer(Job{TargetAction, strconv.Itoa(i)})
		}
	}()

	invalidCount := 0
	validCount := 0
	for !finished() {
		work, more := q.Acquire()
		if !more {
			invalidCount++
			assert.False(t, more)
			assert.Equal(t, ActionInvalid, work.Action)
		} else {
			validCount++
			assert.True(t, more)
			assert.Equal(t, TargetAction, work.Action)
		}
	}

	assert.GreaterOrEqual(t, int64(time.Since(startTime)), int64(WorkCount*WaitTime), "work time less than expected")

	assert.NotEqual(t, 0, invalidCount, "invalid count should not be zero")
	assert.InDelta(t, WorkCount, validCount, 1.5)
}
