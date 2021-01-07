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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	"nocalhost/pkg/nhctl/log"
)

func (c *ClientGoUtils) DeleteConfigMapByName(name string) error {
	//var err error
	//if namespace == "" {
	//	namespace, err = c.GetDefaultNamespace()
	//	if err != nil {
	//		return err
	//	}
	//}
	return c.ClientSet.CoreV1().ConfigMaps(c.namespace).Delete(c.ctx, name, metav1.DeleteOptions{})
}

func (c *ClientGoUtils) Delete(yamlPath string) error {
	if yamlPath == "" {
		return errors.New("yaml path can not be empty")
	}
	//if namespace == "" {
	//	namespace = "default"
	//}

	filebytes, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(filebytes), 100)
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		gr, err := restmapper.GetAPIGroupResources(c.ClientSet.Discovery())
		if err != nil {
			return err
		}

		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			log.Fatal(err)
		}

		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			//if namespace != "" {
			unstructuredObj.SetNamespace(c.namespace)
			//} else if unstructuredObj.GetNamespace() == "" {
			//	unstructuredObj.SetNamespace("default")
			//}
			dri = c.dynamicClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = c.dynamicClient.Resource(mapping.Resource)
		}

		propagationPolicy := metav1.DeletePropagationBackground
		err = dri.Delete(context.Background(), unstructuredObj.GetName(), metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if err != nil {
			return err
		}

		fmt.Printf("%s/%s deleted\n", unstructuredObj.GetKind(), unstructuredObj.GetName())

	}
	return nil
}
