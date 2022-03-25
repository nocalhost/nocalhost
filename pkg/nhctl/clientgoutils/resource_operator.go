/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
	"strings"
)

func (c *ClientGoUtils) NewFactory() cmdutil.Factory {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	kubeConfigFlags.KubeConfig = &c.kubeConfigFilePath
	kubeConfigFlags.Namespace = &c.namespace
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)
	return f
}

func (c *ClientGoUtils) ApplyAndWaitFor(manifests string, continueOnError bool, flags *ApplyFlags) error {
	return c.doApplyAndWait(NewResourceFromStr(manifests), continueOnError, flags)
}

func (c *ClientGoUtils) ApplyAndWait(files []string, continueOnError bool, flags *ApplyFlags) error {
	reader := NewManifestResourceReader(files)

	loadResource, err := reader.LoadResource()
	if err != nil && !continueOnError {
		return err
	}

	return c.doApplyAndWait(loadResource, continueOnError, flags)
}

func (c *ClientGoUtils) doApplyAndWait(resource *Resource, continueOnError bool, flags *ApplyFlags) error {
	//goland:noinspection GoNilness
	if flags != nil && flags.BeforeApply != nil {
		if err := (flags.BeforeApply)(resource.String()); err != nil {
			return err
		}
	}

	if flags != nil && flags.DoApply {
		//goland:noinspection GoNilness
		for _, manifest := range resource.arr() {
			if manifest == "" || strings.Trim(manifest, " ") == "" {
				continue
			}

			if err := c.applyUnstructuredResource(manifest, true, flags); err != nil {
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
			return DeleteResourceInfo(r)
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
