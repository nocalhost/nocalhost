/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package app

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"net"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/nocalhost"
	nocalhostDb "nocalhost/internal/nhctl/nocalhost/db"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

var (
	ErrNotFound = errors.New("Application not found")
	indent      = 70
)

type Application struct {
	Name       string
	NameSpace  string
	KubeConfig string
	AppType    string

	// may be nil, only for install or upgrade
	// dir use to load the user's resource
	ResourceTmpDir string

	appMeta *appmeta.ApplicationMeta
	client  *clientgoutils.ClientGoUtils
}

type SvcDependency struct {
	Name string   `json:"name" yaml:"name"`
	Type string   `json:"type" yaml:"type"`
	Jobs []string `json:"jobs" yaml:"jobs,omitempty"`
	Pods []string `json:"pods" yaml:"pods,omitempty"`
}

func (a *Application) GetAppMeta() *appmeta.ApplicationMeta {
	return a.appMeta
}

func (a *Application) moveProfileFromFileToLeveldb() error {
	profileV2 := &profile.AppProfileV2{}

	fBytes, err := ioutil.ReadFile(a.getProfileV2Path())
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = yaml.Unmarshal(fBytes, profileV2)
	if err != nil {
		return errors.Wrap(err, "")
	}
	log.Log("Move profile to leveldb")

	//a.profileV2 = profileV2
	return nocalhost.UpdateProfileV2(a.NameSpace, a.Name, profileV2)
}

// When new a application, kubeconfig is required to get meta in k8s cluster
// KubeConfig can be acquired from profile in leveldb
func NewApplication(name string, ns string, kubeconfig string, initClient bool) (*Application, error) {

	var err error
	app := &Application{
		Name:       name,
		NameSpace:  ns,
		KubeConfig: kubeconfig,
	}

	if app.appMeta, err = nocalhost.GetApplicationMeta(app.Name, app.NameSpace, app.KubeConfig); err != nil {
		return nil, err
	}

	// 1. first try load profile from local or earlier version
	// 2. check should generate secret for adapt earlier version
	// 3. try load application meta from secret
	// 4. update kubeconfig for profile
	// 5. init go client inner Application

	if err := app.tryLoadProfileFromLocal(); err != nil {
		return nil, err
	}

	// if appMeta is not installed but application installed in earlier version
	// should make a fake installation and generate an application meta
	if app.generateSecretForEarlierVer() {

		// load app meta if generate secret for earlier verion
		if app.appMeta, err = nocalhost.GetApplicationMeta(app.Name, app.NameSpace, app.KubeConfig); err != nil {
			return nil, err
		}
	}

	if !app.appMeta.IsInstalled() {
		return nil, errors.Wrap(ErrNotFound, fmt.Sprintf("%s-%s not found", app.NameSpace, app.Name))
	}

	// if still not present
	// load from secret
	profileV2, err := nocalhost.GetProfileV2(app.NameSpace, app.Name)
	if err != nil {
		profileV2 = generateProfileFromConfig(app.appMeta.Config)
		if err = nocalhost.UpdateProfileV2(app.NameSpace, app.Name, profileV2); err != nil {
			return nil, err
		}
	}
	app.AppType = profileV2.AppType

	if kubeconfig != "" && kubeconfig != profileV2.Kubeconfig {
		if err := app.UpdateProfile(
			func(p *profile.AppProfileV2) error {
				p.Kubeconfig = kubeconfig
				return nil
			},
		); err != nil {
			return nil, err
		}
	}

	if initClient {
		if app.client, err = clientgoutils.NewClientGoUtils(app.KubeConfig, app.NameSpace); err != nil {
			return nil, err
		}
	}

	return app, nil
}

func (a *Application) generateSecretForEarlierVer() bool {

	a.GetHomeDir()
	profileV2, err := a.GetProfile()
	if err != nil {
		return false
	}

	if a.HasBeenGenerateSecret() {
		return false
	}

	if profileV2 != nil && !profileV2.Secreted && a.appMeta.IsNotInstall() &&
		a.Name != nocalhost.DefaultNocalhostApplication {
		a.AppType = profileV2.AppType

		defer func() {
			log.Logf("Mark application %s in ns %s has been secreted", a.Name, a.NameSpace)
			_ = a.UpdateProfile(
				func(p *profile.AppProfileV2) error {
					p.Secreted = true
					return nil
				},
			)
		}()

		if err := a.appMeta.Initial(); err != nil {
			log.ErrorE(err, "")
			return true
		}
		log.Logf("Earlier version installed application found, generate a secret...")

		profileV2.GenerateIdentifierIfNeeded()
		_ = nocalhost.UpdateProfileV2(a.NameSpace, a.Name, profileV2)

		// config、manifest is missing while adaption update
		a.appMeta.Config = a.newConfigFromProfile()
		a.appMeta.DepConfigName = profileV2.DependencyConfigMapName
		a.appMeta.Ns = a.NameSpace
		a.appMeta.ApplicationType = appmeta.AppTypeOf(a.AppType)

		_ = a.appMeta.Update()

		a.client, err = a.appMeta.GetClient()
		if err != nil {
			log.Error(err)
		}
		switch a.AppType {
		case string(appmeta.Manifest), string(appmeta.ManifestLocal), string(appmeta.ManifestGit):
			_ = a.InstallManifest(a.appMeta, a.getResourceDir(), false)
		case string(appmeta.KustomizeGit):
			_ = a.InstallKustomize(a.appMeta, a.getResourceDir(), false)
		default:
		}

		for _, svc := range profileV2.SvcProfile {
			if svc.Developing {
				_ = a.appMeta.SvcDevStart(svc.Name, base.SvcType(svc.Type), profileV2.Identifier)
			}
		}

		a.appMeta.ApplicationState = appmeta.INSTALLED
		_ = a.appMeta.Update()

		log.Logf("Application %s in ns %s is completed secreted", a.Name, a.NameSpace)
		return false
	}

	a.MarkAsGenerated()

	return false
}

func (a *Application) ReloadCfg(reloadFromMeta, silence bool) error {
	secretCfg := a.appMeta.Config
	for _, config := range secretCfg.ApplicationConfig.ServiceConfigs {
		if err := a.ReloadSvcCfg(config.Name, base.SvcTypeOf(config.Type), reloadFromMeta, silence); err != nil {
			log.LogE(err)
		}
	}

	return nil
}

// ReloadSvcCfg try load config from cm first
// then load from local under associateDir/.nocalhost/config.yaml
// at last load config from local profile
func (a *Application) ReloadSvcCfg(svcName string, svcType base.SvcType, reloadFromMeta, silence bool) error {

	if a.loadSvcCfgFromCmIfValid(svcName, svcType, silence) {
		return nil
	}

	if a.loadSvcCfgFromLocalIfValid(svcName, svcType, silence) {
		return nil
	}

	preCheck, err := a.Controller(svcName, svcType).GetProfile()
	if err != nil {
		return err
	}

	// skip the case do not need to reload cfg
	if preCheck.LocalConfigLoaded == false && preCheck.CmConfigLoaded == false && !reloadFromMeta {
		return nil
	}

	return a.Controller(svcName, svcType).UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {

			if reloadFromMeta {
				svcProfile.ServiceConfigV2 = a.appMeta.Config.GetSvcConfigV2(svcName, svcType)
				if !silence {
					metaInfo := fmt.Sprintf("[name: %s serviceType: %s]", svcName, svcType)
					log.Infof(
						fmt.Sprintf(
							"%-"+strconv.Itoa(indent)+"s %s",
							metaInfo,
							"Load nocalhost svc config from application config (secret)",
						),
					)
				}
			}

			svcProfile.LocalConfigLoaded = false
			svcProfile.CmConfigLoaded = false
			return nil
		},
	)
}

func (a *Application) loadSvcCfgFromCmIfValid(svcName string, svcType base.SvcType, silence bool) bool {
	metaInfo := fmt.Sprintf("[name: %s serviceType: %s]", svcName, svcType)
	hint := func(format string, s ...string) {
		if !silence {
			var output string
			if len(s) == 0 {
				output = format
			} else {
				output = fmt.Sprintf(format, s)
			}

			coloredoutput.Hint(
				"%-"+strconv.Itoa(indent)+"s %s",
				metaInfo,
				output,
			)
		}
	}

	configMap, err := a.GetConfigMap(appmeta.ConfigMapName(a.appMeta.Application))
	if err != nil {
		return false
	}

	cfgStr := configMap.Data[appmeta.CmConfigKey]
	if cfgStr == "" {
		return false
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		hint("Load nocalhost svc config from cm fail, err: %s", err.Error())
		return false
	}

	tmpFp := fp.NewFilePath(dir).RelOrAbs("config.yaml")

	err = tmpFp.WriteFile(cfgStr)
	if err != nil {
		hint("Load nocalhost svc config from cm fail, err: %s", err.Error())
		return false
	}

	var svcCfg *profile.ServiceConfigV2
	if svcCfg, err = doLoadProfileFromSvcConfig(tmpFp, svcName, svcType); svcCfg == nil {
		if svcCfg, _ = doLoadProfileFromAppConfig(tmpFp, svcName, svcType); svcCfg == nil {
			if err != nil {
				hint("Load nocalhost svc config from cm fail, err: %s", err.Error())
			}
			return false
		}
	}

	// means should cm cfg is valid, persist to profile
	if err := a.Controller(svcName, svcType).UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
			hint("Success load svc config from cm")
			svcCfg.Name = svcName
			svcCfg.Type = svcType.String()

			svcProfile.ServiceConfigV2 = svcCfg
			svcProfile.CmConfigLoaded = true
			svcProfile.LocalConfigLoaded = false
			return nil
		},
	); err != nil {
		hint("Load nocalhost svc config from cm fail, fail while updating svc profile, err: %s", err.Error())
		return false
	}
	return true
}

func (a *Application) loadSvcCfgFromLocalIfValid(svcName string, svcType base.SvcType, silence bool) bool {
	metaInfo := fmt.Sprintf("[name: %s serviceType: %s]", svcName, svcType)
	hint := func(format string, s ...string) {
		if !silence {
			var output string
			if len(s) == 0 {
				output = format
			} else {
				output = fmt.Sprintf(format, s)
			}

			coloredoutput.Hint(
				"%-"+strconv.Itoa(indent)+"s %s",
				metaInfo,
				output,
			)
		}
	}

	p, err := a.GetProfile()
	if err != nil {
		return false
	}

	svcProfile := p.SvcProfileV2(svcName, svcType.String())

	if svcProfile.Associate == "" {
		return false
	}

	configFile := fp.NewFilePath(svcProfile.Associate).
		RelOrAbs(DefaultGitNocalhostDir).
		RelOrAbs(DefaultConfigNameInGitNocalhostDir)

	if err = configFile.CheckExist(); err != nil {
		return false
	}

	var svcCfg *profile.ServiceConfigV2
	if svcCfg, err = doLoadProfileFromSvcConfig(configFile, svcName, svcType); svcCfg == nil {
		if svcCfg, _ = doLoadProfileFromAppConfig(configFile, svcName, svcType); svcCfg == nil {
			if err != nil {
				hint("Load nocalhost svc config from local fail, err: %s", err.Error())
			}
			return false
		}
	}

	// means should load svc cfg from local
	if err := a.Controller(svcName, svcType).UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
			hint("Success load svc config from local file %s", configFile.Abs())
			svcCfg.Name = svcName
			svcCfg.Type = svcType.String()

			svcProfile.ServiceConfigV2 = svcCfg
			svcProfile.LocalConfigLoaded = true
			svcProfile.CmConfigLoaded = false
			return nil
		},
	); err != nil {
		hint("Load nocalhost svc config from local fail, fail while updating svc profile, err: %s", err.Error())
		return false
	}
	return true
}

func doLoadProfileFromSvcConfig(configFile *fp.FilePathEnhance, svcName string, svcType base.SvcType) (
	*profile.ServiceConfigV2, error,
) {
	config, err := RenderConfigForSvc(configFile.Path)
	if err != nil {
		return nil, err
	}

	if len(config) == 1 && config[0].Name == "" {
		return config[0], nil
	}

	for _, svcConfig := range config {
		if svcConfig.Name == svcName && base.SvcTypeOf(svcConfig.Type) == svcType {
			return svcConfig, nil
		}
	}

	return nil, errors.New("Local config loaded, but no valid config found")
}

func doLoadProfileFromAppConfig(configFile *fp.FilePathEnhance, svcName string, svcType base.SvcType) (
	*profile.ServiceConfigV2, error,
) {
	appConfig, err := RenderConfig(configFile.Path)
	if err != nil {
		return nil, err
	}

	return appConfig.GetSvcConfigV2(svcName, svcType), nil
}

func (a *Application) newConfigFromProfile() *profile.NocalHostAppConfigV2 {
	if bys, err := ioutil.ReadFile(a.GetConfigV2Path()); err == nil {
		p := &profile.NocalHostAppConfigV2{}
		if err = yaml.Unmarshal(bys, p); err == nil {
			return p
		}
	}
	profileV2, _ := a.GetProfile()
	return &profile.NocalHostAppConfigV2{
		ConfigProperties: &profile.ConfigProperties{
			Version: "v2",
		},
		ApplicationConfig: &profile.ApplicationConfig{
			Name:           a.Name,
			Type:           profileV2.AppType,
			ResourcePath:   profileV2.ResourcePath,
			IgnoredPath:    profileV2.IgnoredPath,
			PreInstall:     profileV2.PreInstall,
			Env:            profileV2.Env,
			EnvFrom:        profileV2.EnvFrom,
			ServiceConfigs: loadServiceConfigsFromProfile(profileV2.SvcProfile),
		},
	}
}

func loadServiceConfigsFromProfile(profiles []*profile.SvcProfileV2) []*profile.ServiceConfigV2 {
	var configs = []*profile.ServiceConfigV2{}

	for _, p := range profiles {
		configs = append(
			configs, &profile.ServiceConfigV2{
				Name:                p.Name,
				Type:                p.Type,
				PriorityClass:       p.PriorityClass,
				DependLabelSelector: p.DependLabelSelector,
				ContainerConfigs:    p.ContainerConfigs,
			},
		)
	}

	return configs
}

func (a *Application) tryLoadProfileFromLocal() (err error) {
	if db, err := nocalhostDb.OpenApplicationLevelDB(a.NameSpace, a.Name, true); err != nil {
		if err = nocalhostDb.CreateApplicationLevelDB(a.NameSpace, a.Name, true); err != nil { // Init leveldb dir
			return err
		}
	} else {
		_ = db.Close()
	}

	// try load from db first
	// then try load from disk(to supports earlier version)
	if _, err = nocalhost.GetProfileV2(a.NameSpace, a.Name); err != nil {
		if _, err := os.Stat(a.getProfileV2Path()); err == nil {

			// need not care what happen
			_ = a.moveProfileFromFileToLeveldb()
		}
	}

	return nil
}

func (a *Application) GetProfile() (*profile.AppProfileV2, error) {
	return nocalhost.GetProfileV2(a.NameSpace, a.Name)
}

func (a *Application) GetProfileCompel() *profile.AppProfileV2 {
	v2, err := nocalhost.GetProfileV2(a.NameSpace, a.Name)
	clientgoutils.Must(err)
	return v2
}

func (a *Application) UpdateProfile(modify func(*profile.AppProfileV2) error) error {
	p, err := a.getProfileForUpdate()
	if err != nil {
		return err
	}
	defer p.CloseDb()

	if err := modify(p); err != nil {
		return err
	}
	return p.Save()
}

// You need to closeDB for profile explicitly
func (a *Application) getProfileForUpdate() (*profile.AppProfileV2, error) {
	return profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
}

func (a *Application) LoadConfigFromLocalV2() (*profile.NocalHostAppConfigV2, error) {

	isV2, err := a.checkIfAppConfigIsV2()
	if err != nil {
		return nil, err
	}

	if !isV2 {
		log.Log("Upgrade config V1 to V2 ...")
		err = a.UpgradeAppConfigV1ToV2()
		if err != nil {
			return nil, err
		}
	}

	config := &profile.NocalHostAppConfigV2{}
	rbytes, err := ioutil.ReadFile(a.GetConfigV2Path())
	if err != nil {
		return nil, errors.New(fmt.Sprintf("fail to load configFile : %s", a.GetConfigV2Path()))
	}
	if err = yaml.Unmarshal(rbytes, config); err != nil {
		re, _ := regexp.Compile("remoteDebugPort: \"[0-9]*\"")
		rep := re.ReplaceAllString(string(rbytes), "")
		if err = yaml.Unmarshal([]byte(rep), config); err != nil {
			return nil, errors.Wrap(err, "")
		}
	}

	return config, nil
}

type HelmFlags struct {
	Debug    bool
	Wait     bool
	Set      []string
	Values   string
	Chart    string
	RepoName string
	RepoUrl  string
	Version  string
}

func (a *Application) GetApplicationConfigV2() *profile.ApplicationConfig {
	return a.appMeta.Config.ApplicationConfig
}

func (a *Application) GetAppProfileV2() *profile.ApplicationConfig {
	profileV2, _ := a.GetProfile()
	return &profile.ApplicationConfig{
		ResourcePath: profileV2.ResourcePath,
		IgnoredPath:  profileV2.IgnoredPath,
		PreInstall:   profileV2.PreInstall,
		Env:          profileV2.Env,
		EnvFrom:      profileV2.EnvFrom,
	}
}

func (a *Application) SaveAppProfileV2(config *profile.ApplicationConfig) error {
	return a.UpdateProfile(
		func(p *profile.AppProfileV2) error {
			p.ResourcePath = config.ResourcePath
			p.IgnoredPath = config.IgnoredPath
			p.PreInstall = config.PreInstall
			p.Env = config.Env
			p.EnvFrom = config.EnvFrom
			return nil
		},
	)
}

type PortForwardOptions struct {
	Pid     int      `json:"pid" yaml:"pid"`
	DevPort []string // 8080:8080 or :8080 means random localPort
	PodName string   // directly port-forward pod
	//ServiceType string   // service type such deployment
	Way         string // port-forward way, value is manual or devPorts
	RunAsDaemon bool
	Forward     bool
	Follow      bool // will stock until send ctrl+c or occurs error
}

type PortForwardEndOptions struct {
	Port string // 8080:8080
}

func (a *Application) Controller(name string, svcType base.SvcType) *controller.Controller {
	return &controller.Controller{
		NameSpace: a.NameSpace,
		AppName:   a.Name,
		Name:      name,
		Type:      svcType,
		Client:    a.client,
		AppMeta:   a.appMeta,
	}
}

func (a *Application) GetConfigFile() (string, error) {
	configFile, err := ioutil.ReadFile(a.GetConfigPath())
	if err == nil {
		return string(configFile), err
	}
	return "", err
}

func (a *Application) GetDescription() *profile.AppProfileV2 {
	appProfile, _ := a.GetProfile()
	if appProfile != nil {
		meta, err := nocalhost.GetApplicationMeta(a.Name, a.NameSpace, a.KubeConfig)
		if err != nil {
			log.LogE(err)
			return nil
		}
		appProfile.Installed = meta.IsInstalled()
		devMeta := meta.DevMeta

		// first iter from local svcProfile
		for _, svcProfile := range appProfile.SvcProfile {
			svcType := base.SvcTypeOf(svcProfile.Type)

			svcProfile.Developing = meta.CheckIfSvcDeveloping(svcProfile.ActualName, svcType)
			svcProfile.Possess = a.appMeta.SvcDevModePossessor(
				svcProfile.ActualName, svcType,
				appProfile.Identifier,
			)

			if m := devMeta[svcType.Alias()]; m != nil {
				delete(m, svcProfile.ActualName)
			}
		}

		// then gen the fake profile for remote svc
		for svcTypeAlias, m := range devMeta {
			for svcName, _ := range m {
				svcProfile := appProfile.SvcProfileV2(svcName, string(svcTypeAlias.Origin()))

				svcProfile.Developing = true
				svcProfile.Possess = a.appMeta.SvcDevModePossessor(
					svcProfile.ActualName, svcTypeAlias.Origin(),
					appProfile.Identifier,
				)
			}
		}

		return appProfile
	}
	return nil
}

func (a *Application) ListContainersByDeployment(depName string) ([]corev1.Container, error) {
	pods, err := a.client.ListPodsByDeployment(depName)
	if err != nil {
		return nil, err
	}
	if pods == nil || len(pods.Items) == 0 {
		return nil, errors.New("No pod found in deployment ???")
	}
	return pods.Items[0].Spec.Containers, nil
}

func (a *Application) SendPortForwardTCPHeartBeat(addressWithPort string) error {
	conn, err := net.Dial("tcp", addressWithPort)
	if err != nil || conn == nil {
		return errors.New(fmt.Sprintf("connect port-forward heartbeat address fail, %s", addressWithPort))
	}
	// GET /heartbeat HTTP/1.1
	_, err = conn.Write([]byte("ping"))
	return errors.Wrap(err, "send port-forward heartbeat fail")
}

func (a *Application) PortForwardAPod(req clientgoutils.PortForwardAPodRequest) error {
	return a.client.PortForwardAPod(req)
}

// set pid file empty
func (a *Application) SetPidFileEmpty(filePath string) error {
	return os.Remove(filePath)
}

func (a *Application) CleanUpTmpResources() error {
	log.Log("Clean up tmp resources...")
	return errors.Wrap(
		os.RemoveAll(a.ResourceTmpDir),
		fmt.Sprintf("fail to remove resources dir %s", a.ResourceTmpDir),
	)
}

func (a *Application) CleanupResources() error {
	log.Info("Remove resource files...")
	homeDir := a.GetHomeDir()
	return errors.Wrap(
		os.RemoveAll(homeDir),
		fmt.Sprintf("fail to remove resources dir %s", homeDir),
	)
}

func (a *Application) Uninstall(force bool) error {
	return a.appMeta.Uninstall(force)
}

func (a *Application) IsAnyServiceInDevMode() bool {
	for _, m := range a.appMeta.DevMeta {
		if len(m) > 0 {
			return true
		}
	}
	return false
}

func (a *Application) PortForwardFollow(podName string, localPort int, remotePort int, kubeconfig, ns string) error {
	client, err := clientgoutils.NewClientGoUtils(kubeconfig, ns)
	if err != nil {
		panic(err)
	}

	fps := []*clientgoutils.ForwardPort{{LocalPort: localPort, RemotePort: remotePort}}

	pf, err := client.CreatePortForwarder(podName, fps)
	if err != nil {
		panic(err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- pf.ForwardPorts()
	}()
	go func() {
		for {
			select {
			case <-pf.Ready:
				fmt.Printf("Forwarding from 127.0.0.1:%d -> %d\n", localPort, remotePort)
				fmt.Printf("Forwarding from [::1]:%d -> %d\n", localPort, remotePort)
				return
			}
		}
	}()
	for {
		select {
		case <-errChan:
			fmt.Println("err")
		}
	}
}
