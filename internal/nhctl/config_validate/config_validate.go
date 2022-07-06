/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package config_validate

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
	"reflect"
	"strings"
)

var (
	DNS1123      = "DNS1123"
	WorkLoads    = "WorkLoads"
	SyncType     = "SyncType"
	SyncMode     = "SyncMode"
	Quantity     = "Quantity"
	StorageClass = "StorageClass"
	PortForward  = "PortForward"
	Port         = "Port"
	Container    = "Container"
	Language     = "Language"

	SUPPORT_SC = "NOCALHOST_SUPPORT_SC"
	CONTAINERS = "NOCALHOST_CONTAINERS"
	validate   = validator.New()
)

func init() {
	_ = validate.RegisterValidationWithErrorMsg(DNS1123, IsDNS1123Label)
	_ = validate.RegisterValidationWithErrorMsg(WorkLoads, IsSupportsWorkLoads)
	_ = validate.RegisterValidationWithErrorMsg(SyncType, IsSyncType)
	_ = validate.RegisterValidationWithErrorMsg(SyncMode, IsSyncMode)
	_ = validate.RegisterValidationWithErrorMsg(Quantity, IsQuantity)
	_ = validate.RegisterValidationWithErrorMsg(StorageClass, StorageClassSupported)
	_ = validate.RegisterValidationWithErrorMsg(PortForward, PortForwardCheck)
	_ = validate.RegisterValidationWithErrorMsg(Port, PortCheck)
	_ = validate.RegisterValidationWithErrorMsg(Container, ContainerCheck)
	_ = validate.RegisterValidationWithErrorMsg(Language, LanguageCheck)

	validate.RegisterTagNameFunc(
		func(field reflect.StructField) string {
			tags := field.Tag.Get("yaml")
			return strings.Split(tags, ",")[0]
		},
	)
}

// PrepareForConfigurationValidate some validation relies on K8s resource, etc.
// so we should query them first
// and use os.setEnv to pass those condition
func PrepareForConfigurationValidate(client *clientgoutils.ClientGoUtils, containers []v1.Container) {
	if len(containers) > 0 {
		cs := ""
		for _, container := range containers {
			cs += container.Name + "\n"
		}
		_ = os.Setenv(CONTAINERS, cs)
	}

	if client == nil || client.ClientSet == nil {
		return
	}

	if list, err := client.ClientSet.StorageV1().StorageClasses().List(
		context.TODO(), metav1.ListOptions{},
	); err != nil {
		_ = os.Unsetenv(SUPPORT_SC)
		return
	} else {
		storageClasses := ""
		for _, item := range list.Items {
			storageClasses += item.Name + "\n"
		}
		_ = os.Setenv(SUPPORT_SC, storageClasses)
	}
}

func IsDNS1123Label(fl validator.FieldLevel) string {
	val := fl.Field().String()

	errs := validation.IsDNS1123Label(val)
	return hintIfNoPass(
		val == "" || len(errs) == 0,
		func() string {
			return strings.Join(errs, "\n")
		},
	)
}

func IsSupportsWorkLoads(fl validator.FieldLevel) string {
	_, err := nocalhost.SvcTypeOfMutate(fl.Field().String())
	return hintIfNoPass(
		err == nil,
		func() string {
			return "Current work load is not supported yet"
		},
	)
}

func IsSyncType(fl validator.FieldLevel) string {
	val := fl.Field().String()

	return hintIfNoPass(
		val == "" ||
			val == _const.DefaultSyncType ||
			val == _const.SendOnlySyncType ||
			val == _const.SendOnlySyncTypeAlias,
		func() string {
			return fmt.Sprintf("Must be %s or %s", _const.DefaultSyncType, _const.SendOnlySyncTypeAlias)
		},
	)
}

func IsSyncMode(fl validator.FieldLevel) string {
	val := fl.Field().String()

	return hintIfNoPass(
		val == "" ||
			val == _const.PatternMode ||
			val == _const.GitIgnoreMode,
		func() string {
			return fmt.Sprintf("Must be %s or %s", _const.PatternMode, _const.GitIgnoreMode)
		},
	)
}

func IsQuantity(fl validator.FieldLevel) string {
	val := fl.Field().String()
	if val == "" {
		return ""
	}

	_, err := resource.ParseQuantity(val)
	return hintIfNoPass(
		err == nil, func() string {
			return err.Error()
		},
	)
}

func StorageClassSupported(fl validator.FieldLevel) string {
	val := fl.Field().String()

	// if can not get storage class, escape to validate it
	supportsEnv, ok := os.LookupEnv(SUPPORT_SC)
	if ok {
		if val == "" {
			return ""
		}

		split := strings.Split(supportsEnv, "\n")

		set := sets.NewString(split...)
		return hintIfNoPass(
			set.Has(val),
			func() string {
				return fmt.Sprintf(
					"we found your cluster only supports storage class %s,"+
						" config for storageclass %v do not matched", split, val,
				)
			},
		)
	} else {
		return ""
	}
}

func LanguageCheck(fl validator.FieldLevel) string {
	val := fl.Field().String()

	if val == "" {
		return ""
	}

	languages := []string{"node", "java", "go", "python", "php", "ruby"}
	set := sets.NewString(languages...)
	return hintIfNoPass(
		set.Has(val),
		func() string {
			return fmt.Sprintf("language %s is unsupported, only %v supported", val, languages)
		},
	)
}

func ContainerCheck(fl validator.FieldLevel) string {
	val := fl.Field().String()

	// if can not get storage class, escape to validate it
	supportsEnv, ok := os.LookupEnv(CONTAINERS)
	if ok {
		split := strings.Split(supportsEnv, "\n")

		set := sets.NewString(split...)
		return hintIfNoPass(
			set.Has(val),
			func() string {
				return fmt.Sprintf(
					"we found your workloads with container %s,"+
						" container specified named '%v' do not matched", split, val,
				)
			},
		)
	} else {
		return ""
	}
}

func PortForwardCheck(fl validator.FieldLevel) string {
	val := fl.Field().String()

	_, _, err := utils.GetPortForwardForString(val)
	return hintIfNoPass(
		err == nil,
		func() string {
			return err.Error()
		},
	)
}

func hintIfNoPass(result bool, hint func() string) string {
	if !result {
		return hint()
	}

	return ""
}

// Validate the fields.
func Validate(s *profile.ServiceConfigV2) error {
	err := validate.Struct(s)

	if _, ok := err.(validator.ValidationErrors); ok {
		validationErrors := err.(validator.ValidationErrors)

		errMsg := ""

		if len(validationErrors) > 0 {
			errMsg += "[Configuration Validate Error]: "
		}

		for i, validationError := range validationErrors {
			errMsg +=
				fmt.Sprintf(
					"\n\n(%v) Error on field '%s', value '%v', hint: %s. ",
					i, validationError.Namespace(), validationError.Value(), validationError.Msg(),
				)
		}

		if errMsg != "" {
			return errors.New(errMsg)
		}
	}
	return err
}

func PortCheck(fl validator.FieldLevel) string {
	val := fl.Field().Int()

	l, r, err := utils.GetPortForwardForString(fmt.Sprintf("%v", val))
	return hintIfNoPass(
		err == nil && l == r,
		func() string {
			if err != nil {
				return err.Error()
			}

			return "You must defined a TCP port number with range of [0, 65535]"
		},
	)
}
