package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/vpn/util"
)

type PodController struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	resource  string
	name      string
	f         func() error
}

func NewPodController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, resource, name string) *PodController {
	return &PodController{
		factory:   factory,
		clientset: clientset,
		resource:  resource,
		namespace: namespace,
		name:      name,
	}
}

// ScaleToZero TODO needs to create a same pod name, but with different labels for using to click
func (pod *PodController) ScaleToZero() (map[string]string, []v1.ContainerPort, error) {
	topController := util.GetTopController(pod.factory, pod.clientset, pod.namespace, fmt.Sprintf("%s/%s", pod.resource, pod.name))
	// controllerBy is empty
	if len(topController.Name) == 0 || len(topController.Resource) == 0 {
		get, err := pod.clientset.CoreV1().Pods(pod.namespace).Get(context.TODO(), pod.name, metav1.GetOptions{})
		if err != nil {
			return nil, nil, err
		}
		pod.f = func() error {
			_, err = pod.clientset.CoreV1().Pods(pod.namespace).Create(context.TODO(), get, metav1.CreateOptions{})
			if err != nil {
				log.Warnln(err)
			}
			return err
		}
		_ = pod.clientset.CoreV1().Pods(pod.namespace).Delete(context.TODO(), pod.name, metav1.DeleteOptions{})
		return get.GetLabels(), get.Spec.Containers[0].Ports, nil
	}
	object, err := util.GetUnstructuredObject(pod.factory, pod.namespace, fmt.Sprintf("%s/%s", topController.Resource, topController.Name))
	helper := resource.NewHelper(object.Client, object.Mapping)
	pod.f = func() error {
		_, err = helper.Create(pod.namespace, true, object.Object)
		return err
	}
	if _, err = helper.Delete(pod.namespace, object.Name); err != nil {
		return nil, nil, err
	}
	return util.GetLabelSelector(object.Object).MatchLabels, util.GetPorts(object.Object), err
}

func (pod *PodController) Cancel() error {
	return pod.f()
}

func (pod PodController) getResource() string {
	return "pods"
}

func (pod *PodController) Reset() error {
	get, err := pod.clientset.CoreV1().
		Pods(pod.namespace).
		Get(context.TODO(), toInboundPodName(pod.getResource(), pod.name), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if a := get.GetAnnotations()[util.OriginData]; len(a) != 0 {
		var r unstructured.Unstructured
		if err = json.Unmarshal([]byte(a), &r); err != nil {
			return err
		}
		client, err := pod.factory.DynamicClient()
		if err != nil {
			return err
		}
		mapper, err := pod.factory.ToRESTMapper()
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
