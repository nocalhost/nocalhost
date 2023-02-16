/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/dev_dir"
	"nocalhost/internal/nhctl/envsubst"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/nocalhost"
	nocalhostDb "nocalhost/internal/nhctl/nocalhost/db"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"strconv"
)

var (
	// do not change this error message
	ErrNotFound   = errors.New("Application not found")
	ErrInstalling = errors.New("Application is installing")
	indent        = 70
)

type Application struct {
	Name       string
	NameSpace  string
	KubeConfig string
	AppType    string
	Identifier string

	// may be nil, only for install or upgrade
	// dir use to load the user's resource
	ResourceTmpDir string
	shouldClean    bool

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

// NewApplication When new a application, kubeconfig is required to get meta in k8s cluster
// KubeConfig can be acquired from profile in leveldb
func NewApplication(name string, ns string, kubeconfig string, initClient bool) (*Application, error) {
	return newApplication(name, ns, kubeconfig, nil, initClient)
}

func NewApplicationM(name string, ns string, kubeconfig string, meta *appmeta.ApplicationMeta, initClient bool) (*Application, error) {
	return newApplication(name, ns, kubeconfig, meta, initClient)
}

func newApplication(name string, ns string, kubeconfig string, meta *appmeta.ApplicationMeta, initClient bool) (*Application, error) {

	var err error
	if kubeconfig == "" { // use default config
		kubeconfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
	}
	app := &Application{
		Name:       name,
		NameSpace:  ns,
		KubeConfig: kubeconfig,
	}

	if _, err = os.Stat(app.KubeConfig); err != nil {
		return nil, err
	}

	if meta == nil {
		if app.appMeta, err = nocalhost.GetApplicationMeta(app.Name, app.NameSpace, app.KubeConfig); err != nil {
			return nil, err
		}
	} else {
		app.appMeta = meta
	}

	// 1. first try load profile from local or earlier version
	// 2. (x deprecated) check should generate secret for adapt earlier version
	// 3. try load application meta from secret
	// 4. update kubeconfig for profile
	// 5. init go client inner Application

	if app.appMeta.IsUnknown() {
		return nil, errors.New(fmt.Sprintf("%s-%s state is UNKNOWN", app.NameSpace, app.Name))
	}

	if app.appMeta.IsNotInstall() {
		return nil, errors.Wrap(ErrNotFound, fmt.Sprintf("%s-%s not found", app.NameSpace, app.Name))
	}

	if err = app.appMeta.GenerateNidINE(); err != nil {
		return nil, err
	}

	if err = nocalhost.MigrateNsDirToSupportNidIfNeeded(app.Name, app.NameSpace, app.appMeta.NamespaceId); err != nil {
		return nil, err
	}

	if err := app.createLevelDbIfNotExists(); err != nil {
		return nil, err
	}

	// load from secret
	profileV2, err := nocalhost.GetProfileV2(app.NameSpace, app.Name, app.appMeta.NamespaceId)
	if err != nil {
		return nil, err
	}
	// Migrate config to meta
	if app.appMeta.Config == nil || !app.appMeta.Config.Migrated {
		if len(profileV2.SvcProfile) > 0 {
			c := app.newConfigFromProfile()
			for _, sc := range c.ApplicationConfig.ServiceConfigs {
				for _, scc := range sc.ContainerConfigs {
					if scc.Dev != nil {
						scc.Dev.Image = utils.ReplaceCodingcorpString(scc.Dev.Image)
						scc.Dev.GitUrl = utils.ReplaceCodingcorpString(scc.Dev.GitUrl)
					}
				}
			}
			app.appMeta.Config = c
			app.appMeta.Config.Migrated = true
			if err = app.appMeta.Update(); err != nil {
				return nil, err
			}
		}
	}

	if profileV2.Identifier == "" {
		if err = app.UpdateProfile(
			func(v2 *profile.AppProfileV2) error {
				v2.GenerateIdentifierIfNeeded()
				return nil
			},
		); err != nil {
			return nil, err
		}
		if profileV2, err = app.GetProfile(); err != nil {
			return nil, err
		}
	}

	app.AppType = profileV2.AppType
	app.Identifier = profileV2.Identifier

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

// Deprecated: this method is no need any more, because config is always load from secrets now
func (a *Application) ReloadCfg(reloadFromMeta, silence bool) error {
	secretCfg := a.appMeta.Config
	for _, config := range secretCfg.ApplicationConfig.ServiceConfigs {
		if err := a.ReloadSvcCfg(config.Name, base.SvcType(config.Type), reloadFromMeta, silence); err != nil {
			log.LogE(err)
		}
	}

	return nil
}

// ReloadSvcCfg try load config from cm first
// then load from local under associateDir/.nocalhost/config.yaml
// at last load config from local profile
func (a *Application) ReloadSvcCfg(svcName string, svcType base.SvcType, reloadFromMeta, silence bool) error {

	if a.loadSvcCfgFromLocalIfValid(svcName, svcType, silence) {
		return nil
	}

	if a.loadSvcCfmFromAnnotationIfValid(svcName, svcType, silence) {
		return nil
	}

	a.loadSvcCfgFromCmIfValid(svcName, svcType, silence)
	return nil
}

func (a *Application) loadSvcCfmFromAnnotationIfValid(svcName string, svcType base.SvcType, silence bool) bool {
	hint := hintFunc(svcName, svcType, silence)

	mw, err := a.GetObjectMeta(svcName, svcType.String())
	if err != nil {
		return false
	}

	if mw.GetObjectMeta() == nil {
		return false
	}

	if mw.GetObjectMeta().GetAnnotations() == nil {
		return false
	}

	if v, ok := mw.GetObjectMeta().GetAnnotations()[appmeta.AnnotationKey]; !ok || v == "" {
		return false
	} else {
		_, // local config should not contain app config
			svcCfg, err := LoadSvcCfgFromStrIfValid(v, svcName, svcType)
		if err != nil {
			hint(
				"Load nocalhost svc config from [Resource:%s, Name:%s] annotation fail, err: %s",
				mw.GetObjectMeta().GetResourceVersion(), mw.GetObjectMeta().GetName(), err.Error(),
			)
			return false
		}
		if svcCfg == nil {
			hint("Load nocalhost svc config from annotations success, but can not find corresponding config.")
			return false
		}

		a.appMeta.Config.SetSvcConfigV2(*svcCfg)
		if a.appMeta.Update() != nil {
			log.WarnE(err, "Failed to update svc config to meta")
			return false
		}

		// means should cm cfg is valid, persist to profile
		var c *controller.Controller
		if c, err = a.Controller(svcName, svcType); err == nil {
			err = c.UpdateSvcProfile(
				func(svcProfile *profile.SvcProfileV2) error {
					hint("Success load svc config from annotation")

					svcProfile.Name = svcName
					svcProfile.Type = svcType.String()
					svcProfile.LocalConfigLoaded = false
					svcProfile.AnnotationsConfigLoaded = true
					svcProfile.CmConfigLoaded = false
					return nil
				},
			)
		}
		if err != nil {
			hint(
				"Load nocalhost svc config from [Resource:%s, Name:%s] annotation fail, fail while updating svc profile, err: %s",
				mw.GetObjectMeta().GetResourceVersion(), mw.GetObjectMeta().GetName(), err.Error(),
			)
			return false
		}
		return true
	}
}

func (a *Application) loadSvcCfgFromCmIfValid(svcName string, svcType base.SvcType, silence bool) bool {
	hint := hintFunc(svcName, svcType, silence)

	configMap, err := a.GetConfigMap(appmeta.ConfigMapName(a.appMeta.Application))
	if err != nil {
		return false
	}

	cfgStr := configMap.Data[appmeta.CmConfigKey]
	if cfgStr == "" {
		return false
	}

	_, // local config should not contain app config
		svcCfg, err := LoadSvcCfgFromStrIfValid(cfgStr, svcName, svcType)
	if err != nil {
		hint("Load nocalhost svc config from cm fail, err: %s", err.Error())
		return false
	}
	if svcCfg == nil {
		hint("Load nocalhost svc config from cm success, but can not find corresponding config.")
		return false
	}

	a.appMeta.Config.SetSvcConfigV2(*svcCfg)
	if a.appMeta.Update() != nil {
		log.WarnE(err, "Failed to update svc config to meta")
		return false
	}

	// means should cm cfg is valid, persist to profile
	var c *controller.Controller
	if c, err = a.Controller(svcName, svcType); err == nil {
		err = c.UpdateSvcProfile(
			func(svcProfile *profile.SvcProfileV2) error {
				hint("Success load svc config from cm")

				svcProfile.Name = svcName
				svcProfile.Type = svcType.String()
				svcProfile.LocalConfigLoaded = false
				svcProfile.AnnotationsConfigLoaded = false
				svcProfile.CmConfigLoaded = true
				return nil
			},
		)
	}
	if err != nil {
		hint("Load nocalhost svc config from cm fail, fail while updating svc profile, err: %s", err.Error())
		return false
	}
	return true
}

// LoadSvcCfgFromStrIfValid
// CAUTION: appCfg may nil!
func LoadSvcCfgFromStrIfValid(config string, svcName string, svcType base.SvcType) (
	*profile.NocalHostAppConfigV2, *profile.ServiceConfigV2, error) {
	var svcCfg *profile.ServiceConfigV2
	var appCfg *profile.NocalHostAppConfigV2
	var err error
	if svcCfg, _ = doLoadProfileFromSvcConfig(
		envsubst.TextRenderItem(config), svcName, svcType,
	); svcCfg == nil {
		if appCfg, svcCfg, err = doLoadProfileFromAppConfig(
			envsubst.TextRenderItem(config), svcName, svcType,
		); err != nil {
			return nil, nil, errors.New(fmt.Sprintf("can not load cfg, may has syntax error, Content: %s", config))
		}
	}

	return appCfg, svcCfg, nil
}

func (a *Application) loadSvcCfgFromLocalIfValid(svcName string, svcType base.SvcType, silence bool) bool {
	hint := hintFunc(svcName, svcType, silence)
	var err error

	meta := a.GetAppMeta()
	pack := dev_dir.NewSvcPack(
		meta.Ns,
		meta.NamespaceId,
		meta.Application,
		svcType,
		svcName,
		"",
	)

	associatePath := pack.GetAssociatePath()

	if associatePath == "" {
		return false
	}

	configFile := fp.NewFilePath(string(associatePath)).
		RelOrAbs(DefaultGitNocalhostDir).
		RelOrAbs(DefaultConfigNameInGitNocalhostDir)

	if err = configFile.CheckExist(); err != nil {
		return false
	}

	var svcCfg *profile.ServiceConfigV2
	if svcCfg, err = doLoadProfileFromSvcConfig(
		envsubst.LocalFileRenderItem{FilePathEnhance: configFile}, svcName, svcType,
	); svcCfg == nil {
		if _, // local config should not contain app config
			svcCfg, _ = doLoadProfileFromAppConfig(
			envsubst.LocalFileRenderItem{FilePathEnhance: configFile}, svcName, svcType,
		); svcCfg == nil {
			if err != nil {

				hint("Load nocalhost svc config from local fail, err: %s", err.Error())
			}
			return false
		}
	}

	a.appMeta.Config.SetSvcConfigV2(*svcCfg)
	if a.appMeta.Update() != nil {
		log.WarnE(err, "Failed to update svc config to meta")
		return false
	}

	// means should load svc cfg from local
	var c *controller.Controller
	if c, err = a.Controller(svcName, svcType); err == nil {
		err = c.UpdateSvcProfile(
			func(svcProfile *profile.SvcProfileV2) error {
				hint("Success load svc config from local file %s", configFile.Abs())
				svcCfg.Name = svcName
				svcCfg.Type = svcType.String()

				//svcProfile.ServiceConfigV2 = svcCfg
				svcProfile.LocalConfigLoaded = true
				svcProfile.AnnotationsConfigLoaded = false
				svcProfile.CmConfigLoaded = false
				return nil
			},
		)
	}
	if err != nil {
		hint("Load nocalhost svc config from local fail, fail while updating svc profile, err: %s", err.Error())
		return false
	}
	return true
}

func hintFunc(svcName string, svcType base.SvcType, silence bool) func(string, ...string) {
	metaInfo := fmt.Sprintf("[name: %s serviceType: %s]", svcName, svcType)
	return func(format string, s ...string) {
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
}

func doLoadProfileFromSvcConfig(renderItem envsubst.RenderItem, svcName string, svcType base.SvcType) (
	*profile.ServiceConfigV2, error,
) {
	config, err := RenderConfigForSvc(renderItem)
	if err != nil {
		return nil, err
	}

	if len(config) == 1 && config[0].Name == "" {
		config[0].Name = svcName
		config[0].Type = svcType.String()
		return config[0], nil
	}

	for _, svcConfig := range config {
		if svcConfig.Name == svcName && base.SvcType(svcConfig.Type) == svcType {
			return svcConfig, nil
		}
	}

	return nil, errors.New("Local config loaded, but no valid config found")
}

func doLoadProfileFromAppConfig(configFile envsubst.RenderItem, svcName string, svcType base.SvcType) (
	*profile.NocalHostAppConfigV2, *profile.ServiceConfigV2, error,
) {
	appConfig, err := RenderConfig(configFile)
	if err != nil {
		return nil, nil, err
	}

	return appConfig, appConfig.GetSvcConfigV2(svcName, svcType), nil
}

func (a *Application) newConfigFromProfile() *profile.NocalHostAppConfigV2 {

	profileV2, _ := a.GetProfile()
	return &profile.NocalHostAppConfigV2{
		ConfigProperties: profile.ConfigProperties{
			Version: "v2",
		},
		ApplicationConfig: profile.ApplicationConfig{
			Name:           a.Name,
			Type:           profileV2.AppType,
			ResourcePath:   profileV2.ResourcePath,
			IgnoredPath:    profileV2.IgnoredPath,
			PreInstall:     profileV2.PreInstall,
			ServiceConfigs: loadServiceConfigsFromProfile(profileV2.SvcProfile),
		},
	}
}

func loadServiceConfigsFromProfile(profiles []*profile.SvcProfileV2) []*profile.ServiceConfigV2 {
	var configs = []*profile.ServiceConfigV2{}

	for _, p := range profiles {
		//if p.ServiceConfigV2 != nil {
		//	configs = append(configs, p.ServiceConfigV2)
		//} else {
		configs = append(
			configs, &profile.ServiceConfigV2{
				Name: p.GetName(),
				Type: p.GetType(),
			},
		)
		//}
	}

	return configs
}

func (a *Application) createLevelDbIfNotExists() (err error) {
	if db, err := nocalhostDb.OpenApplicationLevelDB(a.NameSpace, a.Name, a.appMeta.NamespaceId, true); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			p := profile.AppProfileV2{}
			p.GenerateIdentifierIfNeeded()
			pBytes, _ := yaml.Marshal(&p)
			return nocalhostDb.CreateApplicationLevelDBWithProfile(
				a.NameSpace, a.Name, a.appMeta.NamespaceId, profile.ProfileV2Key(a.NameSpace, a.Name), pBytes, true,
			)
		}
		return err
	} else {
		_ = db.Close()
	}
	return nil
}

func (a *Application) GetProfile() (*profile.AppProfileV2, error) {
	return nocalhost.GetProfileV2(a.NameSpace, a.Name, a.appMeta.NamespaceId)
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
	return profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name, a.appMeta.NamespaceId)
}

//func (a *Application) LoadConfigFromLocalV2() (*profile.NocalHostAppConfigV2, error) {
//
//	isV2, err := a.checkIfAppConfigIsV2()
//	if err != nil {
//		return nil, err
//	}
//
//	if !isV2 {
//		log.Log("Upgrade config V1 to V2 ...")
//		err = a.UpgradeAppConfigV1ToV2()
//		if err != nil {
//			return nil, err
//		}
//	}
//
//	config := &profile.NocalHostAppConfigV2{}
//	rbytes, err := ioutil.ReadFile(a.GetConfigV2Path())
//	if err != nil {
//		return nil, errors.New(fmt.Sprintf("fail to load configFile : %s", a.GetConfigV2Path()))
//	}
//	if err = yaml.Unmarshal(rbytes, config); err != nil {
//		re, _ := regexp.Compile("remoteDebugPort: \"[0-9]*\"")
//		rep := re.ReplaceAllString(string(rbytes), "")
//		if err = yaml.Unmarshal([]byte(rep), config); err != nil {
//			return nil, errors.Wrap(err, "")
//		}
//	}
//
//	return config, nil
//}

type HelmFlags struct {
	Debug    bool
	Wait     bool
	Set      []string
	Values   []string
	Chart    string
	RepoName string
	RepoUrl  string
	Version  string
}

func (a *Application) GetApplicationConfigV2() *profile.ApplicationConfig {
	return &a.appMeta.Config.ApplicationConfig
}

func (a *Application) SaveAppProfileV2(config *profile.ApplicationConfig) error {
	return a.UpdateProfile(
		func(p *profile.AppProfileV2) error {
			p.ResourcePath = config.ResourcePath
			p.IgnoredPath = config.IgnoredPath
			p.PreInstall = config.PreInstall
			//p.Env = config.Env
			//p.EnvFrom = config.EnvFrom
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

func (a *Application) Controller(name string, svcType base.SvcType) (*controller.Controller, error) {
	if a.Identifier == "" {
		return nil, errors.New("Application's identifier cannot be nil")
	}
	return controller.NewController(a.NameSpace, name, a.Name, a.Identifier, svcType, a.client, a.appMeta)
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
		meta := a.appMeta
		appProfile.Installed = meta.IsInstalled()
		devMeta := meta.DevMeta

		// first iter from local svcProfile
		for _, svcProfile := range appProfile.SvcProfile {
			appmeta.FillingExtField(svcProfile, meta, a.Name, a.NameSpace, appProfile.Identifier)

			if m := devMeta[base.SvcType(svcProfile.GetType()).Alias()]; m != nil {
				delete(m, svcProfile.GetName())
			}
		}

		// then gen the fake profile for remote svc
		for svcTypeAlias, m := range devMeta {
			for svcName, _ := range m {
				if !appmeta.HasDevStartingSuffix(svcName) {
					svcProfile := appProfile.SvcProfileV2(svcName, string(svcTypeAlias.Origin()))
					appmeta.FillingExtField(svcProfile, meta, a.Name, a.NameSpace, appProfile.Identifier)
				}
			}
		}

		return appProfile
	}
	return nil
}

func (a *Application) PortForward(pod string, localPort, remotePort int, readyChan, stopChan chan struct{}, g genericclioptions.IOStreams) error {
	return a.client.ForwardPortForwardByPod(pod, localPort, remotePort, readyChan, stopChan, g)
}

func (a *Application) CleanUpTmpResources() error {
	if !a.shouldClean {
		return nil
	}
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

func (a *Application) PortForwardFollow(podName string, localPort int, remotePort int, okChan chan struct{}) error {
	client, err := clientgoutils.NewClientGoUtils(a.KubeConfig, a.NameSpace)
	if err != nil {
		return err
	}
	fps := []*clientgoutils.ForwardPort{{LocalPort: localPort, RemotePort: remotePort}}
	pf, err := client.CreatePortForwarder(podName, fps, nil, nil, genericclioptions.IOStreams{})
	if err != nil {
		return err
	}
	errChan := make(chan error, 1)
	go func() {
		if err = pf.ForwardPorts(); err != nil {
			errChan <- err
		}
	}()
	go func() {
		for {
			select {
			case <-pf.Ready:
				fmt.Printf("Forwarding from 127.0.0.1:%d -> %d\n", localPort, remotePort)
				fmt.Printf("Forwarding from [::1]:%d -> %d\n", localPort, remotePort)
				if okChan != nil {
					okChan <- struct{}{}
				}
				return
			}
		}
	}()
	return <-errChan
}

func (a *Application) InitService(svcName string, svcType string) (*controller.Controller, error) {
	if svcName == "" {
		return nil, errors.New("please use -d to specify a k8s workload")
	}
	st, err := nocalhost.SvcTypeOfMutate(svcType)
	if err != nil {
		return nil, err
	}
	return a.Controller(svcName, st)
}

func (a *Application) InitAndCheckIfSvcExist(svcName string, svcType string) (*controller.Controller, error) {
	nocalhostSvc, err := a.InitService(svcName, svcType)
	if err != nil {
		return nil, err
	}
	return nocalhostSvc, nocalhostSvc.CheckIfExist()
}
