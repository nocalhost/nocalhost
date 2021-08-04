/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
)

var accessor = meta.NewAccessor()

//const (
//	AppManagedByLabel                   = "app.kubernetes.io/managed-by"
//	AppManagedByNocalhost               = "nocalhost"
//	NocalhostReleaseNameAnnotation      = "meta.nocalhost.sh/release-name"
//	NocalhostReleaseNamespaceAnnotation = "meta.nocalhost.sh/release-namespace"
//)

//func checkOwnership(obj runtime.Object, releaseName, releaseNamespace string) error {
//	lbls, err := accessor.Labels(obj)
//	if err != nil {
//		return err
//	}
//	annos, err := accessor.Annotations(obj)
//	if err != nil {
//		return err
//	}
//
//	var errs []error
//	if err := requireValue(lbls, AppManagedByLabel, AppManagedByNocalhost); err != nil {
//		errs = append(errs, fmt.Errorf("label validation error: %s", err))
//	}
//	if err := requireValue(annos, NocalhostReleaseNameAnnotation, releaseName); err != nil {
//		errs = append(errs, fmt.Errorf("annotation validation error: %s", err))
//	}
//	if err := requireValue(annos, NocalhostReleaseNamespaceAnnotation, releaseNamespace); err != nil {
//		errs = append(errs, fmt.Errorf("annotation validation error: %s", err))
//	}
//
//	if len(errs) > 0 {
//		err := errors.New("invalid ownership metadata")
//		for _, e := range errs {
//			err = fmt.Errorf("%w; %s", err, e)
//		}
//		return err
//	}
//
//	return nil
//}

func requireValue(meta map[string]string, k, v string) error {
	actual, ok := meta[k]
	if !ok {
		return fmt.Errorf("missing key %q: must be set to %q", k, v)
	}
	if actual != v {
		return fmt.Errorf("key %q must equal %q: current value is %q", k, v, actual)
	}
	return nil
}

func AddMetas(ut *unstructured.Unstructured, flags *ApplyFlags) {
	ut.SetAnnotations(mergeStrStrMaps(ut.GetAnnotations(), flags.MergeableAnnotation))
	ut.SetLabels(mergeStrStrMaps(ut.GetLabels(), flags.MergeableLabel))
}

func addLabels(labels map[string]string) resource.VisitorFunc {
	return func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}

		if err := mergeLabels(info.Object, labels); err != nil {
			return fmt.Errorf(
				"%s labels could not be updated: %s",
				resourceString(info), err,
			)
		}
		return nil
	}
}

func addAnnotations(annotations map[string]string) resource.VisitorFunc {
	return func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}

		if err := mergeAnnotations(info.Object, annotations); err != nil {
			return fmt.Errorf(
				"%s annotations could not be updated: %s",
				resourceString(info), err,
			)
		}

		return nil
	}
}

func resourceString(info *resource.Info) string {
	_, k := info.Mapping.GroupVersionKind.ToAPIVersionAndKind()
	return fmt.Sprintf(
		"%s %q in namespace %q",
		k, info.Name, info.Namespace,
	)
}

func mergeLabels(obj runtime.Object, labels map[string]string) error {
	current, err := accessor.Labels(obj)
	if err != nil {
		return err
	}
	return accessor.SetLabels(obj, mergeStrStrMaps(current, labels))
}

func mergeAnnotations(obj runtime.Object, annotations map[string]string) error {
	current, err := accessor.Annotations(obj)
	if err != nil {
		return err
	}
	return accessor.SetAnnotations(obj, mergeStrStrMaps(current, annotations))
}

// merge two maps, always taking the value on the right
func mergeStrStrMaps(current, desired map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range current {
		result[k] = v
	}
	for k, desiredVal := range desired {
		result[k] = desiredVal
	}
	return result
}
