package reconcile

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	kubecache "k8s.io/client-go/tools/cache"
)

func TestReconciler_objectCache(t *testing.T) {
	const (
		objectKey   = "test"
		objectValue = "test"
	)
	informer := kubecache.NewSharedIndexInformer(nil, &corev1.Pod{}, 0, nil)
	r := NewKubeInformerReconciler(context.TODO(), "test.objectCache", informer, &HandleFuncs{})

	r.updateObjectByKey(objectKey, nil, objectValue)
	old, latest := r.getObjectByKey(objectKey)
	if s, ok := latest.(string); !ok {
		t.Errorf("new cache not updated")
	} else if s != objectValue {
		t.Errorf("new cache not expected: %q", s)
	}

	if s, ok := old.(string); !ok {
		t.Errorf("old cache not updated")
	} else if s != objectValue {
		t.Errorf("old cache not expected: %q", s)
	}
}
