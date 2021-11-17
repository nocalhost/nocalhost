package pkg

import (
	"context"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/vpn/util"
	"strconv"
)

type StatefulsetController struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
	f         func() error
}

func NewStatefulsetController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *StatefulsetController {
	return &StatefulsetController{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

func (c *StatefulsetController) ScaleToZero() (map[string]string, []v1.ContainerPort, error) {
	scale, err := c.clientset.AppsV1().StatefulSets(c.namespace).Get(context.TODO(), c.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	c.f = func() error {
		_, err = c.clientset.AppsV1().
			StatefulSets(c.namespace).
			UpdateScale(context.TODO(), c.name, &autoscalingv1.Scale{
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.name,
					Namespace: c.namespace,
				},
				Spec: autoscalingv1.ScaleSpec{
					Replicas: *scale.Spec.Replicas,
				},
			}, metav1.UpdateOptions{})
		return err
	}
	_, err = c.clientset.AppsV1().
		StatefulSets(c.namespace).
		UpdateScale(context.TODO(), c.name, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.name,
				Namespace: c.namespace,
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: 0,
			},
		}, metav1.UpdateOptions{})
	if err != nil {
		return nil, nil, err
	}
	return scale.Spec.Template.Labels, scale.Spec.Template.Spec.Containers[0].Ports, nil
}

func (c *StatefulsetController) Cancel() error {
	return c.f()
}

func (c *StatefulsetController) getResource() string {
	return "statefulsets"
}

func (c *StatefulsetController) Reset() error {
	get, err := c.clientset.CoreV1().
		Pods(c.namespace).
		Get(context.TODO(), toInboundPodName(c.getResource(), c.name), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if o := get.GetAnnotations()[util.OriginData]; len(o) != 0 {
		if n, err := strconv.Atoi(o); err == nil {
			util.UpdateReplicasScale(c.clientset, c.namespace, util.ResourceTupleWithScale{
				Resource: c.getResource(),
				Name:     c.name,
				Scale:    n,
			})
		}
	}
	return nil
}
