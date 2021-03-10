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
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/apply"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"strings"

	//"k8s.io/kubectl/pkg/util"
	"nocalhost/pkg/nhctl/log"
	"os"
)

type ApplyFlags struct {

	// There is currently no need to delete labels, so similar support is not provided
	MergeableLabel      map[string]string
	MergeableAnnotation map[string]string
}

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
	obj, err := helper.Patch(info.Namespace, info.Name, types.StrategicMergePatchType, data, &metav1.PatchOptions{Force: &forceConflicts, FieldManager: "kubectl"})
	if err != nil {
		return errors.Wrap(err, "")
	}
	return errors.Wrap(info.Refresh(obj, true), "")
}

//func (c *ClientGoUtils) CreateResourceInfo(info *resource.Info) error {
//	helper := resource.NewHelper(info.Client, info.Mapping)
//	obj, err := helper.Create(info.Namespace, true, info.Object)
//	if err != nil {
//		return errors.Wrap(err, "")
//	}
//	return errors.Wrap(info.Refresh(obj, true), "")
//}

// Similar to `kubectl apply`, but apply a resourceInfo instead a file
func (c *ClientGoUtils) ApplyResourceInfo(info *resource.Info, af *ApplyFlags) error {
	o, err := c.generateCompletedApplyOption("", af)
	if err != nil {
		return err
	}
	o.SetObjects([]*resource.Info{info})
	o.ToPrinter = func(operation string) (printers.ResourcePrinter, error) {
		o.PrintFlags.NamePrintFlags.Operation = operation
		cmdutil.PrintFlagsWithDryRunStrategy(o.PrintFlags, o.DryRunStrategy)
		return &runtimeObjectPrinter{Operation: operation, Name: info.Name}, nil
	}
	o.IOStreams = genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stdout} // don't print log to stderr
	return o.Run()
}

// Similar to `kubectl apply -f `
func (c *ClientGoUtils) Apply(file string, af *ApplyFlags) error {
	o, err := c.generateCompletedApplyOption(file, af)

	if err != nil {
		return err
	}
	return o.Run()
}

func (c *ClientGoUtils) generateCompletedApplyOption(file string, af *ApplyFlags, ) (*apply.ApplyOptions, error) {
	var err error
	ioStreams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr} // don't print log to stderr
	o := apply.NewApplyOptions(ioStreams)
	o.DeleteFlags.FileNameFlags.Filenames = &[]string{file}
	o.OpenAPIPatch = true

	f := c.newFactory()
	// From o.Complete
	o.ServerSideApply = false
	o.ForceConflicts = false
	o.DryRunStrategy = cmdutil.DryRunNone
	o.DynamicClient, err = f.DynamicClient()
	if err != nil {
		return nil, err
	}
	discoveryClient, err := f.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	o.DryRunVerifier = resource.NewDryRunVerifier(o.DynamicClient, discoveryClient)
	o.FieldManager = apply.FieldManagerClientSideApply
	o.DeleteOptions, err = o.DeleteFlags.ToOptions(o.DynamicClient, o.IOStreams)
	if err != nil {
		return nil, err
	}
	err = o.DeleteOptions.FilenameOptions.RequireFilenameOrKustomize()
	if err != nil {
		return nil, err
	}
	o.OpenAPISchema, _ = f.OpenAPISchema()
	o.Validator, err = f.Validator(true)
	if err != nil {
		return nil, err
	}
	o.Builder = f.NewBuilder()
	o.Mapper, err = f.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	o.Namespace, o.EnforceNamespace, err = f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}
	o.ToPrinter = func(operation string) (printers.ResourcePrinter, error) {
		o.PrintFlags.NamePrintFlags.Operation = operation
		cmdutil.PrintFlagsWithDryRunStrategy(o.PrintFlags, o.DryRunStrategy)
		return o.PrintFlags.ToPrinter()
	}

	// injection for the objects before apply
	o.PreProcessorFn = func() error {
		var resourceList ResourceList

		resourceList, err := o.GetObjects()
		if err != nil {
			return err
		}

		// inject nocalhost label and annotations
		if af != nil {
			err = resourceList.Visits([]resource.VisitorFunc{addLabels(af.MergeableLabel), addAnnotations(af.MergeableAnnotation)})
		}
		return nil
	}
	return o, nil
}

type runtimeObjectPrinter struct {
	Operation string
	Name      string
}

func (r *runtimeObjectPrinter) PrintObj(obj runtime.Object, writer io.Writer) error {
	log.Infof("Resource(%s) %s %s", obj.GetObjectKind().GroupVersionKind().Kind, r.Name, r.Operation)
	return nil
}

func (c *ClientGoUtils) GetResourceFromIo(reader io.Reader, validate bool) ([]*resource.Info, error) {
	f := c.newFactory()
	builder := f.NewBuilder()
	v, err := f.Validator(validate)
	if err != nil {
		return nil, err
	}
	result, err := builder.
		Unstructured().
		Schema(v).
		ContinueOnError().
		NamespaceParam(c.namespace).DefaultNamespace().
		Stream(reader, "").
		Flatten().
		Do().
		Infos()
	return result, scrubValidationError(err)
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
		ContinueOnError().
		NamespaceParam(c.namespace).DefaultNamespace().
		FilenameParam(true, &filenames).
		//LabelSelectorParam(o.Selector).
		Flatten().Do()

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
}

// scrubValidationError removes kubectl info from the message.
func scrubValidationError(err error) error {
	if err == nil {
		return nil
	}
	const stopValidateMessage = "if you choose to ignore these errors, turn validation off with --validate=false"

	if strings.Contains(err.Error(), stopValidateMessage) {
		return errors.New(strings.ReplaceAll(err.Error(), "; "+stopValidateMessage, ""))
	}
	return err
}
