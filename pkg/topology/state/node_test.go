/*
Copyright (C) 2022-2023 Traefik Labs

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hubv1alpha1 "github.com/traefik/hub-agent-kubernetes/pkg/crd/api/hub/v1alpha1"
	hubfake "github.com/traefik/hub-agent-kubernetes/pkg/crd/generated/client/hub/clientset/versioned/fake"
	traefikcrdfake "github.com/traefik/hub-agent-kubernetes/pkg/crd/generated/client/traefik/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	kscheme "k8s.io/client-go/kubernetes/scheme"
)

func TestFetcher_GetNodes(t *testing.T) {
	err := hubv1alpha1.AddToScheme(kscheme.Scheme)
	require.NoError(t, err)

	hubObjects := []runtime.Object{
		&hubv1alpha1.API{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "api-1",
			},
			Spec: hubv1alpha1.APISpec{
				Service: hubv1alpha1.APIService{
					Name: "service-1",
				},
			},
		},
	}

	kubeObjects := []runtime.Object{
		&v1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "service-1",
				Labels:    map[string]string{"kubernetes.io/service-name": "service-1"},
			},
			Endpoints: []v1.Endpoint{
				{Hostname: stringPtr("pod-1"), NodeName: stringPtr("node-1")},
				{Hostname: stringPtr("pod-2"), NodeName: stringPtr("node-2")},
				{Hostname: stringPtr("pod-3"), NodeName: stringPtr("node-2")},
				{Hostname: stringPtr("pod-4"), NodeName: nil},
			},
		},
		&v1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "not-exposed",
				Name:      "service-1",
				Labels:    map[string]string{"kubernetes.io/service-name": "service-1"},
			},
			Endpoints: []v1.Endpoint{
				{Hostname: stringPtr("pod-5"), NodeName: stringPtr("node-3")},
			},
		},
		&v1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "service-3",
				Labels:    map[string]string{"kubernetes.io/service-name": "service-3"},
			},
			Endpoints: []v1.Endpoint{
				{Hostname: stringPtr("pod-6"), NodeName: stringPtr("node-3")},
			},
		},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-2"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-3"}},
	}

	kubeClient := kubefake.NewSimpleClientset(kubeObjects...)
	traefikClient := traefikcrdfake.NewSimpleClientset()
	hubClient := hubfake.NewSimpleClientset(hubObjects...)

	f, err := watchAll(context.Background(), kubeClient, traefikClient, hubClient, "v1.20.1")
	require.NoError(t, err)

	nodes, err := f.getNodes()
	require.NoError(t, err)

	assert.Equal(t, Nodes{RunningAPIs: 2, Total: 3}, nodes)
}
