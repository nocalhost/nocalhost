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
	"github.com/jonboulle/clockwork"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/apply"
	//"k8s.io/kubectl/pkg/util"
	"nocalhost/pkg/nhctl/log"
	"os"
	"time"
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

func (c *ClientGoUtils) UpdateResourceInfoByServerSide(info *resource.Info) error {
	data, err := runtime.Encode(unstructured.UnstructuredJSONScheme, info.Object)
	if err != nil {
		return errors.Wrap(err, "")
	}

	helper := resource.NewHelper(info.Client, info.Mapping)
	forceConflicts := true
	obj, err := helper.Patch(info.Namespace, info.Name, types.ApplyPatchType, data, &metav1.PatchOptions{Force: &forceConflicts, FieldManager: "kubectl"})
	if err != nil {
		return errors.Wrap(err, "")
	}
	return errors.Wrap(info.Refresh(obj, true), "")
}

func (c *ClientGoUtils) UpdateResourceInfoByClientSide(info *resource.Info) error {
	f := c.newFactory()
	helper := resource.NewHelper(info.Client, info.Mapping)
	openAPISchema, err := f.OpenAPISchema()
	if err != nil {
		return errors.Wrap(err, "")
	}
	modified, err := runtime.Encode(unstructured.UnstructuredJSONScheme, info.Object)
	//modified, err := util.GetModifiedConfiguration(info.Object, true, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return errors.Wrap(err, "")
	}
	patcher := &apply.Patcher{
		Mapping:           info.Mapping,
		Helper:            helper,
		Overwrite:         true,
		BackOff:           clockwork.NewRealClock(),
		Force:             true,
		CascadingStrategy: metav1.DeletePropagationBackground,
		Timeout:           time.Hour,
		GracePeriod:       -1,
		OpenapiSchema:     openAPISchema,
		Retries:           3,
	}

	_, _, err = patcher.Patch(info.Object, modified, info.Source, info.Namespace, info.Name, os.Stderr)
	return errors.Wrap(err, "")
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
