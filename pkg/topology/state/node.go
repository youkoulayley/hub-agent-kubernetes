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
	"fmt"

	discovery "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (f *Fetcher) getNodes() (Nodes, error) {
	apis, err := f.hub.Hub().V1alpha1().APIs().Lister().List(labels.Everything())
	if err != nil {
		return Nodes{}, fmt.Errorf("list apis: %w", err)
	}

	nodes := make(map[string]struct{})
	for _, api := range apis {
		serviceSelector := labels.SelectorFromSet(map[string]string{"kubernetes.io/service-name": api.Spec.Service.Name})

		var endpoints []*discovery.EndpointSlice
		endpoints, err = f.k8s.Discovery().V1().EndpointSlices().Lister().EndpointSlices(api.Namespace).List(serviceSelector)
		if err != nil {
			return Nodes{}, fmt.Errorf("get endpoint slices %q in %q: %w", api.Spec.Service.Name, api.Namespace, err)
		}

		for _, endpoint := range endpoints {
			for _, e := range endpoint.Endpoints {
				if e.NodeName == nil {
					continue
				}

				nodes[*e.NodeName] = struct{}{}
			}
		}
	}

	allNodes, err := f.k8s.Core().V1().Nodes().Lister().List(labels.Everything())
	if err != nil {
		return Nodes{}, fmt.Errorf("get nodes: %w", err)
	}

	return Nodes{RunningAPIs: len(nodes), Total: len(allNodes)}, nil
}
