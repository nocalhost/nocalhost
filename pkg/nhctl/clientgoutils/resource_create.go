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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"strings"

	"nocalhost/pkg/nhctl/log"
)

func (c *ClientGoUtils) newFactory() cmdutil.Factory {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.KubeConfig = &c.kubeConfigFilePath
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)
	return f
}

func (c *ClientGoUtils) ApplyForCreate(files []string, continueOnError bool) error {
	return c.apply(files, continueOnError, Create)
}

func (c *ClientGoUtils) ApplyForDelete(files []string, continueOnError bool) error {
	return c.apply(files, continueOnError, Delete)
}

type applyAction string

const (
	Delete applyAction = "Delete"
	Create applyAction = "Create"
)

func (c *ClientGoUtils) apply(files []string, continueOnError bool, action applyAction) error {
	if len(files) == 0 {
		return errors.New("files must not be nil")
	}

	f := c.newFactory()
	builder := f.NewBuilder()
	validate, err := f.Validator(true)
	if err != nil {
		if continueOnError {
			log.Warnf("build validator err:", err.Error())
		} else {
			return errors.Wrap(err, "")
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

	if result == nil {
		return errors.New("result is nil")
	}
	if result.Err() != nil {
		if continueOnError {
			log.WarnE(err, "error occurs in results")
		} else {
			return errors.Wrap(result.Err(), "")
		}
	}

	infos, err := result.Infos()
	if err != nil {
		if continueOnError {
			log.WarnE(err, "error occurs in results")
		} else {
			return errors.Wrap(err, "")
		}
	}

	if len(infos) == 0 {
		return errors.New("no result info")
	}

	log.Infof("%s %d resources", action, len(infos))
	for _, info := range infos {
		helper := resource.NewHelper(info.Client, info.Mapping)
		var obj runtime.Object
		if action == Create {
			obj, err = helper.Create(info.Namespace, true, info.Object)
		} else if action == Delete {
			propagationPolicy := metav1.DeletePropagationBackground
			obj, err = helper.DeleteWithOptions(info.Namespace, info.Name, &metav1.DeleteOptions{
				PropagationPolicy: &propagationPolicy,
			})
		}
		if err != nil {
			if continueOnError {
				log.WarnE(err, fmt.Sprintf("Failed to %s resource %s: %s", strings.ToLower(string(action)), info.Name, err.Error()))
				continue
			}
			return errors.Wrap(err, "")
		}
		info.Refresh(obj, true)
		if action == Create {
			log.Infof("Resource(%s) %s %s", info.Object.GetObjectKind().GroupVersionKind().Kind, info.Name, "created")
		} else if action == Delete {
			log.Infof("Resource(%s) %s %s", info.Object.GetObjectKind().GroupVersionKind().Kind, info.Name, "deleted")
		}
	}
	return nil
}
