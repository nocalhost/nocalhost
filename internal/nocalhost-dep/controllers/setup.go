/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"nocalhost/internal/nocalhost-dep/controllers/vcluster"
)

// Setup create controllers
func Setup(mgr ctrl.Manager) error {
	return vcluster.Setup(mgr)
}
