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
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"

	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

const (
	NotInstalled = iota
	ShouldBeInstalled
	Installed
	ShouldBeDeleted

	Unselected = NotInstalled
	Selected   = ShouldBeInstalled
)

type MeshManager interface {
	InitMeshDevSpace(*MeshDevInfo) error
	UpdateMeshDevSpace(*MeshDevInfo) error
	InjectMeshDevSpace(*MeshDevInfo) error
	GetBaseDevSpaceAppInfo(*MeshDevInfo) []MeshDevApp
	GetAPPInfo(*MeshDevInfo) ([]MeshDevApp, error)
	RefreshCache() error
}

type meshManager struct {
	mu     sync.Mutex
	client *clientgo.GoClient
	cache  cache
	//meshDevInfo MeshDevInfo
}

type MeshDevInfo struct {
	BaseNamespace    string       `json:"-"`
	MeshDevNamespace string       `json:"-"`
	Header           model.Header `json:"header"`
	APPS             []MeshDevApp `json:"apps"`
}

type MeshDevApp struct {
	Name      string            `json:"name"`
	Workloads []MeshDevWorkload `json:"workloads"`
}

type MeshDevWorkload struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}

func (m *meshManager) InitMeshDevSpace(info *MeshDevInfo) error {
	return m.initMeshDevSpace(info)
}

func (m *meshManager) UpdateMeshDevSpace(info *MeshDevInfo) error {
	if err := m.setWorkloadStatus(info); err != nil {
		return err
	}
	return m.InjectMeshDevSpace(info)
}

func (m *meshManager) InjectMeshDevSpace(info *MeshDevInfo) error {
	// get dev space workloads from cache
	ws := make(map[string]int)
	for _, a := range info.APPS {
		for _, w := range a.Workloads {
			ws[w.Kind+"/"+w.Name] = w.Status
		}
	}
	irs := make([]unstructured.Unstructured, 0)
	drs := make([]unstructured.Unstructured, 0)
	for _, r := range m.cache.GetDeploymentsListByNameSpace(info.BaseNamespace) {
		if ws[r.GetKind()+"/"+r.GetName()] == ShouldBeInstalled {
			irs = append(irs, r)
			continue
		}
		if ws[r.GetKind()+"/"+r.GetName()] == ShouldBeDeleted {
			drs = append(drs, r)
		}
	}

	g, _ := errgroup.WithContext(context.Background())
	// apply workloads
	g.Go(func() error {
		return m.applyWorkloadsToMeshDevSpace(irs, info)
	})

	// update base dev space vs
	g.Go(func() error {
		return m.updateVirtualserviceOnBaseDevSpace(irs, drs, info)
	})

	// delete workloads
	g.Go(func() error {
		return m.deleteWorkloadsFromMeshDevSpace(drs, info)
	})
	return g.Wait()
}

func (m *meshManager) GetBaseDevSpaceAppInfo(info *MeshDevInfo) []MeshDevApp {
	appNames := make([]string, 0)
	appInfo := make([]MeshDevApp, 0)
	appConfigsTmp := newAppMatcher(m.cache.GetSecretsListByNameSpace(info.BaseNamespace)).
		namePrefix(appmeta.SecretNamePrefix).
		match()
	for _, c := range appConfigsTmp {
		name := c.GetName()[len(appmeta.SecretNamePrefix):]
		if name == nocalhost.DefaultNocalhostApplication {
			continue
		}

		val, found, err := unstructured.NestedString(c.UnstructuredContent(), "type")
		if !found || err != nil {
			continue
		}
		if val != appmeta.SecretType {
			continue
		}

		appNames = append(appNames, name)
		w := make([]MeshDevWorkload, 0)
		for _, r := range newAppMatcher(m.cache.GetDeploymentsListByNameSpace(info.BaseNamespace)).app(name).match() {
			w = append(w, MeshDevWorkload{
				Kind: r.GetKind(),
				Name: r.GetName(),
			})
		}
		appInfo = append(appInfo, MeshDevApp{
			Name:      name,
			Workloads: w,
		})
	}

	// default.application
	w := make([]MeshDevWorkload, 0)
	for _, r := range newAppMatcher(m.cache.GetDeploymentsListByNameSpace(info.BaseNamespace)).
		excludeApps(appNames).
		match() {
		w = append(w, MeshDevWorkload{
			Kind: r.GetKind(),
			Name: r.GetName(),
		})
	}
	appInfo = append(appInfo, MeshDevApp{
		Name:      nocalhost.DefaultNocalhostApplication,
		Workloads: w,
	})

	return appInfo
}

func (m *meshManager) GetAPPInfo(info *MeshDevInfo) ([]MeshDevApp, error) {
	status := make(map[string]struct{})
	for _, r := range m.cache.GetDeploymentsListByNameSpace(info.MeshDevNamespace) {
		status[r.GetKind()+"/"+r.GetName()] = struct{}{}
	}

	apps := m.GetBaseDevSpaceAppInfo(info)
	for i, a := range apps {
		for j, w := range a.Workloads {
			if _, ok := status[w.Kind+"/"+w.Name]; ok {
				apps[i].Workloads[j].Status = Selected
			}
		}
	}
	return apps, nil
}

func (m *meshManager) RefreshCache() error {
	return m.buildCache()
}

func (m *meshManager) deleteWorkloadsFromMeshDevSpace(drs []unstructured.Unstructured, info *MeshDevInfo) error {
	for _, r := range drs {
		log.Debugf("delete the workload %s/%s from %s", r.GetKind(), r.GetName(), info.MeshDevNamespace)
		if err := meshDevModify(info.MeshDevNamespace, &r); err != nil {
			return err
		}
		err := m.client.Delete(&r)
		if err != nil {
			log.Errorf("%+v", err)
			continue
		}
		vs, err := genVirtualServiceForMeshDevSpace(info.BaseNamespace, r)
		if err != nil {
			log.Errorf("%+v", err)
			continue
		}
		log.Debugf("apply the Virtualservice/%s to the base namespace %s", r.GetName(), r.GetNamespace())
		if _, err := m.client.Apply(vs); err != nil {
			log.Errorf("%+v", err)
		}
	}
	return nil
}

func (m *meshManager) applyWorkloadsToMeshDevSpace(irs []unstructured.Unstructured, info *MeshDevInfo) error {
	for _, r := range irs {
		log.Debugf("inject the workload %s/%s to %s", r.GetKind(), r.GetName(), info.MeshDevNamespace)
		if err := meshDevModify(info.MeshDevNamespace, &r); err != nil {
			return err
		}
		if _, err := m.client.Apply(&r); err != nil {
			return err
		}
		// delete vs form mesh dev space
		vs := &unstructured.Unstructured{}
		vs.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "networking.istio.io",
			Version: "v1alpha3",
			Kind:    "VirtualService",
		})
		vs.SetNamespace(r.GetNamespace())
		vs.SetName(r.GetName())

		log.Debugf("delete the Virtualservice/%s from the base namespace %s", r.GetName(), r.GetNamespace())
		err := m.client.Delete(vs)
		if err != nil {
			log.Debug(err)
		}
	}
	return nil
}

func (m *meshManager) updateVirtualserviceOnBaseDevSpace(irs, drs []unstructured.Unstructured, info *MeshDevInfo) error {
	// TODO, create vs by service name, not workload name
	// TODO, just update if the vs already exists
	if info.Header.TraceKey != "" || info.Header.TraceValue != "" {
		for _, r := range irs {
			vs, err := genVirtualServiceForBaseDevSpace(
				info.BaseNamespace,
				info.MeshDevNamespace,
				r.GetName(),
				info.Header,
			)
			if err != nil {
				return err
			}
			log.Debugf("apply the Virtualservice/%s to the base namespace %s", r.GetName(), info.BaseNamespace)
			_, err = m.client.Apply(vs)
			if err != nil {
				return err
			}
		}
	}

	// TODO just delete the header match form vs
	for _, r := range drs {
		vs := &unstructured.Unstructured{}
		vs.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "networking.istio.io",
			Version: "v1alpha3",
			Kind:    "VirtualService",
		})
		vs.SetNamespace(r.GetNamespace())
		vs.SetName(r.GetName())

		log.Debugf("delete the Virtualservice/%s from the base namespace %s", r.GetName(), info.BaseNamespace)
		err := m.client.Delete(vs)
		if err != nil {
			log.Debug(err)
		}
	}
	return nil
}

func (m *meshManager) initMeshDevSpace(info *MeshDevInfo) error {
	// apply app config
	appConfigsTmp := newAppMatcher(m.cache.GetSecretsListByNameSpace(info.BaseNamespace)).
		namePrefix(appmeta.SecretNamePrefix).
		match()
	for _, c := range appConfigsTmp {
		name := c.GetName()[len(appmeta.SecretNamePrefix):]
		if name == nocalhost.DefaultNocalhostApplication {
			continue
		}
		val, found, err := unstructured.NestedString(c.UnstructuredContent(), "type")
		if !found || err != nil {
			continue
		}
		if val != appmeta.SecretType {
			continue
		}

		if err := meshDevModify(info.MeshDevNamespace, &c); err != nil {
			return err
		}
		_, err = m.client.Apply(&c)
		if err != nil {
			return err

		}
	}
	// get svc, gen vs
	svcs := m.cache.GetServicesListByNameSpace(info.BaseNamespace)
	vss := make([]v1alpha3.VirtualService, len(svcs))
	for i := range svcs {
		if err := meshDevModify(info.MeshDevNamespace, &svcs[i]); err != nil {
			return err
		}
		vs, err := genVirtualServiceForMeshDevSpace(info.BaseNamespace, svcs[i])
		if err != nil {
			return err
		}
		vss[i] = *vs
	}

	// apply svc and vs
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		for _, svc := range svcs {
			_, err := m.client.Apply(&svc)
			if err != nil {
				return err
			}
		}
		return nil
	})

	g.Go(func() error {
		for _, vs := range vss {
			_, err := m.client.Apply(&vs)
			if err != nil {
				return err
			}

		}
		return nil
	})

	return g.Wait()
}

func (m *meshManager) setInformerFactory() {
	m.cache.informers = dynamicinformer.NewDynamicSharedInformerFactory(m.client.DynamicClient, time.Minute)
}

func (m *meshManager) buildCache() error {
	m.cache.build()
	return nil
}
func (m *meshManager) getMeshDevSpaceWorkloads(info *MeshDevInfo) []MeshDevWorkload {
	w := make([]MeshDevWorkload, 0)
	for _, r := range m.cache.GetDeploymentsListByNameSpace(info.MeshDevNamespace) {
		w = append(w, MeshDevWorkload{
			Kind:   r.GetKind(),
			Name:   r.GetName(),
			Status: Installed,
		})
	}
	return w
}

func (m *meshManager) setWorkloadStatus(info *MeshDevInfo) error {
	log.Debug("set workloads status")
	devWs := m.getMeshDevSpaceWorkloads(info)
	devMap := make(map[string]MeshDevWorkload)
	for _, w := range devWs {
		devMap[w.Kind+"/"+w.Name] = w
	}
	apps := info.APPS
	for i, a := range apps {
		for j, w := range a.Workloads {
			if w.Status == Selected && devMap[w.Kind+"/"+w.Name].Status == Installed {
				apps[i].Workloads[j].Status = Installed
			}
			if w.Status == Unselected && devMap[w.Kind+"/"+w.Name].Status == Installed {
				apps[i].Workloads[j].Status = ShouldBeDeleted
			}
		}
	}
	info.APPS = apps
	return nil

}

func NewMeshManager(client *clientgo.GoClient) MeshManager {
	m := &meshManager{}
	m.client = client
	m.setInformerFactory()
	return m
}
