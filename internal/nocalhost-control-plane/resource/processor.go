/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package resource

import (
	"context"
	"fmt"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/log"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"go.uber.org/zap"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"nocalhost/internal/nocalhost-control-plane/common"
	"nocalhost/internal/nocalhost-control-plane/k8s"
	"nocalhost/internal/nocalhost-control-plane/pkg/util"
	"strings"
	"sync"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	corev1 "k8s.io/client-go/informers/core/v1"
	k8scache "k8s.io/client-go/tools/cache"
)

type meshInfo struct {
	uuid string
	// headerKey+headerValue --> route info
	dev    map[headerPair]*routeInfo
	origin *routeInfo
}

type headerPair struct {
	headerKey string
	headerVal string
}

type routeInfo struct {
	name      sets.String
	port      sets.Int32
	endpoints sets.String
}

type Processor struct {
	mu          sync.RWMutex
	PodInformer corev1.PodInformer
	Snapshot    cache.SnapshotCache
	Log         log.Logger
}

func (s *Processor) Start(ctx context.Context) {
	s.Log.Debugf("start snapshot")
	s.Log.Debugf("start pod informer")
	factory := informers.NewSharedInformerFactoryWithOptions(k8s.ClientSet, time.Second*10, informers.WithNamespace(k8s.Namespace))
	s.PodInformer = factory.Core().V1().Pods()
	s.PodInformer.Informer().AddEventHandler(k8scache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			switch t := obj.(type) {
			case *v1.Pod:
				return t.GetAnnotations() != nil && t.GetAnnotations()[common.MeshKey] == "true"
			default:
				return false
			}
		},
		Handler: k8scache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				s.Log.Debugf("add pod: %s", pod.Name)
				if util.IsPodReady(pod) {
					if err := s.parse(ctx, pod.GetAnnotations()[common.MeshUUIDKEY]); err != nil {
						s.Log.Errorf("parse snapshot with pod %s error: %v", pod.Name, err)
					}
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				pod := newObj.(*v1.Pod)
				s.Log.Debugf("update pod: %s", pod.Name)
				if util.IsPodReady(pod) {
					if err := s.parse(ctx, pod.GetAnnotations()[common.MeshUUIDKEY]); err != nil {
						s.Log.Errorf("parse snapshot with pod %s error: %v", pod.Name, err)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				s.Log.Debugf("delete pod: %s", pod.Name)
				if util.IsPodReady(pod) {
					s.Log.Debugf("the pod deleted: %s", pod.Name)
					s.Log.Infof("clean snapshot for: %s", pod.Name)
					s.Snapshot.ClearSnapshot(pod.Name)
					if err := s.parse(ctx, pod.GetAnnotations()[common.MeshUUIDKEY]); err != nil {
						s.Log.Errorf("parse snapshot with pod %s error: %v", pod.Name, err)
					}
				}
			},
		},
	})
	factory.Start(ctx.Done())
	factory.WaitForCacheSync(ctx.Done())
}

func listPods(uuid string) (devPods []v1.Pod, originPods []v1.Pod, err error) {
	list, err := k8s.ClientSet.CoreV1().Pods(k8s.Namespace).List(context.Background(), v12.ListOptions{})
	if err != nil {
		return
	}
	for _, pod := range list.Items {
		if anno := pod.GetAnnotations(); anno != nil && anno[common.MeshUUIDKEY] == uuid {
			if anno[common.MeshTypeKEY] == common.MeshDevType {
				devPods = append(devPods, pod)
			}
			if anno[common.MeshTypeKEY] == common.MeshOriginType {
				originPods = append(originPods, pod)
			}
		}
	}
	return
}

func (s *Processor) getMeshInfo(uuid string) (meshInfo, error) {
	s.Log.Debugf("get mesh info by uuid: %s", uuid)
	info := meshInfo{
		uuid: uuid,
		dev:  map[headerPair]*routeInfo{},
		origin: &routeInfo{
			name:      sets.NewString(),
			port:      sets.NewInt32(),
			endpoints: sets.NewString(),
		},
	}

	devPods, originPods, err := listPods(info.uuid)
	if err != nil {
		return info, err
	}
	for _, pod := range devPods {
		if pod.DeletionTimestamp != nil {
			continue
		}
		h := headerPair{
			headerKey: pod.Annotations[common.MeshHeaderKey],
			headerVal: pod.Annotations[common.MeshHeaderVal],
		}
		ports := sets.NewInt32()
		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				ports.Insert(port.ContainerPort)
			}
		}
		if v, found := info.dev[h]; found {
			v.name.Insert(pod.Name)
			v.endpoints.Insert(pod.Status.PodIP)
			v.port.Insert(ports.List()...)
		} else {
			info.dev[h] = &routeInfo{
				name:      sets.NewString(pod.Name),
				port:      sets.NewInt32(ports.List()...),
				endpoints: sets.NewString(pod.Status.PodIP),
			}
		}
	}

	for _, pod := range originPods {
		if pod.DeletionTimestamp != nil {
			continue
		}
		ports := sets.NewInt32()
		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				ports.Insert(port.ContainerPort)
			}
		}
		info.origin.name.Insert(pod.GetName())
		info.origin.endpoints.Insert(pod.Status.PodIP)
		info.origin.port.Insert(ports.List()...)
	}

	var sb strings.Builder
	sb.WriteString("dev:\n")
	for pair, r := range info.dev {
		sb.WriteString(fmt.Sprintf("%s=%s, pod name: %#v\n", pair.headerKey, pair.headerVal, *r))
	}
	sb.WriteString("origin:\n")
	sb.WriteString(fmt.Sprintf("%#v", *info.origin))
	s.Log.Debugf("the mesh info: \n%s", sb.String())
	return info, nil
}

func (s *Processor) parse(ctx context.Context, uuid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := s.getMeshInfo(uuid)
	if err != nil {
		return err
	}

	for _, name := range info.origin.name.List() {
		listeners := make([]types.Resource, 0)
		routers := make([]types.Resource, 0)
		clusters := make([]types.Resource, 0)
		endpoints := make([]types.Resource, 0)
		for _, port := range info.origin.port.List() {
			unique := fmt.Sprintf("%s-%v", name, port)
			listeners = append(listeners, buildListener(unique, uint32(port)))
			var rr []*route.Route
			for pair, routes := range info.dev {
				headerPort := fmt.Sprintf("%s-%s-%v", pair.headerKey, pair.headerVal, port)
				clusters = append(clusters, buildCluster(headerPort))
				endpoints = append(endpoints, buildEndpoint(headerPort, routes.endpoints.List(), uint32(port)))
				rr = append(rr, ToRoute(headerPort, map[string]string{pair.headerKey: pair.headerVal}))
			}
			rr = append(rr, defaultRoute(common.PassthroughCluster))
			routers = append(routers, &route.RouteConfiguration{
				Name: unique,
				VirtualHosts: []*route.VirtualHost{
					{
						Name:    "local_service",
						Domains: []string{"*"},
						Routes:  rr,
					},
				},
			})
		}

		s.Log.Infof("parse snapshot for: %s", name)
		snapshot, err := cache.NewSnapshot(time.Now().String(), map[resource.Type][]types.Resource{
			resource.ListenerType: listeners,
			resource.RouteType:    routers,
			resource.ClusterType:  clusters,
			resource.EndpointType: endpoints,
		})
		if err != nil {
			s.Log.Errorf("%v", err)
			continue
		}
		if err = s.Snapshot.SetSnapshot(ctx, name, snapshot); err != nil {
			s.Log.Errorf("%v", err)
			continue
		}
	}

	for _, r := range info.dev {
		for _, name := range r.name.List() {

			listeners := make([]types.Resource, 0)
			routers := make([]types.Resource, 0)
			clusters := make([]types.Resource, 0)
			endpoints := make([]types.Resource, 0)
			for _, port := range r.port.List() {
				unique := fmt.Sprintf("%s-%v", name, port)
				listeners = append(listeners, buildListener(unique, uint32(port)))
				var rr []*route.Route
				for pair, routes := range info.dev {
					headerPort := fmt.Sprintf("%s-%s-%v", pair.headerKey, pair.headerVal, port)
					clusters = append(clusters, buildCluster(headerPort))
					endpoints = append(endpoints, buildEndpoint(headerPort, routes.endpoints.List(), uint32(port)))
					rr = append(rr, ToRoute(headerPort, map[string]string{pair.headerKey: pair.headerVal}))
				}
				clusters = append(clusters, buildCluster("origin"))
				endpoints = append(endpoints, buildEndpoint("origin", info.origin.endpoints.List(), uint32(port)))
				rr = append(rr, defaultRoute("origin"))
				routers = append(routers, &route.RouteConfiguration{
					Name: unique,
					VirtualHosts: []*route.VirtualHost{
						{
							Name:    "local_service",
							Domains: []string{"*"},
							Routes:  rr,
						},
					},
				})
			}

			s.Log.Infof("parse snapshot for: %s", name)
			snapshot, err := cache.NewSnapshot(time.Now().String(), map[resource.Type][]types.Resource{
				resource.ListenerType: listeners,
				resource.RouteType:    routers,
				resource.ClusterType:  clusters,
				resource.EndpointType: endpoints,
			})
			if err != nil {
				s.Log.Errorf("%v", err)
				continue
			}
			if err := s.Snapshot.SetSnapshot(ctx, name, snapshot); err != nil {
				s.Log.Errorf("%v", err)
				continue
			}
		}
	}
	return nil
}

func NewProcessor(logger log.Logger) *Processor {
	snapshot := &Processor{
		Snapshot: cache.NewSnapshotCache(true, cache.IDHash{}, nil),
	}
	if logger == nil {
		logTemp, _ := zap.NewDevelopment()
		snapshot.Log = logTemp.Sugar()
	} else {
		snapshot.Log = logger
	}
	return snapshot
}
