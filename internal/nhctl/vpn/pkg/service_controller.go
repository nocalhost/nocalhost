package pkg

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/vpn/util"
)

type ServiceController struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
}

func NewServiceController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *ServiceController {
	return &ServiceController{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

func (s *ServiceController) ScaleToZero() (map[string]string, []v1.ContainerPort, string, error) {
	get, err := s.clientset.CoreV1().Services(s.namespace).Get(context.TODO(), s.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, "", err
	}

	object, err := util.GetUnstructuredObject(s.factory, s.namespace, fmt.Sprintf("services/%s", s.name))
	if err != nil {
		return nil, nil, "", err
	}
	asSelector, _ := metav1.LabelSelectorAsSelector(util.GetLabelSelector(object.Object))
	podList, _ := s.clientset.CoreV1().Pods(s.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: asSelector.String(),
	})
	if len(podList.Items) == 0 {
		var ports []v1.ContainerPort
		for _, port := range get.Spec.Ports {
			ports = append(ports, v1.ContainerPort{
				Name:          port.Name,
				ContainerPort: port.Port,
				Protocol:      port.Protocol,
			})
		}
		return get.Spec.Selector, ports, "", nil
	}
	// if podList is not one, needs to merge ???
	podController := NewPodController(s.factory, s.clientset, s.namespace, s.getResource(), podList.Items[0].Name)
	return podController.ScaleToZero()
}

func (s *ServiceController) Cancel() error {
	return s.Reset()
}

func (s ServiceController) getResource() string {
	return "services"
}

func (s *ServiceController) Reset() error {
	podController := NewPodController(s.factory, s.clientset, s.namespace, s.getResource(), s.name)
	return podController.Reset()
}
