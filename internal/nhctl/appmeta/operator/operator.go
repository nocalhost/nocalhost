/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package operator

import (
	"context"
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_handler/item"
	"nocalhost/internal/nhctl/resouce_cache"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	k8sutil "nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"strings"
)

var ErrorButSkip = "Error while uninstall application but skipped,"

type ClientGoUtilClient struct {
	ClientInner     *clientgoutils.ClientGoUtils
	Dc              dynamic.Interface
	KubeconfigBytes []byte
}

func (cso *ClientGoUtilClient) ExecHook(appName, ns, manifests string) error {
	return cso.ClientInner.ApplyAndWaitFor(
		manifests, true,
		clientgoutils.StandardNocalhostMetas(appName, ns).
			SetDoApply(true).
			SetBeforeApply(nil),
	)
}
func (cso *ClientGoUtilClient) CleanManifest(manifests string) {
	resource := clientgoutils.NewResourceFromStr(manifests)

	//goland:noinspection GoNilness
	infos, err := resource.GetResourceInfo(cso.ClientInner, true)
	utils.ShouldI(err, "Error while loading the manifest able to be deleted: "+manifests)

	for _, info := range infos {
		utils.ShouldI(clientgoutils.DeleteResourceInfo(info), "Failed to delete resource "+info.Name)
	}
}
func (cso *ClientGoUtilClient) Create(ns string, secret *corev1.Secret) (*corev1.Secret, error) {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Create(
		cso.ClientInner.GetContext(), secret, metav1.CreateOptions{},
	)
}

func (cso *ClientGoUtilClient) Get(ns string, secretName string) (*corev1.Secret, error) {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Get(
		cso.ClientInner.GetContext(), secretName, metav1.GetOptions{},
	)
}

func (cso *ClientGoUtilClient) Update(ns string, secret *corev1.Secret) (*corev1.Secret, error) {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Update(
		cso.ClientInner.GetContext(), secret, metav1.UpdateOptions{},
	)
}
func (cso *ClientGoUtilClient) Delete(ns, name string) error {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Delete(
		cso.ClientInner.GetContext(), name, metav1.DeleteOptions{},
	)
}
func (cso *ClientGoUtilClient) GetKubeconfigBytes() []byte {
	return cso.KubeconfigBytes
}

func (cso *ClientGoUtilClient) getCustomResourceDaemon(app, ns string) item.App {
	s, err := resouce_cache.GetSearcherWithLRU(cso.KubeconfigBytes, ns)
	if err != nil {
		log.Infof("Error while uninstall application: %s", err.Error())
		return item.App{}
	}

	result := item.App{Name: app}

	// clean the custom resource with annotations
	for _, entry := range resouce_cache.GroupToTypeMap {
		resources := make([]item.Resource, 0, len(entry.V))
		for _, resource := range entry.V {
			rs := strings.Split(resource, ".")
			if len(rs) < 1 {
				continue
			}
			resource = strings.ToLower(rs[0])
			resourceList, err := s.Criteria().
				ResourceType(resource).
				AppName(app).
				Namespace(ns).
				ShowHidden(true).
				Query()
			if err == nil {
				items := make([]item.Item, 0, len(resourceList))
				for _, v := range resourceList {
					items = append(
						items, item.Item{
							Metadata: v,
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

func (cso *ClientGoUtilClient) getCustomResource(app, ns string) (result item.App) {
	result = item.App{}

	cli, err := daemon_client.GetDaemonClient(utils.IsSudoUser())
	if err != nil {
		log.Infof("%s error while initial daemon cli", ErrorButSkip)
		return
	}

	data, err := cli.SendGetResourceInfoCommand(
		k8sutil.GetOrGenKubeConfigPath(string(cso.KubeconfigBytes)),
		ns, app, "all", "", map[string]string{}, true,
	)

	if err != nil {
		log.Infof("%s error while getting resource info from daemon %s", ErrorButSkip, err)
		return
	}

	if data == nil {
		log.Infof("%s error while getting resource info from daemon, get nothing from daemon", ErrorButSkip)
		return
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		log.Infof("%s error while getting resource info from daemon, marshal fail: %s", ErrorButSkip, err)
		return
	}

	var itemResult item.Result
	_ = json.Unmarshal(bytes, &itemResult)

	for _, appItem := range itemResult.Application {
		if appItem.Name == app {
			return appItem
		}
	}
	return
}

func (cso *ClientGoUtilClient) CleanCustomResource(app, ns string) {
	var applicationPack item.App
	if _const.IsDaemon {
		applicationPack = cso.getCustomResourceDaemon(app, ns)
	} else {
		applicationPack = cso.getCustomResource(app, ns)
	}

	for _, group := range applicationPack.Groups {
		for _, resource := range group.List {
			for _, omItem := range resource.List {
				objectMeta := k8sutil.GetObjectMetaData(omItem.Metadata)

				if objectMeta == nil {
					log.Infof(
						"%s fail to getting object meta from %s-%s",
						ErrorButSkip, group.GroupName, resource.Name,
					)
					continue
				}

				strs := strings.Split(resource.Name, ".")
				resourceType := resource.Name
				if len(strs) > 0 {
					resourceType = strs[0]
				}
				cso.doCleanCustomResource(
					schema.GroupVersionResource{
						Group:    objectMeta.GroupVersionKind().Group,
						Version:  objectMeta.GroupVersionKind().Version,
						Resource: resourceType,
					}, objectMeta.Namespace, objectMeta.GroupVersionKind().Kind, objectMeta.Name,
				)
			}
		}
	}
}

// delete all resources with specify annotations
func (cso *ClientGoUtilClient) doCleanCustomResource(gvr schema.GroupVersionResource, ns, kind, name string) {
	if ns == "" || name == "" {
		return
	}

	if cso.Dc == nil {
		log.Infof("%s dynamic client init fail", ErrorButSkip)
		return
	}

	if err := cso.Dc.Resource(gvr).Namespace(ns).Delete(context.TODO(), name, metav1.DeleteOptions{}); err != nil {
		log.Infof("%s custom resources %s-%s deleting fail, %s", ErrorButSkip, gvr.String(), name, err)
		return
	}

	log.Infof("Resource(%s) %s deleted ", kind, name)
}
