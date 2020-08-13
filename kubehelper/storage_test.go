package kubehelper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateCSIDriverLister(t *testing.T) {
	tests := []struct {
		name     string
		resource *metav1.APIResourceList
	}{
		{
			name: "v1",
			resource: &metav1.APIResourceList{
				GroupVersion: "storage.k8s.io/v1",
				APIResources: []metav1.APIResource{{
					Name:         "csidrivers",
					SingularName: "",
					Namespaced:   false,
					Kind:         "CSIDriver",
				}},
			},
		},
		{
			name: "v1beta1",
			resource: &metav1.APIResourceList{
				GroupVersion: "storage.k8s.io/v1beta1",
				APIResources: []metav1.APIResource{{
					Name:         "csidrivers",
					SingularName: "",
					Namespaced:   false,
					Kind:         "CSIDriver",
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k := fake.NewSimpleClientset()
			c := CreateCSIDriverClient([]*metav1.APIResourceList{test.resource}, k)

			switch test.name {
			case "v1":
				assert.NotNil(t, c.V1Client)
			case "v1beta1":
				assert.NotNil(t, c.V1b1Client)
			default:
				assert.FailNow(t, "unknown version")
			}
		})
	}
}
