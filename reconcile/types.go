package reconcile

import (
	"errors"
	"time"

	"arhat.dev/pkg/backoff"
	"arhat.dev/pkg/log"
	"arhat.dev/pkg/queue"
)

var (
	resultCacheNotFound = &Result{Err: errors.New("cache not found")}
)

type Interface interface {
	Start() error
	ReconcileUntil(stop <-chan struct{})
	Schedule(job queue.Job, delay time.Duration) error
	CancelSchedule(job queue.Job) bool
}

type Options struct {
	Logger          log.Interface
	BackoffStrategy *backoff.Strategy
	Workers         int
	RequireCache    bool
	Handlers        HandleFuncs
}

func (o Options) ResolveNil() *Options {
	result := &Options{
		Logger:          o.Logger,
		BackoffStrategy: o.BackoffStrategy,
		Workers:         o.Workers,
		Handlers:        o.Handlers,
	}

	if result.Logger == nil {
		result.Logger = log.NoOpLogger
	}

	if result.BackoffStrategy == nil {
		result.BackoffStrategy = backoff.NewStrategy(300*time.Millisecond, 10*time.Second, 1.5, 3)
	}

	return result
}

type Result struct {
	NextAction    queue.JobAction
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

func (h *HandleFuncs) ResolveNil() *HandleFuncs {
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
