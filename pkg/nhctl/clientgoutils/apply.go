/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/apply"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/const"
	//"k8s.io/kubectl/pkg/util"
	"nocalhost/pkg/nhctl/log"
	"os"
)

func StandardNocalhostMetas(releaseName, releaseNamespace string) *ApplyFlags {
	return &ApplyFlags{
		MergeableLabel: map[string]string{
			_const.AppManagedByLabel: _const.AppManagedByNocalhost,
		},

		MergeableAnnotation: map[string]string{
			_const.NocalhostApplicationName:      releaseName,
			_const.NocalhostApplicationNamespace: releaseNamespace,
		},
		DoApply: true,
	}
}

type ApplyFlags struct {

	// There is currently no need to delete labels, so similar support is not provided
	MergeableLabel      map[string]string
	MergeableAnnotation map[string]string

	// apply if set to true
	DoApply     bool
	BeforeApply func(string) error
}

func (a *ApplyFlags) SetBeforeApply(fun func(string) error) *ApplyFlags {
	a.BeforeApply = fun
	return a
}

func (a *ApplyFlags) SetDoApply(doApply bool) *ApplyFlags {
	a.DoApply = doApply
	return a
}

func DeleteResourceInfo(info *resource.Info) error {
	helper := resource.NewHelper(info.Client, info.Mapping)
	propagationPolicy := metav1.DeletePropagationBackground
	obj, err := helper.DeleteWithOptions(
		info.Namespace, info.Name, &metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		},
	)
	if err != nil {
		return errors.Wrap(err, "")
	}
	if err := errors.Wrap(info.Refresh(obj, true), ""); err != nil {
		return err
	}

	operation := "deleted"
	groupKind := info.Mapping.GroupVersionKind
	log.Infof("Resource(%s) %s %s", groupKind.Kind, info.Name, operation)
	return nil
}

// Similar to `kubectl apply`, but apply a resourceInfo instead a file
func (c *ClientGoUtils) ApplyResourceInfo(info *resource.Info, af *ApplyFlags) error {
	if af == nil {
		af = &ApplyFlags{}
	}
	o, err := c.generateCompletedApplyOption(af)
	if err != nil {
		return err
	}
	o.SetObjects([]*resource.Info{info})
	o.ToPrinter = func(operation string) (printers.ResourcePrinter, error) {
		o.PrintFlags.NamePrintFlags.Operation = operation
		cmdutil.PrintFlagsWithDryRunStrategy(o.PrintFlags, o.DryRunStrategy)
		return &runtimeObjectPrinter{Operation: operation, Name: info.Name}, nil
	}
	o.IOStreams = genericclioptions.IOStreams{
		In: os.Stdin, Out: os.Stdout, ErrOut: os.Stdout,
	} // don't print log to stderr
	return o.Run()
}

func (c *ClientGoUtils) generateCompletedApplyOption(af *ApplyFlags) (*apply.ApplyOptions, error) {
	var err error
	ioStreams := genericclioptions.IOStreams{
		In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr,
	} // don't print log to stderr
	o := apply.NewApplyOptions(ioStreams)
	o.DeleteFlags.FileNameFlags.Filenames = &[]string{""}
	o.OpenAPIPatch = true

	f := c.NewFactory()
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
		err = resourceList.Visits(
			[]resource.VisitorFunc{
				addLabels(af.MergeableLabel), addAnnotations(af.MergeableAnnotation),
			},
		)
		return nil
	}
	return o, nil
}

// GetResourceInfoFromString Str is in json format (Can be a yaml ?)
func (c *ClientGoUtils) GetResourceInfoFromString(str string, continueOnError bool) ([]*resource.Info, error) {
	return c.GetResourceInfoFromReader(bytes.NewBufferString(str), continueOnError)
}

func (c *ClientGoUtils) GetResourceInfoFromReader(reader io.Reader, continueOnError bool) ([]*resource.Info, error) {

	f := c.NewFactory()
	builder := f.NewBuilder()
	validate, err := f.Validator(true)
	if err != nil {
		if continueOnError {
			log.Warnf("Build validator err: %v", err.Error())
		} else {
			return nil, errors.WithStack(err)
		}
	}
	if continueOnError {
		builder.ContinueOnError()
	}
	result := builder.
		Unstructured().
		Schema(validate).
		NamespaceParam(c.namespace).
		DefaultNamespace().
		Stream(reader, "").
		Flatten().
		Do()

	if result.Err() != nil {
		if continueOnError {
			log.WarnE(errors.Wrap(result.Err(), ""), "error occurs in results")
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

type runtimeObjectPrinter struct {
	Operation string
	Name      string
}

func (r *runtimeObjectPrinter) PrintObj(obj runtime.Object, writer io.Writer) error {
	log.Infof("Resource(%s) %s %s", obj.GetObjectKind().GroupVersionKind().Kind, r.Name, r.Operation)
	return nil
}
