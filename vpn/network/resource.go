package network

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newSshConfigmap(privateKey, publicKey []byte) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        DNSPOD,
			Namespace:   DefaultNamespace,
			Annotations: map[string]string{refCountKey: "0"},
		},
		Data: map[string]string{"authorized": string(publicKey), "privateKey": string(privateKey)},
	}
}

func newDnsPodDeployment() *appsv1.Deployment {
	one := int32(1)
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "apps",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DNSPOD,
			Namespace: DefaultNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": DNSPOD}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DNSPOD,
					Namespace: DefaultNamespace,
					Labels:    map[string]string{"app": DNSPOD},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  DNSPOD,
						Image: "naison/dnsserver:latest",
						Ports: []v1.ContainerPort{
							{ContainerPort: 53, Protocol: v1.ProtocolTCP},
							{ContainerPort: 53, Protocol: v1.ProtocolUDP},
							{ContainerPort: 22},
						},
						ImagePullPolicy: v1.PullAlways,
						VolumeMounts: []v1.VolumeMount{{
							Name:      "ssh-key",
							MountPath: "/root",
						}},
					}},
					Volumes: []v1.Volume{{
						Name: "ssh-key",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: DNSPOD,
								},
								Items: []v1.KeyToPath{{
									Key:  "authorized",
									Path: "authorized_keys",
								}},
							},
						},
					}},
				},
			},
		},
	}
}

func newDnsPodService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DNSPOD,
			Namespace: DefaultNamespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Name: "tcp", Protocol: v1.ProtocolTCP, Port: 53, TargetPort: intstr.FromInt(53)},
				{Name: "udp", Protocol: v1.ProtocolUDP, Port: 53, TargetPort: intstr.FromInt(53)},
				{Name: "ssh", Port: 22, TargetPort: intstr.FromInt(22)},
			},
			Selector: map[string]string{"app": DNSPOD},
			Type:     v1.ServiceTypeClusterIP,
		},
	}
}

func newPod(podName, namespace string, labels map[string]string, port []v1.ContainerPort) *v1.Pod {
	labels["nocalhost"] = "nocalhost"
	labels["name"] = podName
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Image:           "naison/dnsserver:latest",
				Ports:           port,
				Name:            podName,
				ImagePullPolicy: v1.PullAlways,
				VolumeMounts: []v1.VolumeMount{{
					Name:      "ssh-key",
					MountPath: "/root",
				}},
			}},
			Volumes: []v1.Volume{{
				Name: "ssh-key",
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: DNSPOD,
						},
						Items: []v1.KeyToPath{{
							Key:  "authorized",
							Path: "authorized_keys",
						}},
					},
				},
			}},
		},
	}
}
