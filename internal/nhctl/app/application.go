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
	"io/ioutil"
	"net"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/nocalhost"
	nocalhostDb "nocalhost/internal/nhctl/nocalhost/db"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/svc"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
)

const (

	// default is a special app type, it can be uninstalled neither installed
	// it's a virtual application to managed that those manifest out of Nocalhost management
	DefaultNocalhostApplication           = "default.application"
	DefaultNocalhostApplicationOperateErr = "default.application is a virtual application " +
		"to managed that those manifest out of Nocalhost" +
		" management so can't be install, uninstall, reset, etc."

	HelmReleaseName = "meta.helm.sh/release-name"
)

var (
	ErrNotFound = errors.New("Application not found")
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
		return nil, ErrNotFound
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
		//app.profileV2.Kubeconfig = kubeconfig
		p, err := profile.NewAppProfileV2ForUpdate(app.NameSpace, app.Name)
		if err != nil {
			return nil, err
		}
		p.Kubeconfig = kubeconfig
		_ = p.Save()

		if err = p.CloseDb(); err != nil {
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

	if profileV2 != nil && !profileV2.Secreted && a.appMeta.IsNotInstall() && a.Name != DefaultNocalhostApplication {
		a.AppType = profileV2.AppType

		defer func() {
			log.Logf("Mark application %s in ns %s has been secreted", a.Name, a.NameSpace)
			//a.profileV2.Secreted = true
			p, _ := profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
			p.Secreted = true
			p.Save()
			p.CloseDb()
			//_ = nocalhost.UpdateProfileV2(a.NameSpace, a.Name, a.profileV2)
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

		a.client = a.appMeta.GetClient()
		switch a.AppType {
		case string(appmeta.Manifest), string(appmeta.ManifestLocal):
			_ = a.InstallManifest(a.appMeta, a.getResourceDir(), false)
		case string(appmeta.KustomizeGit):
			_ = a.InstallKustomize(a.appMeta, a.getResourceDir(), false)
		default:
		}

		for _, svc := range profileV2.SvcProfile {
			if svc.Developing {
				_ = a.appMeta.SvcDevStart(svc.Name, appmeta.SvcType(svc.Type), profileV2.Identifier)
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

// You need to closeDB for profile explicitly
func (a *Application) GetProfileForUpdate() (*profile.AppProfileV2, error) {
	return profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
}

func (a *Application) SaveProfile(p *profile.AppProfileV2) error {
	return nocalhost.UpdateProfileV2(a.NameSpace, a.Name, p)
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

// Todo hxx
func (a *Application) IsAnyServiceInDevMode() bool {
	return len(a.appMeta.DevMeta) > 0 && len(a.appMeta.DevMeta[appmeta.DEPLOYMENT]) > 0
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
	profileV2, err := profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	profileV2.ResourcePath = config.ResourcePath
	profileV2.IgnoredPath = config.IgnoredPath
	profileV2.PreInstall = config.PreInstall
	profileV2.Env = config.Env
	profileV2.EnvFrom = config.EnvFrom

	return profileV2.Save()
}

type PortForwardOptions struct {
	Pid     int      `json:"pid" yaml:"pid"`
	DevPort []string // 8080:8080 or :8080 means random localPort
	PodName string   // directly port-forward pod
	//ServiceType string   // service type such deployment
	Way         string // port-forward way, value is manual or devPorts
	RunAsDaemon bool
	Forward     bool
}

type PortForwardEndOptions struct {
	Port string // 8080:8080
}

func (a *Application) Controller(name string, svcType appmeta.SvcType) *svc.Controller {
	return &svc.Controller{
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

func (a *Application) GetDescription() string {
	appProfile, _ := a.GetProfile()
	desc := ""
	if appProfile != nil {
		meta, err := nocalhost.GetApplicationMeta(a.Name, a.NameSpace, a.KubeConfig)
		if err != nil {
			log.LogE(err)
			return ""
		}
		appProfile.Installed = meta.IsInstalled()
		for _, svcProfile := range appProfile.SvcProfile {
			svcProfile.Developing = meta.CheckIfSvcDeveloping(svcProfile.ActualName, appmeta.SvcType(svcProfile.Type))
			svcProfile.Possess = a.appMeta.SvcDevModePossessor(svcProfile.ActualName, appmeta.SvcType(svcProfile.Type),
				appProfile.Identifier)
		}
		bytes, err := yaml.Marshal(appProfile)
		if err == nil {
			desc = string(bytes)
		}
	}
	return desc
}

//func (a *Application) GetSvcDescription(svcName string) string {
//	appProfile, _ := a.GetProfile()
//	desc := ""
//	profile := appProfile.SvcProfileV2(svcName)
//	if profile != nil {
//		profile.Developing = a.appMeta.CheckIfSvcDeveloping(svcName, appmeta.SvcType(profile.Type))
//		profile.Possess = a.appMeta.DeploymentDevModePossessor(svcName, appProfile.Identifier)
//		bytes, err := yaml.Marshal(profile)
//		if err == nil {
//			desc = string(bytes)
//		}
//	}
//	return desc
//}

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
	return errors.Wrap(os.RemoveAll(a.ResourceTmpDir),
		fmt.Sprintf("fail to remove resources dir %s", a.ResourceTmpDir),
	)
}

func (a *Application) CleanupResources() error {
	log.Info("Remove resource files...")
	homeDir := a.GetHomeDir()
	return errors.Wrap(os.RemoveAll(homeDir),
		fmt.Sprintf("fail to remove resources dir %s", homeDir),
	)
}

func (a *Application) Uninstall() error {
	return a.appMeta.Uninstall()
}
