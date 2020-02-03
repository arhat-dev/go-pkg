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
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	ErrWorkDuplicate  = errors.New("work duplicated")
	ErrWorkConflict   = errors.New("work conflicted")
	ErrWorkCounteract = errors.New("work counteracted")
	ErrWorkInvalid    = errors.New("work invalid")
)

type WorkAction uint8

const (
	ActionInvalid WorkAction = iota
	ActionAdd
	ActionUpdate
	ActionDelete
	ActionCleanup
)

var actionNames = map[WorkAction]string{
	ActionInvalid: "Invalid",
	ActionAdd:     "Add",
	ActionUpdate:  "Update",
	ActionDelete:  "Delete",
	ActionCleanup: "Cleanup",
}

func (t WorkAction) String() string {
	return actionNames[t]
}

type Work struct {
	Action WorkAction
	Key    interface{}
}

func (w Work) String() string {
	if s, ok := w.Key.(fmt.Stringer); ok {
		return w.Action.String() + "/" + s.String()
	}

	return fmt.Sprintf("%s/%v", w.Action.String(), w.Key)
}

// NewWorkQueue will create a stopped new work queue,
// you can offer work to it, but any acquire will fail until
// you have called its Resume()
func NewWorkQueue() *WorkQueue {
	// prepare a closed channel for this work queue
	hasWork := make(chan struct{})
	close(hasWork)

	return &WorkQueue{
		queue: make([]Work, 0, 16),
		index: make(map[Work]int),

		// set work queue to closed
		hasWork:    hasWork,
		chanClosed: true,
		mu:         new(sync.RWMutex),

		closed: 1,
	}
}

// WorkQueue
type WorkQueue struct {
	queue []Work
	index map[Work]int

	hasWork    chan struct{}
	chanClosed bool
	mu         *sync.RWMutex

	// protected by atomic
	closed uint32
}

func (q *WorkQueue) has(action WorkAction, key interface{}) bool {
	_, ok := q.index[Work{Action: action, Key: key}]
	return ok
}

func (q *WorkQueue) add(w Work) {
	q.index[w] = len(q.queue)
	q.queue = append(q.queue, w)
}

func (q *WorkQueue) delete(action WorkAction, key interface{}) {
	workToDelete := Work{Action: action, Key: key}
	if idx, ok := q.index[workToDelete]; ok {
		delete(q.index, workToDelete)
		q.queue = append(q.queue[:idx], q.queue[idx+1:]...)

		// refresh index
		for i, w := range q.queue {
			q.index[w] = i
		}
	}
}

func (q *WorkQueue) Remains() []Work {
	q.mu.RLock()
	defer q.mu.RUnlock()

	works := make([]Work, len(q.queue))
	for i, w := range q.queue {
		works[i] = Work{Action: w.Action, Key: w.Key}
	}
	return works
}

func (q *WorkQueue) Find(key interface{}) (Work, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, t := range []WorkAction{ActionAdd, ActionUpdate, ActionDelete, ActionCleanup} {
		i, ok := q.index[Work{Action: t, Key: key}]
		if ok {
			return q.queue[i], true
		}
	}

	return Work{}, false
}

// Acquire a work item from the work queue
// if shouldAcquireMore is false, w will be an empty work
func (q *WorkQueue) Acquire() (w Work, shouldAcquireMore bool) {
	// wait until we have got some work to do
	// or we have stopped stopped work queue
	<-q.hasWork

	if q.isClosed() {
		return Work{Action: ActionInvalid}, false
	}

	q.mu.Lock()
	defer func() {
		if len(q.queue) == 0 {
			if !q.isClosed() {
				q.hasWork = make(chan struct{})
			}
		}

		q.mu.Unlock()
	}()

	if len(q.queue) == 0 {
		return Work{Action: ActionInvalid}, true
	}

	w = q.queue[0]
	q.delete(w.Action, w.Key)

	return w, true
}

// Offer a work item to the work queue
// if offered work was not added, an error result will return, otherwise nil
func (q *WorkQueue) Offer(action WorkAction, key interface{}) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if action == ActionInvalid {
		return ErrWorkInvalid
	}

	newJob := Work{Action: action, Key: key}
	_, dup := q.index[newJob]
	if dup {
		return ErrWorkDuplicate
	}

	switch action {
	case ActionAdd:
		if q.has(ActionUpdate, key) {
			return ErrWorkConflict
		}

		q.add(newJob)
	case ActionUpdate:
		if q.has(ActionAdd, key) || q.has(ActionDelete, key) {
			return ErrWorkConflict
		}

		q.add(newJob)
	case ActionDelete:
		// pod need to be deleted
		if q.has(ActionAdd, key) {
			// cancel according create work
			q.delete(ActionAdd, key)
			return ErrWorkCounteract
		}

		if q.has(ActionUpdate, key) {
			// if you want to delete it now, update operation doesn't matter any more
			q.delete(ActionUpdate, key)
		}

		q.add(newJob)
	case ActionCleanup:
		// cleanup job only requires no duplication

		q.add(newJob)
	}

	// we reach here means we have added some work to the queue
	// we should signal those consumers to go for it
	select {
	case <-q.hasWork:
		// we can reach here means q.hasWork has been closed
	default:
		// release the signal
		close(q.hasWork)
		// mark the channel closed to prevent a second close which would panic
		q.chanClosed = true
	}

	return nil
}

// Resume do nothing but mark you can perform acquire
// actions to the work queue
func (q *WorkQueue) Resume() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.chanClosed && len(q.queue) == 0 {
		// reopen signal channel for wait
		q.hasWork = make(chan struct{})
		q.chanClosed = false
	}

	atomic.StoreUint32(&q.closed, 0)
}

// Pause do nothing but mark this work queue is closed,
// you should not perform acquire actions to the work queue
func (q *WorkQueue) Pause() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.chanClosed {
		// close wait channel to prevent wait
		close(q.hasWork)
		q.chanClosed = true
	}

	atomic.StoreUint32(&q.closed, 1)
}

func (q *WorkQueue) isClosed() bool {
	return atomic.LoadUint32(&q.closed) == 1
}
