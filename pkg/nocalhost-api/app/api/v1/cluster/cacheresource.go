/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/retry"
	"math/rand"
	"nocalhost/internal/nocalhost-api/cache"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"strconv"
	"sync"
	"time"
)

// resourceCache for cache resources(like cpu, memory, storage, pod number...), init cache with expire 15 seconds
var resourceCache = cache.NewCache(time.Second * 120)
var defaultValue = []model.Resource{
	{ResourceName: v1.ResourcePods, Capacity: 0, Used: 0, Percentage: 0},
	{ResourceName: v1.ResourceCPU, Capacity: 0, Used: 0, Percentage: 0},
	{ResourceName: v1.ResourceMemory, Capacity: 0, Used: 0, Percentage: 0},
	{ResourceName: v1.ResourceStorage, Capacity: 0, Used: 0, Percentage: 0},
}

var lock = sync.Mutex{}
var cacheRunnable = sync.Map{}

func Add(kubeconfig string) {
	lock.Lock()
	defer lock.Unlock()
	load, ok := cacheRunnable.Load(kubeconfig)
	if ok && load != nil {
		return
	}
	if value, loaded := cacheRunnable.LoadAndDelete(kubeconfig); loaded && value != nil {
		value.(context.CancelFunc)()
	}
	ctx, cancelFunc := context.WithCancel(context.TODO())
	cacheRunnable.Store(kubeconfig, cancelFunc)
	go func() {
		after := time.NewTicker(time.Second * 30)
		c := make(chan struct{}, 1)
		c <- struct{}{}
		for {
			select {
			case <-ctx.Done():
				after.Stop()
				close(c)
				return
			case <-after.C:
				c <- struct{}{}
			case <-c:
				go reload(kubeconfig)
			}
		}
	}()
}

func Remove(kubeconfig string) {
	lock.Lock()
	defer lock.Unlock()
	if cancelFunc, found := cacheRunnable.LoadAndDelete(kubeconfig); found && cancelFunc != nil {
		cancelFunc.(context.CancelFunc)()
	}
}

// remove unneeded kubeconfig goroutines
func Merge(availableKubeConfigs map[string]string) {
	lock.Lock()
	defer lock.Unlock()
	needsToBeDelete := make([]string, 0)
	cacheRunnable.Range(func(key, value interface{}) bool {
		if kubeconfig, found := availableKubeConfigs[key.(string)]; !found || len(kubeconfig) == 0 {
			needsToBeDelete = append(needsToBeDelete, key.(string))
		}
		return true
	})
	for _, kubeconfig := range needsToBeDelete {
		if value, loaded := cacheRunnable.LoadAndDelete(kubeconfig); loaded && value != nil {
			value.(context.CancelFunc)()
		}
	}
}

func GetFromCache(kubeconfig string) []model.Resource {
	get, found := resourceCache.Get(kubeconfig)
	if found && get != nil {
		return get.([]model.Resource)
	} else {
		return defaultValue
	}
}

func reload(kubeconfig string) {
	var resources []model.Resource
	err := retry.OnError(wait.Backoff{
		Steps:    5,
		Duration: time.Duration(rand.Intn(4000)) * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}, func(err error) bool { return err != nil }, func() error {
		temp, err := fetchData(kubeconfig)
		if err != nil {
			return err
		} else {
			resources = temp
			return nil
		}
	})
	if err != nil || len(resources) == 0 {
		return
	}
	// if resources is default value
	if isSame(resources, defaultValue) {
		// if resourceCache found not default value, then not need to replace it
		if v, found := resourceCache.Get(kubeconfig); found && !isSame(v.([]model.Resource), defaultValue) {
			return
		}
	}
	resourceCache.Set(kubeconfig, resources)
}

// fetchData info by using metrics-api
func fetchData(kubeconfig string) ([]model.Resource, error) {
	goclient, err := clientgo.NewGoClient([]byte(kubeconfig))
	if err != nil {
		log.Warn(fmt.Sprintf("init goclinet error, error: %v", err))
		return defaultValue, err
	}
	restClient, err := goclient.GetRestClient()
	if err != nil {
		log.Warn(fmt.Sprintf("init restclient error, error: %v", err))
		return defaultValue, err
	}
	list, err := goclient.GetClusterNode()
	if err != nil {
		log.Warn(fmt.Sprintf("get clusterNode error, error: %v", err))
		return defaultValue, err
	}
	summaries := make([]model.Summary, len(list.Items), len(list.Items))
	wg := sync.WaitGroup{}
	wg.Add(len(list.Items))
	for i, node := range list.Items {
		node := node
		i := i
		go func() {
			// using metrics-api to get nodes stats summary
			defer wg.Done()
			var bytes []byte

			err = retry.OnError(wait.Backoff{
				Steps:    3,
				Duration: time.Duration(rand.Intn(4000)) * time.Millisecond,
				Factor:   1.0,
				Jitter:   0.1,
			}, func(err error) bool {
				return err != nil
			}, func() error {
				temp, errs := restClient.Get().
					Timeout(time.Second * 10).
					Throttle(flowcontrol.NewTokenBucketRateLimiter(1000, 1000)).
					RequestURI(fmt.Sprintf("/api/v1/nodes/%s/proxy/stats/summary", node.Name)).
					DoRaw(context.Background())
				if errs != nil {
					return errs
				} else {
					bytes = temp
					return nil
				}
			})
			if err != nil {
				log.Warnf("get stats summary error, err: %v, ignore", err)
			}
			var s model.Summary
			_ = json.Unmarshal(bytes, &s)
			summaries[i] = s
		}()
	}
	wg.Wait()
	summaries = summaries[0:]

	var cpuTotal, memoryTotal, storageTotal, podTotal int64
	var cpuUsed, memoryUsed, storageUsed, podUsed int64
	for _, node := range list.Items {
		cpuTotal += node.Status.Capacity.Cpu().MilliValue()
		// convert bytes to mega bytes (B --> MB)
		memoryTotal += node.Status.Capacity.Memory().Value() / 1024 / 1024
		storageTotal += node.Status.Capacity.StorageEphemeral().Value() / 1024 / 1024
		podTotal += node.Status.Capacity.Pods().Value()
	}
	for _, summary := range summaries {
		cpuUsed += resource.NewScaledQuantity(int64(summary.Node.CPU.UsageNanoCores), resource.Nano).
			ScaledValue(resource.Milli)
		// convert bytes to mega bytes (B --> MB)
		memoryUsed += int64(summary.Node.Memory.WorkingSetBytes) / 1024 / 1024
		storageUsed += int64(summary.Node.Fs.UsedBytes) / 1024 / 1024
		podUsed += int64(len(summary.Pods))
	}
	// if all data is 0, then needs to retry
	if cpuTotal+memoryTotal+storageTotal+podTotal+cpuUsed+memoryUsed+storageUsed+podUsed == 0 {
		return defaultValue, errors.New("all info is zero")
	}

	resources := make([]model.Resource, 0, 4)
	resources = append(resources, model.Resource{
		ResourceName: v1.ResourcePods,
		Capacity:     float64(podTotal),
		Used:         float64(podUsed),
		Percentage:   Div(float64(podUsed), float64(podTotal)),
	}, model.Resource{
		ResourceName: v1.ResourceCPU,
		Capacity:     Div(float64(cpuTotal), 1000),
		Used:         Div(float64(cpuUsed), 1000),
		Percentage:   Div(float64(cpuUsed), float64(cpuTotal)),
	}, model.Resource{
		ResourceName: v1.ResourceMemory,
		Capacity:     Div(float64(memoryTotal), 1024),
		Used:         Div(float64(memoryUsed), 1024),
		Percentage:   Div(float64(memoryUsed), float64(memoryTotal)),
	}, model.Resource{
		ResourceName: v1.ResourceStorage,
		Capacity:     Div(float64(storageTotal), 1024),
		Used:         Div(float64(storageUsed), 1024),
		Percentage:   Div(float64(storageUsed), float64(storageTotal)),
	})
	return resources, nil
}

func Div(a float64, b float64) float64 {
	if b == 0 {
		b = 1
	}
	if float, err := strconv.ParseFloat(fmt.Sprintf("%.2f", a/b), 64); err == nil {
		return float
	}
	return 0
}

func isSame(a, b []model.Resource) bool {
	maps := map[v1.ResourceName]model.Resource{}
	for _, m := range a {
		maps[m.ResourceName] = m
	}
	for _, m := range b {
		if v, found := maps[m.ResourceName]; !found || !v.Equals(m) {
			return false
		}
	}
	return true
}
