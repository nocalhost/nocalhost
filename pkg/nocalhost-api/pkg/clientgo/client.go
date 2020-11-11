/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clientgo

import (
	"context"
	authorizationv1 "k8s.io/api/authorization/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"math/rand"
	"nocalhost/internal/nocalhost-api/global"
	"strconv"
	"time"
)

type GoClient struct {
	client *kubernetes.Clientset
}

func NewGoClient(kubeconfig []byte) (*GoClient, error) {
	c, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	client := &GoClient{
		client: clientSet,
	}
	return client, nil
}

// When create sub namespace for developer, label should set "nocalhost" for nocalhost-dep admission webhook muting
// when create nocalhost-reserved namesapce, label should set "nocalhost-reserved"
func (c *GoClient) CreateNS(namespace, label string) (bool, error) {
	if label == "" {
		if namespace == global.NocalhostSystemNamespace {
			label = global.NocalhostSystemNamespaceLabel
		} else {
			label = global.NocalhostDevNamespaceLabel
		}
	}
	labels := make(map[string]string, 0)
	labels["env"] = label
	nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: labels}}
	_, err := c.client.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// nocalhost create namespace use the rule: nocal+userId+4 random word, exp: nocal4esZc
func (c *GoClient) GenerateNsName(userId uint64) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, 4)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return "nocal" + strconv.Itoa(int(userId)) + string(b)
}

// check if admin for kubeconfig use SelfSubjectAccessReview
// check https://kubernetes.io/zh/docs/reference/access-authn-authz/authorization/
func (c *GoClient) IsAdmin() (bool, error) {
	arg := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: "*",
				Group:     "*",
				Verb:      "*",
				Name:      "*",
				Version:   "*",
				Resource:  "*",
			},
		},
	}

	response, err := c.client.AuthorizationV1().SelfSubjectAccessReviews().Create(context.TODO(), arg, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return response.Status.Allowed, nil
}

// create serviceAccount for namespace(Authorization cluster for developer)
// default name is nocalhost
func (c *GoClient) CreateServiceAccount(name, namespace string) (bool, error) {
	if name == "" {
		name = global.NocalhostDevServiceAccountName
	}
	arg := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	_, err := c.client.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), arg, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// bind roles for serviceAccount
// this use for given default serviceAccount default:view case by initContainer need use kubectl get pods....(clusterRole=view)
// and this will use for bind developer serviceAccount roles(clusterRole=nocalhost-roles)

/*
default serviceAccount default:view:
kubectl create rolebinding default-view \
        --clusterrole=view \
        --serviceaccount={namespace}:default \
        --namespace={namespace}
*/
func (c *GoClient) CreateRoleBinding(name, namespace, clusterRole, toServiceAccount string) (bool, error) {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if clusterRole != "" {
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRole,
		}
	}
	if toServiceAccount != "" {
		roleBinding.Subjects = append(roleBinding.Subjects, rbacv1.Subject{
			Kind:      rbacv1.ServiceAccountKind,
			APIGroup:  "",
			Namespace: namespace,
			Name:      toServiceAccount,
		})
	}
	_, err := c.client.RbacV1().RoleBindings(namespace).Create(context.TODO(), roleBinding, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// create user role for single namespace
// name default nocalhost-role
//  default create every developer can access all resource for he's namespace
func (c *GoClient) CreateRole(name, namespace string) (bool, error) {
	role := &rbacv1.Role{}
	role.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
	role.Rules = append(role.Rules, rbacv1.PolicyRule{
		Verbs:     []string{""},
		Resources: []string{""},
		APIGroups: []string{""},
	})
	_, err := c.client.RbacV1().Roles(namespace).Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// deploy nocalhost-dep
// now all value has set by default
// TODO this might better read from database manfest
func (c *GoClient) DeployNocalhostDep(image, namespace string) (bool, error) {
	if image == "" {
		image = "codingcorp-docker.pkg.coding.net/nocalhost/public/dep-installer-job:latest"
	}
	var ttl int32 = 1
	var backOff int32 = 1
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "nocalhost-dep-installer-",
			Namespace:    global.NocalhostSystemNamespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttl,
			BackoffLimit:            &backOff,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: "kubeconfig",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "nocalhost-kubeconfig",
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "nocalhost-dep-installer",
							Image: image,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kubeconfig",
									MountPath: "/.kube/config",
									SubPath:   "config",
								},
							},
							Command: []string{"/nocalhost/installer.sh"},
						},
					},
				},
			},
		},
	}
	_, err := c.client.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// create admin kubeconfig in cluster for admission webhook
func (c *GoClient) CreateConfigMap(name, namespace, key, value string) (bool, error) {
	configMapData := make(map[string]string, 0)
	configMapData[key] = value

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: configMapData,
	}
	_, err := c.client.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// get serviceAccount secret
func (c *GoClient) GetSecret(name, namespace string) (*corev1.Secret, error) {
	return c.client.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// get serviceAccount
func (c *GoClient) GetServiceAccount(name, namespace string) (*corev1.ServiceAccount, error) {
	return c.client.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}
