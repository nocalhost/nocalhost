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

type ReplicasController struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
	f         func() error
}

func NewReplicasController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *ReplicasController {
	return &ReplicasController{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

func (c *ReplicasController) ScaleToZero() (map[string]string, []v1.ContainerPort, error) {
	updateScale, err2 := c.clientset.AppsV1().ReplicaSets(c.namespace).Get(context.TODO(), c.name, metav1.GetOptions{})
	if err2 != nil {
		return nil, nil, err2
	}
	_, err := c.clientset.AppsV1().ReplicaSets(c.namespace).UpdateScale(context.TODO(), c.name, &autoscalingv1.Scale{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.name,
			Namespace: c.namespace,
		},
		Spec: autoscalingv1.ScaleSpec{
			Replicas: int32(0),
		},
	}, metav1.UpdateOptions{})
	if err != nil {
		return nil, nil, err
	}
	c.f = func() error {
		_, err = c.clientset.AppsV1().ReplicaSets(c.namespace).
			UpdateScale(context.TODO(), c.name, &autoscalingv1.Scale{
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.name,
					Namespace: c.namespace,
				},
				Spec: autoscalingv1.ScaleSpec{
					Replicas: *updateScale.Spec.Replicas,
				},
			}, metav1.UpdateOptions{})
		return err
	}
	return updateScale.Spec.Template.Labels, updateScale.Spec.Template.Spec.Containers[0].Ports, nil
}

func (c *ReplicasController) Cancel() error {
	return c.f()
}

func (c *ReplicasController) getResource() string {
	return "replicasets"
}

func (c *ReplicasController) Reset() error {
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
