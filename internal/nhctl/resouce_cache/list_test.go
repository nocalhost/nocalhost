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

package resouce_cache

import (
	"fmt"
	"io/ioutil"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/pkg/nhctl/log"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	b, _ := ioutil.ReadFile("/Users/naison/t")
	search, err := GetSearcher(string(b), "nh7wump", false)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			query, e := search.Criteria().Kind(&corev1.Namespace{}).Query()
			if e != nil {
				fmt.Println(e)
			}
			for _, ns := range query {
				fmt.Println(ns.(*corev1.Namespace).Namespace)
			}

			deploymentList, err := search.Criteria().ResourceType("deployments").AppName("default.application").Query()
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(deploymentList)
			deployments, _ := search.Criteria().Kind(&v1.Deployment{}).Namespace("test").Query()
			fmt.Print("\ntest: ")
			for _, name := range deployments {
				fmt.Print(name.(*v1.Deployment).Name + " ")
			}

			fmt.Println("pods in service 1")
			list, _ := search.Criteria().Kind(&v1.Deployment{}).AppName("1").Query()
			for _, pod := range list {
				fmt.Print(pod.(metav1.Object).GetName() + " ")
			}

			fmt.Print("\nkube-system: ")
			//keys := inform.GetIndexer().ListKeys()
			//for _, k := range keys {
			//    fmt.Print(k + " ")
			//}
			deployme, ok := search.Criteria().Kind(&v1.Deployment{}).Namespace("test").ResourceName("productpage").QueryOne()
			//for _, name := range deployme {
			//}
			if ok != nil {
				fmt.Print(deployme.(metav1.Object).GetCreationTimestamp().String() + " ")
			}
			time.Sleep(time.Second * 5)
		}
	}()
	search.Start()
}

//func TestConvert(t *testing.T) {
//	b, _ := ioutil.ReadFile("/Users/naison/tke")
//	s, _ := GetSearcher(string(b), "")
//
//	i, _ := s.GetByResourceAndNamespace("Pods", "", "default")
//	for _, dep := range i {
//		fmt.Println(dep.(metav1.Object).GetName())
//	}
//
//	search, err := GetSearcher(string(b), "")
//	if err != nil {
//		log.Fatal(err)
//	}
//	list, _ := search.GetAllByType(&extensionsv1beta1.Ingress{})
//	for _, i := range list {
//		fmt.Println(i.(metav1.Object).GetName())
//	}
//	search.Stop()
//}

func TestGetDeployment(t *testing.T) {
	b, _ := ioutil.ReadFile("/Users/naison/zzz")
	s, _ := GetSearcher(string(b), "", false)

	i, _ := s.Criteria().ResourceType("Pods").Namespace("default").Query()
	for _, dep := range i {
		fmt.Println(dep.(metav1.Object).GetName())
	}
	fmt.Println("-----------")
	i, e := s.Criteria().ResourceType("deployments").AppName("nocalhost").Namespace("nocalhost").Query()
	if e != nil {
		log.Error(e)
	}
	for _, k := range i {
		fmt.Println(k.(metav1.Object).GetName())
	}
}

func TestGetPods(t *testing.T) {
	b, _ := ioutil.ReadFile("/Users/naison/zzz")
	s, _ := GetSearcher(string(b), "", false)
	i, e := s.Criteria().ResourceType("pods").Namespace("default").Query()
	if e != nil {
		log.Error(e)
	}
	for _, dep := range i {
		fmt.Println(dep.(metav1.Object).GetName())
	}
	SortByNameAsc(i)
	fmt.Println("after sort by create timestamp asc")
	for _, dep := range i {
		fmt.Println(dep.(metav1.Object).GetName())
	}

}

func TestGetDefault(t *testing.T) {
	b, _ := ioutil.ReadFile("/Users/naison/zzz")
	s, _ := GetSearcher(string(b), "nocalhost", false)
	i, e := s.Criteria().ResourceType("deployments").
		ResourceName("nocalhost-api").
		AppName("nocalhost").
		Namespace("nocalhost").Query()
	if e != nil {
		log.Error(e)
	}
	for _, dep := range i {
		fmt.Println(dep.(metav1.Object).GetName())
	}

	/*i, e = s.GetByAppAndNs(&v1.Deployment{}, "default.application", "default")
	if e != nil {
		log.Error(e)
	}
	for _, dep := range i {
		fmt.Println(dep.(metav1.Object).GetName())
	}*/
}

func TestGetWithNsHaveNoPermission(t *testing.T) {
	b, _ := ioutil.ReadFile("/Users/naison/ZZZ")
	s, _ := GetSearcher(string(b), "nh2qpiv", false)
	i, e := s.Criteria().ResourceType("deployments").
		AppName("bookinfo").ResourceName("details").QueryOne()
	if e != nil {
		log.Error(e)
	}
	fmt.Println(i.(metav1.Object).GetName())
	//for _, dep := range i {
	//	fmt.Println(dep.(metav1.Object).GetName())
	//}
}

func TestGetNamespace(t *testing.T) {
	kubeconfigBytes, _ := ioutil.ReadFile("/Users/naison/zzz")
	s, _ := GetSearcher(string(kubeconfigBytes), "", false)
	list, er := s.Criteria().ResourceType("namespaces").Namespace("default").Query()
	if er != nil {
		fmt.Println(er)
	}
	for _, dep := range list {
		fmt.Println(dep.(metav1.Object).GetName())
	}
}

func TestGetDeploy(t *testing.T) {
	kubeconfigBytes, _ := ioutil.ReadFile("/Users/naison/zzz")
	s, _ := GetSearcher(string(kubeconfigBytes), "", false)
	list, er := s.Criteria().Kind(&v1.Deployment{}).Namespace("default").Query()
	if er != nil {
		fmt.Println(er)
	}
	for _, dep := range list {
		fmt.Println(dep.(metav1.Object).GetName())
	}
}

func TestNewLRU(t *testing.T) {
	lru := NewLRU(4)
	lru.Add("a", 1)
	lru.Add("b", 1)
	lru.Get("a")
	lru.Delete("b")
	lru.Add("c", 2)
	lru.Add("d", 2)
	lru.Add("c", 2)
	lru.Get("a")
	fmt.Println(lru.cache)
}
