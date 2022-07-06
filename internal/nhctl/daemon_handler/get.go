/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/appmeta_manager"
	"nocalhost/internal/nhctl/common/base"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/daemon_handler/item"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/resouce_cache"
	"nocalhost/internal/nhctl/vpn/util"
	k8sutil "nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"sort"
	"strings"
	"sync"
	"time"
)

var svcProfileCacheMap = cache.NewLRUExpireCache(10000)

func InvalidCache(ns, nid, appName string) {
	svcProfileCacheMap.Remove(fmt.Sprintf("%s/%s/%s", ns, nid, appName))
}

func getServiceProfile(
	f func(resourceType string) (resouce_cache.GvkGvrWithAlias, error),
	ns,
	appName string,
	kubeconfigBytes []byte,
) map[string]map[string]*profile.SvcProfileV2 {
	serviceProfileMap := make(map[string]map[string]*profile.SvcProfileV2)
	if appName == "" || ns == "" {
		return serviceProfileMap
	}
	description := GetDescriptionDaemon(ns, appName, kubeconfigBytes)
	if description != nil {
		//appMeta := appmeta_manager.GetApplicationMeta(ns, appName, kubeconfigBytes)
		for _, svcProfileV2 := range description.SvcProfile {
			if svcProfileV2 != nil {
				//svcProfileV2.DevModeType = appMeta.GetCurrentDevModeTypeOfWorkload(
				//	svcProfileV2.Name, base.SvcType(svcProfileV2.Type), description.Identifier,
				//)
				name := strings.ToLower(svcProfileV2.GetType())
				if mapping, err := f(svcProfileV2.GetType()); err == nil {
					name = mapping.GetFullName()
				}
				if len(serviceProfileMap[name]) == 0 {
					serviceProfileMap[name] = map[string]*profile.SvcProfileV2{}
				}
				serviceProfileMap[name][svcProfileV2.GetName()] = svcProfileV2
				//serviceMap[name+"/"+svcProfileV2.GetName()] = svcProfileV2
			}
		}
	}
	return serviceProfileMap
}

func GetDescriptionDaemon(ns, appName string, kubeconfigBytes []byte) *profile.AppProfileV2 {
	var appProfile *profile.AppProfileV2
	// deep copy
	marshal, err := json.Marshal(appmeta_manager.GetApplicationMeta(ns, appName, kubeconfigBytes))
	if err != nil {
		return nil
	}

	var meta appmeta.ApplicationMeta
	if err = json.Unmarshal(marshal, &meta); err != nil {
		return nil
	}

	appProfileCache, found := svcProfileCacheMap.Get(fmt.Sprintf("%s/%s/%s", ns, meta.NamespaceId, appName))
	if !found || appProfileCache == nil {
		if appProfile, err = nocalhost.GetProfileV2(ns, appName, meta.NamespaceId); err == nil {
			svcProfileCacheMap.Add(fmt.Sprintf("%s/%s/%s", ns, meta.NamespaceId, appName), appProfile, time.Second*2)
		} else {
			log.Error(err)
			return nil
		}
	} else {
		appProfile = appProfileCache.(*profile.AppProfileV2)
	}

	if appProfile != nil {

		appProfile.Installed = meta.IsInstalled()
		devMeta := meta.DevMeta

		// sort svc profiles based on name to prevent:
		// in DevMode(Duplicate), check $(workload)-duplicate first and delete $(workload)-duplicate-$(id) key-value of devMeta,
		// and when checking $(workload), DevModeType is NONE, devStatus NONE, expected STARTED
		// example:
		// after $(workload) start DevMode(Duplicate)
		// app profile just have one record (from remote secret) about this workload: $(workload)-duplicate-$(id):$(id)
		// CheckIfSvcDeveloping
		// 1. check $(workload): hit, devMode duplicate, devStatus STARTED, delete $(workload) in devMeta, but not deleted because not exist this key
		// 2. check $(workload)-duplicate-$(id): hit, devMode replace, devStatus STARTED, delete $(workload)-duplicate-$(id) in devMeta, deleted
		// but if not sort, happen probably
		// 1. check $(workload)-duplicate-$(id): hit, devMode replace, devStatus STARTED, delete $(workload)-duplicate-$(id) in meta, deleted
		// 2. check $(workload): miss, devMode none, devStatus None
		// step 2 is not as expected
		// so must check $(workload) before check $(workload)-duplicate-$(id)
		sort.Slice(appProfile.SvcProfile, func(i, j int) bool {
			vi := appProfile.SvcProfile[i]
			vj := appProfile.SvcProfile[j]
			return vi.Name < vj.Name
		})

		// first iter from local svcProfile
		for _, svcProfile := range appProfile.SvcProfile {
			if svcProfile == nil {
				continue
			}
			svcProfile.DevModeType = meta.GetCurrentDevModeTypeOfWorkload(
				svcProfile.Name, base.SvcType(svcProfile.Type), appProfile.Identifier,
			)

			//if svcProfile.ServiceConfigV2 == nil {
			//	svcProfile.ServiceConfigV2 = &profile.ServiceConfigV2{
			//		Name: svcProfile.GetName(),
			//		Type: base.Deployment.String(),
			//		ContainerConfigs: []*profile.ContainerConfig{
			//			{
			//				Dev: &profile.ContainerDevConfig{
			//					Image:   profile.DefaultDevImage,
			//					WorkDir: profile.DefaultWorkDir,
			//				},
			//			},
			//		},
			//	}
			//}

			appmeta.FillingExtField(svcProfile, &meta, appName, ns, appProfile.Identifier)

			if m := devMeta[base.SvcType(svcProfile.GetType()).Alias()]; m != nil {
				delete(m, svcProfile.GetName())
			}
		}

		// then gen the fake profile for remote svc
		for svcTypeAlias, m := range devMeta {
			for svcName, _ := range m {
				if !appmeta.HasDevStartingSuffix(svcName) {
					svcProfile := appProfile.SvcProfileV2(svcName, string(svcTypeAlias.Origin()))
					appmeta.FillingExtField(svcProfile, &meta, appName, ns, appProfile.Identifier)
				}
			}
		}
		return appProfile
	}
	return nil
}

func HandleGetResourceInfoRequest(request *command.GetResourceInfoCommand) (interface{}, error) {
	KubeConfigBytes, err := ioutil.ReadFile(request.KubeConfig)
	if err != nil {
		return nil, err
	}

	ns := getNamespace(request.Namespace, KubeConfigBytes)
	switch request.Resource {
	case "all":
		s, err := resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns)
		if err != nil {
			return nil, err
		}
		if len(request.AppName) != 0 {
			nid := getNidByAppName(ns, request.KubeConfig, request.AppName)
			return item.Result{
				Namespace:   ns,
				Application: []item.App{getApp(ns, request.AppName, nid, s, request.Label, request.ShowHidden)},
			}, nil
		}
		// it's cluster kubeconfig
		if len(request.Namespace) == 0 {
			nsObjectList, err := s.Criteria().ResourceType("namespaces").Query()
			if err == nil && nsObjectList != nil && len(nsObjectList) > 0 {
				result := make([]item.Result, 0, len(nsObjectList))
				//// try to init a cluster level searcher
				//searcher, err2 := resouce_cache.GetSearcher(KubeConfigBytes, "")
				//if err2 != nil {
				//	return nil
				//}
				ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)
				defer cancelFunc()
				var wg = &sync.WaitGroup{}
				wg.Add(len(nsObjectList))
				var lock sync.Mutex
				for _, nsObject := range nsObjectList {
					go func(finalNs string) {
						app := getApplicationByNs(
							finalNs, request.KubeConfig, s, request.Label, request.ShowHidden,
						)
						lock.Lock()
						result = append(result, app)
						lock.Unlock()
						wg.Done()
					}(nsObject.(metav1.Object).GetName())
				}
				go func() {
					wg.Wait()
					cancelFunc()
				}()
				<-ctx.Done()
				return result, nil
			}
		}
		return getApplicationByNs(ns, request.KubeConfig, s, request.Label, request.ShowHidden), nil

	case "app", "application":
		// init searcher for cache async
		go resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns)
		if len(request.ResourceName) == 0 {
			return ParseApplicationsResult(ns, GetAllValidApplicationWithDefaultApp(ns, KubeConfigBytes)), nil
		} else {
			meta := appmeta_manager.GetApplicationMeta(ns, request.ResourceName, KubeConfigBytes)
			return ParseApplicationsResult(ns, []*appmeta.ApplicationMeta{meta}), nil
		}
	case "crd-list":
		s, err := resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns)
		if err != nil {
			return nil, errors.New("fail to get searcher")
		}
		data, err := s.Criteria().
			ResourceType("crds").
			Label(request.Label).
			ShowHidden(request.ShowHidden).
			Query()
		if err != nil {
			return nil, nil
		}

		type gvkr struct {
			schema.GroupVersionKind
			Resource   string
			Namespaced bool
		}

		crdGvkList := make([]*gvkr, 0)
		agrs, err := resouce_cache.GetApiGroupResources(KubeConfigBytes, ns)
		if err != nil {
			return nil, err
		}

		for _, datum := range data {
			um, ok := datum.(*unstructured.Unstructured)
			if !ok {
				continue
			}
			crd, err := resouce_cache.ConvertRuntimeObjectToCRD(um)
			if err != nil {
				log.WarnE(err, "")
				continue
			}
			gs := resouce_cache.ConvertCRDToGgwa(crd, agrs)
			for _, ggwa := range gs {
				g := gvkr{}
				g.GroupVersionKind = ggwa.Gvk
				g.Resource = ggwa.Gvr.Resource
				g.Namespaced = ggwa.Namespaced
				crdGvkList = append(crdGvkList, &g)
			}
		}

		if len(request.ResourceName) == 0 {
			result := make([]item.Item, 0, len(crdGvkList))
			for _, datum := range crdGvkList {
				result = append(result, item.Item{Metadata: datum})
			}
			return result[0:], nil
		} else {
			return item.Item{Metadata: crdGvkList[0]}, nil
		}
	case "ns", "namespace", "namespaces":
		s, err := resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns)
		if err != nil {
			return nil, errors.New("fail to get searcher")
		}
		data, err := s.Criteria().
			ResourceType(request.Resource).
			ResourceName(request.ResourceName).
			Label(request.Label).
			ShowHidden(request.ShowHidden).
			Query()
		if err != nil {
			return nil, nil
		}
		// resource namespace filter status is active
		result := make([]item.Item, 0, 0)
		for _, datum := range data {
			if um, ok := datum.(metav1.Object); ok {
				if um.GetDeletionTimestamp() == nil {
					result = append(result, item.Item{Metadata: datum})
				}
			}
		}
		// add default namespace if can't list namespace
		if len(result) == 0 {
			result = append(result, item.Item{Metadata: &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}})
		}
		for _, i := range result {
			go GetOrGenerateConfigMapWatcher(KubeConfigBytes, i.Metadata.(metav1.Object).GetName(), nil)
			if connectInfo.IsSameCluster(KubeConfigBytes) {
				i.VPN = &item.VPNInfo{
					Mode:   ConnectMode.String(),
					Status: connectInfo.Status(),
					IP:     connectInfo.getIPIfIsMe(KubeConfigBytes, ns),
				}
			}
		}
		if len(request.ResourceName) == 0 {
			return result[0:], nil
		} else {
			return result[0], nil
		}

	default:
		request.ClientStack = ""
		s, err := resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns)
		if err != nil {
			return nil, err
		}
		serviceMap := make(map[string]map[string]*profile.SvcProfileV2)
		if len(request.AppName) != 0 {
			//nid := getNidByAppName(ns, request.KubeConfig, request.AppName)
			serviceMap = getServiceProfile(s.GetResourceInfo, ns, request.AppName, KubeConfigBytes)
		}
		var belongsToMe = NewSet()
		var reverseReversed = sets.NewString()
		if load, ok := GetReverseInfo().Load(util.GenerateKey(KubeConfigBytes, ns)); ok {
			belongsToMe.Insert(load.(*status).reverse.GetBelongToMeResources().List()...)
			reverseReversed.Insert(load.(*status).reverse.ReversedResource().List()...)
		}

		items, err := s.Criteria().
			ResourceType(request.Resource).
			ResourceName(request.ResourceName).
			AppName(request.AppName).
			Namespace(ns).
			ShowHidden(request.ShowHidden).
			Label(request.Label).
			Query()

		if err != nil {
			return nil, err
		}
		//mapping, err := s.GetResourceInfo(request.Resource)
		result := make([]item.Item, 0, len(items))
		for _, i := range items {
			tempItem := item.Item{Metadata: i}
			if mapping, err := s.GetResourceInfo(request.Resource); err == nil {
				//var tt string
				//if nocalhost.IsBuildInGvk(&mapping.Gvk) {
				//	tt = strings.ToLower(mapping.Gvk.Kind)
				//} else {
				//	tt := fmt.Sprintf("%s.%s.%s", mapping.Gvr.Resource, mapping.Gvr.Version, mapping.Gvr.Group)
				//}
				tempItem.Description = &profile.SvcProfileV2{
					DevPortForwardList: make([]*profile.DevPortForward, 0),
				}
				if tm, ok := serviceMap[mapping.GetFullName()]; ok {
					if d, ok := tm[i.(metav1.Object).GetName()]; ok {
						tempItem.Description = d
					}
				}

				n := fmt.Sprintf(
					"%s.%s.%s/%s",
					mapping.Gvr.Resource, mapping.Gvr.Version, mapping.Gvr.Group, i.(metav1.Object).GetName(),
				)
				if revering := belongsToMe.HasKey(n) || reverseReversed.Has(n); revering {
					tempItem.VPN = &item.VPNInfo{
						Status:      belongsToMe.Get(n).status(),
						Mode:        ReverseMode.String(),
						BelongsToMe: belongsToMe.HasKey(n),
						IP:          connectInfo.getIPIfIsMe(KubeConfigBytes, ns),
					}
				}
			}
			result = append(result, tempItem)
		}
		// get all resource in namespace
		if len(request.ResourceName) == 0 {
			return result, nil
		} else if len(result) > 0 {
			return result[0], nil
		}
		return result, nil
	}
}

func getNamespace(namespace string, kubeconfigBytes []byte) (ns string) {
	if len(namespace) != 0 {
		ns = namespace
		return
	}
	if config, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes); err == nil {
		ns, _, _ = config.Namespace()
		return
	}
	return
}

func getApplicationByNs(namespace, kubeconfigPath string, search *resouce_cache.Searcher, label map[string]string, showHidden bool) item.Result {
	result := item.Result{Namespace: namespace}
	nameAndNidList := GetAllApplicationWithDefaultApp(namespace, kubeconfigPath)
	okChan := make(chan struct{}, 2)
	go func() {
		time.Sleep(time.Second * 10)
		okChan <- struct{}{}
	}()
	var wg sync.WaitGroup
	var lock sync.Mutex
	for _, name := range nameAndNidList {
		wg.Add(1)
		go func(finalName, nid string) {
			app := getApp(namespace, finalName, nid, search, label, showHidden)
			lock.Lock()
			result.Application = append(result.Application, app)
			lock.Unlock()
			wg.Done()
		}(name.Application, name.NamespaceId)
	}
	go func() {
		wg.Wait()
		okChan <- struct{}{}
	}()
	<-okChan
	return result
}

func getApp(namespace, appName, nid string, search *resouce_cache.Searcher, label map[string]string, showHidden bool) item.App {
	result := item.App{Name: appName}
	//profileMap := getServiceProfile(namespace, appName, nid, search.GetKubeconfigBytes())
	//for _, entry := range resouce_cache.GroupToTypeMap {
	resources := make([]item.Resource, 0)
	for _, alias := range search.SupportSchemaList {
		//resource := strings.Join([]string{alias.Gvr.Resource, alias.Gvr.Version, alias.Gvr.Group}, ".")
		resource := alias.GetFullName()
		resourceList, err := search.Criteria().
			ResourceType(resource).
			AppName(appName).
			Namespace(namespace).
			Label(label).
			ShowHidden(showHidden).
			Query()
		if err == nil {
			items := make([]item.Item, 0, len(resourceList))
			for _, v := range resourceList {
				items = append(
					items, item.Item{
						//Metadata: v, Description: profileMap[resource+"/"+v.(metav1.Object).GetName()],
						Metadata: v,
					},
				)
			}
			resources = append(resources, item.Resource{Name: resource, List: items})
		}
	}
	result.Groups = []item.Group{{GroupName: "", List: resources}}
	return result
}

func SortApplication(metas []*appmeta.ApplicationMeta) {
	if metas == nil {
		return
	}
	sort.SliceStable(
		metas, func(i, j int) bool {
			var n1, n2 string
			if metas[i] != nil {
				n1 = metas[i].Application
			}
			if metas[j] != nil {
				n2 = metas[j].Application
			}
			if n1 < n2 {
				return false
			}
			return true
		},
	)
}

func ParseApplicationsResult(namespace string, metas []*appmeta.ApplicationMeta) []*model.Namespace {
	var result []*model.Namespace
	ns := &model.Namespace{
		Namespace:   namespace,
		Application: []*model.ApplicationInfo{},
	}
	for _, meta := range metas {
		ns.Application = append(
			ns.Application, &model.ApplicationInfo{
				Name: meta.Application,
				Type: string(meta.ApplicationType),
			},
		)
	}

	sort.Slice(
		ns.Application, func(i, j int) bool {
			return ns.Application[i].Name > ns.Application[j].Name
		},
	)

	result = append(result, ns)
	return result
}

type AppNameAndNid struct {
	Name string
	Nid  string
}

func getNidByAppName(namespace, kubeconfig, appName string) string {
	kubeconfigBytes, _ := ioutil.ReadFile(kubeconfig)
	meta := appmeta_manager.GetApplicationMeta(namespace, appName, kubeconfigBytes)
	return meta.NamespaceId
}

func GetAllValidApplicationWithDefaultApp(ns string, KubeConfigBytes []byte) []*appmeta.ApplicationMeta {
	result := make(map[string]*appmeta.ApplicationMeta, 0)

	// first get apps from secret
	// then get apps from annotations
	// merge then
	//
	// the result should contains all meta from secret except 'uninstall'
	// the result from annotations should escape the meta is 'protected'
	appFromMeta := GetAllApplicationWithDefaultApp(ns, k8sutil.GetOrGenKubeConfigPath(string(KubeConfigBytes)))
	appFromMetaMapping := make(map[string]*appmeta.ApplicationMeta, 0)
	for _, meta := range appFromMeta {
		appFromMetaMapping[meta.Application] = meta
	}

	appFromAnno := resouce_cache.GetAllAppNameByNamespace(KubeConfigBytes, ns).UnsortedList()

	// first add all meta from anno,
	// but if meta from annotations is protected, escape it
	for _, s := range appFromAnno {
		if meta, ok := appFromMetaMapping[s]; ok && meta.ProtectedFromReInstall() {
			continue
		}

		result[s] = &appmeta.ApplicationMeta{ApplicationType: appmeta.ManifestLocal, Application: s}
	}

	// then if meta from secret is installed, put it into result
	for _, meta := range appFromMeta {
		if meta.IsInstalled() || meta.IsInstalling() {
			result[meta.Application] = meta
		}
	}

	displayedMeta := make([]*appmeta.ApplicationMeta, 0)
	for _, meta := range result {
		displayedMeta = append(displayedMeta, meta)
	}

	sort.Slice(
		displayedMeta, func(i, j int) bool {
			return displayedMeta[i].Application > displayedMeta[j].Application
		},
	)
	return displayedMeta
}

// GetAllApplicationWithDefaultApp will not to create default application if default application not found
// note: this func will return all app meta, includes invalid application(uninstalled)
func GetAllApplicationWithDefaultApp(namespace, kubeconfigPath string) []*appmeta.ApplicationMeta {
	kubeconfigBytes, _ := ioutil.ReadFile(kubeconfigPath)
	applicationMetaList := appmeta_manager.GetApplicationMetas(
		namespace, kubeconfigBytes,
		func(meta *appmeta.ApplicationMeta) bool {
			return true
		},
	)

	var foundDefaultApp bool
	for _, meta := range applicationMetaList {
		if meta.Application == _const.DefaultNocalhostApplication {
			foundDefaultApp = true
		}
	}
	if !foundDefaultApp {
		applicationMetaList = append(
			applicationMetaList, appmeta.FakeAppMeta(namespace, _const.DefaultNocalhostApplication),
		)
	}
	SortApplication(applicationMetaList)
	return applicationMetaList
}
