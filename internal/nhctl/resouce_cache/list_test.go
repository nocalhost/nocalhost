/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package resouce_cache

import (
	"fmt"
	"github.com/hashicorp/golang-lru/simplelru"
	"io/ioutil"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/homedir"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

//func TestName(t *testing.T) {
//	b, _ := ioutil.ReadFile("/Users/naison/t")
//	search, err := GetSearcherWithLRU(b, "nh7wump")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	go func() {
//		for {
//			query, e := search.Criteria().Kind(&corev1.Namespace{}).Query()
//			if e != nil {
//				fmt.Println(e)
//			}
//			for _, ns := range query {
//				fmt.Println(ns.(*corev1.Namespace).Namespace)
//			}
//
//			deploymentList, err := search.Criteria().ResourceType("deployments").AppName("default.application").Query()
//			if err != nil {
//				fmt.Println(err)
//			}
//			fmt.Println(deploymentList)
//			deployments, _ := search.Criteria().Kind(&v1.Deployment{}).Namespace("test").Query()
//			fmt.Print("\ntest: ")
//			for _, name := range deployments {
//				fmt.Print(name.(*v1.Deployment).Name + " ")
//			}
//
//			fmt.Println("pods in service 1")
//			list, _ := search.Criteria().Kind(&v1.Deployment{}).AppName("1").Query()
//			for _, pod := range list {
//				fmt.Print(pod.(metav1.Object).GetName() + " ")
//			}
//
//			fmt.Print("\nkube-system: ")
//			//keys := inform.GetIndexer().ListKeys()
//			//for _, k := range keys {
//			//    fmt.Print(k + " ")
//			//}
//			deployme, ok := search.Criteria().Kind(&v1.Deployment{}).Namespace("test").ResourceName("productpage").QueryOne()
//			//for _, name := range deployme {
//			//}
//			if ok != nil {
//				fmt.Print(deployme.(metav1.Object).GetCreationTimestamp().String() + " ")
//			}
//			time.Sleep(time.Second * 5)
//		}
//	}()
//	//search.Start()
//}

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
	bytes, _ := ioutil.ReadFile("/Users/naison/zzz")
	s, _ := GetSearcherWithLRU(bytes, "")

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
	bytes, _ := ioutil.ReadFile("/Users/naison/zzz")
	s, _ := GetSearcherWithLRU(bytes, "")
	i, e := s.Criteria().ResourceType("pods").Namespace("default").Query()
	if e != nil {
		log.Error(e)
	}
	for _, dep := range i {
		fmt.Println(dep.(metav1.Object).GetName())
	}
	//SortByNameAsc(i)
	fmt.Println("after sort by create timestamp asc")
	for _, dep := range i {
		fmt.Println(dep.(metav1.Object).GetName())
	}

}

func TestConcurrence(t *testing.T) {
	//bytes, _ := ioutil.ReadFile("/tmp/test.txt")
	bytes, _ := ioutil.ReadFile(path.Join(utils.GetHomePath(), ".kube/config"))
	//s, err := GetSearcherWithLRU(bytes, "nh2yunf")
	_, _ = GetSearcherWithLRU(bytes, "nocalhost-test")
	time.Sleep(3 * time.Second)

	wg := sync.WaitGroup{}
	//for ii := 0; ii < 10000; ii++ {
	for ii := 0; ii < 1000000; ii++ {
		wg.Add(1)
		go func() {
			//start := time.Now()
			defer func() {
				if err := recover(); err != nil {
					fmt.Println("recovering")
				}
			}()
			s, err := GetSearcherWithLRU(bytes, "nocalhost-test")
			if err != nil {
				//panic(err)
				return
			}
			i, e := s.Criteria().ResourceType("crds").
				ResourceName("").
				//AppName("bookinfo").
				Namespace("").Query()
			if e != nil {
				e.Error()
			}
			for _, dep := range i {
				dep.(metav1.Object).GetName()
			}
			//fmt.Printf("%d Get len %d, takes: %d ms\n", ii, len(i), start.Sub(time.Now()).Microseconds())
		}()
	}

	/*i, e = s.GetByAppAndNs(&v1.Deployment{}, "default.application", "default")
	  if e != nil {
	  	log.Error(e)
	  }
	  for _, dep := range i {
	  	fmt.Println(dep.(metav1.Object).GetName())
	  }*/
}

func TestGet(t *testing.T) {
	//bytes, _ := ioutil.ReadFile("/tmp/admin-large")
	bytes, _ := ioutil.ReadFile(path.Join(utils.GetHomePath(), ".kube/config"))
	//s, err := GetSearcherWithLRU(bytes, "nh2yunf")
	namespace := "nocalhost-test"
	_, _ = GetSearcherWithLRU(bytes, namespace)
	time.Sleep(3 * time.Second)

	s, err := GetSearcherWithLRU(bytes, namespace)
	if err != nil {
		panic(err)
	}
	i, e := s.Criteria().ResourceType("cronjob").
		ResourceName("").
		//AppName("bookinfo").
		Namespace(namespace).Query()
	if e != nil {
		panic(e)
	}
	for _, dep := range i {
		fmt.Println(dep.(metav1.Object).GetName())
	}
	//fmt.Printf("%d Get len %d, takes: %d ms\n", ii, len(i), start.Sub(time.Now()).Microseconds())

	/*i, e = s.GetByAppAndNs(&v1.Deployment{}, "default.application", "default")
	  if e != nil {
	  	log.Error(e)
	  }
	  for _, dep := range i {
	  	fmt.Println(dep.(metav1.Object).GetName())
	  }*/
}

func TestGetWithNsHaveNoPermission(t *testing.T) {
	bytes, _ := ioutil.ReadFile("/Users/naison/ZZZ")
	s, _ := GetSearcherWithLRU(bytes, "nh2qpiv")
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
	kubeconfigBytes, _ := ioutil.ReadFile("/Users/naison/.kube/config")
	s, _ := GetSearcherWithLRU(kubeconfigBytes, "default")
	list, er := s.Criteria().ResourceType("pods").Namespace("test").Query()
	if er != nil {
		fmt.Println(er)
	}
	for _, dep := range list {
		fmt.Println(dep.(metav1.Object).GetName())
	}
}

//func TestGetDeploy(t *testing.T) {
//	kubeconfigBytes, _ := ioutil.ReadFile("/Users/naison/.kube/config")
//	s, _ := GetSearcherWithLRU(kubeconfigBytes, "")
//	list, er := s.Criteria().Kind(&v1.Deployment{}).Namespace("default").Query()
//	if er != nil {
//		fmt.Println(er)
//	}
//	for _, dep := range list {
//		fmt.Println(dep.(metav1.Object).GetName())
//	}
//}

func TestNewLRU(t *testing.T) {
	lru, _ := simplelru.NewLRU(2, nil)
	lru.Add("a", 1)
	lru.Add("b", 1)
	lru.Get("a")
	lru.Remove("b")
	lru.Add("c", 2)
	lru.Add("d", 2)
	lru.Add("c", 2)
	lru.Get("a")
	fmt.Println(lru.Keys())
}

func TestApiResource(t *testing.T) {
	join := filepath.Join("/Users/naison/Downloads/app/reviews", "config")
	file, _ := ioutil.ReadFile(join)

	config, err := clientcmd.RESTConfigFromKubeConfig(file)
	if err != nil {
		log.Fatal(err)
	}
	//config.Timeout = time.Second * 5
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(10000, 10000)
	clientset, err1 := kubernetes.NewForConfig(config)
	if err1 != nil {
		log.Fatal(err1)
	}
	_, err2 := restmapper.GetAPIGroupResources(clientset)
	if err2 != nil {
		log.Fatal(err2)
	}

	cc, err := clientset.ServerPreferredResources()
	fmt.Println(len(cc))
	fmt.Println(k8serrors.IsServiceUnavailable(err))
	if err != nil {
		log.Fatal(err)
	}
}

//func TestNoListNamespacePermission(t *testing.T) {
//	join := filepath.Join(homedir.HomeDir(), ".kube", "minikube")
//	kubeconfigBytes, _ := ioutil.ReadFile(join)
//	s, _ := GetSearcherWithLRU(kubeconfigBytes, "default")
//	list, er := s.Criteria().Kind(&corev1.Namespace{}).Namespace("default").Query()
//	if er != nil {
//		fmt.Println(er)
//	}
//	for _, dep := range list {
//		fmt.Println(dep.(metav1.Object).GetName())
//	}
//}

func TestEventHandler(t *testing.T) {
	join := filepath.Join(homedir.HomeDir(), ".kube", "config")
	kubeconfigBytes, _ := ioutil.ReadFile(join)
	config, _ := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	// default value is flowcontrol.NewTokenBucketRateLimiter(5, 10)
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(10000, 10000)
	clientset, _ := kubernetes.NewForConfig(config)
	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		clientset, time.Second*5, informers.WithNamespace("kubevela-foo1"),
	)
	resource, _ := informerFactory.ForResource(corev1.SchemeGroupVersion.WithResource("pods"))
	resource.Informer().AddEventHandler(NewResourceEventHandlerFuncs(
		resource, kubeconfigBytes, schema.GroupVersionResource{Resource: "pods"}),
	)
	resource1, _ := informerFactory.ForResource(v1.SchemeGroupVersion.WithResource("deployments"))
	resource1.Informer().AddEventHandler(
		NewResourceEventHandlerFuncs(resource1, kubeconfigBytes, schema.GroupVersionResource{Resource: "deployments"}),
	)
	informerFactory.Start(make(chan struct{}))
	informerFactory.WaitForCacheSync(make(chan struct{}))
	for {
		select {
		default:
			fmt.Println(GetAllAppNameByNamespace(kubeconfigBytes, "kubevela-foo1"))
			time.Sleep(time.Second * 5)
		}
	}
}
