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
	"errors"
	"fmt"
	v1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"math/rand"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"strconv"
	"time"
)

type GoClient struct {
	clusterIpAccessMode bool
	client              *kubernetes.Clientset
	Config              []byte
}

// use this go client generator to avoid out-cluster/in-cluster network issues
func NewAdminGoClient(kubeconfig []byte) (*GoClient, error) {

	// first try to access cluster normally

	client, originErr := newGoClient(kubeconfig)
	if originErr == nil && client != nil {
		originErr = client.requireClusterAdminClient()

		if originErr == nil {
			client.clusterIpAccessMode = false
			client.Config = kubeconfig

			return client, nil
		}
	}

	// then try to access current cluster's kube api-server

	client, err, newConfig := newGoClientUseCurrentClusterHost(kubeconfig)
	if err == nil && client != nil {
		err = client.requireClusterAdminClient()

		if err == nil {
			client.clusterIpAccessMode = true
			client.Config = newConfig

			fmt.Printf("Create k8s Go Client with 'clusterIpAccessMode' \n")
			return client, nil
		}
	}

	return nil, errors.New("can't not create client go with current kubeconfig")
}

func newGoClient(kubeconfig []byte) (*GoClient, error) {
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

// try to replace the host to access kube-apiserver
func newGoClientUseCurrentClusterHost(kubeconfig []byte) (*GoClient, error, []byte) {

	// Step1. get raw config
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err, nil
	}

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, err, nil
	}

	cluster := rawConfig.Clusters[rawConfig.CurrentContext]
	if cluster == nil {
		return nil, err, nil
	}

	// Step2. get in-cluster config
	configInCluster, err := rest.InClusterConfig()
	if err != nil {
		return nil, err, nil
	}

	// Step3. override the host and new client
	cluster.Server = configInCluster.Host
	newConfig, _ := clientcmd.Write(rawConfig)

	c, err := clientcmd.RESTConfigFromKubeConfig(newConfig)
	if err != nil {
		return nil, err, nil
	}

	clientSet, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err, nil
	}
	client := &GoClient{
		client: clientSet,
	}
	return client, nil, newConfig
}

func (c *GoClient) requireClusterAdminClient() error {
	// check is admin Kubeconfig
	isAdmin, err := c.IsAdmin()
	if err != nil {
		return errno.ErrClusterKubeConnect
	}
	if isAdmin != true {
		return errno.ErrClusterKubeAdmin
	}
	return nil
}

// get deployment
func (c *GoClient) GetDepDeploymentStatus() error {
	deployment, err := c.client.AppsV1().Deployments(global.NocalhostSystemNamespace).Get(context.TODO(), global.NocalhostDepName, metav1.GetOptions{})
	if err != nil {
		return errors.New("nocalhost-dep component not found")
	}
	isAvailable := false
	if deployment.Name != "" {
		for _, status := range deployment.Status.Conditions {
			if status.Type == v1.DeploymentAvailable {
				isAvailable = true
			}
		}
	}
	if isAvailable {
		// cluster can use
		return nil
	}
	return errors.New("nocalhost-dep is processing")
}

// check if exist namespace
func (c *GoClient) IfNocalhostNameSpaceExist() (bool, error) {
	_, err := c.client.CoreV1().Namespaces().Get(context.TODO(), global.NocalhostSystemNamespace, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
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

// delete namespace, this will delete all resource in namespace
func (c *GoClient) DeleteNS(namespace string) (bool, error) {
	err := c.client.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// nocalhost create namespace use the rule: nocal+userId+4 random word, exp: nocal4esac
// namespace rules must match DNS-1123 label, capital doesn't allow
func (c *GoClient) GenerateNsName(userId uint64) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, 4)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return "nh" + strconv.Itoa(int(userId)) + string(b)
}

// check if admin for kubeconfig use SelfSubjectAccessReview
// check https://kubernetes.io/zh/docs/reference/access-authn-authz/authorization/
// kubectl auth can-i '*' '*'
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

// Create resource quota for namespace. such as:
/**
apiVersion: v1
  kind: ResourceQuota
  metadata:
    name: namespace-name
  spec:
    hard:
      limits.cpu: "10"
      requests.cpu: "10"
      limits.memory: "48Gi"
      requests.memory: "40Gi"
      persistentvolumeclaims: "10"
      services.loadbalancers: "10"
      requests.storage: "20Gi"
*/
func (c *GoClient) CreateResourceQuota(name, namespace, reqMem, reqCpu, limitsMem, limitsCpu, storageCapacity, ephemeralStorage, pvcCount, lbCount string) (bool, error) {

	resourceQuota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}

	resourceList := make(map[corev1.ResourceName]resource.Quantity)
	if len(reqMem) > 0 {
		resourceList[corev1.ResourceRequestsMemory] = resource.MustParse(reqMem)
	}
	if len(reqCpu) > 0 {
		resourceList[corev1.ResourceRequestsCPU] = resource.MustParse(reqCpu)
	}
	if len(limitsMem) > 0 {
		resourceList[corev1.ResourceLimitsMemory] = resource.MustParse(limitsMem)
	}
	if len(limitsCpu) > 0 {
		resourceList[corev1.ResourceLimitsCPU] = resource.MustParse(limitsCpu)
	}
	if len(storageCapacity) > 0 {
		resourceList[corev1.ResourceRequestsStorage] = resource.MustParse(storageCapacity)
	}
	if len(ephemeralStorage) > 0 {
		resourceList[corev1.ResourceEphemeralStorage] = resource.MustParse(ephemeralStorage)
	}
	if len(pvcCount) > 0 {
		resourceList[corev1.ResourcePersistentVolumeClaims] = resource.MustParse(pvcCount)
	}
	if len(lbCount) > 0 {
		resourceList[corev1.ResourceServicesLoadBalancers] = resource.MustParse(lbCount)
	}
	if (len(resourceList)) < 1 {
		return true, nil
	}
	resourceQuota.Spec = corev1.ResourceQuotaSpec{
		Hard: resourceList,
	}
	_, err := c.client.CoreV1().ResourceQuotas(namespace).Create(context.TODO(), resourceQuota, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, err
}

func (c *GoClient) DeleteResourceQuota(name, namespace string) (bool, error) {
	err := c.client.CoreV1().ResourceQuotas(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return false, err
	}
	return true, err
}

// create default resource quota for container. such as:
/**
apiVersion: v1
kind: LimitRange
metadata:
  name: limits
spec:
  limits:
  - default:
      cpu: 200m
      memory: 512Mi
    defaultRequest:
      cpu: 100m
      memory: 128Mi
    type: Container
*/
func (c *GoClient) CreateLimitRange(name, namespace, reqMem, limitsMem, reqCpu, limitsCpu, ephemeralStorage string) (bool, error) {
	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}

	limits := make(map[corev1.ResourceName]resource.Quantity)
	if len(limitsMem) > 0 {
		limits[corev1.ResourceMemory] = resource.MustParse(limitsMem)
	}
	if len(limitsCpu) > 0 {
		limits[corev1.ResourceCPU] = resource.MustParse(limitsCpu)
	}
	if len(ephemeralStorage) > 0 {
		limits[corev1.ResourceEphemeralStorage] = resource.MustParse(ephemeralStorage)
	}

	requests := make(map[corev1.ResourceName]resource.Quantity)
	if len(reqMem) > 0 {
		requests[corev1.ResourceMemory] = resource.MustParse(reqMem)
	}
	if len(reqCpu) > 0 {
		requests[corev1.ResourceCPU] = resource.MustParse(reqCpu)
	}

	if len(limits) < 1 && len(requests) < 1 {
		return true, nil
	}
	limitRange.Spec.Limits = append(limitRange.Spec.Limits, corev1.LimitRangeItem{
		Default:        limits,
		DefaultRequest: requests,
		Type:           corev1.LimitTypeContainer,
	})
	_, err := c.client.CoreV1().LimitRanges(namespace).Create(context.TODO(), limitRange, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, err
}

func (c *GoClient) DeleteLimitRange(name, namespace string) (bool, error) {
	err := c.client.CoreV1().LimitRanges(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return false, err
	}
	return true, err
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
func (c *GoClient) CreateRoleBinding(name, namespace, role, toServiceAccount string) (bool, error) {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if role != "" {
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     role,
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

// create clusterRoleBinding
// role=admin
func (c *GoClient) CreateClusterRoleBinding(name, namespace, role, toServiceAccount string) (bool, error) {
	roleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if role != "" {
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     role,
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
	_, err := c.client.RbacV1().ClusterRoleBindings().Create(context.TODO(), roleBinding, metav1.CreateOptions{})
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
		Verbs:     []string{"*"},
		Resources: []string{"*"},
		APIGroups: []string{"*"},
	})
	_, err := c.client.RbacV1().Roles(namespace).Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// deploy nocalhsot resource such as priorityclass
func (c *GoClient) DeployNocalhostResource() error {
	priorityClass := schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: global.NocalhostDefaultPriorityclassName,
		},
		Value:       global.NocalhostDefaultPriorityclassDefaultValue,
		Description: "This priority class should be used for Nocalhost service pods only.",
	}
	_, err := c.client.SchedulingV1().PriorityClasses().Create(context.TODO(), &priorityClass, metav1.CreateOptions{})
	return err
}

// deploy nocalhost-dep
// now all value has set by default
// TODO this might better read from database manifest
func (c *GoClient) DeployNocalhostDep(image, namespace, serviceAccount string) (bool, error) {
	tag := "latest"
	if global.Branch == global.NocalhostDefaultReleaseBranch {
		tag = global.Version
	}
	if global.Branch != global.NocalhostDefaultReleaseBranch && global.Branch != "default" {
		tag = global.CommitId
	}
	if image == "" {
		image = "codingcorp-docker.pkg.coding.net/nocalhost/public/dep-installer-job:" + tag
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
					Containers: []corev1.Container{
						{
							Name:    "nocalhost-dep-installer",
							Image:   image,
							Command: []string{"/nocalhost/installer.sh"},
						},
					},
					ServiceAccountName: serviceAccount,
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

// deploy pre pull images
// use DaemonSet InitContainer make sure every Node pull images https://kubernetes.io/zh/docs/concepts/workloads/controllers/daemonset/
// when started it should kill himself
func (c *GoClient) DeployPrePullImages(images []string, namespace string) (bool, error) {
	if namespace == "" {
		namespace = global.NocalhostSystemNamespace
	}
	// initContainer
	initContainer := make([]corev1.Container, 0)
	for key, image := range images {
		sContainer := corev1.Container{
			Name:    "prepull" + strconv.Itoa(key),
			Image:   image,
			Command: []string{"echo", "done"},
		}
		initContainer = append(initContainer, sContainer)
	}

	daemonSet := &v1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      global.NocalhostPrePullDSName,
			Namespace: namespace,
		},
		Spec: v1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": global.NocalhostPrePullDSName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": global.NocalhostPrePullDSName},
				},
				Spec: corev1.PodSpec{
					InitContainers: initContainer,
					Containers: []corev1.Container{
						{
							Name:    "kubectl",
							Image:   "codingcorp-docker.pkg.coding.net/nocalhost/public/kubectl:latest",
							Command: []string{"kubectl", "delete", "ds", global.NocalhostPrePullDSName, "-n", global.NocalhostSystemNamespace},
						},
					},
					ServiceAccountName: global.NocalhostSystemNamespaceServiceAccount,
				},
			},
		},
	}
	_, err := c.client.AppsV1().DaemonSets(namespace).Create(context.TODO(), daemonSet, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create admin kubeconfig in cluster for admission webhook
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

// Get serviceAccount secret
func (c *GoClient) GetSecret(name, namespace string) (*corev1.Secret, error) {
	return c.client.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// Get serviceAccount
func (c *GoClient) GetServiceAccount(name, namespace string) (*corev1.ServiceAccount, error) {
	return c.client.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// Get cluster node
func (c *GoClient) GetClusterNode() (*corev1.NodeList, error) {
	return c.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
}

// Get cluster version
func (c *GoClient) GetClusterVersion() (*version.Info, error) {
	return c.client.DiscoveryClient.ServerVersion()
}

// Watch serviceAccount
// Bug Fix in Tencent TKE servieAccount secret will not return immediately
func (c *GoClient) WatchServiceAccount(name, namespace string) (*corev1.ServiceAccount, error) {
	resourceWatchTimeoutSeconds := int64(30)
	log.Infof("GET ServiceAccount name %s, namespace %s: ", name, namespace)
	var serviceAccount *corev1.ServiceAccount
	watcher, err := c.client.CoreV1().ServiceAccounts(namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  fields.Set{"metadata.name": name}.AsSelector().String(),
		Watch:          true,
		TimeoutSeconds: &resourceWatchTimeoutSeconds,
	})
	if err != nil {

	}
	for event := range watcher.ResultChan() {
		if event.Type == watch.Added {
			log.Infof("ServiceAccount added")
			//serviceAccount = event.Object.(*corev1.ServiceAccount)
			//coloredoutput.Infof("ServiceAccount %s", serviceAccount)
			// Tencent TKE can not return secrets immediately
			break
		}
	}

	// watch serviceAccount secret
	// var secret *corev1.Secret
	// TKE unknow: the server rejected our request for an unknown reason (get secrets) can not watch secret
	//swatcher, err := c.client.CoreV1().Secrets(namespace).Watch(context.TODO(), metav1.ListOptions{
	//	//FieldSelector:  fields.Set{"metadata.annotations.kubernetes.io/service-account.name": name}.AsSelector().String(),
	//	FieldSelector:  "metadata.annotations.kubernetes.io/service-account.name=" + name,
	//	Watch:          true,
	//	TimeoutSeconds: &resourceWatchTimeoutSeconds,
	//})
	//if err != nil {
	//	coloredoutput.Infof("err %s", err)
	//}
	//for sevent := range swatcher.ResultChan() {
	//	if sevent.Type == watch.Added {
	//		coloredoutput.Infof("ServiceAccount Secret added")
	//		//secret = sevent.Object.(*corev1.Secret)
	//		coloredoutput.Infof("ServiceAccount Secret added %s")
	//		break
	//	}
	//}

	// loop and wait for serviceAccountToken Create,especially for TKE is slow
	// wait 30S
	i := 0
	for {
		if i > 300 {
			break
		}
		serviceAccount, err = c.client.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if serviceAccount != nil && len(serviceAccount.Secrets) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
		i++
	}
	return serviceAccount, nil
}

func (c *GoClient) GetStorageClassList() (*storagev1.StorageClassList, error) {
	return c.client.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
}
