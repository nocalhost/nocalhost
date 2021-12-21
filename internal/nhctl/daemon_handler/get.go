/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	k8sutil "nocalhost/pkg/nhctl/k8sutils"
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
				svcProfileV2.DevModeType = appMeta.GetCurrentDevModeTypeOfWorkload(
					svcProfileV2.Name, base.SvcTypeOf(svcProfileV2.Type), description.Identifier,
				)
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
			nid := getNidByAppName(ns, request.KubeConfig, request.AppName)
			return item.Result{
				Namespace:   ns,
				Application: []item.App{getApp(ns, request.AppName, nid, s, request.Label, request.ShowHidden)},
			}
		}
		// it's cluster kubeconfig
		if len(request.Namespace) == 0 {
			nsObjectList, err := s.Criteria().ResourceType("namespaces").Query()
			if err == nil && nsObjectList != nil && len(nsObjectList) > 0 {
				result := make([]item.Result, 0, len(nsObjectList))
				ctx, cancelFunc := context.WithTimeout(context.TODO(), time.Second*10)
				wg := &sync.WaitGroup{}
				lock := &sync.Mutex{}
				for _, nsObject := range nsObjectList {
					wg.Add(1)
					go func(finalNs string) {
						defer wg.Done()
						app := getApplicationByNs(finalNs, request.KubeConfig, s, request.Label, request.ShowHidden)
						lock.Lock()
						result = append(result, app)
						lock.Unlock()
					}(nsObject.(metav1.Object).GetName())
				}
				go func() {
					wg.Wait()
					cancelFunc()
				}()
				<-ctx.Done()
				return result
			}
		}
		return getApplicationByNs(ns, request.KubeConfig, s, request.Label, request.ShowHidden)

	case "app", "application":
		// init searcher for cache async
		go resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns)
		if len(request.ResourceName) == 0 {
			return ParseApplicationsResult(ns, GetAllValidApplicationWithDefaultApp(ns, KubeConfigBytes))
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
			ShowHidden(request.ShowHidden).
			Query()
		if err != nil || len(data) == 0 {
			return nil
		}
		// resource namespace filter status is active
		result := make([]item.Item, 0, 0)
		for _, datum := range data {
			if datum.(metav1.Object).GetDeletionTimestamp() == nil {
				result = append(result, item.Item{Metadata: datum})
			}
		}
		if err != nil || len(result) == 0 {
			return nil
		}
		if len(request.ResourceName) == 0 {
			return result[0:]
		} else {
			return result[0]
		}

	default:
		s, err := resouce_cache.GetSearcherWithLRU(KubeConfigBytes, ns)
		if err != nil {
			return nil
		}
		serviceMap := make(map[string]*profile.SvcProfileV2)
		if len(request.AppName) != 0 {
			nid := getNidByAppName(ns, request.KubeConfig, request.AppName)
			serviceMap = getServiceProfile(ns, request.AppName, nid, KubeConfigBytes)
		}

		items, err := s.Criteria().
			ResourceType(request.Resource).
			ResourceName(request.ResourceName).
			AppName(request.AppName).
			Namespace(ns).
			ShowHidden(request.ShowHidden).
			Label(request.Label).
			Query()
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
		// get all resource in namespace
		if len(request.ResourceName) == 0 {
			return result
		} else {
			return result[0]
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

func getApplicationByNs(namespace, kubeconfigPath string, search *resouce_cache.Searcher, label map[string]string, showHidden bool) item.Result {
	result := item.Result{Namespace: namespace}
	nameAndNidList := GetAllApplicationWithDefaultApp(namespace, kubeconfigPath)
	ctx, cancelFunc := context.WithTimeout(context.TODO(), time.Second*10)

	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	for _, name := range nameAndNidList {
		wg.Add(1)
		go func(finalName, nid string) {
			defer wg.Done()
			app := getApp(namespace, finalName, nid, search, label, showHidden)
			lock.Lock()
			result.Application = append(result.Application, app)
			lock.Unlock()
		}(name.Application, name.NamespaceId)
	}
	go func() {
		wg.Wait()
		cancelFunc()
	}()
	<-ctx.Done()
	return result
}

func getApp(namespace, appName, nid string, search *resouce_cache.Searcher, label map[string]string, showHidden bool) item.App {
	result := item.App{Name: appName}
	profileMap := getServiceProfile(namespace, appName, nid, search.GetKubeconfigBytes())
	for _, entry := range resouce_cache.GroupToTypeMap {
		resources := make([]item.Resource, 0, len(entry.V))
		for _, resource := range entry.V {
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
