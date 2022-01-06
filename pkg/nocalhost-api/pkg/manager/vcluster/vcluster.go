/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package vcluster

import (
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	helmv1alpha1 "nocalhost/internal/nocalhost-dep/controllers/vcluster/api/v1alpha1"
	"nocalhost/internal/nocalhost-dep/controllers/vcluster/helper"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
)

const (
	defaultResync = 10 * time.Minute
)

type Manager interface {
	GetInfo(spaceName, namespace string) (model.VirtualClusterInfo, error)
	GetKubeConfig(spaceName, namespace string) (string, string, error)
	Update(spaceName, namespace, clusterName string, v *model.VirtualClusterInfo) error
	Create(spaceName, namespace, clusterName string, v *model.VirtualClusterInfo) error
	close()
}

type manager struct {
	mu        sync.Mutex
	client    *clientgo.GoClient
	informers dynamicinformer.DynamicSharedInformerFactory
	stopCh    chan struct{}
}

var _ Manager = &manager{}

func (m *manager) GetInfo(spaceName, namespace string) (model.VirtualClusterInfo, error) {
	name := global.VClusterPrefix + namespace
	info := model.VirtualClusterInfo{}
	vc, err := m.getVirtualCluster(name, namespace)
	if err != nil {
		info.Status = string(helmv1alpha1.Unknown)
		return info, err
	}
	info.Status = string(vc.Status.Phase)
	info.Version = vc.GetChartVersion()
	info.Values = vc.GetValues()
	info.ServiceType = corev1.ServiceType(vc.GetServiceType())
	return info, nil
}

func (m *manager) GetKubeConfig(spaceName, namespace string) (string, string, error) {
	name := global.VClusterPrefix + namespace
	vc, err := m.getVirtualCluster(name, namespace)
	if err != nil {
		return "", "", err
	}

	if vc.Status.Phase != helmv1alpha1.Ready {
		return "", "", errors.New("virtual cluster is not ready")
	}

	kubeConfig, err := base64.StdEncoding.DecodeString(vc.Status.AuthConfig)
	if err != nil {
		return "", "", err
	}
	serviceType := vc.GetServiceType()
	return string(kubeConfig), serviceType, nil
}

func (m *manager) Update(spaceName, namespace, clusterName string, v *model.VirtualClusterInfo) error {
	name := global.VClusterPrefix + namespace
	vc, err := m.getVirtualCluster(name, namespace)
	if err != nil {
		return err
	}

	if vc.GetValues() == v.Values &&
		vc.GetServiceType() == string(v.ServiceType) &&
		vc.GetChartVersion() == v.Version &&
		vc.GetSpaceName() == spaceName {
		return nil
	}

	vc.SetValues(v.Values)
	vc.SetChartVersion(v.Version)
	annotations := vc.GetAnnotations()
	annotations[helmv1alpha1.ServiceTypeKey] = string(v.ServiceType)
	annotations[helmv1alpha1.Timestamp] = strconv.Itoa(int(time.Now().UnixNano()))
	annotations[helmv1alpha1.SpaceName] = spaceName
	vc.SetAnnotations(annotations)
	vc.SetManagedFields(nil)

	_, err = m.client.Apply(vc)
	return err
}

func (m *manager) Create(spaceName, namespace, clusterName string, v *model.VirtualClusterInfo) error {
	name := global.VClusterPrefix + namespace
	vc := &helmv1alpha1.VirtualCluster{}
	vc.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "helm.nocalhost.dev",
		Version: "v1alpha1",
		Kind:    "VirtualCluster",
	})
	vc.SetName(name)
	vc.SetNamespace(namespace)
	vc.SetValues(v.Values)
	vc.SetChartName("vcluster")
	vc.SetChartRepo(global.NocalhostChartRepository)
	vc.SetChartVersion(v.Version)
	annotations := map[string]string{
		helmv1alpha1.ServiceTypeKey: string(v.ServiceType),
		helmv1alpha1.SpaceName:      spaceName,
		helmv1alpha1.ClusterName:    clusterName,
		helmv1alpha1.Timestamp:      strconv.Itoa(int(time.Now().UnixNano())),
	}
	vc.SetAnnotations(annotations)

	vc.Status.Phase = helmv1alpha1.Upgrading

	_, err := m.client.Apply(vc)
	return err
}

func (m *manager) PortForward(spaceName, namespace string) error {
	return nil
}

func (m *manager) vcInformer() informers.GenericInformer {
	m.mu.Lock()
	defer m.mu.Unlock()
	informer := m.informers.ForResource(schema.GroupVersionResource{
		Group:    "helm.nocalhost.dev",
		Version:  "v1alpha1",
		Resource: "virtualclusters",
	})
	m.informers.Start(m.stopCh)
	m.informers.WaitForCacheSync(m.stopCh)
	return informer
}

func (m *manager) getVirtualCluster(name, namespace string) (*helmv1alpha1.VirtualCluster, error) {
	informer := m.vcInformer()
	informer.Lister()
	obj, exists, err := informer.Informer().GetIndexer().GetByKey(namespace + "/" + name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !exists {
		return nil, errors.Errorf("virtual cluster not found: %s/%s", namespace, name)
	}
	vc := &helmv1alpha1.VirtualCluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		obj.(*unstructured.Unstructured).UnstructuredContent(), vc); err != nil {
		return nil, errors.WithStack(err)
	}
	return vc, nil
}

func (m *manager) getVirtualClusterList() (*helmv1alpha1.VirtualClusterList, error) {
	informer := m.vcInformer()
	informer.Lister()
	objs := informer.Informer().GetIndexer().List()
	vcList := &helmv1alpha1.VirtualClusterList{}
	for i := 0; i < len(objs); i++ {
		vc := &helmv1alpha1.VirtualCluster{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
			objs[i].(*unstructured.Unstructured).UnstructuredContent(), vc); err != nil {
			return nil, errors.WithStack(err)
		}
		vcList.Items = append(vcList.Items, *vc)
	}
	return vcList, nil
}

func (m *manager) close() {
	close(m.stopCh)
}

func newManager(client *clientgo.GoClient) Manager {
	return &manager{
		client:    client,
		informers: dynamicinformer.NewDynamicSharedInformerFactory(client.DynamicClient, defaultResync),
	}
}

func PortForwardAndGetKubeConfig(namespace string, config *rest.Config, c kubernetes.Interface, stopChan <-chan struct{}) ([]byte, error) {
	name := global.VClusterPrefix + namespace

	// 1. get vcluster pod
	pod, err := helper.GetVClusterPod(name, namespace, time.Second, time.Second*5, c)
	if err != nil {
		return nil, err
	}

	// 2. get vcluster kubeconfig
	OriginalKubeConfig, err := helper.GetVClusterKubeConfigFromPod(pod, config, c)
	if err != nil {
		return nil, err
	}

	// 3. port forward
	req := c.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		Timeout(time.Second * 5).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	var stdout, stderr io.Writer
	errChan := make(chan error)
	readyChan := make(chan struct{})
	fw, err := portforward.NewOnAddresses(dialer, []string{"localhost"}, []string{":8443"}, stopChan, readyChan, stdout, stderr)
	if err != nil {
		return nil, err
	}

	go func() {
		errChan <- fw.ForwardPorts()
	}()

	select {
	case err = <-errChan:
		return nil, err
	case <-readyChan:
	case <-stopChan:
		return nil, errors.New("stopped before ready")
	}

	ports, err := fw.GetPorts()
	if err != nil {
		return nil, err
	}
	if len(ports) == 0 {
		return nil, errors.Errorf("no port is forwarded")
	}
	port := ports[0].Local

	// 4. get change kubeconfig
	kubeConfig, err := clientcmd.Load(OriginalKubeConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ctx := kubeConfig.Contexts[kubeConfig.CurrentContext]
	if ctx == nil {
		return nil, errors.Errorf("vcluster kubeconfig context not found")
	}
	if _, ok := kubeConfig.Clusters[ctx.Cluster]; !ok {
		return nil, errors.Errorf("vcluster kubeconfig cluster not found")
	}
	server := kubeConfig.Clusters[ctx.Cluster].Server
	server = strings.Replace(server, "8443", strconv.Itoa(int(port)), -1)
	kubeConfig.Clusters[ctx.Cluster].Server = server
	out, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return out, nil
}
