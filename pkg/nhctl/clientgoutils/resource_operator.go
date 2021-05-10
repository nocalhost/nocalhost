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

package clientgoutils

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/pkg/nhctl/log"
)

func (c *ClientGoUtils) NewFactory() cmdutil.Factory {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.KubeConfig = &c.kubeConfigFilePath
	kubeConfigFlags.Namespace = &c.namespace
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)
	return f
}

func (c *ClientGoUtils) ApplyAndWait(files []string, continueOnError bool, flags *ApplyFlags) error {
	reader := NewManifestResourceReader(files)

	loadResource, err := reader.LoadResource()
	if err != nil && !continueOnError {
		return err
	}

	//goland:noinspection GoNilness
	if flags != nil && flags.BeforeApply != nil {
		if err := (flags.BeforeApply)(loadResource.String()); err != nil {
			return err
		}
	}

	if flags != nil && flags.DoApply {
		//goland:noinspection GoNilness
		for _, manifest := range loadResource.arr() {
			if err = c.applyUnstructuredResource(manifest, true, flags); err != nil {
				if !continueOnError {
					return err
				}

				if te, ok := err.(*TypedError); ok {
					log.Warnf("Invalid yaml: %s", te.Mes)
				} else {
					log.Warnf("Fail to install manifest : %s", err.Error())
				}
			}
		}
	}
	return nil
}

func (c *ClientGoUtils) Apply(files []string, continueOnError bool, flags *ApplyFlags, kustomize string) error {

	return c.renderManifestAndThen(
		files, continueOnError, flags, kustomize,
		func(c *ClientGoUtils, r *resource.Info) error {
			return c.ApplyResourceInfo(r, flags)
		},
	)
}

// useless temporally
func (c *ClientGoUtils) Delete(files []string, continueOnError bool, flags *ApplyFlags, kustomize string) error {

	return c.renderManifestAndThen(
		files, continueOnError, flags, kustomize,
		func(c *ClientGoUtils, r *resource.Info) error {

			// for now the apply flag used to adding annotations
			// while apply resource
			// delete resource need not to do that
			return c.DeleteResourceInfo(r)
		},
	)
}

// useless temporally
func (c *ClientGoUtils) renderManifestAndThen(
	files []string, continueOnError bool, flags *ApplyFlags,
	kustomize string, doForResourceInfo func(c *ClientGoUtils, r *resource.Info) error,
) error {
	var reader ResourceReader

	if kustomize == "" {
		reader = NewManifestResourceReader(files)
	} else {
		reader = NewKustomizeResourceReader(kustomize)
	}

	loadResource, err := reader.LoadResource()
	if err != nil && !continueOnError {
		return err
	}

	//goland:noinspection GoNilness
	infos, err := loadResource.GetResourceInfo(c, continueOnError)
	if err != nil {
		log.Logf("Error while resolve [ResourceInfo] from [Info] %s, err:%s", loadResource, err)
		if !continueOnError {
			return err
		}
	}

	if flags != nil && flags.BeforeApply != nil {
		if err := (flags.BeforeApply)(loadResource.String()); err != nil {
			return err
		}
	}

	if flags != nil && flags.DoApply {
		for _, info := range infos {
			if err := doForResourceInfo(c, info); err != nil && !continueOnError {
				return errors.Wrap(err, "Error while apply resourceInfo")
			}
		}
	}
	return nil
}

func (c *ClientGoUtils) applyUnstructuredResource(manifest string, wait bool, flags *ApplyFlags) error {
	obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode([]byte(manifest), nil, nil)
	if err != nil {
		return &TypedError{ErrorType: InvalidYaml, Mes: err.Error()}
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		//return errors.Wrap(err, fmt.Sprintf("[Invalid Yaml] fail to parse resource obj"))
		return &TypedError{ErrorType: InvalidYaml, Mes: err.Error()}
	}

	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

	gr, err := restmapper.GetAPIGroupResources(c.ClientSet.Discovery())
	if err != nil {
		return errors.Wrap(err, "")
	}

	mapper := restmapper.NewDiscoveryRESTMapper(gr)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return errors.Wrap(err, "")
	}

	var dri dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		unstructuredObj.SetNamespace(c.namespace)
		dri = c.GetDynamicClient().Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
	} else {
		dri = c.GetDynamicClient().Resource(mapping.Resource)
	}

	AddMetas(unstructuredObj, flags)

	obj2, err := dri.Create(c.ctx, unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("fail to create %s", unstructuredObj.GetName()))
	}

	log.Infof("%s %s created", obj2.GetKind(), obj2.GetName())

	if wait {
		err = c.WaitJobToBeReady(obj2.GetName(), "metadata.name")
		if err != nil {
			//PrintlnErr("fail to wait", err)
			return err
		}
	}
	return nil
}
