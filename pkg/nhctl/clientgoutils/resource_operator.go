/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/pkg/nhctl/log"
)

func (c *ClientGoUtils) newFactory() cmdutil.Factory {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.KubeConfig = &c.kubeConfigFilePath
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
	return nil
}

func (c *ClientGoUtils) Apply(files []string, continueOnError bool, flags *ApplyFlags, kustomize string) error {
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

	for _, info := range infos {
		_ = c.ApplyResourceInfo(info, flags)
	}
	return nil
}

// useless temporally
func (c *ClientGoUtils) Delete(files []string, continueOnError bool, flags *ApplyFlags, kustomize string) error {
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

	for _, info := range infos {
		_ = c.DeleteResourceInfo(info)
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
