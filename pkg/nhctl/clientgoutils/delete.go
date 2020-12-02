package clientgoutils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
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

func (c *ClientGoUtils) DeleteConfigMapByName(name string, namespace string) error {
	var err error
	if namespace == "" {
		namespace, err = c.GetDefaultNamespace()
		if err != nil {
			return err
		}
	}
	return c.ClientSet.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (c *ClientGoUtils) Delete(yamlPath string, namespace string) error {
	if yamlPath == "" {
		return errors.New("yaml path can not be empty")
	}
	if namespace == "" {
		namespace = "default"
	}

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
			if namespace != "" {
				unstructuredObj.SetNamespace(namespace)
			} else if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
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
