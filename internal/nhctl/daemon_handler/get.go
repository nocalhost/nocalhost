/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/cache"
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
	"nocalhost/pkg/nhctl/log"
	"sort"
	"strings"
	"sync"
	"time"
)

var svcProfileCacheMap = cache.NewLRUExpireCache(10000)

func getServiceProfile(ns, appName, nid string, kubeconfigBytes []byte) map[string]*profile.SvcProfileV2 {
	serviceMap := make(map[string]*profile.SvcProfileV2)
	if appName == "" || ns == "" {
		return serviceMap
	}
	description := GetDescriptionDaemon(ns, appName, nid, kubeconfigBytes)
	if description != nil {
		appMeta := appmeta_manager.GetApplicationMeta(ns, appName, kubeconfigBytes)
		for _, svcProfileV2 := range description.SvcProfile {
			if svcProfileV2 != nil {
				svcProfileV2.DevModeType = appMeta.GetCurrentDevModeTypeOfWorkload(svcProfileV2.Name, base.SvcTypeOf(svcProfileV2.Type), description.Identifier)
				name := strings.ToLower(svcProfileV2.GetType()) + "s"
				serviceMap[name+"/"+svcProfileV2.GetName()] = svcProfileV2
			}
		}
	}
	return serviceMap
}

func GetDescriptionDaemon(ns, appName, nid string, kubeconfigBytes []byte) *profile.AppProfileV2 {
	var appProfile *profile.AppProfileV2
	var err error
	appProfileCache, found := svcProfileCacheMap.Get(fmt.Sprintf("%s/%s/%s", ns, nid, appName))
	if !found || appProfileCache == nil {
		if appProfile, err = nocalhost.GetProfileV2(ns, appName, nid); err == nil {
			svcProfileCacheMap.Add(fmt.Sprintf("%s/%s/%s", ns, nid, appName), appProfile, time.Second*2)
		}
	} else {
		appProfile = appProfileCache.(*profile.AppProfileV2)
	}
	if err != nil {
		log.Error(err)
		return nil
	}
	if appProfile != nil {
		// deep copy
		marshal, err := json.Marshal(appmeta_manager.GetApplicationMeta(ns, appName, kubeconfigBytes))
		if err != nil {
			return nil
		}

		var meta appmeta.ApplicationMeta
		if err = json.Unmarshal(marshal, &meta); err != nil {
			return nil
		}

		appProfile.Installed = meta.IsInstalled()
		devMeta := meta.DevMeta

		// first iter from local svcProfile
		for _, svcProfile := range appProfile.SvcProfile {
			if svcProfile == nil {
				continue
			}
			if svcProfile.ServiceConfigV2 == nil {
				svcProfile.ServiceConfigV2 = &profile.ServiceConfigV2{
					Name: svcProfile.GetName(),
					Type: base.Deployment.String(),
					ContainerConfigs: []*profile.ContainerConfig{
						{
							Dev: &profile.ContainerDevConfig{
								Image:   profile.DefaultDevImage,
								WorkDir: profile.DefaultWorkDir,
							},
						},
					},
				}
			}

			appmeta.FillingExtField(svcProfile, &meta, appName, ns, appProfile.Identifier)

			if m := devMeta[base.SvcTypeOf(svcProfile.GetType()).Alias()]; m != nil {
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

func HandleGetResourceInfoRequest(request *command.GetResourceInfoCommand) interface{} {
	KubeConfigBytes, _ := ioutil.ReadFile(request.KubeConfig)
	ns := getNamespace(request.Namespace, KubeConfigBytes)
	switch request.Resource {
	case "all":
		s, err := resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns)
		if err != nil {
			return nil
		}
		if len(request.AppName) != 0 {
			availableAppName := getAvailableAppName(ns, request.KubeConfig)
			nid := ""
			nameList := make([]string, 0)
			for _, aa := range availableAppName {
				nameList = append(nameList, aa.Name)
				if aa.Name == request.AppName {
					nid = aa.Nid
				}
				break
			}

			return item.Result{
				Namespace:   ns,
				Application: []item.App{getApp(nameList, ns, request.AppName, nid, s, request.Label)},
			}
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
				okChan := make(chan struct{}, 2)
				go func() {
					time.Sleep(time.Second * 10)
					okChan <- struct{}{}
				}()
				var wg sync.WaitGroup
				var lock sync.Mutex
				for _, nsObject := range nsObjectList {
					wg.Add(1)
					go func(finalNs metav1.Object) {
						app := getApplicationByNs(finalNs.GetName(), request.KubeConfig, s, request.Label)
						lock.Lock()
						result = append(result, app)
						lock.Unlock()
						wg.Done()
					}(nsObject.(metav1.Object))
				}
				go func() {
					wg.Wait()
					okChan <- struct{}{}
				}()
				<-okChan
				return result
			}
		}
		return getApplicationByNs(ns, request.KubeConfig, s, request.Label)

	case "app", "application":
		// init searcher for cache async
		go func() { _, _ = resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns) }()
		if request.ResourceName == "" {
			return ParseApplicationsResult(ns, GetAllApplicationWithDefaultApp(ns, request.KubeConfig))
		} else {
			meta := appmeta_manager.GetApplicationMeta(ns, request.ResourceName, KubeConfigBytes)
			return ParseApplicationsResult(ns, []*appmeta.ApplicationMeta{meta})
		}

	case "ns", "namespace", "namespaces":
		s, err := resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns)
		if err != nil {
			return nil
		}
		data, err := s.Criteria().
			ResourceType(request.Resource).
			ResourceName(request.ResourceName).
			Label(request.Label).
			Query()
		// resource namespace filter status is active
		availableData := make([]interface{}, 0, 0)
		for _, datum := range data {
			if datum.(*v1.Namespace).Status.Phase == v1.NamespaceActive {
				availableData = append(availableData, datum)
			}
		}
		if err != nil || len(availableData) == 0 {
			return nil
		}
		if len(request.ResourceName) == 0 {
			result := make([]item.Item, 0, len(availableData))
			for _, datum := range availableData {
				result = append(result, item.Item{Metadata: datum})
			}
			return result[0:]
		} else {
			return item.Item{Metadata: availableData[0]}
		}

	default:
		s, err := resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns)
		if err != nil {
			return nil
		}
		serviceMap := make(map[string]*profile.SvcProfileV2)
		appNameList := make([]string, 0, 0)
		if len(request.AppName) != 0 {
			appNameAndNidList := getAvailableAppName(ns, request.KubeConfig)
			nid := ""
			for _, an := range appNameAndNidList {
				appNameList = append(appNameList, an.Name)
				if an.Name == request.AppName {
					nid = an.Nid
				}
			}
			serviceMap = getServiceProfile(ns, request.AppName, nid, KubeConfigBytes)
		}

		c := s.Criteria().
			ResourceType(request.Resource).
			ResourceName(request.ResourceName).
			AppName(request.AppName).
			AppNameNotIn(appNameList...).
			Namespace(ns).
			Label(request.Label)
		// get all resource in namespace
		if len(request.ResourceName) == 0 {
			items, err := c.Query()
			if err != nil || len(items) == 0 {
				return nil
			}
			result := make([]item.Item, 0, len(items))
			for _, i := range items {
				tempItem := item.Item{Metadata: i}
				if mapping, err := s.GetResourceInfo(request.Resource); err == nil {
					tempItem.Description = serviceMap[mapping.Gvr.Resource+"/"+i.(metav1.Object).GetName()]
				}
				result = append(result, tempItem)
			}
			return result
		} else {
			// get specify resource name in namespace
			one, err := c.QueryOne()
			if err != nil || one == nil {
				return nil
			}
			i := item.Item{Metadata: one}
			if mapping, err := s.GetResourceInfo(request.Resource); err == nil {
				i.Description = serviceMap[mapping.Gvr.Resource+"/"+one.(metav1.Object).GetName()]
			}
			return i
		}
	}
}

func getNamespace(namespace string, kubeconfigBytes []byte) (ns string) {
	if namespace != "" {
		ns = namespace
		return
	}
	config, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err == nil && config != nil {
		ns, _, _ = config.Namespace()
		return ns
	}
	return ""
}

func getApplicationByNs(namespace, kubeconfigPath string, search *resouce_cache.Searcher, label map[string]string) item.Result {
	result := item.Result{Namespace: namespace}
	nameAndNidList := getAvailableAppName(namespace, kubeconfigPath)
	nameList := make([]string, 0)
	for _, n := range nameAndNidList {
		nameList = append(nameList, n.Name)
	}
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
			app := getApp(nameList, namespace, finalName, nid, search, label)
			lock.Lock()
			result.Application = append(result.Application, app)
			lock.Unlock()
			wg.Done()
		}(name.Name, name.Nid)
	}
	go func() {
		wg.Wait()
		okChan <- struct{}{}
	}()
	<-okChan
	return result
}

func getApp(name []string, namespace, appName, nid string, search *resouce_cache.Searcher, label map[string]string) item.App {
	result := item.App{Name: appName}
	profileMap := getServiceProfile(namespace, appName, nid, search.GetKubeconfigBytes())
	for _, entry := range resouce_cache.GroupToTypeMap {
		resources := make([]item.Resource, 0, len(entry.V))
		for _, resource := range entry.V {
			resourceList, err := search.Criteria().
				ResourceType(resource).
				AppName(appName).
				AppNameNotIn(name...).
				Namespace(namespace).
				Label(label).
				Query()
			if err == nil {
				items := make([]item.Item, 0, len(resourceList))
				for _, v := range resourceList {
					items = append(
						items, item.Item{
							Metadata: v, Description: profileMap[resource+"/"+v.(metav1.Object).GetName()],
						},
					)
				}
				resources = append(resources, item.Resource{Name: resource, List: items})
			}
		}
		result.Groups = append(result.Groups, item.Group{GroupName: entry.K, List: resources})
	}
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

func getAvailableAppName(namespace, kubeconfig string) []AppNameAndNid {
	applicationMetaList := GetAllApplicationWithDefaultApp(namespace, kubeconfig)
	var availableAppName []AppNameAndNid
	for _, meta := range applicationMetaList {
		if meta != nil {
			availableAppName = append(availableAppName, AppNameAndNid{Name: meta.Application, Nid: meta.NamespaceId})
		}
	}
	sort.SliceStable(availableAppName, func(i, j int) bool { return availableAppName[i].Name < availableAppName[j].Name })
	return availableAppName
}

// GetAllApplicationWithDefaultApp will not to create default application if default application not found
func GetAllApplicationWithDefaultApp(namespace, kubeconfigPath string) []*appmeta.ApplicationMeta {
	kubeconfigBytes, _ := ioutil.ReadFile(kubeconfigPath)
	applicationMetaList := appmeta_manager.GetApplicationMetas(namespace, kubeconfigBytes)
	var foundDefaultApp bool
	for _, meta := range applicationMetaList {
		if meta.Application == _const.DefaultNocalhostApplication {
			foundDefaultApp = true
			break
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
