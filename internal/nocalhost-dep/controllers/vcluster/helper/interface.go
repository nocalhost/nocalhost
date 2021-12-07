/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

type Actions interface {
	Get(name string, opts ...GetOption) (*release.Release, error)
	Install(name, namespace string, chrt *chart.Chart, opts ...InstallOption) (*release.Release, error)
	Upgrade(name, namespace string, chrt *chart.Chart, opts ...UpgradeOption) (*release.Release, error)
	Uninstall(name string, opts ...UninstallOption) (*release.UninstallReleaseResponse, error)
	GetChart(chartRef string) (*chart.Chart, error)
	GetState(name string) ActionState
}

type AuthConfig interface {
	Get(name, namespace string) (string, error)
}
