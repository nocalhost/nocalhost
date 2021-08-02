/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package app

import (
	"nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/clientgoutils"
)

func StandardNocalhostMetas(releaseName, releaseNamespace string) *clientgoutils.ApplyFlags {
	return &clientgoutils.ApplyFlags{
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
