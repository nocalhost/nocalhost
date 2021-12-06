/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"

	helmv1alpha1 "nocalhost/internal/nocalhost-dep/controllers/vcluster/api/v1alpha1"
)

type GetOption func(*action.Get) error
type InstallOption func(*action.Install) error
type UpgradeOption func(*action.Upgrade) error
type UninstallOption func(*action.Uninstall) error

type ActionState string

const (
	ActionInstall ActionState = "install"
	ActionUpgrade ActionState = "upgrade"
	ActionError   ActionState = "error"
)

type actions struct {
	mu      sync.Mutex
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

func (a *actions) GetChart(chartRef string) (*chart.Chart, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := fmt.Sprintf("%s/%s/%s", a.cpo.RepoURL, chartRef, a.cpo.Version)
	var err error
	chrt, ok := ChartCache.Get(key)
	if ok {
		a.cfg.Log("found chart in cache")
		return chrt, nil
	}
	chrt, err = a.loadChart(chartRef)
	if err != nil {
		return nil, err
	}
	a.cfg.Log("add chart to cache")
	ChartCache.Add(key, chrt)
	return chrt, nil
}

func (a *actions) GetState(name string) ActionState {
	_, err := a.Get(name)
	if err != nil {
		if errors.Cause(err) == driver.ErrReleaseNotFound {
			return ActionInstall
		}
		return ActionError
	}

	return ActionUpgrade
}

func (a *actions) loadChart(chartRef string) (*chart.Chart, error) {
	var out strings.Builder
	var chrt *chart.Chart
	dl := downloader.ChartDownloader{
		Out:     &out,
		Keyring: "",
		Verify:  downloader.VerifyNever,
		Getters: getter.All(a.setting),
		Options: []getter.Option{
			getter.WithBasicAuth(a.cpo.Username, a.cpo.Password),
			getter.WithTLSClientConfig(a.cpo.CertFile, a.cpo.KeyFile, a.cpo.CaFile),
			getter.WithInsecureSkipVerifyTLS(a.cpo.InsecureSkipTLSverify),
		},
		RepositoryConfig: a.setting.RepositoryConfig,
		RepositoryCache:  a.setting.RepositoryCache,
	}

	if a.cpo.RepoURL != "" {
		chartURL, err := repo.FindChartInAuthAndTLSRepoURL(
			a.cpo.RepoURL, a.cpo.Username, a.cpo.Password, chartRef, a.cpo.Version, a.cpo.CertFile,
			a.cpo.KeyFile, a.cpo.CaFile, a.cpo.InsecureSkipTLSverify, getter.All(a.setting))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		chartRef = chartURL
	}

	dest, err := os.MkdirTemp("", "vc")
	defer func() {
		a.cfg.Log("remove temp dir: %s", dest)
		if err := os.RemoveAll(dest); err != nil {
			a.cfg.Log("can not remove temp dir: %s", err)
		}
	}()

	if err != nil {
		return nil, errors.WithStack(err)
	}
	saved, _, err := dl.DownloadTo(chartRef, a.cpo.Version, dest)
	if err != nil {
		log.Log.Error(err, out.String())
		return nil, errors.Wrap(err, "can not download chart")
	}
	a.cfg.Log("save chart to: %s", saved)

	if chrt, err = loader.Load(saved); err != nil {
		return nil, errors.Wrap(err, "can not load chart")
	}
	return chrt, nil
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
	lg := log.Log.WithName("helm")
	lg.Info(fmt.Sprintf(format, v...))
}

func NewActions(vc *helmv1alpha1.VirtualCluster, config *rest.Config) (Actions, error) {
	cfg := new(action.Configuration)
	if err := cfg.Init(
		newRESTClientGetter(config, vc.GetNamespace()),
		vc.GetNamespace(),
		"secret",
		helmLog); err != nil {
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
