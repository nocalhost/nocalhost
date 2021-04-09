/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	nocalhostDb "nocalhost/internal/nhctl/nocalhost/db"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	port_forward "nocalhost/internal/nhctl/port-forward"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
)

const (

	// default is a special app type, it can be uninstalled neither installed
	// it's a virtual application to managed that those manifest out of Nocalhost management
	DefaultNocalhostApplication           = "default.application"
	DefaultNocalhostApplicationOperateErr = "default.application is a virtual application to managed that those manifest out of Nocalhost management so can't be install, uninstall, reset, etc."

	HelmReleaseName               = "meta.helm.sh/release-name"
	AppManagedByLabel             = "app.kubernetes.io/managed-by"
	AppManagedByNocalhost         = "nocalhost"
	NocalhostApplicationName      = "dev.nocalhost/application-name"
	NocalhostApplicationNamespace = "dev.nocalhost/application-namespace"
)

var (
	ErrNotFound = errors.New("Application not found")
)

type Application struct {
	Name       string
	NameSpace  string
	KubeConfig string

	// may nil, only for install or upgrade
	// dir use to load the user's resource
	ResourceTmpDir string

	appMeta *appmeta.ApplicationMeta

	// profileV2 is created and saved to leveldb when `install`
	// profileV2 will not be nil if you use NewApplication a get a Application
	// you can only get const data from it, such as Namespace,AppType...
	// don't save it to leveldb directly
	profileV2 *profile.AppProfileV2

	client *clientgoutils.ClientGoUtils

	// for upgrade
	upgradeSortedPreInstallManifest []string
	upgradeInstallManifest          []string
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

	return nocalhost.UpdateProfileV2(a.NameSpace, a.Name, profileV2)
}

// When new a application, kubeconfig is required to get meta in k8s cluster
// KubeConfig can be acquired from profile in leveldb
func NewApplication(name string, ns string, kubeconfig string, initClient bool) (*Application, error) {

	app := &Application{
		Name:       name,
		NameSpace:  ns,
		KubeConfig: kubeconfig,
	}

	var err error
	if app.appMeta, err = nocalhost.GetApplicationMeta(app.Name, app.NameSpace, app.KubeConfig); err != nil {
		return nil, err
	}
	if !app.appMeta.IsInstalled() {
		return nil, ErrNotFound
	}

	if db, err := nocalhostDb.OpenApplicationLevelDB(app.NameSpace, app.Name, true); err != nil {
		err = nocalhostDb.CreateApplicationLevelDB(app.NameSpace, app.Name, true) // Init leveldb dir
		if err != nil {
			return nil, err
		}
	} else {
		_ = db.Close()
	}

	if app.profileV2, err = nocalhost.GetProfileV2(app.NameSpace, app.Name); err != nil {
		if _, err := os.Stat(app.getProfileV2Path()); err == nil { // todo: hjh move to version upgrading
			if err = app.moveProfileFromFileToLeveldb(); err != nil {
				return nil, err
			}
		}

		app.profileV2 = generateProfileFromConfig(app.appMeta.Config)
		if err = nocalhost.UpdateProfileV2(app.NameSpace, app.Name, app.profileV2); err != nil {
			return nil, err
		}
	}

	//if len(appProfile.PreInstall) == 0 && len(app.configV2.ApplicationConfig.PreInstall) > 0 {
	//	appProfile.PreInstall = app.configV2.ApplicationConfig.PreInstall
	//	if err = nocalhost.UpdateProfileV2(app.NameSpace, app.Name, appProfile); err != nil {
	//		return nil, err
	//	}
	//}

	if kubeconfig != "" && kubeconfig != app.profileV2.Kubeconfig {
		app.profileV2.Kubeconfig = kubeconfig
		if err = nocalhost.UpdateProfileV2(app.NameSpace, app.Name, app.profileV2); err != nil {
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

func (a *Application) GetProfile() (*profile.AppProfileV2, error) {
	return nocalhost.GetProfileV2(a.NameSpace, a.Name)
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
		return nil, errors.Wrap(err, "")
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

func (a *Application) IsAnyServiceInDevMode() bool {
	appProfile, _ := a.GetProfile()
	for _, svc := range appProfile.SvcProfile {
		if svc.Developing {
			return true
		}
	}
	return false
}

// Deprecated
//func (a *Application) GetSvcConfigV2(svcName string) *profile.ServiceConfigV2 {
//	for _, config := range a.appMeta.Config.ApplicationConfig.ServiceConfigs {
//		if config.Name == svcName {
//			return config
//		}
//	}
//	return nil
//}

func (a *Application) GetApplicationConfigV2() *profile.ApplicationConfig {
	return a.appMeta.Config.ApplicationConfig
}

func (a *Application) SaveSvcProfileV2(svcName string, config *profile.ServiceConfigV2) error {

	profileV2, err := profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcPro := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcPro != nil {
		config.Name = svcName
		svcPro.ServiceConfigV2 = config
	} else {
		config.Name = svcName
		svcPro = &profile.SvcProfileV2{
			ServiceConfigV2: config,
			ActualName:      svcName,
		}
		profileV2.SvcProfile = append(profileV2.SvcProfile, svcPro)
	}

	return profileV2.Save()
}

func (a *Application) GetAppProfileV2() *profile.ApplicationConfig {
	//a.LoadAppProfileV2()
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

	//a.AppProfileV2.ResourcePath = config.ResourcePath
	//a.AppProfileV2.IgnoredPath = config.IgnoredPath
	//a.AppProfileV2.PreInstall = config.PreInstall
	//a.AppProfileV2.Env = config.Env
	//a.AppProfileV2.EnvFrom = config.EnvFrom

	profileV2.ResourcePath = config.ResourcePath
	profileV2.IgnoredPath = config.IgnoredPath
	profileV2.PreInstall = config.PreInstall
	profileV2.Env = config.Env
	profileV2.EnvFrom = config.EnvFrom

	return profileV2.Save()
}

func (a *Application) RollBack(svcName string, reset bool) error {
	clientUtils := a.client
	//clientUtils.deployment

	dep, err := clientUtils.GetDeployment(svcName)
	if err != nil {
		return err
	}

	rss, err := clientUtils.GetSortedReplicaSetsByDeployment(svcName)
	if err != nil {
		log.WarnE(err, "Failed to get rs list")
		return err
	}

	// Find previous replicaSet
	if len(rss) < 2 {
		log.Warn("No history to roll back")
		return nil
	}

	var r *v1.ReplicaSet
	var originalPodReplicas *int32
	for _, rs := range rss {
		if rs.Annotations == nil {
			continue
		}
		// Mark the original revision
		if rs.Annotations[DevImageRevisionAnnotationKey] == DevImageRevisionAnnotationValue {
			r = rs
			if rs.Annotations[DevImageOriginalPodReplicasAnnotationKey] != "" {
				podReplicas, _ := strconv.Atoi(rs.Annotations[DevImageOriginalPodReplicasAnnotationKey])
				podReplicas32 := int32(podReplicas)
				originalPodReplicas = &podReplicas32
			}
		}
	}
	if r == nil {
		if !reset {
			return errors.New("Failed to find the proper revision to rollback")
		} else {
			r = rss[0]
		}
	}

	dep.Spec.Template = r.Spec.Template
	if originalPodReplicas != nil {
		dep.Spec.Replicas = originalPodReplicas
	}

	//spinner := utils.NewSpinner(" Rolling container's revision back...")
	//spinner.Start()
	//dep, err = clientUtils.UpdateDeployment(dep, true)
	log.Info(" Deleting current revision...")
	err = clientUtils.DeleteDeployment(dep.Name, false)
	if err != nil {
		return err
	}

	log.Info(" Recreating original revision...")
	dep.ResourceVersion = ""
	if len(dep.Annotations) == 0 {
		dep.Annotations = make(map[string]string, 0)
	}
	dep.Annotations["nocalhost-dep-ignore"] = "true"

	// Add labels and annotations
	if dep.Labels == nil {
		dep.Labels = make(map[string]string, 0)
	}
	dep.Labels[AppManagedByLabel] = AppManagedByNocalhost

	if dep.Annotations == nil {
		dep.Annotations = make(map[string]string, 0)
	}
	dep.Annotations[NocalhostApplicationName] = a.Name
	dep.Annotations[NocalhostApplicationNamespace] = a.NameSpace

	_, err = clientUtils.CreateDeployment(dep)
	if err != nil {
		if strings.Contains(err.Error(), "initContainers") && strings.Contains(err.Error(), "Duplicate") {
			log.Warn("[Warning] Nocalhost-dep needs to update")
		}
		return err
	}

	//spinner.Stop()
	//if err != nil {
	//	coloredoutput.Fail("Failed to roll revision back")
	//} else {
	//	coloredoutput.Success("Workload has been rollback")
	//}

	return err
}

type PortForwardOptions struct {
	Pid         int      `json:"pid" yaml:"pid"`
	DevPort     []string // 8080:8080 or :8080 means random localPort
	PodName     string   // directly port-forward pod
	ServiceType string   // service type such deployment
	Way         string   // port-forward way, value is manual or devPorts
	RunAsDaemon bool
	Forward     bool
}

type PortForwardEndOptions struct {
	Port string // 8080:8080
}

func (a *Application) CheckIfSvcExist(name string, svcType SvcType) (bool, error) {
	switch svcType {
	case Deployment:
		//ctx, _ := context.WithTimeout(context.TODO(), DefaultClientGoTimeOut)
		dep, err := a.client.GetDeployment(name)
		if err != nil {
			return false, errors.Wrap(err, "")
		}
		if dep == nil {
			return false, nil
		} else {
			return true, nil
		}
	case StatefulSet:
		dep, err := a.client.GetStatefulSet(name)
		if err != nil {
			return false, errors.Wrap(err, "")
		}
		if dep == nil {
			return false, nil
		} else {
			return true, nil
		}
	case DaemonSet:
		dep, err := a.client.GetDaemonSet(name)
		if err != nil {
			return false, errors.Wrap(err, "")
		}
		if dep == nil {
			return false, nil
		} else {
			return true, nil
		}
	case Job:
		dep, err := a.client.GetJobs(name)
		if err != nil {
			return false, errors.Wrap(err, "")
		}
		if dep == nil {
			return false, nil
		} else {
			return true, nil
		}
	case CronJob:
		dep, err := a.client.GetCronJobs(name)
		if err != nil {
			return false, errors.Wrap(err, "")
		}
		if dep == nil {
			return false, nil
		} else {
			return true, nil
		}
	default:
		return false, errors.New("unsupported svc type")
	}
}

func isContainerReadyAndRunning(containerName string, pod *corev1.Pod) bool {
	if len(pod.Status.ContainerStatuses) == 0 {
		return false
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName && status.Ready && status.State.Running != nil {
			return true
		}
	}
	return false
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
			svcProfile.Developing = meta.CheckIfDeploymentDeveloping(svcProfile.ActualName)
		}
		bytes, err := yaml.Marshal(appProfile)
		if err == nil {
			desc = string(bytes)
		}
	}
	return desc
}

func (a *Application) GetSvcDescription(svcName string) string {
	appProfile, _ := a.GetProfile()
	desc := ""
	profile := appProfile.FetchSvcProfileV2FromProfile(svcName)
	if profile != nil {
		profile.Developing = a.appMeta.CheckIfDeploymentDeveloping(svcName)
		bytes, err := yaml.Marshal(profile)
		if err == nil {
			desc = string(bytes)
		}
	}
	return desc
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

// Role: If set to "SYNC", means it is a pf used for syncthing
func (a *Application) PortForward(deployment, podName string, localPort, remotePort int, role string) error {
	if localPort == 0 || remotePort == 0 {
		return errors.New(fmt.Sprintf("Port-forward %d:%d failed", localPort, remotePort))
	}

	if isAvailable := ports.IsTCP4PortAvailable("0.0.0.0", localPort); isAvailable {
		log.Infof("Port %d is available", localPort)
	} else {
		return errors.New(fmt.Sprintf("Port %d is unavailable", localPort))
	}

	isAdmin := utils.IsSudoUser()
	client, err := daemon_client.NewDaemonClient(isAdmin)
	if err != nil {
		return err
	}
	nhResource := &model.NocalHostResource{
		NameSpace:   a.NameSpace,
		Application: a.Name,
		Service:     deployment,
		PodName:     podName,
	}

	if err = client.SendStartPortForwardCommand(nhResource, localPort, remotePort, role); err != nil {
		return err
	} else {
		log.Infof("Port-forward %d:%d has been started", localPort, remotePort)
		return a.SetPortForwardedStatus(deployment, true) //  todo: move port-forward start
	}
}

func (a *Application) CheckPidPortStatus(ctx context.Context, deployment string, sLocalPort, sRemotePort int, lock *sync.Mutex) {
	for {
		select {
		case <-ctx.Done():
			log.Info("Stop Checking port status")
			//_ = a.UpdatePortForwardStatus(deployment, sLocalPort, sRemotePort, portStatus, "Stopping")
			return
		default:
			portStatus := port_forward.PidPortStatus(os.Getpid(), sLocalPort)
			log.Infof("Checking Port %d:%d's status: %s", sLocalPort, sRemotePort, portStatus)
			lock.Lock()
			_ = a.UpdatePortForwardStatus(deployment, sLocalPort, sRemotePort, portStatus, "Check Pid")
			lock.Unlock()
			//}
			<-time.After(2 * time.Minute)
		}
	}
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

func (a *Application) GetBackgroundSyncPortForwardPid(deployment string, isTrunc bool) (int, string, error) {
	f, err := ioutil.ReadFile(a.GetApplicationBackGroundPortForwardPidFile(deployment))
	if err != nil {
		return 0, a.GetApplicationBackGroundPortForwardPidFile(deployment), err
	}
	port, err := strconv.Atoi(string(f))
	if err != nil {
		return 0, a.GetApplicationBackGroundPortForwardPidFile(deployment), err
	}
	if isTrunc {
		_ = a.SetPidFileEmpty(a.GetApplicationBackGroundPortForwardPidFile(deployment))
	}
	return port, a.GetApplicationBackGroundPortForwardPidFile(deployment), nil
}

func (a *Application) GetBackgroundSyncThingPid(deployment string, isTrunc bool) (int, string, error) {
	f, err := ioutil.ReadFile(a.GetApplicationSyncThingPidFile(deployment))
	if err != nil {
		return 0, a.GetApplicationSyncThingPidFile(deployment), err
	}
	port, err := strconv.Atoi(string(f))
	if err != nil {
		return 0, a.GetApplicationSyncThingPidFile(deployment), err
	}
	if isTrunc {
		_ = a.SetPidFileEmpty(a.GetApplicationBackGroundPortForwardPidFile(deployment))
	}
	return port, a.GetApplicationSyncThingPidFile(deployment), nil
}

func (a *Application) GetBackgroundOnlyPortForwardPid(deployment string, isTrunc bool) (int, string, error) {
	f, err := ioutil.ReadFile(a.GetApplicationOnlyPortForwardPidFile(deployment))
	if err != nil {
		return 0, a.GetApplicationOnlyPortForwardPidFile(deployment), err
	}
	port, err := strconv.Atoi(string(f))
	if err != nil {
		return 0, a.GetApplicationOnlyPortForwardPidFile(deployment), err
	}
	if isTrunc {
		_ = a.SetPidFileEmpty(a.GetApplicationBackGroundPortForwardPidFile(deployment))
	}
	return port, a.GetApplicationOnlyPortForwardPidFile(deployment), nil
}

func (a *Application) WriteBackgroundSyncPortForwardPidFile(deployment string, pid int) error {
	file, err := os.OpenFile(a.GetApplicationBackGroundPortForwardPidFile(deployment), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.New("fail open application file sync background port-forward pid file")
	}
	defer file.Close()
	sPid := strconv.Itoa(pid)
	_, err = file.Write([]byte(sPid))
	return errors.Wrap(err, "")
}

func (a *Application) GetSyncthingLocalDirFromProfileSaveByDevStart(svcName string, options *DevStartOptions) (*DevStartOptions, error) {
	appProfile, _ := a.GetProfile()
	svcProfile := appProfile.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return options, errors.New("get " + svcName + " profile fail, please reinstall application")
	}
	options.LocalSyncDir = svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin
	return options, nil
}

func (a *Application) GetPodsFromDeployment(deployment string) (*corev1.PodList, error) {
	return a.client.ListPodsByDeployment(deployment)
}

func (a *Application) GetDefaultPodName(ctx context.Context, svc string, t SvcType) (string, error) {
	var (
		podList *corev1.PodList
		err     error
	)
	for {
		select {
		case <-ctx.Done():
			return "", errors.New(fmt.Sprintf("Fail to get %s' pod", svc))
		default:
			switch t {
			case Deployment:
				podList, err = a.GetPodsFromDeployment(svc)
				if err != nil {
					return "", err
				}
			case StatefulSet:
				podList, err = a.GetClient().ListPodsByStatefulSet(svc)
				if err != nil {
					return "", err
				}
			default:
				return "", errors.New(fmt.Sprintf("Service type %s not support", t))
			}
		}
		if podList == nil || len(podList.Items) == 0 {
			log.Infof("Pod of %s has not been ready, waiting for it...", svc)
			time.Sleep(time.Second)
		} else {
			return podList.Items[0].Name, nil
		}
	}
}

func (a *Application) GetNocalhostDevContainerPod(deployment string) (string, error) {
	checkPodsList, err := a.GetPodsFromDeployment(deployment)
	if err != nil {
		return "", err
	}
	found := false
	for _, pod := range checkPodsList.Items {
		if pod.Status.Phase == "Running" {
			for _, container := range pod.Spec.Containers {
				if container.Name == DefaultNocalhostSideCarName {
					found = true
					break
				}
			}
			if found {
				return pod.Name, nil
			}
		}
	}
	return "", errors.New("dev container not found")
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
	return errors.Wrap(os.RemoveAll(a.ResourceTmpDir), fmt.Sprintf("fail to remove resources dir %s", a.ResourceTmpDir))
}

func (a *Application) CleanupResources() error {
	log.Info("Remove resource files...")
	homeDir := a.GetHomeDir()
	return errors.Wrap(os.RemoveAll(homeDir), fmt.Sprintf("fail to remove resources dir %s", homeDir))
}

func (a *Application) Uninstall() error {
	return a.appMeta.Uninstall()
}
