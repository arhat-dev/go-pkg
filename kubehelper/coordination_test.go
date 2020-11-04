/*
Copyright 2020 The arhat.dev Authors.

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

package kubehelper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateLeaseClient(t *testing.T) {
	tests := []struct {
		name     string
		resource *metav1.APIResourceList
	}{
		{
			name: "v1",
			resource: &metav1.APIResourceList{
				GroupVersion: "coordination.k8s.io/v1",
				APIResources: []metav1.APIResource{{
					Name:         "leases",
					SingularName: "",
					Namespaced:   true,
					Kind:         "Lease",
				}},
			},
		},
		{
			name: "v1beta1",
			resource: &metav1.APIResourceList{
				GroupVersion: "coordination.k8s.io/v1beta1",
				APIResources: []metav1.APIResource{{
					Name:         "leases",
					SingularName: "",
					Namespaced:   true,
					Kind:         "Lease",
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k := fake.NewSimpleClientset()
			c := CreateLeaseClient([]*metav1.APIResourceList{test.resource}, k, "")

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
