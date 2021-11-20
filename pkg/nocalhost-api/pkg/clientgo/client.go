/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgo

import (
	"context"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	istio "istio.io/client-go/pkg/clientset/versioned"
	apiappsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

const (
	DepInstaller   = "dep-installer-job"
	Dep            = "nocalhost-dep"
	NocalhostLabel = "nocalhost-managed"
	Api            = "nocalhost-api"
)

var LimitedRules = &rbacv1.PolicyRule{
	Verbs:     []string{"get", "list"},
	Resources: []string{"resourcequotas", "roles"},
	APIGroups: []string{"*"},
}

type GoClient struct {
	mu sync.Mutex

	clusterIpAccessMode bool
	client              *kubernetes.Clientset
	DynamicClient       dynamic.Interface
	istioClient         *istio.Clientset
	mapper              *restmapper.DeferredDiscoveryRESTMapper
	restConfig          *rest.Config
	Config              []byte
}

type InitResult struct {
	goClient *GoClient
	err      error
}

// new client with time out
func NewAdminGoClient(kubeconfig []byte) (*GoClient, error) {
	return NewAdminGoClientWithTimeout(kubeconfig, time.Second*5)
}

func NewAdminGoClientWithTimeout(kubeconfig []byte, duration time.Duration) (*GoClient, error) {
	initCh := make(chan *InitResult)

	go func() {
		client, err := newAdminGoClientTimeUnreliable(kubeconfig)
		initCh <- &InitResult{
			goClient: client,
			err:      err,
		}
	}()

	select {
	case res := <-initCh:
		return res.goClient, res.err

	case <-time.After(duration):
		log.Infof("Initial k8s Go Client timeout!")
		return nil, errno.ErrClusterTimeout
	}
}

// use this go client generator to avoid out-cluster/in-cluster network issues
func newAdminGoClientTimeUnreliable(kubeconfig []byte) (*GoClient, error) {

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

			log.Infof("Initial k8s Go Client with 'clusterIpAccessMode' ")
			return client, nil
		}
	}

	if originErr != nil {
		if strings.Contains(originErr.Error(), "client connection lost") {
			log.Infof("Failed to connect to the kube-api, may cause by the network connectivity")
			return nil, errno.ErrClusterTimeout
		} else {
			log.Infof("Failed to connect to the kube-api: %s \n", originErr.Error())
		}
	}

	if err != nil {
		log.Infof("Failed to try connect to the cluster-inner kube-api: %s \n", err.Error())
	}

	return nil, errors.New("can't not create client go with current kubeconfig")
}

func NewGoClient(kubeconfig []byte) (*GoClient, error) {
	return newGoClient(kubeconfig)
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

	dynamicClient, err := dynamic.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	client := &GoClient{
		client:        clientSet,
		DynamicClient: dynamicClient,
		restConfig:    c,
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

	currentContext := rawConfig.Contexts[rawConfig.CurrentContext]
	if currentContext == nil {
		return nil, errno.ErrClusterContext, nil
	}

	cluster := rawConfig.Clusters[currentContext.Cluster]
	if cluster == nil {
		return nil, errno.ErrClusterName, nil
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

	dynamicClient, err := dynamic.NewForConfig(c)
	if err != nil {
		return nil, err, nil
	}

	client := &GoClient{
		client:        clientSet,
		DynamicClient: dynamicClient,
		restConfig:    c,
	}
	return client, nil, newConfig
}

func (c *GoClient) requireClusterAdminClient() error {
	// check is admin Kubeconfig
	isAdmin, err := c.IsAdmin()
	if err != nil {
		log.Infof("Error occurred while checking authentication: ", err.Error())
		return errno.ErrClusterKubeConnect
	}
	if isAdmin != true {
		return errno.ErrClusterKubeAdmin
	}
	return nil
}

// get deployment
func (c *GoClient) GetDepDeploymentStatus() error {
	deployment, err := c.client.AppsV1().Deployments(global.NocalhostSystemNamespace).Get(
		context.TODO(), global.NocalhostDepName, metav1.GetOptions{},
	)
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
func (c *GoClient) CreateNS(namespace string, labels map[string]string) (bool, error) {
	if labels == nil {
		labels = make(map[string]string, 0)
	}
	if labels["env"] == "" {
		if namespace == global.NocalhostSystemNamespace {
			labels["env"] = global.NocalhostSystemNamespaceLabel
		} else {
			labels["env"] = global.NocalhostDevNamespaceLabel
		}
	}
	nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: labels}}
	_, err := c.client.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *GoClient) ExistNs(namespace string) (bool, *corev1.Namespace) {
	get, err := c.client.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})

	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	}
	if get != nil {
		return true, get
	}
	return false, nil
}

func (c *GoClient) ExistDeployment(namespace, deployment string) (bool, *apiappsV1.Deployment) {
	get, err := c.client.AppsV1().Deployments(namespace).Get(context.TODO(), deployment, metav1.GetOptions{})

	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	}
	if get != nil {
		return true, get
	}
	return false, nil
}

func (c *GoClient) ExistClusterRoleBinding(roleBinding string) (bool, *rbacv1.ClusterRoleBinding) {
	get, err := c.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), roleBinding, metav1.GetOptions{})

	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	}
	if get != nil {
		return true, get
	}
	return false, nil
}

func (c *GoClient) ExistPriorityClass(name string) (bool, *schedulingv1.PriorityClass) {
	get, err := c.client.SchedulingV1().PriorityClasses().Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	}
	if get != nil {
		return true, get
	}
	return false, nil
}

func (c *GoClient) ExistServiceAccount(namespace, name string) (bool, *corev1.ServiceAccount) {
	get, err := c.client.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	}
	if get != nil {
		return true, get
	}
	return false, nil
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

	response, err := c.client.AuthorizationV1().SelfSubjectAccessReviews().Create(
		context.TODO(), arg, metav1.CreateOptions{},
	)
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

	m := map[string]string{}
	m[_const.NocalhostRoleBindingLabelKey] = _const.NocalhostRoleBindingLabelVal
	m[NocalhostLabel] = time.Now().Format("20060102150405")
	arg := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: m},
	}
	_, err := c.client.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), arg, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// nocalhost-dep watch the service account, so make some change to refresh the cache
func (c *GoClient) RefreshServiceAccount(name, namespace string) {
	if name == "" {
		name = global.NocalhostDevServiceAccountName
	}

	if sa, err := c.client.CoreV1().ServiceAccounts(namespace).Get(
		context.TODO(), name, metav1.GetOptions{},
	); err == nil {
		if sa.Labels == nil {
			sa.Labels = map[string]string{}
		}

		sa.Labels[NocalhostLabel] = time.Now().Format("20060102150405")
		_, _ = c.client.CoreV1().ServiceAccounts(namespace).Update(context.TODO(), sa, metav1.UpdateOptions{})
	} else {
		err = retry.OnError(
			retry.DefaultBackoff, func(err error) bool {
				return !k8serrors.IsAlreadyExists(err)
			}, func() error {
				_, err = c.CreateServiceAccount(name, namespace)
				return err
			},
		)

		log.Errorf("Error while create sa %s after multiple retry!", name)
	}
}

// CreateResourceQuota Initial resource quota for namespace
func (c *GoClient) CreateResourceQuota(
	name, namespace, reqMem, reqCpu, limitsMem,
	limitsCpu, storageCapacity, ephemeralStorage, pvcCount, lbCount string,
) (
	bool, error,
) {

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

// CreateLimitRange create default resource quota for container
func (c *GoClient) CreateLimitRange(name, namespace, reqMem, limitsMem, reqCpu, limitsCpu, ephemeralStorage string) (
	bool, error,
) {
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
	limitRange.Spec.Limits = append(
		limitRange.Spec.Limits, corev1.LimitRangeItem{
			Default:        limits,
			DefaultRequest: requests,
			Type:           corev1.LimitTypeContainer,
		},
	)
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

// CreateRoleBinding bind roles for serviceAccount
// this use for given default serviceAccount default:view case by initContainer need
// use kubectl get pods....(clusterRole=view)
// and this will use for bind developer serviceAccount roles(clusterRole=nocalhost-roles)
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
		roleBinding.Subjects = append(
			roleBinding.Subjects, rbacv1.Subject{
				Kind:      rbacv1.ServiceAccountKind,
				APIGroup:  "",
				Namespace: namespace,
				Name:      toServiceAccount,
			},
		)
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
		roleBinding.Subjects = append(
			roleBinding.Subjects, rbacv1.Subject{
				Kind:      rbacv1.ServiceAccountKind,
				APIGroup:  "",
				Namespace: namespace,
				Name:      toServiceAccount,
			},
		)
	}
	_, err := c.client.RbacV1().ClusterRoleBindings().Create(context.TODO(), roleBinding, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *GoClient) UpdateRole(name, namespace string, rbacRule []rbacv1.PolicyRule) error {

	before, err := c.client.RbacV1().Roles(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	before.Rules = rbacRule

	_, err = c.client.RbacV1().Roles(namespace).Update(context.TODO(), before, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *GoClient) UpdateClusterRole(name string, rbacRule []rbacv1.PolicyRule) error {

	before, err := c.client.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	before.Rules = rbacRule

	_, err = c.client.RbacV1().ClusterRoles().Update(context.TODO(), before, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *GoClient) UpdateRoleBindingForNocalhostLabel(name, namespace string) error {

	before, err := c.client.RbacV1().RoleBindings(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	labels := before.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	labels[_const.NocalhostRoleBindingLabelKey] = _const.NocalhostRoleBindingLabelVal

	_, err = c.client.RbacV1().RoleBindings(namespace).Update(context.TODO(), before, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *GoClient) UpdateClusterRoleBindingForNocalhostLabel(name string) error {

	before, err := c.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	labels := before.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	labels[_const.NocalhostRoleBindingLabelKey] = _const.NocalhostRoleBindingLabelVal

	_, err = c.client.RbacV1().ClusterRoleBindings().Update(context.TODO(), before, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// create roleBinding role=admin
func (c *GoClient) AppendRoleBinding(name, namespace, role, toServiceAccount, toServiceAccountNs string) error {
	rb, err := c.client.RbacV1().RoleBindings(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	var update = false
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	} else if k8serrors.IsNotFound(err) {

		rb = &rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "RoleBinding"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,

				// for role_binding_watcher
				Labels: map[string]string{
					_const.NocalhostRoleBindingLabelKey: _const.NocalhostRoleBindingLabelVal,
				},
			},
		}

		if role != "" {
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     role,
			}
		}

	} else {

		// skip if present
		for _, subject := range rb.Subjects {
			if subject.Name == toServiceAccount {
				return nil
			}
		}

		update = true
	}

	if rb.Labels == nil {
		rb.Labels = map[string]string{}
	}

	// label for watch
	rb.Labels[NocalhostLabel] = time.Now().Format("20060102150405")

	//  auth admin for current role binding ns
	if toServiceAccount != "" {
		rb.Subjects = append(
			rb.Subjects, rbacv1.Subject{
				Kind:      rbacv1.ServiceAccountKind,
				Namespace: toServiceAccountNs,
				Name:      toServiceAccount,
			},
		)
	}

	if update {
		_, err = c.client.RbacV1().RoleBindings(namespace).Update(context.TODO(), rb, metav1.UpdateOptions{})
	} else {
		_, err = c.client.RbacV1().RoleBindings(namespace).Create(context.TODO(), rb, metav1.CreateOptions{})
	}
	return err
}

// create roleBinding role=admin
func (c *GoClient) AppendClusterRoleBinding(name, role, toServiceAccount, toServiceAccountNs string) error {
	crb, err := c.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})

	var update = false
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	} else if k8serrors.IsNotFound(err) {

		crb = &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{APIVersion: rbacv1.SchemeGroupVersion.String(), Kind: "RoleBinding"},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,

				// for cluster_role_binding_watcher
				Labels: map[string]string{
					_const.NocalhostRoleBindingLabelKey: _const.NocalhostRoleBindingLabelVal,
				},
			},
		}

		if role != "" {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     role,
			}
		}

	} else {

		// skip if present
		for _, subject := range crb.Subjects {
			if subject.Name == toServiceAccount {
				return nil
			}
		}

		update = true
	}

	if crb.Labels == nil {
		crb.Labels = map[string]string{}
	}

	// label for watch
	crb.Labels[NocalhostLabel] = time.Now().Format("20060102150405")

	//  auth admin for current role binding ns
	if toServiceAccount != "" {
		crb.Subjects = append(
			crb.Subjects, rbacv1.Subject{
				Kind:      rbacv1.ServiceAccountKind,
				Namespace: toServiceAccountNs,
				Name:      toServiceAccount,
			},
		)
	}

	if update {
		_, err = c.client.RbacV1().ClusterRoleBindings().Update(context.TODO(), crb, metav1.UpdateOptions{})
	} else {
		_, err = c.client.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb, metav1.CreateOptions{})
	}
	return err
}

func (c *GoClient) RemoveClusterRoleBinding(name, toServiceAccount, toServiceAccountNs string) error {
	crb, err := c.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	if len(crb.Subjects) == 0 {
		return nil
	}

	for i, subject := range crb.Subjects {
		if subject.Name == toServiceAccount && subject.Namespace == toServiceAccountNs {
			if i == len(crb.Subjects)-1 {
				crb.Subjects = append(crb.Subjects[:i])
			} else {
				crb.Subjects = append(crb.Subjects[:i], crb.Subjects[i+1:]...)
			}

			break
		}
	}

	_, err = c.client.RbacV1().ClusterRoleBindings().Update(context.TODO(), crb, metav1.UpdateOptions{})
	return err
}

func (c *GoClient) RemoveRoleBinding(name, namespace, toServiceAccount, toServiceAccountNs string) error {
	rb, err := c.client.RbacV1().RoleBindings(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	if len(rb.Subjects) == 0 {
		return nil
	}

	for i, subject := range rb.Subjects {
		if subject.Name == toServiceAccount && subject.Namespace == toServiceAccountNs {
			if i == len(rb.Subjects)-1 {
				rb.Subjects = append(rb.Subjects[:i])
			} else {
				rb.Subjects = append(rb.Subjects[:i], rb.Subjects[i+1:]...)
			}

			break
		}
	}

	_, err = c.client.RbacV1().RoleBindings(namespace).Update(context.TODO(), rb, metav1.UpdateOptions{})
	return err
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

	//// resource quota && role's permission is limited
	//rule, err := getPolicyRule(c)
	//if err != nil {
	//	return false, err
	//}
	//role.Rules = append(role.Rules, *rule...)

	role.Rules = []rbacv1.PolicyRule{
		{
			Verbs:     []string{"*"},
			Resources: []string{"*"},
			APIGroups: []string{"*"},
		},
	}

	_, err := c.client.RbacV1().Roles(namespace).Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// cluster admin role for nocalhost
func (c *GoClient) CreateClusterRole(name string, rule []rbacv1.PolicyRule) (bool, error) {
	role := &rbacv1.ClusterRole{}
	role.ObjectMeta = metav1.ObjectMeta{
		Name: name,
		Labels: map[string]string{
			_const.NocalhostRoleBindingLabelKey: _const.NocalhostRoleBindingLabelVal,
		},
	}

	role.Rules = rule

	_, err := c.client.RbacV1().ClusterRoles().Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func getPolicyRule(c *GoClient) (*[]rbacv1.PolicyRule, error) {
	var result []rbacv1.PolicyRule

	_, lists, err := c.client.ServerGroupsAndResources()
	if err != nil {
		return &result, err
	}

	for _, list := range lists {
		for _, apiResource := range list.APIResources {
			var resourceName = apiResource.Name

			if apiResource.Namespaced && !isLimitedRules(resourceName) {

				result = append(
					result, rbacv1.PolicyRule{
						Verbs:     apiResource.Verbs,
						Resources: []string{resourceName},
						APIGroups: []string{"*"},
					},
				)
			}
		}
	}

	result = append(result, *LimitedRules)
	return &result, nil
}

func isLimitedRules(rn string) bool {
	return rn == "resourcequotas" || rn == "roles" || strings.HasPrefix(rn, "resourcequotas/") ||
		strings.HasPrefix(
			rn, "roles/",
		)
}

// deploy priorityclass
func (c *GoClient) CreateNocalhostPriorityClass() error {
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
func (c *GoClient) DeployNocalhostDep(namespace, serviceAccount, tag string) (bool, error) {
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
							Image:   c.MatchedArtifactVersion(DepInstaller, tag),
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
// use DaemonSet InitContainer make sure every Node pull images
// https://kubernetes.io/zh/docs/concepts/workloads/controllers/daemonset/
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
							Name:  "kubectl",
							Image: "nocalhost-docker.pkg.coding.net/nocalhost/public/kubectl:latest",
							Command: []string{
								"kubectl", "delete", "ds", global.NocalhostPrePullDSName, "-n",
								global.NocalhostSystemNamespace,
							},
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

// Initial admin kubeconfig in cluster for admission webhook
func (c *GoClient) CreateConfigMapWithValue(name, namespace, key, value string) (bool, error) {
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

// Get serviceAccount
func (c *GoClient) GetServiceAccount(name, namespace string) (*corev1.ServiceAccount, error) {
	return c.client.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// Get cluster node
func (c *GoClient) GetClusterNode() (*corev1.NodeList, error) {
	return c.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
}

func (c *GoClient) GetClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error) {
	return c.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
}

func (c *GoClient) ListClusterRoleBindingByLabel(label string) (*rbacv1.ClusterRoleBindingList, error) {
	return c.client.RbacV1().ClusterRoleBindings().List(
		context.TODO(), metav1.ListOptions{
			LabelSelector: label,
		},
	)
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
	watcher, err := c.client.CoreV1().ServiceAccounts(namespace).Watch(
		context.TODO(), metav1.ListOptions{
			FieldSelector:  fields.Set{"metadata.name": name}.AsSelector().String(),
			Watch:          true,
			TimeoutSeconds: &resourceWatchTimeoutSeconds,
		},
	)
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
	//	//FieldSelector:  fields.Set{"metadata.annotations.kubernetes.io/service-account.name": name}.
	//	AsSelector().String(),
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

	// loop and wait for serviceAccountToken Initial,especially for TKE is slow
	// wait 30S
	i := 0
	for {
		if i > 300 {
			break
		}
		serviceAccount, err = c.client.CoreV1().ServiceAccounts(namespace).Get(
			context.TODO(), name, metav1.GetOptions{},
		)
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

// Sprintf the specify artifact while image == ""
// or use the default image from param
func (c *GoClient) MatchedArtifactVersion(artifact, tags string) string {
	tag := global.Version

	if tag == "" {
		tag = global.CommitId
	}

	if tags != "" {
		tag = tags
	}

	return fmt.Sprintf("nocalhost-docker.pkg.coding.net/nocalhost/public/%s:%s", artifact, tag)
}

func (c *GoClient) ListJobs(namespace string) (*batchv1.JobList, error) {
	return c.client.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{})
}

func (c *GoClient) DeleteJob(namespace, name string) error {
	return c.client.BatchV1().Jobs(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (c *GoClient) ListPods(namespace string) (*corev1.PodList, error) {
	return c.client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
}

func (c *GoClient) DeletePod(namespace, name string) error {
	return c.client.CoreV1().Pods(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (c *GoClient) GetRestClient() (*restclient.RESTClient, error) {
	c.restConfig.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	c.restConfig.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	return restclient.RESTClientFor(c.restConfig)
}

func (c *GoClient) Clientset() *kubernetes.Clientset {
	return c.client
}

// IsNamespaceExist check if exist namespace
func (c *GoClient) IsNamespaceExist(ns string) (bool, error) {
	_, err := c.client.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}
