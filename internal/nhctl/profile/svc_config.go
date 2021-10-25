/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package profile

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"nocalhost/internal/nhctl/common/base"
	_const "nocalhost/internal/nhctl/const"
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

	SUPPORT_SC = "NOCALHOST_SUPPORT_SC"
	CONTAINERS = "NOCALHOST_CONTAINERS"

	validate = validator.New()
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

	validate.RegisterTagNameFunc(
		func(field reflect.StructField) string {
			tags := field.Tag.Get("yaml")
			return strings.Split(tags, ",")[0]
		},
	)
}

type ServiceConfigV2 struct {
	Name                string               `validate:"required" json:"name" yaml:"name"`
	Type                string               `validate:"required" json:"serviceType" yaml:"serviceType"`
	PriorityClass       string               `json:"priorityClass,omitempty" yaml:"priorityClass,omitempty"`
	DependLabelSelector *DependLabelSelector `json:"dependLabelSelector,omitempty" yaml:"dependLabelSelector,omitempty"`
	ContainerConfigs    []*ContainerConfig   `validate:"dive" json:"containers" yaml:"containers"`
}

type ContainerConfig struct {
	Name    string                  `validate:"Container" json:"name" yaml:"name"`
	Hub     *HubConfig              `json:"hub" yaml:"hub,omitempty"`
	Install *ContainerInstallConfig `json:"install,omitempty" yaml:"install,omitempty"`
	Dev     *ContainerDevConfig     `json:"dev" yaml:"dev"`
}

func (s *ServiceConfigV2) GetContainerConfig(container string) *ContainerConfig {
	if s == nil {
		return nil
	}
	for _, c := range s.ContainerConfigs {
		if c.Name == container {
			return c
		}
	}
	return nil
}

func (s *ServiceConfigV2) GetDefaultContainerDevConfig() *ContainerDevConfig {
	if len(s.ContainerConfigs) == 0 {
		return nil
	}
	return s.ContainerConfigs[0].Dev
}

// Compatible for v1
// Finding `containerName` config, if not found, use the first container config
func (s *ServiceConfigV2) GetContainerDevConfigOrDefault(containerName string) *ContainerDevConfig {
	if containerName == "" {
		return s.GetDefaultContainerDevConfig()
	}
	config := s.GetContainerDevConfig(containerName)
	if config == nil {
		config = s.GetDefaultContainerDevConfig()
	}
	return config
}

func (s *ServiceConfigV2) GetContainerDevConfig(containerName string) *ContainerDevConfig {
	for _, devConfig := range s.ContainerConfigs {
		if devConfig.Name == containerName {
			return devConfig.Dev
		}
	}
	return nil
}

// Validate the fields.
func (s *ServiceConfigV2) Validate() error {
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
	_, err := base.SvcTypeOfMutate(fl.Field().String())
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

	_, _, err := GetPortForwardForString(val)
	return hintIfNoPass(
		err == nil,
		func() string {
			return err.Error()
		},
	)
}

func PortCheck(fl validator.FieldLevel) string {
	val := fl.Field().Int()

	l, r, err := GetPortForwardForString(fmt.Sprintf("%v", val))
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

func hintIfNoPass(result bool, hint func() string) string {
	if !result {
		return hint()
	}

	return ""
}
