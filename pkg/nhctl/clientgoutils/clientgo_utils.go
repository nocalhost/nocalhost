/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"context"
	"fmt"
	"io/ioutil"
	"k8s.io/api/batch/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/flowcontrol"
	"nocalhost/internal/nhctl/utils"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	appsV1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	batchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	batchV1beta1 "k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"nocalhost/pkg/nhctl/log"
)

type ClientGoUtils struct {
	kubeConfigFilePath      string
	restConfig              *restclient.Config
	restMapper              meta.RESTMapper
	ClientSet               *kubernetes.Clientset
	dynamicClient           dynamic.Interface //
	ClientConfig            clientcmd.ClientConfig
	namespace               string
	includeDeletedResources bool
	ctx                     context.Context
	labels                  map[string]string
	fieldSelector           string

	gvrCache     map[string]schema.GroupVersionResource
	gvrCacheLock sync.Mutex
}

type PortForwardAPodRequest struct {
	// listenAddress
	Listen []string
	// Pod is the selected pod for this port forwarding
	Pod corev1.Pod
	// LocalPort is the local port that will be selected to expose the PodPort
	LocalPort int
	// PodPort is the target port for the pod
	PodPort int
	// Steams configures where to write or read input from
	Streams genericclioptions.IOStreams
	// StopCh is the channel used to manage the port forward lifecycle
	StopCh <-chan struct{}
	// ReadyCh communicates when the tunnel is ready to receive traffic
	ReadyCh chan struct{}
}

// NewClientGoUtils If namespace is not specified, use namespace defined in kubeconfig
// If namespace is not specified and can not get from kubeconfig, return error
func NewClientGoUtils(kubeConfigPath string, namespace string) (*ClientGoUtils, error) {
	var (
		err error
	)

	if kubeConfigPath == "" { // use default config
		kubeConfigPath = filepath.Join(utils.GetHomePath(), ".kube", "config")
	}

	abs, err := filepath.Abs(kubeConfigPath)
	if err != nil {
		return nil, errors.Wrap(err, "please make sure kubeconfig path is reachable")
	}
	kubeConfigPath = abs

	client := &ClientGoUtils{
		kubeConfigFilePath: kubeConfigPath,
		namespace:          namespace,
	}

	client.ClientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}},
	)

	client.restConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	// set default rateLimiter to 100, in case of throttling request
	client.restConfig.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(1000, 2000)

	if client.ClientSet, err = kubernetes.NewForConfig(client.restConfig); err != nil {
		return nil, errors.Wrap(err, "")
	}

	if client.dynamicClient, err = dynamic.NewForConfig(client.restConfig); err != nil {
		return nil, errors.Wrap(err, "")
	}

	if client.restMapper, err = client.NewFactory().ToRESTMapper(); err != nil {
		return nil, errors.Wrap(err, "")
	}

	if client.namespace == "" {
		client.namespace, err = client.GetDefaultNamespace()
		if err != nil {
			return nil, err
		}
	}

	client.ctx = context.TODO()

	return client, nil
}

func GetKubeContentFromPath(kubePath string) ([]byte, error) {
	if kubePath == "" { // use default config
		kubePath = filepath.Join(utils.GetHomePath(), ".kube", "config")
	}

	abs, err := filepath.Abs(kubePath)
	if err != nil {
		return nil, errors.Wrap(err, "please make sure kubeconfig path is reachable")
	}

	bys, err := ioutil.ReadFile(abs)
	return bys, errors.Wrap(err, "")
}

func (c *ClientGoUtils) KubeConfigFilePath() string {
	return c.kubeConfigFilePath
}

// NameSpace Set ClientGoUtils's namespace
func (c *ClientGoUtils) NameSpace(namespace string) *ClientGoUtils {
	c.namespace = namespace
	return c
}

func (c *ClientGoUtils) GetNameSpace() string {
	return c.namespace
}

// Context Set ClientGoUtils's Context
func (c *ClientGoUtils) Context(ctx context.Context) *ClientGoUtils {
	c.ctx = ctx
	return c
}

func (c *ClientGoUtils) Labels(labels map[string]string) *ClientGoUtils {
	cc := c.GetCopy()
	cc.labels = labels
	return cc
}

func (c *ClientGoUtils) GetCopy() *ClientGoUtils {
	cc := *c
	cc.labels = map[string]string{}
	for s, s2 := range c.labels {
		cc.labels[s] = s2
	}
	return &cc
}

func (c *ClientGoUtils) FieldSelector(f string) *ClientGoUtils {
	cc := c.GetCopy()
	cc.fieldSelector = f
	return cc
}

func (c *ClientGoUtils) IncludeDeletedResources(i bool) *ClientGoUtils {
	c.includeDeletedResources = i
	return c
}

func (c *ClientGoUtils) getListOptions() metav1.ListOptions {
	ops := metav1.ListOptions{}
	if len(c.labels) > 0 {
		ops.LabelSelector = labels.Set(c.labels).String()
	}
	if len(c.fieldSelector) > 0 {
		ops.FieldSelector = c.fieldSelector
	}
	return ops
}

func (c *ClientGoUtils) GetDynamicClient() dynamic.Interface {
	var restConfig *restclient.Config
	restConfig, _ = clientcmd.BuildConfigFromFlags("", c.kubeConfigFilePath)
	dyn, _ := dynamic.NewForConfig(restConfig)
	return dyn
}

func (c *ClientGoUtils) getRestConfig() (*restclient.Config, error) {
	return clientcmd.BuildConfigFromFlags("", c.kubeConfigFilePath)
}

func GetNamespaceFromKubeConfig(kubeConfig string) (string, error) {

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfig},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}},
	)
	ns, _, err := clientConfig.Namespace()
	return ns, errors.Wrap(err, "")
}

func (c *ClientGoUtils) GetDefaultNamespace() (string, error) {
	ns, _, err := c.ClientConfig.Namespace()
	return ns, errors.Wrap(err, "")
}

func (c *ClientGoUtils) GetDiscoveryClient() (*discovery.DiscoveryClient, error) {
	config, err := c.getRestConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(config)
}

func (c *ClientGoUtils) Discovery() {
	discoveryClient, err := c.GetDiscoveryClient()
	if err != nil {
		fmt.Println("failed to get discovery client")
		return
	}

	apiGroups, resourceList, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		fmt.Println("failed to discover")
		return
	}
	fmt.Println("the following api groups found:")
	for _, apiGroup := range apiGroups {
		fmt.Printf("%s %s %s\n", apiGroup.Kind, apiGroup.APIVersion, apiGroup.Name)
	}

	fmt.Println("the following resources found:")
	for _, list := range resourceList {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			fmt.Println("fail to get gv")
		}
		for _, resource := range list.APIResources {
			fmt.Printf("name:%30s group:%15s version:%s\n", resource.Name, gv.Group, gv.Version)
		}
	}
}

func (c *ClientGoUtils) GetDeploymentClient() appsV1.DeploymentInterface {
	return c.ClientSet.AppsV1().Deployments(c.namespace)
}

func (c *ClientGoUtils) GetStatefulSetClient() appsV1.StatefulSetInterface {
	return c.ClientSet.AppsV1().StatefulSets(c.namespace)
}

func (c *ClientGoUtils) GetDaemonSetClient() appsV1.DaemonSetInterface {
	return c.ClientSet.AppsV1().DaemonSets(c.namespace)
}

func (c *ClientGoUtils) GetJobsClient() batchV1.JobInterface {
	return c.ClientSet.BatchV1().Jobs(c.namespace)
}

func (c *ClientGoUtils) GetCronJobsClient() batchV1beta1.CronJobInterface {
	return c.ClientSet.BatchV1beta1().CronJobs(c.namespace)
}

func (c *ClientGoUtils) GetPodClient() coreV1.PodInterface {
	return c.ClientSet.CoreV1().Pods(c.namespace)
}

func (c *ClientGoUtils) GetPod(name string) (*corev1.Pod, error) {
	dep, err := c.GetPodClient().Get(c.ctx, name, metav1.GetOptions{})
	return dep, errors.Wrap(err, "")
}

func (c *ClientGoUtils) GetDaemonSet(name string) (*v1.DaemonSet, error) {
	dep, err := c.GetDaemonSetClient().Get(c.ctx, name, metav1.GetOptions{})
	return dep, errors.Wrap(err, "")
}

func (c *ClientGoUtils) GetStatefulSet(name string) (*v1.StatefulSet, error) {
	dep, err := c.GetStatefulSetClient().Get(c.ctx, name, metav1.GetOptions{})
	return dep, errors.Wrap(err, "")
}

func (c *ClientGoUtils) GetJobs(name string) (*batchv1.Job, error) {
	dep, err := c.GetJobsClient().Get(c.ctx, name, metav1.GetOptions{})
	return dep, errors.Wrap(err, "")
}

func (c *ClientGoUtils) ListJobs() (*batchv1.JobList, error) {
	dep, err := c.GetJobsClient().List(c.ctx, metav1.ListOptions{})
	return dep, errors.Wrap(err, "")
}

func (c *ClientGoUtils) GetCronJobs(name string) (*v1beta1.CronJob, error) {
	dep, err := c.GetCronJobsClient().Get(c.ctx, name, metav1.GetOptions{})
	return dep, errors.Wrap(err, "")
}

func (c *ClientGoUtils) CheckDeploymentReady(name string) (bool, error) {
	deployment, err := c.GetDeployment(name)
	if err != nil {
		return false, err
	}
	for _, c := range deployment.Status.Conditions {
		if c.Type == v1.DeploymentAvailable && c.Status == "True" {
			return true, nil
		}
	}
	return false, nil
}

func (c *ClientGoUtils) ListPodsOfDeployment(deployName string) ([]corev1.Pod, error) {
	podClient := c.GetPodClient()

	podList, err := podClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	result := make([]corev1.Pod, 0)

OuterLoop:
	for _, pod := range podList.Items {
		if !c.includeDeletedResources {
			if pod.DeletionTimestamp != nil {
				continue
			}
		}
		for _, ref := range pod.OwnerReferences {
			if ref.Kind != "ReplicaSet" {
				continue
			}
			rss, _ := c.GetReplicaSetsByDeployment(deployName)
			for _, rs := range rss {
				if rs.Name == ref.Name {
					result = append(result, pod)
					continue OuterLoop
				}
			}
		}
	}
	return result, nil
}

func (c *ClientGoUtils) ListLatestRevisionPodsByDeployment(deployName string) ([]corev1.Pod, error) {
	podClient := c.GetPodClient()

	podList, err := podClient.List(c.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	result := make([]corev1.Pod, 0)

	// Find the latest revision
	replicaSets, err := c.GetReplicaSetsByDeployment(deployName)
	if err != nil {
		log.WarnE(err, "Failed to get replica sets")
		return nil, err
	}
	revisions := make([]int, 0)
	for _, rs := range replicaSets {
		if rs.Annotations["deployment.kubernetes.io/revision"] != "" {
			r, _ := strconv.Atoi(rs.Annotations["deployment.kubernetes.io/revision"])
			revisions = append(revisions, r)
		}
	}

	if len(revisions) < 1 {
		return nil, errors.New("No replicaSets found")
	}

	sort.Ints(revisions)

	latestRevision := revisions[len(revisions)-1]

	var latestRevisionReplicasets *v1.ReplicaSet
	for _, rs := range replicaSets {
		if rs.Annotations["deployment.kubernetes.io/revision"] != "" {
			r, _ := strconv.Atoi(rs.Annotations["deployment.kubernetes.io/revision"])
			if r == latestRevision {
				latestRevisionReplicasets = rs
			}
		}
	}

OuterLoop:
	for _, pod := range podList.Items {
		if pod.OwnerReferences == nil || pod.DeletionTimestamp != nil {
			continue
		}
		for _, ref := range pod.OwnerReferences {
			if ref.Kind != "ReplicaSet" {
				continue
			}

			if latestRevisionReplicasets.Name == ref.Name {
				result = append(result, pod)
				continue OuterLoop
			}
		}
	}
	return result, nil
}

func waitForJob(obj runtime.Object, name string) (bool, error) {
	o, ok := obj.(*batchv1.Job)
	if !ok {
		return true, errors.Errorf("expected %s to be a *batch.Job, got %T", name, obj)
	}

	for _, c := range o.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == "True" {
			log.Infof("Job %s completed", name)
			return true, nil
		} else if c.Type == batchv1.JobFailed && c.Status == "True" {
			log.Infof("Job %s failed", name)
			return true, errors.Errorf("job failed: %s", c.Reason)
		}
	}
	log.Infof("Job %s running", name)

	return false, nil
}

//func (c *ClientGoUtils) PortForwardAPod(req PortForwardAPodRequest) error {
//	path := fmt.Sprintf(
//		"/api/v1/namespaces/%s/pods/%s/portforward",
//		req.Pod.Namespace, req.Pod.Name,
//	)
//	clientConfig, err := c.ClientConfig.ClientConfig()
//	if err != nil {
//		return errors.Wrap(err, "")
//	}
//
//	transport, upgrader, err := spdy.RoundTripperFor(clientConfig)
//	if err != nil {
//		return errors.Wrap(err, "")
//	}
//
//	parseUrl, err := url.Parse(clientConfig.Host)
//	if err != nil {
//		return errors.Wrap(err, "")
//	}
//	parseUrl.Path = path
//	dialer := spdy.NewDialer(
//		upgrader, &http.Client{Transport: transport}, http.MethodPost,
//		//&url.URL{Scheme: schema, Path: path, Host: hostIP},
//		parseUrl,
//	)
//	// fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)}, req.StopCh,
//	//req.ReadyCh, req.Streams.Out, req.Streams.ErrOut)
//	fw, err := NewOnAddresses(
//		dialer, req.Listen, []string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)}, req.StopCh, req.ReadyCh,
//		req.Streams.Out, req.Streams.ErrOut,
//	)
//	if err != nil {
//		return errors.Wrap(err, "")
//	}
//	return errors.Wrap(fw.ForwardPorts(), "")
//}

func (c *ClientGoUtils) GetNodesList() (*corev1.NodeList, error) {
	nodes, err := c.ClientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return &corev1.NodeList{}, errors.Wrap(err, "")
	}
	return nodes, nil
}

func (c *ClientGoUtils) GetService(name string) (*corev1.Service, error) {
	service, err := c.ClientSet.CoreV1().Services(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return &corev1.Service{}, errors.Wrap(err, "")
	}
	return service, nil
}

func (c *ClientGoUtils) GetContext() context.Context {
	return c.ctx
}

func (c *ClientGoUtils) DeleteStatefulSetAndPVC(name string) error {
	_ = c.ClientSet.AppsV1().StatefulSets(c.namespace).Delete(
		c.ctx, name, metav1.DeleteOptions{GracePeriodSeconds: new(int64)},
	)
	pvc, err := c.ClientSet.CoreV1().PersistentVolumeClaims(c.namespace).Get(
		c.ctx, "data-nocalhost-mariadb-0", metav1.GetOptions{},
	)
	if err != nil {
		pvName := pvc.Spec.VolumeName
		_ = c.ClientSet.CoreV1().PersistentVolumeClaims(c.namespace).Delete(
			c.ctx, "data-nocalhost-mariadb-0", metav1.DeleteOptions{},
		)
		_ = c.ClientSet.CoreV1().PersistentVolumes().Delete(c.ctx, pvName, metav1.DeleteOptions{})
	}
	return nil
}

func (c *ClientGoUtils) DeletePod(podName string, wait bool, duration time.Duration) error {
	ctx, cancelFunc := context.WithTimeout(context.TODO(), duration)
	defer cancelFunc()

	err := c.ClientSet.CoreV1().Pods(c.namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if !wait {
		return err
	}
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("not found shadow pod, no need to delete it")
			return nil
		}
		return err
	}
	log.Infof("waiting for pod: %s to be deleted...", podName)
	w, errs := c.ClientSet.CoreV1().Pods(c.namespace).Watch(
		ctx, metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", podName).String(),
			Watch:         true,
		},
	)
	if errs != nil {
		log.Error(errs)
		return errs
	}
out:
	for {
		select {
		case event := <-w.ResultChan():
			if watch.Deleted == event.Type {
				break out
			}
		case <-ctx.Done():
			return errors.New("timeout")
		}
	}
	log.Infof("delete pod: %s successfully", podName)
	return nil
}

// GetControllerOf returns a pointer to the controllerRef if controllee has a controller
func GetControllerOfNoCopy(refs []metav1.OwnerReference) *metav1.OwnerReference {
	for i := range refs {
		if refs[i].Controller != nil && *refs[i].Controller {
			return &refs[i]
		}
	}
	return nil
}
