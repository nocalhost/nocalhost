package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/vpn/util"
)

type CustomResourceDefinitionController struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	resource  string
	name      string
}

func NewCustomResourceDefinitionController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, resource, name string) *CustomResourceDefinitionController {
	return &CustomResourceDefinitionController{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		resource:  resource,
		name:      name,
	}
}

// ScaleToZero TODO needs to create a same pod name, but with different labels for using to click
func (crd *CustomResourceDefinitionController) ScaleToZero() (map[string]string, []v1.ContainerPort, string, error) {
	topController := util.GetTopController(crd.factory, crd.clientset, crd.namespace, fmt.Sprintf("%s/%s", crd.getResource(), crd.name))
	// controllerBy is empty
	if len(topController.Name) == 0 || len(topController.Resource) == 0 {
		get, err := crd.clientset.CoreV1().Pods(crd.namespace).Get(context.TODO(), crd.name, metav1.GetOptions{})
		if err != nil {
			return nil, nil, "", err
		}
		_ = crd.clientset.CoreV1().Pods(crd.namespace).Delete(context.TODO(), crd.name, metav1.DeleteOptions{})
		return get.GetLabels(), get.Spec.Containers[0].Ports, "", nil
	}
	object, err := util.GetUnstructuredObject(crd.factory, crd.namespace, fmt.Sprintf("%s/%s", topController.Resource, topController.Name))
	helper := resource.NewHelper(object.Client, object.Mapping)
	//crd.f = func() error {
	//	_, err = helper.Create(crd.namespace, true, object.Object)
	//	return err
	//}
	if _, err = helper.Delete(crd.namespace, object.Name); err != nil {
		return nil, nil, "", err
	}
	marshal, _ := json.Marshal(object.Object)
	return util.GetLabelSelector(object.Object).MatchLabels, util.GetPorts(object.Object), string(marshal), err
}

func (crd *CustomResourceDefinitionController) Cancel() error {
	return crd.Reset()
}

func (crd CustomResourceDefinitionController) getResource() string {
	return crd.resource
}

func (crd *CustomResourceDefinitionController) Reset() error {
	get, err := crd.clientset.CoreV1().
		Pods(crd.namespace).
		Get(context.TODO(), ToInboundPodName(crd.getResource(), crd.name), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if a := get.GetAnnotations()[util.OriginData]; len(a) != 0 {
		var r unstructured.Unstructured
		if err = json.Unmarshal([]byte(a), &r); err != nil {
			return err
		}
		client, err := crd.factory.DynamicClient()
		if err != nil {
			return err
		}
		mapper, err := crd.factory.ToRESTMapper()
		if err != nil {
			return err
		}
		mapping, err := mapper.RESTMapping(r.GetObjectKind().GroupVersionKind().GroupKind(), r.GetObjectKind().GroupVersionKind().Version)
		if err != nil {
			return err
		}
		_, err = client.Resource(mapping.Resource).Create(context.TODO(), &r, metav1.CreateOptions{})
		return err
	}
	return nil
}
