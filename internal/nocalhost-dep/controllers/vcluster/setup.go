/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package vcluster

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"nocalhost/internal/nocalhost-dep/controllers/vcluster/controllers"
)

// Setup create VirtualCluster controller
func Setup(mgr ctrl.Manager) error {
	return (&controllers.Reconciler{
		Client: mgr.GetClient(),
		Config: mgr.GetConfig(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
}
