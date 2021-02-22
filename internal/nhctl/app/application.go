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
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/syncthing/ports"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/nocalhost"
	port_forward "nocalhost/internal/nhctl/port-forward"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"

	"github.com/pkg/errors"
)

type AppType string

const (
	Helm          AppType = "helmGit"
	HelmRepo      AppType = "helmRepo"
	Manifest      AppType = "rawManifest"
	ManifestLocal AppType = "rawManifestLocal"
	HelmLocal     AppType = "helmLocal"
)

type Application struct {
	Name string
	//config   *NocalHostAppConfig //  this should not be nil
	configV2 *NocalHostAppConfigV2
	//AppProfile               *AppProfile // runtime info, this will not be nil
	AppProfileV2             *AppProfileV2
	client                   *clientgoutils.ClientGoUtils
	sortedPreInstallManifest []string // for pre install
	installManifest          []string // for install
}

type SvcDependency struct {
	Name string   `json:"name" yaml:"name"`
	Type string   `json:"type" yaml:"type"`
	Jobs []string `json:"jobs" yaml:"jobs,omitempty"`
	Pods []string `json:"pods" yaml:"pods,omitempty"`
}

func NewApplication(name string) (*Application, error) {
	app := &Application{
		Name: name,
	}

	err := app.LoadConfigV2()
	if err != nil {
		return nil, err
	}

	err = app.LoadAppProfileV2()
	if err != nil {
		return nil, err
	}

	if len(app.AppProfileV2.PreInstall) == 0 {
		app.AppProfileV2.PreInstall = app.configV2.ApplicationConfig.PreInstall
	}

	app.client, err = clientgoutils.NewClientGoUtils(app.GetKubeconfig(), app.GetNamespace())
	if err != nil {
		return nil, err
	}

	app.convertDevPortForwardList()

	return app, app.SaveProfile()
}

func (a *Application) ReadBeforeWriteProfile() error {
	return a.LoadAppProfileV2()
}

//func (a *Application) InitProfile(profile *AppProfile) {
//	if profile != nil {
//		a.AppProfile = profile
//	}
//}

//func (a *Application) LoadConfig() error {
//	config := &NocalHostAppConfig{}
//	if _, err := os.Stat(a.GetConfigPath()); err != nil {
//		if os.IsNotExist(err) {
//			a.config = config
//			return nil
//		} else {
//			return errors.Wrap(err, "fail to load configs")
//		}
//	}
//	rbytes, err := ioutil.ReadFile(a.GetConfigPath())
//	if err != nil {
//		return errors.New(fmt.Sprintf("fail to load configFile : %s", a.GetConfigPath()))
//	}
//	err = yaml.Unmarshal(rbytes, config)
//	if err != nil {
//		return errors.Wrap(err, err.Error())
//	}
//	a.config = config
//	return nil
//}

func (a *Application) LoadConfigV2() error {

	isV2, err := a.checkIfAppConfigIsV2()
	if err != nil {
		return err
	}

	if !isV2 {
		log.Info("Upgrade config V1 to V2 ...")
		err = a.UpgradeAppConfigV1ToV2()
		if err != nil {
			return err
		}
	}

	config := &NocalHostAppConfigV2{}
	if _, err := os.Stat(a.GetConfigV2Path()); err != nil {
		if os.IsNotExist(err) {
			a.configV2 = config
			return nil
		} else {
			return errors.Wrap(err, "fail to load configs")
		}
	}
	rbytes, err := ioutil.ReadFile(a.GetConfigV2Path())
	if err != nil {
		return errors.New(fmt.Sprintf("fail to load configFile : %s", a.GetConfigV2Path()))
	}
	err = yaml.Unmarshal(rbytes, config)
	if err != nil {
		return errors.Wrap(err, "")
	}
	a.configV2 = config
	return nil
}

func (a *Application) SaveProfile() error {

	v2Bytes, err := yaml.Marshal(a.AppProfileV2)
	if err != nil {
		return errors.Wrap(err, "")
	}

	err = ioutil.WriteFile(a.getProfileV2Path(), v2Bytes, 0644)
	return errors.Wrap(err, "")
}

func (a *Application) downloadResourcesFromGit(gitUrl string, gitRef string) error {
	var (
		err        error
		gitDirName string
	)

	if strings.HasPrefix(gitUrl, "https") || strings.HasPrefix(gitUrl, "git") || strings.HasPrefix(gitUrl, "http") {
		if strings.HasSuffix(gitUrl, ".git") {
			gitDirName = gitUrl[:len(gitUrl)-4]
		} else {
			gitDirName = gitUrl
		}
		strs := strings.Split(gitDirName, "/")
		gitDirName = strs[len(strs)-1] // todo : for default application name
		if len(gitRef) > 0 {
			_, err = tools.ExecCommand(nil, true, "git", "clone", "--branch", gitRef, "--depth", "1", gitUrl, a.getGitDir())
		} else {
			_, err = tools.ExecCommand(nil, true, "git", "clone", "--depth", "1", gitUrl, a.getGitDir())
		}
		if err != nil {
			return errors.Wrap(err, err.Error())
		}
	}
	return nil
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

// if a file is a preInstall/postInstall, it should be ignored in installing
func (a *Application) ignoredInInstall(manifest string) bool {
	for _, pre := range a.sortedPreInstallManifest {
		if pre == manifest {
			return true
		}
	}
	return false
}

func (a *Application) uninstallManifestRecursively() error {
	a.loadInstallManifest()

	if len(a.installManifest) > 0 {
		err := a.client.ApplyForDelete(a.installManifest, true)
		if err != nil {
			log.WarnE(err, "Error occurs when cleaning resources")
			return err
		}
	} else {
		log.Warn("nothing need to be uninstalled ??")
	}
	return nil
}

func (a *Application) preInstall() {

	a.loadSortedPreInstallManifest()

	if len(a.sortedPreInstallManifest) > 0 {
		log.Info("Run pre-install...")
		for _, item := range a.sortedPreInstallManifest {
			err := a.client.Create(item, true, false)
			if err != nil {
				log.Warnf("error occurs when install %s : %s\n", item, err.Error())
			}
		}
	}
}

func (a *Application) cleanPreInstall() {
	a.loadSortedPreInstallManifest()
	if len(a.sortedPreInstallManifest) > 0 {
		log.Debug("Cleaning up pre-install jobs...")
		err := a.client.ApplyForDelete(a.sortedPreInstallManifest, true)
		if err != nil {
			log.Warnf("error occurs when cleaning pre install resources : %s\n", err.Error())
		}
	} else {
		log.Debug("No pre-install job needs to clean up")
	}
}

func (a *Application) GetApplicationSyncDir(deployment string) string {
	dirPath := filepath.Join(a.GetHomeDir(), nocalhost.DefaultBinSyncThingDirName, deployment)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0700)
		if err != nil {
			log.Fatalf("fail to create syncthing directory: %s", dirPath)
		}
	}
	return dirPath
}

//func (a *Application) GetSvcConfig(svcName string) *ServiceDevOptions {
//	a.LoadConfig() // get the latest config
//	if a.config == nil {
//		return nil
//	}
//	for _, config := range a.config.SvcConfigs {
//		if config.Name == svcName {
//			return config
//		}
//	}
//	return nil
//}

func (a *Application) GetSvcConfigV2(svcName string) *ServiceConfigV2 {
	a.LoadConfigV2() // get the latest config
	if a.configV2 == nil {
		return nil
	}
	for _, config := range a.configV2.ApplicationConfig.ServiceConfigs {
		if config.Name == svcName {
			return config
		}
	}
	return nil
}

func (a *Application) GetApplicationConfigV2() *ApplicationConfig {
	a.LoadConfigV2() // get the latest config
	if a.configV2 == nil {
		return nil
	}
	return a.configV2.ApplicationConfig
}

//func (a *Application) SaveSvcProfile(svcName string, config *ServiceDevOptions) error {
//
//	svcPro := a.GetSvcProfile(svcName)
//	if svcPro != nil {
//		config.Name = svcName
//		svcPro.ServiceDevOptions = config
//	} else {
//		config.Name = svcName
//		svcPro = &SvcProfile{
//			ServiceDevOptions: config,
//			ActualName:        svcName,
//		}
//		a.AppProfile.SvcProfile = append(a.AppProfile.SvcProfile, svcPro)
//	}
//
//	return a.SaveProfile()
//}

func (a *Application) SaveSvcProfileV2(svcName string, config *ServiceConfigV2) error {

	svcPro := a.GetSvcProfileV2(svcName)
	if svcPro != nil {
		config.Name = svcName
		svcPro.ServiceConfigV2 = config
	} else {
		config.Name = svcName
		svcPro = &SvcProfileV2{
			ServiceConfigV2: config,
			ActualName:      svcName,
		}
		a.AppProfileV2.SvcProfile = append(a.AppProfileV2.SvcProfile, svcPro)
	}

	return a.SaveProfile()
}

func (a *Application) RollBack(ctx context.Context, svcName string, reset bool) error {
	clientUtils := a.client

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

	spinner := utils.NewSpinner(" Rolling container's revision back...")
	spinner.Start()
	dep, err = clientUtils.UpdateDeployment(dep, metav1.UpdateOptions{}, true)
	spinner.Stop()
	if err != nil {
		coloredoutput.Fail("Failed to roll revision back")
	} else {
		coloredoutput.Success("Workload has been rollback")
	}

	return err
}

type PortForwardOptions struct {
	Pid         int      `json:"pid" yaml:"pid"`
	DevPort     []string // 8080:8080 or :8080 means random localPort
	PodName     string   // directly port-forward pod
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
	default:
		return false, errors.New("unsupported svc type")
	}
	return false, nil
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
	desc := ""
	if a.AppProfileV2 != nil {
		bytes, err := yaml.Marshal(a.AppProfileV2)
		if err == nil {
			desc = string(bytes)
		}
	}
	return desc
}

func (a *Application) GetSvcDescription(svcName string) string {
	desc := ""
	profile := a.GetSvcProfileV2(svcName)
	if profile != nil {
		bytes, err := yaml.Marshal(profile)
		if err == nil {
			desc = string(bytes)
		}
	}
	return desc
}

func (a *Application) FixPortForwardOSArgs(localPort, remotePort []int) {
	var newArg []string
	for _, v := range os.Args {
		match := false
		for key, vv := range remotePort {
			if v == "-p" || v == fmt.Sprintf(":%d", vv) || v == fmt.Sprintf("%d:%d", localPort[key], vv) {
				match = true
			}
		}
		if !match {
			newArg = append(newArg, v)
		}
	}
	for k, v := range localPort {
		newArg = append(newArg, "-p", fmt.Sprintf("%d:%d", v, remotePort[k]))
	}
	os.Args = newArg
}

// for background port-forward
func (a *Application) PortForwardInBackGround(listenAddress []string, deployment, podName string, localPorts, remotePorts []int, way string, forwardActually bool) {
	if len(localPorts) != len(remotePorts) {
		log.Fatalf("dev port forward fail, please check you devPort in config\n")
	}

	if !forwardActually {
		if way == PortForwardDevPorts {
			// AppendDevPortManual
			// if from devPorts, check previously port-forward and add to port-forward list
			portForwardList := a.GetSvcProfileV2(deployment).DevPortForwardList
			for _, v := range portForwardList {
				exist := false
				for _, vv := range localPorts {
					if vv == v.LocalPort {
						exist = true
					}
				}
				if !exist {
					localPorts = append(localPorts, v.LocalPort)
					remotePorts = append(remotePorts, v.RemotePort)
					os.Args = append(os.Args, "-p", fmt.Sprintf("%d:%d", v.LocalPort, v.RemotePort))
				}
			}
		}

		for key, sLocalPort := range localPorts {
			a.EndDevPortForward(deployment, sLocalPort, remotePorts[key]) // kill existed port-forward
			isAvailable := ports.IsPortAvailable("0.0.0.0", sLocalPort)
			devPort := &DevPortForward{
				LocalPort:  sLocalPort,
				RemotePort: remotePorts[key],
				Way:        way,
				Status:     "",
				Updated:    time.Now().Format("2006-01-02 15:04:05"),
			}
			if isAvailable {
				log.Infof("Port %d is available", sLocalPort)
				devPort.Status = "AVAILABLE"
			} else {
				log.Infof("Port %d is unavailable", sLocalPort)
				devPort.Status = "UNAVAILABLE"
			}
			a.AppendPortForward(deployment, devPort)
		}
		_ = a.SetPortForwardedStatus(deployment, true)

		os.Args = append(os.Args, "--forward", "true")
		_, err := daemon.Background(a.GetPortForwardLogFile(deployment), a.GetApplicationBackGroundOnlyPortForwardPidFile(deployment), true)
		if err != nil {
			log.Fatal("Failed to run port-forward background, please try again")
		}
	}

	// isDaemon == false
	var wg sync.WaitGroup
	wg.Add(len(localPorts))

	for key, sLocalPort := range localPorts {

		go func(lPort int, rPort int) {
			for {
				// stopCh control the port forwarding lifecycle. When it gets closed the
				// port forward will terminate
				stopCh := make(chan struct{}, 1)
				// readyCh communicate when the port forward is ready to get traffic
				readyCh := make(chan struct{})
				endCh := make(chan struct{})

				// stream is used to tell the port forwarder where to place its output or
				// where to expect input if needed. For the port forwarding we just need
				// the output eventually
				stream := genericclioptions.IOStreams{
					In:     os.Stdin,
					Out:    os.Stdout,
					ErrOut: os.Stderr,
				}

				go func() {
					select {
					case <-readyCh:
						log.Info("Port forward is ready")
						go func() {
							a.CheckPidPortStatus(endCh, deployment, lPort, rPort)
						}()
						go func() {
							a.SendHeartBeat(endCh, listenAddress[0], lPort)
						}()
						_ = a.SetPortForwardPid(deployment, lPort, rPort, os.Getpid())
					}
				}()

				err := a.PortForwardAPod(clientgoutils.PortForwardAPodRequest{
					Listen: listenAddress,
					Pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      podName,
							Namespace: a.GetNamespace(),
						},
					},
					LocalPort: lPort,
					PodPort:   rPort,
					Streams:   stream,
					StopCh:    stopCh,
					ReadyCh:   readyCh,
				})
				if err != nil {
					if strings.Contains(err.Error(), "unable to listen on any of the requested ports") {
						log.Warnf("Unable to listen on port %d", lPort)
						_ = a.UpdatePortForwardStatus(deployment, lPort, rPort, "DISCONNECTED", fmt.Sprintf("Unable to listen on port %d", lPort))
						wg.Done()
						return
					}
					log.WarnE(err, "Port-forward failed, reconnecting after 30 seconds...")
					close(endCh)
					_ = a.UpdatePortForwardStatus(deployment, lPort, rPort, "RECONNECTING", "Port-forward failed, reconnecting after 30 seconds...")
					<-time.After(30 * time.Second)
				} else {
					log.Warn("Reconnecting after 30 seconds...")
					close(endCh)
					_ = a.UpdatePortForwardStatus(deployment, lPort, rPort, "RECONNECTING", "Reconnecting after 30 seconds...")
					<-time.After(30 * time.Second)
				}
				log.Info("Reconnecting...")
			}
		}(sLocalPort, remotePorts[key])

		// sleep while
		time.Sleep(time.Duration(2) * time.Second)
	}

	wg.Wait()
	log.Info("Stop port forward")
}

func (a *Application) SendHeartBeat(stopCh chan struct{}, listenAddress string, sLocalPort int) {
	for {
		select {
		case <-stopCh:
			log.Infof("Stop sending heart beat to %d", sLocalPort)
			return
		default:
			<-time.After(30 * time.Second)
			log.Infof("try to send port-forward heartbeat to %d", sLocalPort)
			err := a.SendPortForwardTCPHeartBeat(fmt.Sprintf("%s:%v", listenAddress, sLocalPort))
			if err != nil {
				log.Info("send port-forward heartbeat with err %s", err.Error())
			}
		}
	}
}

func (a *Application) CheckPidPortStatus(stopCh chan struct{}, deployment string, sLocalPort, sRemotePort int) {
	for {
		select {
		case <-stopCh:
			log.Info("Stop Checking port status")
			//_ = a.UpdatePortForwardStatus(deployment, sLocalPort, sRemotePort, portStatus, "Stopping")
			return
		default:
			portStatus := port_forward.PidPortStatus(os.Getpid(), sLocalPort)
			log.Infof("Checking Port %d:%d's status: %s", sLocalPort, sRemotePort, portStatus)
			currentStatus := ""
			for _, portForward := range a.GetSvcProfileV2(deployment).DevPortForwardList {
				if portForward.LocalPort == sLocalPort && portForward.RemotePort == sRemotePort {
					currentStatus = portForward.Status
					break
				}
			}
			if currentStatus != portStatus {
				_ = a.UpdatePortForwardStatus(deployment, sLocalPort, sRemotePort, portStatus, "Check Pid")
			}
			<-time.After(2 * time.Minute)
		}
	}
}

func (a *Application) SendPortForwardTCPHeartBeat(addressWithPort string) error {
	conn, err := net.Dial("tcp", addressWithPort)

	if err != nil || conn == nil {
		log.Warnf("connect port-forward heartbeat address fail, %s", addressWithPort)
		return nil
	}
	// GET /heartbeat HTTP/1.1
	_, err = conn.Write([]byte("ping"))
	if err != nil {
		log.Warnf("send port-forward heartbeat fail, %s", err.Error())
	}
	return err
}

func (a *Application) GetMyBinName() string {
	if runtime.GOOS == "windows" {
		return "nhctl.exe"
	}
	return "nhctl"
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
	if err != nil {
		return err
	}
	return nil
}

func (a *Application) GetSyncthingLocalDirFromProfileSaveByDevStart(svcName string, options *DevStartOptions) (*DevStartOptions, error) {
	svcProfile := a.GetSvcProfileV2(svcName)
	if svcProfile == nil {
		return options, errors.New("get " + svcName + " profile fail, please reinstall application")
	}
	options.LocalSyncDir = svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin
	return options, nil
}

func (a *Application) GetPodsFromDeployment(deployment string) (*corev1.PodList, error) {
	return a.client.GetPodsFromDeployment(deployment)
}

func (a *Application) GetNocalhostDevContainerPod(deployment string) (podName string, err error) {
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
				podName = pod.Name
				err = nil
				return
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

func (a *Application) CleanupResources() error {
	log.Info("Remove resource files...")
	homeDir := a.GetHomeDir()
	err := os.RemoveAll(homeDir)
	if err != nil {
		return errors.New(fmt.Sprintf("fail to remove resources dir %s\n", homeDir))
	}
	return nil
}
