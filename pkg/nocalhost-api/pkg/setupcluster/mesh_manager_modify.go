/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package setupcluster

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

func meshDevModifier(ns string, r *unstructured.Unstructured) ([]MeshDevWorkload, error) {
	dependencies := make([]MeshDevWorkload, 0)
	var err error
	switch r.GetKind() {
	case Deployment:
		if dependencies, err = deploymentModifier(r); err != nil {
			return nil, err
		}
	case Service:
		if err := serviceModifier(r); err != nil {
			return nil, err
		}
	}

	if err := commonModifier(ns, r); err != nil {
		return nil, err
	}

	return dependencies, nil
}

func deploymentModifier(rs *unstructured.Unstructured) ([]MeshDevWorkload, error) {
	deploy := &appsv1.Deployment{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rs.UnstructuredContent(), deploy); err != nil {
		return nil, errors.WithStack(err)
	}
	deploy.Status.Reset()

	dependencies := podDependencyModifier(&deploy.Spec.Template.Spec)

	var err error
	if rs.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(deploy); err != nil {
		return nil, errors.WithStack(err)
	}
	return dependencies, nil
}

func serviceModifier(rs *unstructured.Unstructured) error {
	svc := &corev1.Service{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rs.UnstructuredContent(), svc); err != nil {
		return errors.WithStack(err)
	}
	svc.Status.Reset()
	svc.Spec.ClusterIP = ""
	svc.Spec.ClusterIPs = make([]string, 0)

	var err error
	if rs.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(svc); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func commonModifier(ns string, rs *unstructured.Unstructured) error {
	// reset
	resetModifier(rs)

	// set namespace
	rs.SetNamespace(ns)

	// set annotations
	annotations := rs.GetAnnotations()
	if annotations != nil && annotations[nocalhost.NocalhostApplicationNamespace] != "" {
		annotations[nocalhost.NocalhostApplicationNamespace] = ns
	}
	if annotations != nil && annotations[nocalhost.HelmReleaseName] != "" {
		annotations[nocalhost.HelmReleaseName] = ns
	}
	delete(annotations, "deployment.kubernetes.io/revision")
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
	delete(annotations, "control-plane.alpha.kubernetes.io/leader")
	rs.SetAnnotations(annotations)
	return nil
}

func podDependencyModifier(spec *corev1.PodSpec) []MeshDevWorkload {
	// modify the init containers
	initContainersModifier(spec)

	// modify volumes
	return volumeModifier(spec)
}

func initContainersModifier(spec *corev1.PodSpec) {
	initC := spec.InitContainers
	for i := 0; i < len(initC); i++ {
		if strings.HasPrefix(initC[i].Name, "wait-for-pods-") ||
			strings.HasPrefix(initC[i].Name, "wait-for-jobs-") {
			initC = initC[:i+copy(initC[i:], initC[i+1:])]
			i--
		}
	}
	spec.InitContainers = initC
}

func volumeModifier(spec *corev1.PodSpec) []MeshDevWorkload {
	// copy emptyDir, downwardAPI, hostPath, configMap, secret to new namespace, deprecate other volumes
	dependencies := make([]MeshDevWorkload, 0)
	delVolumesMounts := make(map[string]struct{})
	volumes := spec.Volumes
	for i := 0; i < len(volumes); i++ {
		if volumes[i].EmptyDir != nil ||
			volumes[i].DownwardAPI != nil ||
			volumes[i].HostPath != nil {
			continue
		}
		if volumes[i].ConfigMap != nil {
			dependencies = append(dependencies, MeshDevWorkload{
				Kind:   ConfigMap,
				Name:   volumes[i].ConfigMap.Name,
				Status: Selected,
			})
			continue
		}
		if volumes[i].Secret != nil {
			dependencies = append(dependencies, MeshDevWorkload{
				Kind:   Secret,
				Name:   volumes[i].Secret.SecretName,
				Status: Selected,
			})
			continue
		}
		delVolumesMounts[volumes[i].Name] = struct{}{}
		volumes = volumes[:i+copy(volumes[i:], volumes[i+1:])]
		i--
	}
	spec.Volumes = volumes

	// remove volumes mount from containers
	containers := spec.Containers
	for i, c := range containers {
		v := c.VolumeMounts
		for j := 0; j < len(v); j++ {
			if _, ok := delVolumesMounts[v[j].Name]; !ok {
				continue
			}
			v = v[:j+copy(v[j:], v[j+1:])]
			j--
		}
		containers[i].VolumeMounts = v
	}
	spec.Containers = containers

	// remove volumes mount from init containers
	initContainers := spec.InitContainers
	for i, c := range initContainers {
		v := c.VolumeMounts
		for j := 0; j < len(v); j++ {
			if _, ok := delVolumesMounts[v[j].Name]; !ok {
				continue
			}
			v = v[:j+copy(v[j:], v[j+1:])]
			j--
		}
		initContainers[i].VolumeMounts = v
	}
	spec.InitContainers = initContainers

	return dependencies
}

func resetModifier(rs *unstructured.Unstructured) {
	rs.SetGenerateName("")
	rs.SetSelfLink("")
	rs.SetUID("")
	rs.SetResourceVersion("")
	rs.SetGeneration(0)
	rs.SetDeletionGracePeriodSeconds(nil)
	rs.SetOwnerReferences(nil)
	rs.SetFinalizers(nil)
	rs.SetManagedFields(nil)
}

func genVirtualServiceForMeshDevSpace(baseNs string, r unstructured.Unstructured) (*unstructured.Unstructured, error) {
	//if r.GetKind() != "Service" {
	//	return nil, errors.Errorf("The kind of %s is %s, only support Service", r.GetName(), r.GetKind())
	//}
	vs := &v1alpha3.VirtualService{}
	vs.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
		Kind:    "VirtualService",
	})
	vs.SetName(r.GetName())
	vs.SetNamespace(r.GetNamespace())
	labels := r.GetLabels()
	labels["app.kubernetes.io/created-by"] = "nocalhost"
	vs.SetLabels(labels)

	annotations := r.GetAnnotations()
	delete(annotations, "deployment.kubernetes.io/revision")
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
	delete(annotations, "control-plane.alpha.kubernetes.io/leader")
	annotations[nocalhost.AppManagedByLabel] = nocalhost.AppManagedByNocalhost
	vs.SetAnnotations(annotations)
	vs.Spec.Hosts = []string{r.GetName()}
	vs.Spec.Http = []*istiov1alpha3.HTTPRoute{}

	// http route
	host := fmt.Sprintf("%s.%s.%s.%s", r.GetName(), baseNs, "svc", "cluster.local")
	httpRoute := make([]*istiov1alpha3.HTTPRoute, 0)
	httpDsts := make([]*istiov1alpha3.HTTPRouteDestination, 0)
	httpDst := &istiov1alpha3.HTTPRouteDestination{Destination: &istiov1alpha3.Destination{Host: host}}
	httpDsts = append(httpDsts, httpDst)
	http := &istiov1alpha3.HTTPRoute{Route: httpDsts}
	httpRoute = append(httpRoute, http)
	vs.Spec.Http = httpRoute

	// tcp route
	tcpRoute := make([]*istiov1alpha3.TCPRoute, 0)
	tcpDsts := make([]*istiov1alpha3.RouteDestination, 0)
	tcpDst := &istiov1alpha3.RouteDestination{Destination: &istiov1alpha3.Destination{Host: host}}
	tcpDsts = append(tcpDsts, tcpDst)
	tcp := &istiov1alpha3.TCPRoute{Route: tcpDsts}
	tcpRoute = append(tcpRoute, tcp)
	vs.Spec.Tcp = tcpRoute

	var err error
	rs := &unstructured.Unstructured{}
	rs.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(vs)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return rs, nil
}

func genVirtualServiceForBaseDevSpace(baseNs, devNs, name string, header model.Header) (*unstructured.Unstructured, error) {
	if header.TraceKey == "" || header.TraceValue == "" {
		return nil, errors.New("can not find tracing header")
	}
	vs := &v1alpha3.VirtualService{}
	vs.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
		Kind:    "VirtualService",
	})
	vs.SetName(name)
	vs.SetNamespace(baseNs)
	vs.SetLabels(map[string]string{
		nocalhost.AppManagedByLabel:    nocalhost.AppManagedByNocalhost,
		"app.kubernetes.io/created-by": "nocalhost",
	})
	vs.SetAnnotations(map[string]string{
		nocalhost.AppManagedByLabel:    nocalhost.AppManagedByNocalhost,
		"app.kubernetes.io/created-by": "nocalhost",
	})
	vs.Spec.Hosts = []string{name}
	vs.Spec.Http = []*istiov1alpha3.HTTPRoute{}

	// http route
	host := fmt.Sprintf("%s.%s.%s.%s", name, devNs, "svc", "cluster.local")
	httpRoutes := make([]*istiov1alpha3.HTTPRoute, 0)
	httpDsts := make([]*istiov1alpha3.HTTPRouteDestination, 0)
	httpDst := &istiov1alpha3.HTTPRouteDestination{Destination: &istiov1alpha3.Destination{Host: host}}
	httpDsts = append(httpDsts, httpDst)
	headers := make(map[string]*istiov1alpha3.StringMatch)
	// set exact match header
	headers[header.TraceKey] = &istiov1alpha3.StringMatch{
		MatchType: &istiov1alpha3.StringMatch_Exact{
			Exact: header.TraceValue,
		},
	}

	http := &istiov1alpha3.HTTPRoute{
		Name: global.NocalhostDevNamespaceLabel + "-" + devNs,
		Match: []*istiov1alpha3.HTTPMatchRequest{
			{
				Headers: headers,
			},
		},
		Route: httpDsts,
	}
	httpRoutes = append(httpRoutes, http)

	//default http route
	defaultHost := fmt.Sprintf("%s.%s.%s.%s", name, baseNs, "svc", "cluster.local")
	defaultHttpDsts := make([]*istiov1alpha3.HTTPRouteDestination, 0)
	defaultHttpDst := &istiov1alpha3.HTTPRouteDestination{Destination: &istiov1alpha3.Destination{Host: defaultHost}}
	defaultHttpDsts = append(defaultHttpDsts, defaultHttpDst)
	defaultHttp := &istiov1alpha3.HTTPRoute{Route: defaultHttpDsts}
	httpRoutes = append(httpRoutes, defaultHttp)

	vs.Spec.Http = httpRoutes

	var err error
	rs := &unstructured.Unstructured{}
	rs.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(vs)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return rs, nil
}

func addHeaderToVirtualService(rs *unstructured.Unstructured, devNs, svcName string, header model.Header) error {
	resetModifier(rs)

	if header.TraceKey == "" || header.TraceValue == "" {
		log.Debugf("can not find tracing header to update virtual service on the namespace %s",
			rs.GetNamespace())
	}

	vs := &v1alpha3.VirtualService{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rs.UnstructuredContent(), vs); err != nil {
		return errors.WithStack(err)
	}
	name := svcName
	route := vs.Spec.Http
	for i := 0; i < len(route); i++ {
		if route[i].GetName() == name {
			route = route[:i+copy(route[i:], route[i+1:])]
			i--
		}
	}

	// add header
	host := fmt.Sprintf("%s.%s.%s.%s", name, devNs, "svc", "cluster.local")
	httpDsts := make([]*istiov1alpha3.HTTPRouteDestination, 0)
	httpDst := &istiov1alpha3.HTTPRouteDestination{Destination: &istiov1alpha3.Destination{Host: host}}
	httpDsts = append(httpDsts, httpDst)
	headers := make(map[string]*istiov1alpha3.StringMatch)
	// set exact match header
	headers[header.TraceKey] = &istiov1alpha3.StringMatch{
		MatchType: &istiov1alpha3.StringMatch_Exact{
			Exact: header.TraceValue,
		},
	}

	http := &istiov1alpha3.HTTPRoute{
		Name: global.NocalhostDevNamespaceLabel + "-" + devNs,
		Match: []*istiov1alpha3.HTTPMatchRequest{
			{
				Headers: headers,
			},
		},
		Route: httpDsts,
	}
	route = append(route, &istiov1alpha3.HTTPRoute{})
	copy(route[1:], route[:])
	route[0] = http
	vs.Spec.Http = route

	var err error
	rs.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(vs)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func deleteHeaderFromVirtualService(rs *unstructured.Unstructured, devNs string, header model.Header) error {
	resetModifier(rs)

	vs := &v1alpha3.VirtualService{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rs.UnstructuredContent(), vs); err != nil {
		return errors.WithStack(err)
	}
	name := global.NocalhostDevNamespaceLabel + "-" + devNs
	route := vs.Spec.Http
	for i := 0; i < len(route); i++ {
		if route[i].GetName() == name {
			route = route[:i+copy(route[i:], route[i+1:])]
			i--
		}
	}
	vs.Spec.Http = route

	var err error
	rs.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(vs)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
