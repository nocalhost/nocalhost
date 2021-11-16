package pkg

import (
	"context"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
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

func (replicas *ReplicasController) ScaleToZero() (map[string]string, []v1.ContainerPort, error) {
	updateScale, err2 := replicas.clientset.AppsV1().ReplicaSets(replicas.namespace).Get(context.TODO(), replicas.name, metav1.GetOptions{})
	if err2 != nil {
		return nil, nil, err2
	}
	_, err := replicas.clientset.AppsV1().ReplicaSets(replicas.namespace).UpdateScale(context.TODO(), replicas.name, &autoscalingv1.Scale{
		ObjectMeta: metav1.ObjectMeta{
			Name:      replicas.name,
			Namespace: replicas.namespace,
		},
		Spec: autoscalingv1.ScaleSpec{
			Replicas: int32(0),
		},
	}, metav1.UpdateOptions{})
	if err != nil {
		return nil, nil, err
	}
	replicas.f = func() error {
		_, err = replicas.clientset.AppsV1().ReplicaSets(replicas.namespace).
			UpdateScale(context.TODO(), replicas.name, &autoscalingv1.Scale{
				ObjectMeta: metav1.ObjectMeta{
					Name:      replicas.name,
					Namespace: replicas.namespace,
				},
				Spec: autoscalingv1.ScaleSpec{
					Replicas: *updateScale.Spec.Replicas,
				},
			}, metav1.UpdateOptions{})
		return err
	}
	return updateScale.Spec.Template.Labels, updateScale.Spec.Template.Spec.Containers[0].Ports, nil
}

func (replicas *ReplicasController) Cancel() error {
	return replicas.f()
}
