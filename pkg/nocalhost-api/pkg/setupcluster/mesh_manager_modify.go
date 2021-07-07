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
	"nocalhost/internal/nocalhost-api/model"

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
)

func meshDevModifier(ns string, r *unstructured.Unstructured) error {
	switch r.GetKind() {
	case "Deployment":
		if err := deploymentModifier(r); err != nil {
			return err
		}
	case "Service":
		if err := serviceModifier(r); err != nil {
			return err
		}
	}

	return commonModifier(ns, r)
}

func deploymentModifier(rs *unstructured.Unstructured) error {
	dep := &appsv1.Deployment{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rs.UnstructuredContent(), dep); err != nil {
		return errors.WithStack(err)
	}
	dep.Status.Reset()
	var err error
	if rs.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(dep); err != nil {
		return errors.WithStack(err)
	}
	return nil
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
	rs.SetGenerateName("")
	rs.SetSelfLink("")
	rs.SetUID("")
	rs.SetResourceVersion("")
	rs.SetGeneration(0)
	//rs.SetCreationTimestamp(metav1.Time{})
	//rs.SetDeletionTimestamp(&metav1.Time{})
	rs.SetDeletionGracePeriodSeconds(nil)
	rs.SetOwnerReferences(nil)
	rs.SetFinalizers(nil)
	rs.SetManagedFields(nil)

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

func genVirtualServiceForMeshDevSpace(baseNs string, r unstructured.Unstructured) (*v1alpha3.VirtualService, error) {
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
	vs.SetLabels(r.GetLabels())
	vs.SetAnnotations(r.GetAnnotations())
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

	return vs, nil
}

func genVirtualServiceForBaseDevSpace(baseNs, devNs, name string, header model.Header) (*v1alpha3.VirtualService, error) {
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
		Name: global.NocalhostDevNamespaceLabel + "-" + name,
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

	return vs, nil
}
