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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/appmeta_manager"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/resouce_cache"
	"nocalhost/pkg/nhctl/log"
	"sort"
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
						serviceMap[svcProfileV2.Name] = svcProfileV2
					}
				}
			}
		}
	}
	return serviceMap
}

func HandleGetResourceInfoRequest(request *command.GetResourceInfoCommand) interface{} {
	var s *resouce_cache.Searcher
	var err error
	var ns = request.Namespace
	if request.Namespace == "" {
		ns = getNamespace("", []byte(request.KubeConfig))
		s, err = resouce_cache.GetSearcher(request.KubeConfig, ns, false)
	} else {
		s, err = resouce_cache.GetSearcher(request.KubeConfig, request.Namespace, false)
	}
	if err != nil {
		return nil
	}
	switch request.Resource {
	case "all":
		if request.AppName != "" {
			return Result{Namespace: ns, Application: []App{getApp(ns, request.AppName, s)}}
		}
		// means it's cluster kubeconfig
		if request.Namespace == "" {
			nsObjectList, err := s.Criteria().ResourceType("namespaces").Query()
			if err == nil && nsObjectList != nil && len(nsObjectList) > 0 {
				result := make([]Result, 0, len(nsObjectList))
				// try to init a cluster level searcher
				searcher, err2 := resouce_cache.GetSearcher(request.KubeConfig, "", true)
				for _, nsObject := range nsObjectList {
					name := nsObject.(metav1.Object).GetName()
					if err2 != nil {
						log.Error(err2)
						// if cluster level searcher init failed, then try to init a namespace level searcher
						searcher, err2 = resouce_cache.GetSearcher(request.KubeConfig, name, false)
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
		ns = getNamespace(request.Namespace, []byte(request.KubeConfig))
		if request.ResourceName == "" {
			metas := appmeta_manager.GetApplicationMetas(ns, request.KubeConfig)
			if metas != nil {
				sort.SliceStable(metas, func(i, j int) bool {
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
				})
			}
			return metas
		} else {
			return appmeta_manager.GetApplicationMeta(ns, request.ResourceName, request.KubeConfig)
		}
	default:
		ns = getNamespace(request.Namespace, []byte(request.KubeConfig))
		serviceMap := getServiceProfile(ns, request.ResourceName)
		// get all resource in namespace
		var items []interface{}
		if request.ResourceName == "" {
			items, err = s.Criteria().ResourceType(request.Resource).AppName(request.AppName).Namespace(ns).Query()
			if err != nil || len(items) == 0 {
				return nil
			}
			result := make([]Item, 0, len(items))
			for _, i := range items {
				result = append(result, Item{Metadata: i, Description: serviceMap[i.(metav1.Object).GetName()]})
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
			return Item{Metadata: one, Description: serviceMap[one.(metav1.Object).GetName()]}
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

func getApplicationByNs(ns, kubeconfigBytes string, search *resouce_cache.Searcher) Result {
	result := Result{Namespace: ns}
	applicationMetaList := appmeta_manager.GetApplicationMetas(ns, kubeconfigBytes)
	for _, applicationMeta := range applicationMetaList {
		if applicationMeta != nil {
			result.Application = append(result.Application, getApp(ns, applicationMeta.Application, search))
		}
	}
	return result
}

func getApp(namespace, appName string, search *resouce_cache.Searcher) App {
	groupToTypeMap := []struct {
		k string
		v []string
	}{
		{k: "Workloads", v: []string{"deployments", "statefulsets", "daemonsets", "jobs", "cronjobs", "pods"}},
		{k: "Networks", v: []string{"services", "endpoints", "ingresses", "networkpolicies"}},
		{k: "Configurations", v: []string{"configmaps", "secrets", "horizontalpodautoscalers", "resourcequotas", "poddisruptionbudgets"}},
		{k: "Storages", v: []string{"persistentvolumes", "persistentvolumeclaims", "storageclasses"}},
	}
	result := App{Name: appName}
	profileMap := getServiceProfile(namespace, appName)
	for _, entry := range groupToTypeMap {
		resources := make([]Resource, 0, len(entry.v))
		for _, resource := range entry.v {
			resourceList, err := search.Criteria().
				ResourceType(resource).AppName(appName).Namespace(namespace).Query()
			if err == nil {
				items := make([]Item, 0, len(resourceList))
				for _, v := range resourceList {
					items = append(items, Item{Metadata: v, Description: profileMap[v.(metav1.Object).GetName()]})
				}
				resources = append(resources, Resource{Name: resource, List: items})
			}
		}
		result.Groups = append(result.Groups, Group{GroupName: entry.k, List: resources})
	}
	return result
}

type Result struct {
	Namespace   string `json:"namespace"`
	Application []App  `json:"application"`
}

type App struct {
	Name   string  `json:"name"`
	Groups []Group `json:"group"`
}

type Group struct {
	GroupName string     `json:"type"`
	List      []Resource `json:"resource"`
}

type Resource struct {
	Name string `json:"name"`
	List []Item `json:"list"`
}

type Item struct {
	Metadata    interface{}           `json:"data,omitempty"`
	Description *profile.SvcProfileV2 `json:"description,omitempty"`
}
