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
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/appmeta_manager"
	"nocalhost/internal/nhctl/common"
	"nocalhost/internal/nhctl/daemon_server/command"
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
	profileV2, err := nocalhost.GetProfileV2(ns, appName)
	if err != nil {
		log.Error(err)
	}
	if profileV2 != nil {
		nocalhostApp, err2 := app.NewApplication(appName, ns, profileV2.Kubeconfig, true)
		if err2 != nil {
			log.Error(err2)
		}
		if nocalhostApp != nil {
			description := nocalhostApp.GetDescription()
			if description != nil {
				for _, svcProfileV2 := range description.SvcProfile {
					if svcProfileV2 != nil {
						name := strings.ToLower(svcProfileV2.Type) + "s"
						serviceMap[name+"/"+svcProfileV2.Name] = svcProfileV2
					}
				}
			}
		}
	}
	return serviceMap
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
			return Result{Namespace: ns, Application: []App{getApp(names, ns, request.AppName, s)}}
		}
		// means it's cluster kubeconfig
		if request.Namespace == "" {
			nsObjectList, err := s.Criteria().ResourceType("namespaces").Query()
			if err == nil && nsObjectList != nil && len(nsObjectList) > 0 {
				result := make([]Result, 0, len(nsObjectList))
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
			meta := appmeta_manager.GetApplicationMeta(ns, request.ResourceName, string(KubeConfigBytes))
			return ParseApplicationsResult(ns, []*appmeta.ApplicationMeta{meta})
		}
	default:
		ns = getNamespace(request.Namespace, KubeConfigBytes)
		serviceMap := getServiceProfile(ns, request.AppName)
		appNameList := getAvailableAppName(ns, request.KubeConfig)
		// get all resource in namespace
		var items []interface{}
		if request.ResourceName == "" {
			items, err = s.Criteria().
				ResourceType(request.Resource).
				AppName(request.AppName).
				AppNameNotIn(appNameList...).
				Namespace(ns).
				Query()
			if err != nil || len(items) == 0 {
				return nil
			}
			result := make([]Item, 0, len(items))
			for _, i := range items {
				gvr, _ := s.GetGvr(request.Resource)
				result = append(result, Item{
					Metadata:    i,
					Description: serviceMap[gvr.Resource+"/"+i.(metav1.Object).GetName()],
				})
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
			gvr, _ := s.GetGvr(request.Resource)
			return Item{Metadata: one, Description: serviceMap[gvr.Resource+"/"+one.(metav1.Object).GetName()]}
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

func getApplicationByNs(namespace, kubeconfigPath string, search *resouce_cache.Searcher) Result {
	result := Result{Namespace: namespace}
	nameList := getAvailableAppName(namespace, kubeconfigPath)
	for _, name := range nameList {
		result.Application = append(result.Application, getApp(nameList, namespace, name, search))
	}
	return result
}

func getApp(name []string, namespace, appName string, search *resouce_cache.Searcher) App {
	result := App{Name: appName}
	profileMap := getServiceProfile(namespace, appName)
	for _, entry := range resouce_cache.GroupToTypeMap {
		resources := make([]Resource, 0, len(entry.V))
		for _, resource := range entry.V {
			resourceList, err := search.Criteria().
				ResourceType(resource).AppName(appName).AppNameNotIn(name...).Namespace(namespace).Query()
			if err == nil {
				items := make([]Item, 0, len(resourceList))
				for _, v := range resourceList {
					items = append(items, Item{
						Metadata: v, Description: profileMap[resource+"/"+v.(metav1.Object).GetName()],
					})
				}
				resources = append(resources, Resource{Name: resource, List: items})
			}
		}
		result.Groups = append(result.Groups, Group{GroupName: entry.K, List: resources})
	}
	return result
}

type Result struct {
	Namespace   string `json:"namespace" yaml:"namespace"`
	Application []App  `json:"application" yaml:"application"`
}

type App struct {
	Name   string  `json:"name" yaml:"name"`
	Groups []Group `json:"group" yaml:"group"`
}

type Group struct {
	GroupName string     `json:"type" yaml:"type"`
	List      []Resource `json:"resource" yaml:"resource"`
}

type Resource struct {
	Name string `json:"name" yaml:"name"`
	List []Item `json:"list" yaml:"list"`
}

type Item struct {
	Metadata    interface{}           `json:"info,omitempty" yaml:"info"`
	Description *profile.SvcProfileV2 `json:"description,omitempty" yaml:"description"`
}

func SortApplication(metas []*appmeta.ApplicationMeta) {
	if metas == nil {
		return
	}
	sort.SliceStable(metas, func(i, j int) bool {
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
	})
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
	applicationMetaList := appmeta_manager.GetApplicationMetas(namespace, string(kubeconfigBytes))
	var foundDefaultApp bool
	for _, meta := range applicationMetaList {
		if meta.Application == nocalhost.DefaultNocalhostApplication {
			foundDefaultApp = true
			break
		}
	}
	if !foundDefaultApp {
		// try init default application
		utils.ShouldI(common.InitDefaultApplicationInCurrentNs(namespace, kubeconfigPath), "Error while create default application")
		applicationMetaList = appmeta_manager.GetApplicationMetas(namespace, string(kubeconfigBytes))
	}
	SortApplication(applicationMetaList)
	return applicationMetaList
}
