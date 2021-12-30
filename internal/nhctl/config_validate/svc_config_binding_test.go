/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package config_validate

import (
	"errors"
	"github.com/go-playground/validator/v10"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"os"
	"testing"
)

func TestBinding(t *testing.T) {
	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost",
		Type: "deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					DevContainerResources: &profile.ResourceQuota{
						Limits: &profile.QuotaList{
							Memory: "100Mi",
							Cpu:    "1",
						},
						Requests: &profile.QuotaList{
							Memory: "50Gi",
							Cpu:    "500Ti",
						},
					},
					PortForward: []string{
						"8888",
						"8888:8889",
						":8080",
					},
					DebugConfig: &profile.DebugConfig{RemoteDebugPort: 9999},
					PersistentVolumeDirs: []*profile.PersistentVolumeDir{
						{Path: "/path", Capacity: "10Gi"},
					},
					Sync: &profile.SyncConfig{
						Type: _const.DefaultSyncType,
						Mode: _const.GitIgnoreMode,
					},
					StorageClass: "hostpath",
				},
			},
		},
	}, ""); err != nil {
		t.Error(err)
	}
}

func TestQuantity(t *testing.T) {
	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					DevContainerResources: &profile.ResourceQuota{
						Limits: &profile.QuotaList{
							Memory: "100Mii",
						},
					},
				},
			},
		},
	}, Quantity); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					DevContainerResources: &profile.ResourceQuota{
						Limits: &profile.QuotaList{
							Cpu: "1GG",
						},
					},
				},
			},
		},
	}, Quantity); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					DevContainerResources: &profile.ResourceQuota{
						Requests: &profile.QuotaList{
							Memory: "50GGi",
						},
					},
				},
			},
		},
	}, Quantity); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					DevContainerResources: &profile.ResourceQuota{
						Requests: &profile.QuotaList{
							Cpu: "500TTi",
						},
					},
				},
			},
		},
	}, Quantity); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					PersistentVolumeDirs: []*profile.PersistentVolumeDir{
						{Path: "/path", Capacity: "10zGi"},
					},
				},
			},
		},
	}, Quantity); err != nil {
		t.Error(err)
	}
}

func TestSyncMode(t *testing.T) {
	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					Sync: &profile.SyncConfig{
						Mode: "SEND",
					},
				},
			},
		},
	}, SyncMode); err != nil {
		t.Error(err)
	}
}

func TestPortForward(t *testing.T) {
	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					PortForward: []string{
						"99999",
					},
				},
			},
		},
	}, PortForward); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					PortForward: []string{
						"-1",
					},
				},
			},
		},
	}, PortForward); err != nil {
		t.Error(err)
	}
}

func TestPort(t *testing.T) {
	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					DebugConfig: &profile.DebugConfig{RemoteDebugPort: -1},
				},
			},
		},
	}, Port); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					DebugConfig: &profile.DebugConfig{RemoteDebugPort: 99999},
				},
			},
		},
	}, Port); err != nil {
		t.Error(err)
	}
}

func TestSyncType(t *testing.T) {
	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost",
				Dev: &profile.ContainerDevConfig{
					Sync: &profile.SyncConfig{
						Type: "SEND",
					},
				},
			},
		},
	}, SyncType); err != nil {
		t.Error(err)
	}
}

func TestWorkloads(t *testing.T) {
	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-api",
		Type: "Deployment",
	}, ""); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-web",
		Type: "deployment",
	}, ""); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-web",
		Type: "pod",
	}, ""); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost-dep",
		Type: "dep",
	}, WorkLoads); err != nil {
		t.Error(err)
	}
}

func TestDNS1123(t *testing.T) {
	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost_",
		Type: "deployment",
	}, DNS1123); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "!nocalhost",
		Type: "deployment",
	}, DNS1123); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost",
		Type: "deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "-nocalhost",
			},
		},
	}, DNS1123); err != nil {
		t.Error(err)
	}

	if err := validateStructAndExpectFor(&profile.ServiceConfigV2{
		Name: "nocalhost",
		Type: "deployment",
		ContainerConfigs: []*profile.ContainerConfig{
			{
				Name: "nocalhost-",
			},
		},
	}, DNS1123); err != nil {
		t.Error(err)
	}
}

func validateStructAndExpectFor(svcConfig *profile.ServiceConfigV2, expectErrorTag string) error {
	prepare()

	// returns nil or ValidationErrors ( []FieldError )
	err := validate.Struct(svcConfig)
	if err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			validationErrors := err.(validator.ValidationErrors)
			if len(validationErrors) == 1 && validationErrors[0].Tag() == expectErrorTag {
				log.Info(validationErrors.Error())
			} else {
				return errors.New("Unexpect Err: " + validationErrors.Error())
			}
		}
	} else {
		if expectErrorTag != "" {
			return errors.New("No error occur but expect: " + expectErrorTag)
		}
	}

	return nil
}

func prepare() {
	var supportsSc = ""

	client, err := clientgo.NewGoClient([]byte(fp.NewFilePath("~/.kube/config").ReadFile()))
	if err == nil && client != nil {
		list, err := client.GetStorageClassList()
		if err == nil && list != nil {
			for _, item := range list.Items {
				if supportsSc != "" {
					supportsSc += "\n"
				}
				supportsSc += item.GetName()
			}
			_ = os.Setenv(SUPPORT_SC, supportsSc)
		}
	}
}
