/*
Copyright 2021 The Nocalhost Authors.
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
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/resource"
	"nocalhost/pkg/nhctl/log"
)

func (c *ClientGoUtils) DeleteResourceInfo(info *resource.Info) error {
	helper := resource.NewHelper(info.Client, info.Mapping)
	propagationPolicy := metav1.DeletePropagationBackground
	obj, err := helper.DeleteWithOptions(info.Namespace, info.Name, &metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		return errors.Wrap(err, "")
	}
	return errors.Wrap(info.Refresh(obj, true), "")
}

func (c *ClientGoUtils) PatchResourceInfo(info *resource.Info) error {
	data, err := runtime.Encode(unstructured.UnstructuredJSONScheme, info.Object)
	if err != nil {
		return errors.Wrap(err, "")
	}

	helper := resource.NewHelper(info.Client, info.Mapping)
	//forceConflicts := true
	obj, err := helper.Patch(info.Namespace, info.Name, types.ApplyPatchType, data, &metav1.PatchOptions{FieldManager: "kubectl"})
	if err != nil {
		return errors.Wrap(err, "")
	}
	return errors.Wrap(info.Refresh(obj, true), "")
}

func (c *ClientGoUtils) CreateResourceInfo(info *resource.Info) error {
	helper := resource.NewHelper(info.Client, info.Mapping)
	obj, err := helper.Create(info.Namespace, true, info.Object)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return errors.Wrap(info.Refresh(obj, true), "")
}

func (c *ClientGoUtils) GetResourceInfoFromFiles(files []string, continueOnError bool) ([]*resource.Info, error) {

	if len(files) == 0 {
		return nil, errors.New("files must not be nil")
	}

	f := c.newFactory()
	builder := f.NewBuilder()
	validate, err := f.Validator(true)
	if err != nil {
		if continueOnError {
			log.Warnf("Build validator err:", err.Error())
		} else {
			return nil, errors.Wrap(err, "")
		}
	}
	filenames := resource.FilenameOptions{
		Filenames: files,
		Kustomize: "",
		Recursive: false,
	}
	if continueOnError {
		builder.ContinueOnError()
	}
	result := builder.Unstructured().
		Schema(validate).
		NamespaceParam(c.namespace).DefaultNamespace().
		FilenameParam(true, &filenames).
		//LabelSelectorParam(o.Selector).
		Flatten().Do()

	//if result == nil {
	//	return nil, errors.New("result is nil")
	//}
	if result.Err() != nil {
		if continueOnError {
			log.WarnE(err, "error occurs in results")
		} else {
			return nil, errors.Wrap(result.Err(), "")
		}
	}

	infos, err := result.Infos()
	if err != nil {
		if continueOnError {
			log.WarnE(err, "error occurs in results")
		} else {
			return nil, errors.Wrap(err, "")
		}
	}

	return infos, nil
	//if len(infos) == 0 {
	//	return nil, errors.New("no result info")
	//}

	//log.Infof("Find %d resources", len(infos))
}
