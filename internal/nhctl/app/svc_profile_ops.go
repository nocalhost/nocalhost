/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package app

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/profile"
)

func (a *Application) GetSvcProfile(svcName string, svcType base.SvcType) (*profile.SvcProfileV2, error) {
	profileV2, err := a.GetProfile()
	if err != nil {
		return nil, err
	}
	svcProfile := profileV2.SvcProfileV2(svcName, string(svcType))
	if svcProfile == nil {
		return nil, errors.New(
			fmt.Sprintf(
				"Svc Profile not found %s-%s-%s", profileV2.Namespace, svcType.String(), svcName,
			),
		)
	}
	return svcProfile, nil
}
