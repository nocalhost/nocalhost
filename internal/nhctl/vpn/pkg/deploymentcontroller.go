package pkg

import (
	"context"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
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

func (deployment *DeploymentController) ScaleToZero() (map[string]string, []v1.ContainerPort, error) {
	scale, err2 := deployment.clientset.AppsV1().Deployments(deployment.namespace).GetScale(context.TODO(), deployment.name, metav1.GetOptions{})
	if err2 != nil {
		return nil, nil, err2
	}
	deployment.f = func() error {
		_, err := deployment.clientset.AppsV1().Deployments(deployment.namespace).UpdateScale(
			context.TODO(),
			deployment.name,
			&autoscalingv1.Scale{
				ObjectMeta: metav1.ObjectMeta{Name: deployment.name, Namespace: deployment.namespace},
				Spec:       autoscalingv1.ScaleSpec{Replicas: scale.Spec.Replicas},
			},
			metav1.UpdateOptions{},
		)
		return err
	}
	_, err := deployment.clientset.AppsV1().Deployments(deployment.namespace).UpdateScale(
		context.TODO(),
		deployment.name,
		&autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployment.name,
				Namespace: deployment.namespace,
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
	get, err := deployment.clientset.AppsV1().Deployments(deployment.namespace).Get(context.TODO(), deployment.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	return get.Spec.Template.GetLabels(), get.Spec.Template.Spec.Containers[0].Ports, nil
}

func (deployment *DeploymentController) Cancel() error {
	return deployment.f()
}
