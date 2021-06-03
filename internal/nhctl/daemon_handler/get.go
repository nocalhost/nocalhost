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

package daemon_handler

import (
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/appmeta_manager"
	"nocalhost/internal/nhctl/common"
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
)

func getServiceProfile(ns, appName string) map[string]*profile.SvcProfileV2 {
	serviceMap := make(map[string]*profile.SvcProfileV2)

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
	appProfile, err := nocalhost.GetProfileV2(ns, appName)
	if err != nil {
		log.Error(err)
		return nil
	}

	kubeConfigContent := fp.NewFilePath(appProfile.Kubeconfig).ReadFile()

	if appProfile != nil {
		meta := appmeta_manager.GetApplicationMeta(ns, appName, []byte(kubeConfigContent))
		appProfile.Installed = meta.IsInstalled()
		devMeta := meta.DevMeta

		// first iter from local svcProfile
		for _, svcProfile := range appProfile.SvcProfile {
			svcType := appmeta.SvcTypeOf(svcProfile.Type)

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
			return item.Result{Namespace: ns, Application: []item.App{getApp(ns, request.AppName, s)}}
		}
		// means it's cluster kubeconfig
		if request.Namespace == "" {
			nsObjectList, err := s.Criteria().ResourceType("namespaces").Query()
			if err == nil && nsObjectList != nil && len(nsObjectList) > 0 {
				result := make([]item.Result, 0, len(nsObjectList))
				// try to init a cluster level searcher
				searcher, err2 := resouce_cache.GetSearcher(KubeConfigBytes, "", true)
				for _, nsObject := range nsObjectList {
					name := nsObject.(metav1.Object).GetName()
					if err2 != nil {
						log.Error(err2)
						// if cluster level searcher init failed, then try to init a namespace level searcher
						searcher, err2 = resouce_cache.GetSearcher(KubeConfigBytes, name, false)
						if err2 != nil {
							log.Error(err2)
							continue
						}
					}
					result = append(result, getApplicationByNs(name, request.KubeConfig, searcher))
				}
				return result
			}
		}
		return getApplicationByNs(ns, request.KubeConfig, s)
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
		serviceMap := getServiceProfile(ns, request.AppName)
		// get all resource in namespace
		var items []interface{}
		if request.ResourceName == "" {
			items, err = s.Criteria().ResourceType(request.Resource).AppName(request.AppName).Namespace(ns).Query()
			if err != nil || len(items) == 0 {
				return nil
			}
			result := make([]item.Item, 0, len(items))
			for _, i := range items {
				gvr, _ := s.GetGvr(request.Resource)
				result = append(
					result, item.Item{
						Metadata:    i,
						Description: serviceMap[gvr.Resource+"/"+i.(metav1.Object).GetName()],
					},
				)
			}
			return result
		} else {
			// get specify resource name in namespace
			one, err := s.Criteria().
				ResourceType(request.Resource).
				ResourceName(request.ResourceName).
				Namespace(ns).
				AppName(request.AppName).
				QueryOne()
			if err != nil || one == nil {
				return nil
			}
			gvr, _ := s.GetGvr(request.Resource)
			return item.Item{Metadata: one, Description: serviceMap[gvr.Resource+"/"+one.(metav1.Object).GetName()]}
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

func getApplicationByNs(ns, kubeconfigPath string, search *resouce_cache.Searcher) item.Result {
	result := item.Result{Namespace: ns}
	applicationMetaList := InitDefaultAppIfNecessary(ns, kubeconfigPath)
	SortApplication(applicationMetaList)
	for _, applicationMeta := range applicationMetaList {
		if applicationMeta != nil {
			result.Application = append(result.Application, getApp(ns, applicationMeta.Application, search))
		}
	}
	return result
}

func getApp(namespace, appName string, search *resouce_cache.Searcher) item.App {
	groupToTypeMap := []struct {
		k string
		v []string
	}{
		{k: "Workloads", v: []string{"deployments", "statefulsets", "daemonsets", "jobs", "cronjobs", "pods"}},
		{k: "Networks", v: []string{"services", "endpoints", "ingresses", "networkpolicies"}},
		{
			k: "Configurations",
			v: []string{"configmaps", "secrets", "horizontalpodautoscalers", "resourcequotas", "poddisruptionbudgets"},
		},
		{k: "Storages", v: []string{"persistentvolumes", "persistentvolumeclaims", "storageclasses"}},
	}
	result := item.App{Name: appName}
	profileMap := getServiceProfile(namespace, appName)
	for _, entry := range groupToTypeMap {
		resources := make([]item.Resource, 0, len(entry.v))
		for _, resource := range entry.v {
			resourceList, err := search.Criteria().
				ResourceType(resource).AppName(appName).Namespace(namespace).Query()
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
		result.Groups = append(result.Groups, item.Group{GroupName: entry.k, List: resources})
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
			if n1 >= n2 {
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
				Type: meta.ApplicationType,
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

func InitDefaultAppIfNecessary(namespace, kubeconfigPath string) []*appmeta.ApplicationMeta {
	kubeconfigBytes, _ := ioutil.ReadFile(kubeconfigPath)
	applicationMetaList := appmeta_manager.GetApplicationMetas(namespace, kubeconfigBytes)
	var foundDefaultApp bool
	for _, meta := range applicationMetaList {
		if meta.Application == nocalhost.DefaultNocalhostApplication {
			foundDefaultApp = true
			break
		}
	}
	if !foundDefaultApp {
		// try init default application
		utils.ShouldI(
			common.InitDefaultApplicationInCurrentNs(namespace, kubeconfigPath),
			"Error while create default application",
		)
		return appmeta_manager.GetApplicationMetas(namespace, kubeconfigBytes)
	}
	return applicationMetaList
}
