// +build !nocloud,!nokube

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

	"k8s.io/apimachinery/pkg/api/meta"
	kubecache "k8s.io/client-go/tools/cache"

	"arhat.dev/pkg/log"
	"arhat.dev/pkg/queue"
)

func NewKubeInformerReconciler(
	ctx context.Context,
	informer kubecache.SharedInformer,
	options Options,
) *KubeInformerReconciler {
	resolvedOpts := options.ResolveNil()

	r := &KubeInformerReconciler{
		log:  resolvedOpts.Logger,
		Core: NewCore(ctx, resolvedOpts),
	}

	informer.AddEventHandler(kubecache.ResourceEventHandlerFuncs{
		AddFunc:    r.getInformerAddEventFunc(),
		UpdateFunc: r.getInformerUpdateEventFunc(),
		DeleteFunc: r.getInformerDeleteEventFunc(),
	})

	return r
}

type KubeInformerReconciler struct {
	log log.Interface

	*Core
}

func (r *KubeInformerReconciler) getInformerAddEventFunc() func(interface{}) {
	logger := r.log.WithFields(log.String("func", "add"))
	return func(obj interface{}) {
		key, err := r.getKeyOfObject(obj)
		if err != nil {
			logger.I("failed to get key for object", log.Error(err), log.Any("obj", obj))
			return
		}

		logger := logger.WithFields(log.Any("key", key))

		r.Core.Update(key, nil, obj)
		logger.V("scheduling create job")
		err = r.Core.Schedule(queue.Job{Action: queue.ActionAdd, Key: key}, 0)
		if err != nil {
			logger.I("failed to schedule create job", log.Error(err))
		}
	}
}

func (r *KubeInformerReconciler) getInformerUpdateEventFunc() func(old, new interface{}) {
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

		r.Core.Update(key, old, new)
		ts := o.GetDeletionTimestamp()
		if ts != nil && !ts.IsZero() {
			// to be deleted
			logger.V("scheduling delete job")
			err = r.Core.Schedule(queue.Job{Action: queue.ActionDelete, Key: key}, 0)
			if err != nil {
				logger.I("failed to schedule delete job", log.Error(err))
			}
		} else {
			// need to keep old object until user defined update operation is successful
			// so we can calculate actual delta on our own to achieve eventual consensus
			r.Core.Freeze(key, true)

			logger.V("scheduling update job")
			err = r.Core.Schedule(queue.Job{Action: queue.ActionUpdate, Key: key}, 0)
			if err != nil {
				logger.I("failed to schedule update job", log.Error(err))
			}
		}
	}
}

func (r *KubeInformerReconciler) getInformerDeleteEventFunc() func(interface{}) {
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

		r.Core.Update(key, nil, obj)
		logger.V("scheduling cleanup job")
		err := r.Core.Schedule(queue.Job{Action: queue.ActionCleanup, Key: key}, 0)
		if err != nil {
			logger.I("failed to schedule cleanup job", log.Error(err))
		}
	}
}

func (r *KubeInformerReconciler) getKeyOfObject(obj interface{}) (string, error) {
	return kubecache.MetaNamespaceKeyFunc(obj)
}
