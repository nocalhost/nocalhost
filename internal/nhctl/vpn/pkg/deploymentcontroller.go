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

type DeploymentController struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
	f         func() error
}

func NewDeploymentController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *DeploymentController {
	return &DeploymentController{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

func (d *DeploymentController) ScaleToZero() (map[string]string, []v1.ContainerPort, error) {
	scale, err2 := d.clientset.AppsV1().Deployments(d.namespace).GetScale(context.TODO(), d.name, metav1.GetOptions{})
	if err2 != nil {
		return nil, nil, err2
	}
	d.f = func() error {
		_, err := d.clientset.AppsV1().Deployments(d.namespace).UpdateScale(
			context.TODO(),
			d.name,
			&autoscalingv1.Scale{
				ObjectMeta: metav1.ObjectMeta{Name: d.name, Namespace: d.namespace},
				Spec:       autoscalingv1.ScaleSpec{Replicas: scale.Spec.Replicas},
			},
			metav1.UpdateOptions{},
		)
		return err
	}
	_, err := d.clientset.AppsV1().Deployments(d.namespace).UpdateScale(
		context.TODO(),
		d.name,
		&autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      d.name,
				Namespace: d.namespace,
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: int32(0),
			},
		},
		metav1.UpdateOptions{},
	)
	if err != nil {
		return nil, nil, err
	}
	get, err := d.clientset.AppsV1().Deployments(d.namespace).Get(context.TODO(), d.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	return get.Spec.Template.GetLabels(), get.Spec.Template.Spec.Containers[0].Ports, nil
}

func (d *DeploymentController) Cancel() error {
	return d.f()
}

func (d *DeploymentController) getResource() string {
	return "deployments"
}

func (d *DeploymentController) Reset() error {
	get, err := d.clientset.CoreV1().
		Pods(d.namespace).
		Get(context.TODO(), toInboundPodName(d.getResource(), d.name), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if o := get.GetAnnotations()[util.OriginData]; len(o) != 0 {
		if n, err := strconv.Atoi(o); err == nil {
			util.UpdateReplicasScale(d.clientset, d.namespace, util.ResourceTupleWithScale{
				Resource: d.getResource(),
				Name:     d.name,
				Scale:    n,
			})
		}
	}
	return nil
}
