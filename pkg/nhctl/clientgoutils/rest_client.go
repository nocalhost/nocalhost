/*
Copyright 2020 The Nocalhost Authors.
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

package clientgoutils

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const (
	ResourcePods = "pods"
)

func (c *ClientGoUtils) GetRestClient(gv *schema.GroupVersion) (*rest.RESTClient, error) {
	config, err := c.getRestConfig()
	if err != nil {
		return nil, err
	}

	config.APIPath = "api"
	config.GroupVersion = gv
	config.NegotiatedSerializer = scheme.Codecs
	return rest.RESTClientFor(config)
}

/*
	if namespace is empty, use "default" namespace
*/
func (c *ClientGoUtils) GetResourcesByRestClient(gv *schema.GroupVersion, resource string, result runtime.Object) error {
	restClient, err := c.GetRestClient(gv)
	if err != nil {
		return err
	}

	//if namespace == "" {
	//	namespace = "default"
	//}

	return restClient.Get().Namespace(c.namespace).Resource(resource).VersionedParams(&metav1.ListOptions{Limit: 500},
		scheme.ParameterCodec).Do(context.TODO()).Into(result)
}
