/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package daemon_handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/appmeta_manager"
	"nocalhost/internal/nhctl/common"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/daemon_handler/item"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/resouce_cache"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"sort"
	"strings"
	"sync"
	"time"
)

var svcProfileCacheMap = NewCache(time.Second * 2)

func getServiceProfile(ns, appName string) map[string]*profile.SvcProfileV2 {
	serviceMap := make(map[string]*profile.SvcProfileV2)
	if appName == "" || ns == "" {
		return serviceMap
	}
	description := GetDescriptionDaemon(ns, appName)
	if description != nil {
		for _, svcProfileV2 := range description.SvcProfile {
			if svcProfileV2 != nil {
				name := strings.ToLower(svcProfileV2.Type) + "s"
				serviceMap[name+"/"+svcProfileV2.Name] = svcProfileV2
			}
		}
	}

	return serviceMap
}

func GetDescriptionDaemon(ns, appName string) *profile.AppProfileV2 {
	var appProfile *profile.AppProfileV2
	var err error
	appProfileCache, found := svcProfileCacheMap.Get(fmt.Sprintf("%s/%s", ns, appName))
	if !found || appProfileCache == nil {
		if appProfile, err = nocalhost.GetProfileV2(ns, appName); err == nil {
			svcProfileCacheMap.Set(fmt.Sprintf("%s/%s", ns, appName), appProfile)
		}
	} else {
		appProfile = appProfileCache.(*profile.AppProfileV2)
	}
	if err != nil {
		log.Error(err)
		return nil
	}
	if appProfile != nil {
		kubeConfigContent := fp.NewFilePath(appProfile.Kubeconfig).ReadFile()
		// deep copy
		marshal, err := json.Marshal(appmeta_manager.GetApplicationMeta(ns, appName, []byte(kubeConfigContent)))
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
					Name: svcProfile.Name,
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
			svcType := base.SvcTypeOf(svcProfile.Type)

			svcProfile.Developing = meta.CheckIfSvcDeveloping(svcProfile.ActualName, svcType)
			svcProfile.Possess = meta.SvcDevModePossessor(
				svcProfile.ActualName, svcType,
				appProfile.Identifier,
			)

			if m := devMeta[svcType.Alias()]; m != nil {
				delete(m, svcProfile.ActualName)
			}
		}

		// then gen the fake profile for remote svc
		for svcTypeAlias, m := range devMeta {
			for svcName, _ := range m {
				svcProfile := appProfile.SvcProfileV2(svcName, string(svcTypeAlias.Origin()))
				svcProfile.Developing = true
				svcProfile.Possess = meta.SvcDevModePossessor(
					svcProfile.ActualName, svcTypeAlias.Origin(),
					appProfile.Identifier,
				)
			}
		}
		return appProfile
	}
	return nil
}

func HandleGetResourceInfoRequest(request *command.GetResourceInfoCommand) interface{} {
	KubeConfigBytes, _ := ioutil.ReadFile(request.KubeConfig)
	var s *resouce_cache.Searcher
	var err error
	var ns = request.Namespace
	if request.Namespace == "" {
		ns = getNamespace("", KubeConfigBytes)
		s, err = resouce_cache.GetSearcher(KubeConfigBytes, ns, false)
	} else {
		s, err = resouce_cache.GetSearcher(KubeConfigBytes, request.Namespace, false)
	}
	if err != nil {
		return nil
	}
	switch request.Resource {
	case "all":
		if request.AppName != "" {
			names := getAvailableAppName(ns, request.KubeConfig)
			return item.Result{
				Namespace:   ns,
				Application: []item.App{getApp(names, ns, request.AppName, s, request.Label)},
			}
		}
		// means it's cluster kubeconfig
		if request.Namespace == "" {
			nsObjectList, err := s.Criteria().ResourceType("namespaces").Query()
			if err == nil && nsObjectList != nil && len(nsObjectList) > 0 {
				result := make([]item.Result, 0, len(nsObjectList))
				// try to init a cluster level searcher
				searcher, err2 := resouce_cache.GetSearcher(KubeConfigBytes, "", true)
				if err2 != nil {
					return nil
				}
				var wg sync.WaitGroup
				wg.Add(len(nsObjectList))
				okChan := make(chan struct{}, 2)
				go func() {
					time.Sleep(time.Second * 10)
					okChan <- struct{}{}
				}()
				for _, nsObject := range nsObjectList {
					finalNs := nsObject
					go func() {
						name := finalNs.(metav1.Object).GetName()
						result = append(result, getApplicationByNs(name, request.KubeConfig, searcher, request.Label))
						wg.Done()
					}()
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
		ns = getNamespace(request.Namespace, KubeConfigBytes)
		if request.ResourceName == "" {
			return ParseApplicationsResult(ns, InitDefaultAppIfNecessary(ns, request.KubeConfig))
		} else {
			meta := appmeta_manager.GetApplicationMeta(ns, request.ResourceName, KubeConfigBytes)
			return ParseApplicationsResult(ns, []*appmeta.ApplicationMeta{meta})
		}
	default:
		ns = getNamespace(request.Namespace, KubeConfigBytes)

		var serviceMap map[string]*profile.SvcProfileV2
		if request.AppName != "" {
			serviceMap = getServiceProfile(ns, request.AppName)
		}

		appNameList := getAvailableAppName(ns, request.KubeConfig)
		// get all resource in namespace
		var items []interface{}
		if request.ResourceName == "" {
			items, err = s.Criteria().
				ResourceType(request.Resource).
				AppName(request.AppName).
				AppNameNotIn(appNameList...).
				Namespace(ns).
				Label(request.Label).
				Query()
			if err != nil || len(items) == 0 {
				return nil
			}
			result := make([]item.Item, 0, len(items))
			for _, i := range items {
				tempItem := item.Item{Metadata: i}
				if mapping, err := s.GetRestMapping(request.Resource); err == nil {
					tempItem.Description = serviceMap[mapping.Resource.Resource+"/"+i.(metav1.Object).GetName()]
				}
				result = append(result, tempItem)
			}
			return result
		} else {
			// get specify resource name in namespace
			one, err := s.Criteria().
				ResourceType(request.Resource).
				ResourceName(request.ResourceName).
				Namespace(ns).
				AppNameNotIn(appNameList...).
				AppName(request.AppName).
				QueryOne()
			if err != nil || one == nil {
				return nil
			}
			i := item.Item{Metadata: one}
			if mapping, err := s.GetRestMapping(request.Resource); err == nil {
				i.Description = serviceMap[mapping.Resource.Resource+"/"+one.(metav1.Object).GetName()]
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
	nameList := getAvailableAppName(namespace, kubeconfigPath)
	var wg sync.WaitGroup
	wg.Add(len(nameList))
	okChan := make(chan struct{}, 2)
	go func() {
		time.Sleep(time.Second * 10)
		okChan <- struct{}{}
	}()
	for _, name := range nameList {
		finalName := name
		go func() {
			result.Application = append(result.Application, getApp(nameList, namespace, finalName, search, label))
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		okChan <- struct{}{}
	}()
	<-okChan
	return result
}

func getApp(name []string, namespace, appName string, search *resouce_cache.Searcher, label map[string]string) item.App {
	result := item.App{Name: appName}
	profileMap := getServiceProfile(namespace, appName)
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

func getAvailableAppName(namespace, kubeconfig string) []string {
	applicationMetaList := InitDefaultAppIfNecessary(namespace, kubeconfig)
	var availableAppName []string
	for _, meta := range applicationMetaList {
		if meta != nil {
			availableAppName = append(availableAppName, meta.Application)
		}
	}
	sort.SliceStable(availableAppName, func(i, j int) bool { return availableAppName[i] < availableAppName[j] })
	return availableAppName
}

func InitDefaultAppIfNecessary(namespace, kubeconfigPath string) []*appmeta.ApplicationMeta {
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
		// try init default application
		_, err := common.InitDefaultApplicationInCurrentNs(namespace, kubeconfigPath)
		utils.ShouldI(err, "Error while create default application")
		applicationMetaList = appmeta_manager.GetApplicationMetas(namespace, kubeconfigBytes)
	}
	SortApplication(applicationMetaList)
	return applicationMetaList
}
