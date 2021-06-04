package common

import (
	"fmt"
	errors2 "github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os"
)

func InitDefaultApplicationInCurrentNs(namespace string, kubeconfigPath string) (*app.Application, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	baseDir := fp.NewFilePath(tmpDir)
	nocalhostDir := baseDir.RelOrAbs(app.DefaultGitNocalhostDir)
	err = nocalhostDir.Mkdir()
	if err != nil {
		return nil, err
	}

	var cfg = ".default_config"

	err = nocalhostDir.RelOrAbs(cfg).WriteFile("name: nocalhost.default\nmanifestType: rawManifestLocal")
	if err != nil {
		return nil, err
	}

	f := &app_flags.InstallFlags{
		Config:    cfg,
		AppType:   string(appmeta.ManifestLocal),
		LocalPath: baseDir.Abs(),
	}

	application, err := InstallApplication(f, nocalhost.DefaultNocalhostApplication, kubeconfigPath, namespace)
	if errors.IsServerTimeout(err) {
		return application, nil
	}
	return application, err
}

func InstallApplication(flags *app_flags.InstallFlags, applicationName, kubeconfig, namespace string) (*app.Application, error) {
	var err error

	log.Logf("KubeConfig path: %s", kubeconfig)
	bys, err := ioutil.ReadFile(kubeconfig)
	if err != nil {
		return nil, errors2.Wrap(err, "")
	}
	log.Logf("KubeConfig content: %s", string(bys))

	// build Application will create the application meta and it's secret
	// init the application's config
	nocalhostApp, err := app.BuildApplication(applicationName, flags, kubeconfig, namespace)
	if err != nil {
		return nil, err
	}

	// if init appMeta successful, then should remove all things while fail
	defer func() {
		if err != nil {
			coloredoutput.Fail(err.Error())
			log.LogE(err)
			utils.Should(nocalhostApp.Uninstall())
		}
	}()

	appType := nocalhostApp.GetType()
	if appType == "" {
		return nil, errors2.New("--type must be specified")
	}

	// add helmValue in config
	helmValue := nocalhostApp.GetApplicationConfigV2().HelmValues
	for _, v := range helmValue {
		flags.HelmSet = append([]string{fmt.Sprintf("%s=%s", v.Key, v.Value)}, flags.HelmSet...)
	}

	flag := &app.HelmFlags{
		Values:   flags.HelmValueFile,
		Set:      flags.HelmSet,
		Wait:     flags.HelmWait,
		Chart:    flags.HelmChartName,
		RepoUrl:  flags.HelmRepoUrl,
		RepoName: flags.HelmRepoName,
		Version:  flags.HelmRepoVersion,
	}

	err = nocalhostApp.Install(flag)
	_ = nocalhostApp.CleanUpTmpResources()
	return nocalhostApp, err
}
