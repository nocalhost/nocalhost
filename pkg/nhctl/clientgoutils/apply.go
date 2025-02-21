/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"bytes"
	"io"
	"os"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/openapi3"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/apply"
	cmddelete "k8s.io/kubectl/pkg/cmd/delete"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/prune"

	"nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/log"
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
	o.IOStreams = genericiooptions.IOStreams{
		In: os.Stdin, Out: os.Stdout, ErrOut: os.Stdout,
	} // don't print log to stderr
	return o.Run()
}

func (c *ClientGoUtils) generateCompletedApplyOption(af *ApplyFlags) (*apply.ApplyOptions, error) {
	var err error
	ioStreams := genericiooptions.IOStreams{
		In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr,
	} // don't print log to stderr

	f := c.NewFactory()
	recordFlags := genericclioptions.NewRecordFlags()
	deleteFlags := cmddelete.NewDeleteFlags("The files that contain the configurations to apply.")
	deleteFlags.FileNameFlags.Filenames = &[]string{""}
	printFlags := genericclioptions.NewPrintFlags("created").WithTypeSetter(scheme.Scheme)

	// form flags.ToOptions()

	dryRunStrategy := cmdutil.DryRunNone

	dynamicClient, err := f.DynamicClient()
	if err != nil {
		return nil, err
	}

	// allow for a success message operation to be specified at print time
	toPrinter := func(operation string) (printers.ResourcePrinter, error) {
		printFlags.NamePrintFlags.Operation = operation
		cmdutil.PrintFlagsWithDryRunStrategy(printFlags, dryRunStrategy)
		return printFlags.ToPrinter()
	}

	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return nil, err
	}

	deleteOptions, err := deleteFlags.ToOptions(dynamicClient, ioStreams)
	if err != nil {
		return nil, err
	}

	err = deleteOptions.FilenameOptions.RequireFilenameOrKustomize()
	if err != nil {
		return nil, err
	}

	var openAPIV3Root openapi3.Root
	if !cmdutil.OpenAPIV3Patch.IsDisabled() {
		openAPIV3Client, err := f.OpenAPIV3Client()
		if err == nil {
			openAPIV3Root = openapi3.NewRoot(openAPIV3Client)
		} else {
			klog.V(4).Infof("warning: OpenAPI V3 Patch is enabled but is unable to be loaded. Will fall back to OpenAPI V2")
		}
	}

	validationDirective := metav1.FieldValidationStrict
	validator, err := f.Validator(validationDirective)
	if err != nil {
		return nil, err
	}
	builder := f.NewBuilder()
	mapper, err := f.ToRESTMapper()
	if err != nil {
		return nil, err
	}

	namespace, enforceNamespace, err := f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}

	o := &apply.ApplyOptions{

		PrintFlags: printFlags,

		DeleteOptions:   deleteOptions,
		ToPrinter:       toPrinter,
		ServerSideApply: false,
		ForceConflicts:  false,
		FieldManager:    apply.FieldManagerClientSideApply,
		Selector:        "",
		DryRunStrategy:  dryRunStrategy,
		Prune:           false,
		PruneResources:  []prune.Resource{},
		All:             false,
		Overwrite:       true,
		OpenAPIPatch:    true,
		Subresource:     "",

		Recorder:            recorder,
		Namespace:           namespace,
		EnforceNamespace:    enforceNamespace,
		Validator:           validator,
		ValidationDirective: validationDirective,
		Builder:             builder,
		Mapper:              mapper,
		DynamicClient:       dynamicClient,
		OpenAPIGetter:       f,
		OpenAPIV3Root:       openAPIV3Root,

		IOStreams: ioStreams,

		VisitedUids:       sets.New[types.UID](),
		VisitedNamespaces: sets.New[string](),
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
	validate, err := f.Validator(metav1.FieldValidationStrict)
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
