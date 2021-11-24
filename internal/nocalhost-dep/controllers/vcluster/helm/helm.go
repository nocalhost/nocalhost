/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helm

import (
	"fmt"
	"os"
	"strings"

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

	"nocalhost/internal/nocalhost-dep/controllers/vcluster/api/v1alpha1"
)

type Client struct {
	action.ChartPathOptions
	Config   *action.Configuration
	Settings *cli.EnvSettings
	values   chartutil.Values
}

func (c *Client) LoadChart(vc *v1alpha1.VirtualCluster) (*chart.Chart, error) {
	return c.loadChart(vc.GetChartName(), vc.GetChartRepo())
}

func (c Client) Install(vc *v1alpha1.VirtualCluster, chrt *chart.Chart) (*release.Release, error) {
	i := action.NewInstall(c.Config)
	i.Namespace = vc.GetNamespace()
	i.ReleaseName = vc.GetReleaseName()
	i.CreateNamespace = true
	rel, err := i.Run(chrt, c.values)
	if err != nil {
		return nil, errors.Wrapf(err, "can not install helm release: %s", vc.GetReleaseName())
	}
	return rel, nil
}

func (c *Client) SetValues(vc *v1alpha1.VirtualCluster) error {
	vals, err := chartutil.ReadValues([]byte(vc.GetValues()))
	if err != nil {
		return errors.Wrapf(err, "can not set helm values: %s", vc.GetReleaseName())
	}
	c.values = vals
	return nil
}

func (c *Client) Upgrade(vc *v1alpha1.VirtualCluster, chrt *chart.Chart) (*release.Release, error) {
	up := action.NewUpgrade(c.Config)
	up.Namespace = vc.GetNamespace()
	rel, err := up.Run(vc.GetReleaseName(), chrt, c.values)
	if err != nil {
		return nil, errors.Wrapf(err, "can not upgrade helm release: %s", vc.GetReleaseName())
	}
	return rel, nil
}

func (c *Client) UnInstall(vc *v1alpha1.VirtualCluster) error {
	u := action.NewUninstall(c.Config)
	_, err := u.Run(vc.GetReleaseName())
	if err != nil {
		return errors.Wrapf(err, "can not uninstall helm release: %s", vc.GetReleaseName())
	}
	return nil
}

func (c *Client) IsInstalled(vc *v1alpha1.VirtualCluster) bool {
	histClient := action.NewHistory(c.Config)
	histClient.Max = 1
	_, err := histClient.Run(vc.GetReleaseName())
	return !(err == driver.ErrReleaseNotFound)
}

func (c *Client) loadChart(chartRef, repoRUL string) (*chart.Chart, error) {
	var out strings.Builder
	var chrt *chart.Chart
	dl := downloader.ChartDownloader{
		Out:     &out,
		Keyring: "",
		Verify:  downloader.VerifyNever,
		Getters: getter.All(c.Settings),
		Options: []getter.Option{
			getter.WithBasicAuth(c.Username, c.Password),
			getter.WithTLSClientConfig(c.CertFile, c.KeyFile, c.CaFile),
			getter.WithInsecureSkipVerifyTLS(c.InsecureSkipTLSverify),
		},
		RepositoryConfig: c.Settings.RepositoryConfig,
		RepositoryCache:  c.Settings.RepositoryCache,
	}

	c.setRepo(repoRUL)

	if c.RepoURL != "" {
		chartURL, err := repo.FindChartInAuthAndTLSRepoURL(
			c.RepoURL, c.Username, c.Password, chartRef, c.Version, c.CertFile,
			c.KeyFile, c.CaFile, c.InsecureSkipTLSverify, getter.All(c.Settings))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		chartRef = chartURL
	}

	dest, err := os.MkdirTemp("", "vc")
	defer os.RemoveAll(dest)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	saved, _, err := dl.DownloadTo(chartRef, c.Version, dest)
	if err != nil {
		log.Log.Error(err, out.String())
		return nil, errors.Wrap(err, "can not download chart")
	}
	c.Config.Log("save chart to: %s", saved)

	if chrt, err = loader.Load(saved); err != nil {
		return nil, errors.Wrap(err, "can not load chart")
	}
	return chrt, nil
}

func (c *Client) setRepo(repo string) {
	c.RepoURL = repo
}

func NewClient(config *rest.Config, ns string) (*Client, error) {
	kubeConfig := genericclioptions.NewConfigFlags(false)
	kubeConfig.APIServer = &config.Host
	kubeConfig.BearerToken = &config.BearerToken
	kubeConfig.CAFile = &config.CAFile
	kubeConfig.Namespace = &ns

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(kubeConfig, ns, "secret", helmLog); err != nil {
		return nil, errors.Wrap(err, "can not init helm client")
	}

	return &Client{
		Config:   actionConfig,
		Settings: cli.New(),
	}, nil
}

func helmLog(format string, v ...interface{}) {
	log := log.Log.WithName("helm")
	log.Info(fmt.Sprintf(format, v...))
}
