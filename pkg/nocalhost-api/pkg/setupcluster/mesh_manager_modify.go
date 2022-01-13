/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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

	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

const (
	DefaultClusterDomain = "cluster.local"
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
	ports := svc.Spec.Ports
	for i := range ports {
		ports[i].NodePort = 0
	}
	svc.Spec.Ports = ports

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
	if annotations != nil && annotations[_const.NocalhostApplicationNamespace] != "" {
		annotations[_const.NocalhostApplicationNamespace] = ns
	}
	if annotations != nil && annotations[_const.HelmReleaseName] != "" {
		annotations[_const.HelmReleaseName] = ns
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
	dependencies := volumeModifier(spec)
	// get env dependencies
	dependencies = append(dependencies, getEnvDependency(spec)...)
	// get image pull secret dependencies
	dependencies = append(dependencies, getImagePullSecretDependency(spec)...)

	return dependencies
}

func initContainersModifier(spec *corev1.PodSpec) {
	// delete wait init containers
	initC := spec.InitContainers
	for i := 0; i < len(initC); i++ {
		if strings.HasPrefix(initC[i].Name, "wait-for-pods-") ||
			strings.HasPrefix(initC[i].Name, "wait-for-jobs-") ||
			strings.HasPrefix(initC[i].Name, "nocalhost-dependency-waiting-job") {
			initC = initC[:i+copy(initC[i:], initC[i+1:])]
			i--
		}
	}
	spec.InitContainers = initC
}

func volumeModifier(spec *corev1.PodSpec) []MeshDevWorkload {
	// copy emptyDir, downwardAPI, hostPath, configMap, secret to new namespace, deprecate other volumes
	dependencies := make([]MeshDevWorkload, 0)
	delVolumeMounts := make(map[string]struct{})
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
		delVolumeMounts[volumes[i].Name] = struct{}{}
		volumes = volumes[:i+copy(volumes[i:], volumes[i+1:])]
		i--
	}
	spec.Volumes = volumes

	// delete volumes mount from containers
	containers := spec.Containers
	for i, c := range containers {
		v := c.VolumeMounts
		for j := 0; j < len(v); j++ {
			if _, ok := delVolumeMounts[v[j].Name]; !ok {
				continue
			}
			v = v[:j+copy(v[j:], v[j+1:])]
			j--
		}
		containers[i].VolumeMounts = v
	}
	spec.Containers = containers

	// delete volumes mount from init containers
	initContainers := spec.InitContainers
	for i, c := range initContainers {
		v := c.VolumeMounts
		for j := 0; j < len(v); j++ {
			if _, ok := delVolumeMounts[v[j].Name]; !ok {
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

func getImagePullSecretDependency(spec *corev1.PodSpec) []MeshDevWorkload {
	dependencies := make([]MeshDevWorkload, 0)
	for _, secret := range spec.ImagePullSecrets {
		if secret.Name == "" {
			continue
		}
		dependencies = append(dependencies, MeshDevWorkload{
			Kind:   Secret,
			Name:   secret.Name,
			Status: Selected,
		})
	}
	return dependencies
}

func getEnvDependency(spec *corev1.PodSpec) []MeshDevWorkload {
	dependencies := make([]MeshDevWorkload, 0)
	for _, container := range spec.Containers {
		dependencies = append(dependencies, getEnvDependencyFromContainer(container)...)
	}
	for _, container := range spec.InitContainers {
		dependencies = append(dependencies, getEnvDependencyFromContainer(container)...)
	}
	return dependencies
}

func getEnvDependencyFromContainer(container corev1.Container) []MeshDevWorkload {
	dependencies := make([]MeshDevWorkload, 0)
	for _, e := range container.Env {
		if e.ValueFrom == nil {
			continue
		}
		if e.ValueFrom.ConfigMapKeyRef != nil {
			dependencies = append(dependencies, MeshDevWorkload{
				Kind:   ConfigMap,
				Name:   e.ValueFrom.ConfigMapKeyRef.Name,
				Status: Selected,
			})
		}
		if e.ValueFrom.SecretKeyRef != nil {
			dependencies = append(dependencies, MeshDevWorkload{
				Kind:   Secret,
				Name:   e.ValueFrom.SecretKeyRef.Name,
				Status: Selected,
			})
		}
	}

	for _, e := range container.EnvFrom {
		if e.ConfigMapRef != nil {
			dependencies = append(dependencies, MeshDevWorkload{
				Kind:   ConfigMap,
				Name:   e.ConfigMapRef.Name,
				Status: Selected,
			})
		}
		if e.SecretRef != nil {
			dependencies = append(dependencies, MeshDevWorkload{
				Kind:   Secret,
				Name:   e.SecretRef.Name,
				Status: Selected,
			})
		}
	}
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
	annotations[_const.AppManagedByLabel] = _const.AppManagedByNocalhost
	vs.SetAnnotations(annotations)
	vs.Spec.Hosts = []string{r.GetName()}
	vs.Spec.Http = []*istiov1alpha3.HTTPRoute{}

	// http route
	host := fmt.Sprintf("%s.%s.%s.%s", r.GetName(), baseNs, "svc", DefaultClusterDomain)
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

func genVirtualServiceForBaseDevSpace(baseNs, devNs, name string, header model.Header) (
	*unstructured.Unstructured, *istiov1alpha3.HTTPRoute, error) {
	if header.TraceKey == "" || header.TraceValue == "" {
		return nil, nil, errors.New("can not find tracing header")
	}
	log.Debugf("generate %s/%s", "VirtualService", name)
	vs := &v1alpha3.VirtualService{}
	vs.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
		Kind:    "VirtualService",
	})
	vs.SetName(name)
	vs.SetNamespace(baseNs)
	vs.SetLabels(map[string]string{
		_const.AppManagedByLabel:      _const.AppManagedByNocalhost,
		global.NocalhostCreateByLabel: global.NocalhostName,
	})
	vs.SetAnnotations(map[string]string{
		_const.AppManagedByLabel:      _const.AppManagedByNocalhost,
		global.NocalhostCreateByLabel: global.NocalhostName,
	})
	vs.Spec.Hosts = []string{name}
	vs.Spec.Http = []*istiov1alpha3.HTTPRoute{}

	// http route
	host := fmt.Sprintf("%s.%s.%s.%s", name, devNs, "svc", DefaultClusterDomain)
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
		Name: global.NocalhostName + "-" + devNs,
		Match: []*istiov1alpha3.HTTPMatchRequest{
			{
				Name:    fmt.Sprintf("%s-%s-%s", global.NocalhostName, devNs, "tracing-header"),
				Headers: headers,
			},
		},
		Route: httpDsts,
	}
	httpRoutes = append(httpRoutes, http)

	//default http route
	defaultHost := fmt.Sprintf("%s.%s.%s.%s", name, baseNs, "svc", DefaultClusterDomain)
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
		return nil, nil, errors.WithStack(err)
	}
	return rs, http.DeepCopy(), nil
}

func addHeaderToVirtualService(rs *unstructured.Unstructured, svcName string, info *MeshDevInfo) (
	*istiov1alpha3.HTTPRoute, error) {

	rs.SetManagedFields(nil)

	if info.Header.TraceKey == "" || info.Header.TraceValue == "" {
		log.Debugf("can not find tracing header to update virtual service in the namespace %s",
			rs.GetNamespace())
	}

	log.Debugf("add the tracing header %s:%s to %s/%s",
		info.Header.TraceKey, info.Header.TraceValue, rs.GetName(), rs.GetNamespace())
	vs := &v1alpha3.VirtualService{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rs.UnstructuredContent(), vs); err != nil {
		return nil, errors.WithStack(err)
	}
	name := svcName
	routes := vs.Spec.Http
	for i := 0; i < len(routes); i++ {
		if routes[i].GetName() == name {
			routes = routes[:i+copy(routes[i:], routes[i+1:])]
			i--
		}
	}

	// add header
	host := fmt.Sprintf("%s.%s.%s.%s", name, info.MeshDevNamespace, "svc", DefaultClusterDomain)
	httpDsts := make([]*istiov1alpha3.HTTPRouteDestination, 0)
	httpDst := &istiov1alpha3.HTTPRouteDestination{Destination: &istiov1alpha3.Destination{Host: host}}
	httpDsts = append(httpDsts, httpDst)
	headers := make(map[string]*istiov1alpha3.StringMatch)
	// set exact match header
	headers[info.Header.TraceKey] = &istiov1alpha3.StringMatch{
		MatchType: &istiov1alpha3.StringMatch_Exact{
			Exact: info.Header.TraceValue,
		},
	}

	http := &istiov1alpha3.HTTPRoute{
		Name: global.NocalhostName + "-" + info.MeshDevNamespace,
		Match: []*istiov1alpha3.HTTPMatchRequest{
			{
				Name:    fmt.Sprintf("%s-%s-%s", global.NocalhostName, info.MeshDevNamespace, "tracing-header"),
				Headers: headers,
			},
		},
		Route: httpDsts,
	}
	routes = append(routes, &istiov1alpha3.HTTPRoute{})
	copy(routes[1:], routes[:])
	routes[0] = http
	vs.Spec.Http = routes

	var err error
	rs.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(vs)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return http.DeepCopy(), nil
}

func deleteHeaderFromVirtualService(rs *unstructured.Unstructured, info *MeshDevInfo) (bool, error) {
	rs.SetManagedFields(nil)

	vs := &v1alpha3.VirtualService{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rs.UnstructuredContent(), vs); err != nil {
		return false, errors.WithStack(err)
	}
	name := global.NocalhostName + "-" + info.MeshDevNamespace
	route := vs.Spec.Http
	var ok bool
	for i := 0; i < len(route); i++ {
		if route[i].GetName() == name {
			route = route[:i+copy(route[i:], route[i+1:])]
			i--
			ok = true
		}
	}
	vs.Spec.Http = route

	var err error
	rs.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(vs)
	if err != nil {
		return false, errors.WithStack(err)
	}
	return ok, nil
}

func updateHeaderToVirtualService(rs *unstructured.Unstructured, info *MeshDevInfo) (
	[]*istiov1alpha3.HTTPRoute, bool, error) {

	rs.SetManagedFields(nil)
	vs := &v1alpha3.VirtualService{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rs.UnstructuredContent(), vs); err != nil {
		return nil, false, errors.WithStack(err)
	}

	headers := make(map[string]*istiov1alpha3.StringMatch)
	// set exact match header
	headers[info.Header.TraceKey] = &istiov1alpha3.StringMatch{
		MatchType: &istiov1alpha3.StringMatch_Exact{
			Exact: info.Header.TraceValue,
		},
	}

	var updateStatus bool
	ret := make([]*istiov1alpha3.HTTPRoute, 0)
	name := fmt.Sprintf("%s-%s", global.NocalhostName, info.MeshDevNamespace)
	matchName := fmt.Sprintf("%s-%s", name, "tracing-header")
	routes := vs.Spec.Http
	for i := 0; i < len(routes); i++ {
		if routes[i].GetName() != name {
			continue
		}
		route := routes[i].DeepCopy()
		for j, match := range route.Match {
			if match.Name != matchName {
				continue
			}
			value := match.Headers[info.Header.TraceKey]
			if value == nil || value.GetExact() != info.Header.TraceValue {
				log.Debugf("update %s/%s, routes: %s, tracing header: %s",
					rs.GetKind(), rs.GetName(), route.GetName(), info.Header.TraceKey+":"+info.Header.TraceValue)
				route.Match[j] = &istiov1alpha3.HTTPMatchRequest{
					Name:    matchName,
					Headers: headers,
				}
				updateStatus = true
			}
		}
		ret = append(ret, routes[i])
		routes[i] = route
	}
	vs.Spec.Http = routes

	var err error
	rs.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(vs)
	if err != nil {
		return nil, false, errors.WithStack(err)
	}
	return ret, updateStatus, nil
}
