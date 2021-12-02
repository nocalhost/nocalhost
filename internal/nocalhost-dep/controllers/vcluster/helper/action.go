/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"fmt"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"

	helmv1alpha1 "nocalhost/internal/nocalhost-dep/controllers/vcluster/api/v1alpha1"
)

type GetOption func(*action.Get) error
type InstallOption func(*action.Install) error
type UpgradeOption func(*action.Upgrade) error
type UninstallOption func(*action.Uninstall) error

type actions struct {
	cpo     action.ChartPathOptions
	cfg     *action.Configuration
	setting *cli.EnvSettings
	values  chartutil.Values
}

func (a *actions) Get(name string, opts ...GetOption) (*release.Release, error) {
	get := newGet(a)
	for _, o := range opts {
		if err := o(get); err != nil {
			return nil, err
		}
	}
	rel, err := get.Run(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return rel, nil
}

func (a *actions) Install(name, namespace string, chrt *chart.Chart, opts ...InstallOption) (*release.Release, error) {
	i := newInstall(a)
	for _, o := range opts {
		if err := o(i); err != nil {
			return nil, err
		}
	}
	i.Namespace = namespace
	i.ReleaseName = name
	rel, err := i.Run(chrt, a.values)
	if err != nil {
		return nil, errors.Wrapf(err, "can not install helm release: %s", name)
	}
	return rel, nil
}

func (a *actions) Upgrade(name, namespace string, chrt *chart.Chart, opts ...UpgradeOption) (*release.Release, error) {
	up := newUpgrade(a)
	for _, o := range opts {
		if err := o(up); err != nil {
			return nil, err
		}
	}
	up.Namespace = namespace
	rel, err := up.Run(name, chrt, a.values)
	if err != nil {
		return nil, errors.Wrapf(err, "can not upgrade helm release: %s", name)
	}
	return rel, nil
}

func (a *actions) Uninstall(name string, opts ...UninstallOption) (*release.UninstallReleaseResponse, error) {
	u := action.NewUninstall(a.cfg)
	for _, o := range opts {
		if err := o(u); err != nil {
			return nil, err
		}
	}
	rel, err := u.Run(name)
	if err != nil {
		return nil, errors.Wrapf(err, "can not uninstall helm release: %s", name)
	}
	return rel, nil
}

func newGet(a *actions) *action.Get {
	return action.NewGet(a.cfg)
}

func newInstall(a *actions) *action.Install {
	in := action.NewInstall(a.cfg)
	in.ChartPathOptions = a.cpo
	return in
}

func newUpgrade(a *actions) *action.Upgrade {
	up := action.NewUpgrade(a.cfg)
	up.ChartPathOptions = a.cpo
	return up
}

func newRESTClientGetter(config *rest.Config, ns string) genericclioptions.RESTClientGetter {
	kc := genericclioptions.NewConfigFlags(false)
	kc.APIServer = &config.Host
	kc.BearerToken = &config.BearerToken
	kc.CAFile = &config.CAFile
	kc.Namespace = &ns
	return kc
}

func helmLog(format string, v ...interface{}) {
	log := log.Log.WithName("helm")
	log.Info(fmt.Sprintf(format, v...))
}

func NewActions(vc *helmv1alpha1.VirtualCluster, config *rest.Config, ns string) (Actions, error) {
	cfg := new(action.Configuration)
	if err := cfg.Init(newRESTClientGetter(config, ns), ns, "secret", helmLog); err != nil {
		return nil, errors.Wrap(err, "can not init helm client")
	}
	vals, err := chartutil.ReadValues([]byte(vc.GetValues()))
	if err != nil {
		return nil, errors.Wrapf(err, "can not set helm values: %s", vc.GetReleaseName())
	}

	return &actions{
		cpo: action.ChartPathOptions{
			RepoURL: vc.GetChartRepo(),
			Version: vc.GetChartVersion(),
		},
		cfg:     cfg,
		setting: cli.New(),
		values:  vals,
	}, nil
}
