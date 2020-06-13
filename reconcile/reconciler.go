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

package reconcile

import (
	"context"
	"errors"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	kubecache "k8s.io/client-go/tools/cache"

	"arhat.dev/pkg/backoff"
	"arhat.dev/pkg/log"
	"arhat.dev/pkg/queue"
)

var (
	resultCacheNotFound = &Result{Err: errors.New("cache not found")}
)

type Result struct {
	NextAction    queue.WorkAction
	ScheduleAfter time.Duration
	Err           error
}

type (
	SingleObjectHandleFunc  func(obj interface{}) *Result
	CompareObjectHandleFunc func(old, new interface{}) *Result
)

func singleObjectAlwaysSuccess(_ interface{}) *Result {
	return nil
}

func compareObjectAlwaysSuccess(_, _ interface{}) *Result {
	return nil
}

type HandleFuncs struct {
	OnAdded    SingleObjectHandleFunc
	OnUpdated  CompareObjectHandleFunc
	OnDeleting SingleObjectHandleFunc
	OnDeleted  SingleObjectHandleFunc
}

func (h *HandleFuncs) replaceNilFuncWithAlwaysSuccess() *HandleFuncs {
	result := &HandleFuncs{
		OnAdded:    h.OnAdded,
		OnUpdated:  h.OnUpdated,
		OnDeleting: h.OnDeleting,
		OnDeleted:  h.OnDeleted,
	}

	if result.OnAdded == nil {
		result.OnAdded = singleObjectAlwaysSuccess
	}

	if result.OnUpdated == nil {
		result.OnUpdated = compareObjectAlwaysSuccess
	}

	if result.OnDeleting == nil {
		result.OnDeleting = singleObjectAlwaysSuccess
	}

	if result.OnDeleted == nil {
		result.OnDeleted = singleObjectAlwaysSuccess
	}

	return result
}

func NewReconciler(
	ctx context.Context,
	name string,
	informer kubecache.SharedInformer,
	h *HandleFuncs,
) *Reconciler {
	r := &Reconciler{
		ctx: ctx,
		log: log.Log.WithName(name),

		rescheduleQ: queue.NewTimeoutQueue(),
		jobQ:        queue.NewWorkQueue(),
		backoff:     backoff.NewStrategy(100*time.Millisecond, 10*time.Second, 1.5, 3),

		frozenKey: make(map[interface{}]struct{}),
		h:         h.replaceNilFuncWithAlwaysSuccess(),

		oldCache: make(map[interface{}]interface{}),
		cache:    make(map[interface{}]interface{}),

		workingOn: make(map[interface{}]queue.WorkAction),
		mu:        new(sync.RWMutex),
	}

	informer.AddEventHandler(kubecache.ResourceEventHandlerFuncs{
		AddFunc:    r.getAddFunc(),
		UpdateFunc: r.getUpdateFunc(),
		DeleteFunc: r.getDeleteFunc(),
	})

	return r
}

type Reconciler struct {
	ctx context.Context
	log *log.Logger

	rescheduleQ *queue.TimeoutQueue
	jobQ        *queue.WorkQueue
	backoff     *backoff.Strategy

	frozenKey map[interface{}]struct{}
	h         *HandleFuncs

	oldCache map[interface{}]interface{}
	cache    map[interface{}]interface{}

	workingOn map[interface{}]queue.WorkAction
	mu        *sync.RWMutex
}

func (r *Reconciler) Start(stop <-chan struct{}) {
	r.rescheduleQ.Start(stop)
	go func() {
		ch := r.rescheduleQ.TakeCh()

		for t := range ch {
			action := t.Data.(queue.WorkAction)
			err := r.jobQ.Offer(t.Data.(queue.WorkAction), t.Key)
			if err != nil {
				r.log.E("failed to reschedule job after backoff", log.Error(err), log.Any("job", queue.Work{Action: action, Key: t.Key}))
			}
		}
	}()
}

func (r *Reconciler) ReconcileUntil(stop <-chan struct{}) {
	r.jobQ.Resume()
	defer r.jobQ.Pause()

	go func() {
		for {
			job, more := r.jobQ.Acquire()
			if !more {
				return
			}

			r.handleJob(job)
		}
	}()

	select {
	case <-r.ctx.Done():
		return
	case <-stop:
		return
	}
}

func (r *Reconciler) handleJob(job queue.Work) {
	var (
		result *Result
		logger = r.log.WithFields(log.Any("job", job.String()))
	)

	if job.Action == queue.ActionInvalid {
		logger.V("invalid job discarded")
		return
	}

	previous, current := r.getObjectByKey(job.Key)

	logger.V("working on")

	switch job.Action {
	case queue.ActionAdd:
		if current == nil {
			result = resultCacheNotFound
			break
		}

		result = r.h.OnAdded(current)
	case queue.ActionUpdate:
		if previous == nil || current == nil {
			result = resultCacheNotFound
			break
		}

		result = r.h.OnUpdated(previous, current)
		if result == nil || result.Err == nil {
			// updated successfully, no need to keep old cache any more
			r.freezeOldCache(job.Key, false)
		}
	case queue.ActionDelete:
		if current == nil {
			result = resultCacheNotFound
			break
		}

		result = r.h.OnDeleting(current)
	case queue.ActionCleanup:
		if current == nil {
			result = resultCacheNotFound
			break
		}

		result = r.h.OnDeleted(current)

		if result == nil || result.NextAction == queue.ActionInvalid {
			// no further action for this key, check pending jobs with same key
			_, hasPendingJob := r.jobQ.Find(job.Key)
			if !hasPendingJob {
				// no pending job with this key
				logger.I("deleting cache")
				r.deleteObjectByKey(job.Key)
			}
		}
	default:
		logger.I("unknown action")
		return
	}

	if result == nil {
		return
	}

	nA := result.NextAction
	delay := result.ScheduleAfter
	if result.Err != nil {
		nA = job.Action
		if delay == 0 {
			delay = r.backoff.Next(job.Key)
		}
	} else {
		r.backoff.Reset(job.Key)
	}

	if nA == queue.ActionInvalid {
		return
	}

	nextJob := queue.Work{Action: nA, Key: job.Key}
	logger = logger.WithFields(log.Any("nextJob", nextJob))
	if delay > 0 {
		logger.V("scheduling next job with delay", log.Duration("delay", delay))
		err := r.rescheduleQ.OfferWithDelay(nextJob.Key, nextJob.Action, delay)
		if err != nil {
			logger.V("failed to reschedule job with delay", log.Error(err))
		}
	} else {
		logger.V("scheduling next job immediately")
		err := r.jobQ.Offer(nextJob.Action, nextJob.Key)
		if err != nil {
			logger.V("failed to schedule next job", log.Error(err))
		}
	}
}

func (r *Reconciler) getAddFunc() func(interface{}) {
	logger := r.log.WithFields(log.String("func", "add"))
	return func(obj interface{}) {
		key, err := r.getKeyOfObject(obj)
		if err != nil {
			logger.I("failed to get key for object", log.Error(err), log.Any("obj", obj))
			return
		}

		logger := logger.WithFields(log.Any("key", key))

		r.updateObjectByKey(key, nil, obj)
		logger.V("scheduling create job")
		err = r.jobQ.Offer(queue.ActionAdd, key)
		if err != nil {
			logger.I("failed to schedule create job", log.Error(err))
		}
	}
}

func (r *Reconciler) getUpdateFunc() func(old, new interface{}) {
	logger := r.log.WithFields(log.String("func", "update"))
	return func(old, new interface{}) {
		key, err := r.getKeyOfObject(old)
		if err != nil {
			logger.I("failed to get key for object", log.Error(err), log.Any("obj", old))
			return
		}

		o, err := meta.Accessor(new)
		if err != nil {
			logger.I("failed to access object meta", log.Error(err))
			return
		}

		logger := logger.WithFields(log.Any("key", key))

		r.updateObjectByKey(key, old, new)
		ts := o.GetDeletionTimestamp()
		if ts != nil && !ts.IsZero() {
			// to be deleted
			logger.V("scheduling delete job")
			err = r.jobQ.Offer(queue.ActionDelete, key)
			if err != nil {
				logger.I("failed to schedule delete job", log.Error(err))
			}
		} else {
			// need to keep old object until user defined update operation is successful
			// so we can calculate actual delta on our own to achieve eventual consensus
			r.freezeOldCache(key, true)

			logger.V("scheduling update job")
			err = r.jobQ.Offer(queue.ActionUpdate, key)
			if err != nil {
				logger.I("failed to schedule update job", log.Error(err))
			}
		}
	}
}

func (r *Reconciler) getDeleteFunc() func(interface{}) {
	logger := r.log.WithFields(log.String("func", "delete"))
	return func(obj interface{}) {
		var key string
		dfsu, ok := obj.(kubecache.DeletedFinalStateUnknown)
		if ok {
			key = dfsu.Key
			obj = dfsu.Obj
		} else {
			var err error
			key, err = r.getKeyOfObject(obj)
			if err != nil {
				logger.I("failed to get key for object", log.Error(err), log.Any("obj", obj))
				return
			}
		}

		logger := logger.WithFields(log.Any("key", key))

		r.updateObjectByKey(key, nil, obj)
		logger.V("scheduling cleanup job")
		err := r.jobQ.Offer(queue.ActionCleanup, key)
		if err != nil {
			logger.I("failed to schedule cleanup job", log.Error(err))
		}
	}
}

func (r *Reconciler) getKeyOfObject(obj interface{}) (string, error) {
	return kubecache.MetaNamespaceKeyFunc(obj)
}

func (r *Reconciler) freezeOldCache(key interface{}, freeze bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if freeze {
		r.frozenKey[key] = struct{}{}
	} else {
		delete(r.frozenKey, key)
	}
}

func (r *Reconciler) updateObjectByKey(key interface{}, old, latest interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, frozen := r.frozenKey[key]; !frozen {
		if old != nil {
			r.oldCache[key] = old
		} else {
			// move cached to old cached
			if o, ok := r.cache[key]; ok {
				r.oldCache[key] = o
			}
		}
	}

	if latest != nil {
		r.cache[key] = latest

		// fill old cache if not initialized
		if _, ok := r.oldCache[key]; !ok {
			r.oldCache[key] = latest
		}
	}
}

func (r *Reconciler) deleteObjectByKey(key interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.cache, key)
	delete(r.oldCache, key)
}

func (r *Reconciler) getObjectByKey(key interface{}) (old, latest interface{}) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.oldCache[key], r.cache[key]
}
