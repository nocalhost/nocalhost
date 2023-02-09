/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/scheme"
	"nocalhost/pkg/nhctl/log"
	"reflect"
)

func EncodeToJSON(obj runtime.Unstructured) ([]byte, error) {
	serialization, err := runtime.Encode(unstructured.UnstructuredJSONScheme, obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	js, err := yaml.ToJSON(serialization)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return js, nil
}

func PatchDiff(originalInfo, info *resource.Info) error {

	originalJS, err := EncodeToJSON(originalInfo.Object.(runtime.Unstructured))
	if err != nil {
		return err
	}

	editedJS, err := EncodeToJSON(info.Object.(runtime.Unstructured))
	if err != nil {
		return err
	}

	if reflect.DeepEqual(originalJS, editedJS) {
		// no edit, so just skip it.
		return nil
	}

	preconditions := []mergepatch.PreconditionFunc{
		mergepatch.RequireKeyUnchanged("apiVersion"),
		mergepatch.RequireKeyUnchanged("kind"),
		mergepatch.RequireMetadataKeyUnchanged("name"),
		mergepatch.RequireKeyUnchanged("managedFields"),
	}

	// Create the versioned struct from the type defined in the mapping
	// (which is the API version we'll be submitting the patch to)
	versionedObject, err := scheme.Scheme.New(info.Mapping.GroupVersionKind)
	var patchType types.PatchType
	var patch []byte
	switch {
	case runtime.IsNotRegisteredError(err):
		// fall back to generic JSON merge patch
		patchType = types.MergePatchType
		patch, err = jsonpatch.CreateMergePatch(originalJS, editedJS)
		if err != nil {
			return errors.WithStack(err)
		}
		for _, precondition := range preconditions {
			if !precondition(patch) {
				return fmt.Errorf("%s", "At least one of apiVersion, kind and name was changed")
			}
		}
	case err != nil:
		return err
	default:
		patchType = types.StrategicMergePatchType
		patch, err = strategicpatch.CreateTwoWayMergePatch(originalJS, editedJS, versionedObject, preconditions...)
		if err != nil {
			if mergepatch.IsPreconditionFailed(err) {
				return fmt.Errorf("%s", "At least one of apiVersion, kind and name was changed")
			}
			return err
		}
	}

	log.Infof("Patch: %s", string(patch))

	patched, err := resource.NewHelper(info.Client, info.Mapping).
		//WithFieldManager(o.FieldManager). // ???
		Patch(info.Namespace, info.Name, patchType, patch, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	return info.Refresh(patched, true)
}
