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
	"os"
	"regexp"
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
	"nocalhost/internal/nhctl/flock"
	port_forward "nocalhost/internal/nhctl/port-forward"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"

	"github.com/pkg/errors"
)

type AppType string

const (
	Helm     AppType = "helmGit"
	HelmRepo AppType = "helmRepo"
	Manifest AppType = "rawManifest"
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

func NewApplication(name string) (*Application, error) {
	app := &Application{
		Name: name,
	}

	err := app.LoadConfigV2()
	if err != nil {
		return nil, err
	}

	//profile, err := NewAppProfile(app.getProfilePath())
	//if err != nil {
	//	return nil, err
	//}
	//app.AppProfile = profile
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

	return app, nil
}

func (a *Application) ReadBeforeWriteProfile() error {
	//profile, err := NewAppProfile(a.getProfilePath())
	//if err != nil {
	//	return err
	//}
	//a.AppProfile = profile
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
	//a.loadInstallManifest()

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

func (a *Application) cleanPreInstall() {
	//a.loadSortedPreInstallManifest()
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

func (a *Application) IsAnyServiceInDevMode() bool {
	for _, svc := range a.AppProfileV2.SvcProfile {
		if svc.Developing {
			return true
		}
	}
	return false
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

func (a *Application) GetPluginDescription(service string) string {
	desc := ""
	if a.AppProfileV2 != nil {
		// get all service profile
		if service == "" {
			svcProfileForPlugin := make([]*SvcProfileForPlugin, 0)
			for _, value := range a.AppProfileV2.SvcProfile {
				rows := &SvcProfileForPlugin{
					Name:                                   value.Name,
					Type:                                   value.Type,
					GitUrl:                                 value.ContainerConfigs[0].Dev.GitUrl,
					DevImage:                               value.ContainerConfigs[0].Dev.Image,
					WorkDir:                                value.ContainerConfigs[0].Dev.WorkDir,
					Sync:                                   value.SyncDirs,
					DevPort:                                value.ContainerConfigs[0].Dev.PortForward,
					Developing:                             value.Developing,
					PortForwarded:                          value.PortForwarded,
					Syncing:                                value.Syncing,
					LocalAbsoluteSyncDirFromDevStartPlugin: value.LocalAbsoluteSyncDirFromDevStartPlugin,
					DevPortList:                            value.DevPortList,
					SyncedPatterns:                         value.ContainerConfigs[0].Dev.Sync.FilePattern,
					IgnoredPatterns:                        value.ContainerConfigs[0].Dev.Sync.IgnoreFilePattern,
				}
				svcProfileForPlugin = append(svcProfileForPlugin, rows)
			}
			result := &PluginGetApplication{
				Name:                    a.Name,
				ReleaseName:             a.AppProfileV2.ReleaseName,
				Namespace:               a.AppProfileV2.Namespace,
				Kubeconfig:              a.AppProfileV2.Kubeconfig,
				DependencyConfigMapName: a.AppProfileV2.DependencyConfigMapName,
				AppType:                 a.AppProfileV2.AppType,
				Installed:               a.AppProfileV2.Installed,
				ResourcePath:            a.AppProfileV2.ResourcePath,
				SvcProfile:              svcProfileForPlugin,
			}
			bytes, err := yaml.Marshal(result)
			if err == nil {
				desc = string(bytes)
			}
			return desc
		}
		if service != "" {

			svcProfile := a.GetSvcProfileV2(service)
			if svcProfile == nil {
				return desc
			}
			svcProfileForPlugin := &SvcProfileForPlugin{
				Name:                                   svcProfile.Name,
				Type:                                   svcProfile.Type,
				GitUrl:                                 svcProfile.ContainerConfigs[0].Dev.GitUrl,
				DevImage:                               svcProfile.ContainerConfigs[0].Dev.Image,
				WorkDir:                                svcProfile.ContainerConfigs[0].Dev.WorkDir,
				Sync:                                   svcProfile.SyncDirs,
				DevPort:                                svcProfile.ContainerConfigs[0].Dev.PortForward,
				Developing:                             svcProfile.Developing,
				PortForwarded:                          svcProfile.PortForwarded,
				Syncing:                                svcProfile.Syncing,
				LocalAbsoluteSyncDirFromDevStartPlugin: svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin,
				DevPortList:                            svcProfile.DevPortList,
				SyncedPatterns:                         svcProfile.ContainerConfigs[0].Dev.Sync.FilePattern,
				IgnoredPatterns:                        svcProfile.ContainerConfigs[0].Dev.Sync.IgnoreFilePattern,
			}
			result := &PluginGetApplicationService{
				Name:                    a.Name,
				ReleaseName:             a.AppProfileV2.ReleaseName,
				Namespace:               a.AppProfileV2.Namespace,
				Kubeconfig:              a.AppProfileV2.Kubeconfig,
				DependencyConfigMapName: a.AppProfileV2.DependencyConfigMapName,
				AppType:                 a.AppProfileV2.AppType,
				Installed:               a.AppProfileV2.Installed,
				ResourcePath:            a.AppProfileV2.ResourcePath,
				SvcProfile:              svcProfileForPlugin,
			}
			bytes, err := yaml.Marshal(result)
			if err == nil {
				desc = string(bytes)
			}
			return desc
		}
	}
	return desc
}

// record manual port-forward in rawConfig devPorts
func (a *Application) AppendManualPortForwardToRawConfigDevPorts(svcName, way string, localPorts, remotePorts []int) error {
	if way == PortForwardDevPorts {
		return nil
	}
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		return err
	}
	exist := a.GetSvcProfileV2(svcName).ContainerConfigs[0].Dev.PortForward
	for k, v := range localPorts {
		checkPorts := fmt.Sprintf("%d:%d", v, remotePorts[k])
		exist = append(exist, checkPorts)
	}
	newPodList := tools.RemoveDuplicateElement(exist)
	//a.GetSvcProfile(svcName).DevPort = newPodList
	a.GetSvcProfileV2(svcName).ContainerConfigs[0].Dev.PortForward = newPodList
	return a.SaveProfile()
}

func (a *Application) FixPortForwardOSArgs(localPort, remotePort []int) {
	var newArg []string
	for _, v := range os.Args {
		match := false
		for _, vv := range remotePort {
			if v == "-p" || v == fmt.Sprintf(":%d", vv) {
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
func (a *Application) PortForwardInBackGround(listenAddress []string, deployment, podName string, localPorts, remotePorts []int, way string, isDaemon bool) {
	if len(localPorts) != len(remotePorts) {
		log.Fatalf("dev port forward fail, please check you devPort in config\n")
	}
	// wait group
	var wg sync.WaitGroup
	wg.Add(len(localPorts))

	// pid port status chan
	statusChan := make(chan struct{})

	// check if already exist manual port-forward, after dev start, pod will lost connection, should reconnect
	a.AppendDevPortManual(deployment, way, &localPorts, &remotePorts)
	for key, sLocalPort := range localPorts {

		// check if already exist port-forward, and kill old
		_ = a.KillAlreadyExistPortForward(fmt.Sprintf("%d:%d", sLocalPort, remotePorts[key]), deployment)

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

				go func(readyCh chan struct{}) {
					select {
					case <-readyCh:
						log.Info("Port forward is ready")
						// append status each success port-forward
						_ = a.AppendDevPortForward(deployment, fmt.Sprintf("%d:%d", lPort, rPort))
						_ = a.AppendDevPortForwardPID(deployment, fmt.Sprintf("%d:%d-%d", lPort, rPort, os.Getpid()))
						_ = a.SetPortForwardedStatus(deployment, true)
						go func() {
							a.CheckPidPortStatus(endCh, deployment, lPort, rPort, way, statusChan)
						}()
						go func() {
							a.SendHeartBeat(endCh, listenAddress[0], lPort)
						}()
					}
				}(readyCh)

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
						// log.Warnf("Unable to listen on port %d", lPort)
						statusChan <- struct{}{}
						wg.Done()
						return
					}
					log.WarnE(err, "Port-forward failed, reconnecting after 30 seconds...")
					close(endCh)
					<-time.After(30 * time.Second)
				} else {
					log.Warn("Reconnecting after 30 seconds...")
					close(endCh)
					<-time.After(30 * time.Second)
				}
				log.Info("Reconnecting...")
			}
		}(sLocalPort, remotePorts[key])

		// sleep while
		time.Sleep(time.Duration(2) * time.Second)
	}

	// run in background
	if isDaemon {
		for i := 0; i < len(localPorts); i++ {
			<-statusChan
		}
		log.Infof("Get system port status %s", strings.Join(a.GetPortForwardStatus(deployment), ", "))
		_, err := daemon.Background(a.GetPortForwardLogFile(deployment), a.GetApplicationBackGroundOnlyPortForwardPidFile(deployment), true)
		if err != nil {
			log.Fatal("Failed to run port-forward background, please try again")
		}
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

func (a *Application) CheckPidPortStatus(stopCh chan struct{}, deployment string, sLocalPort, sRemotePort int, way string, statusChan chan struct{}) {
	for {
		select {
		case <-stopCh:
			log.Info("Stop Checking port status")
			portStatus := port_forward.PidPortStatus(os.Getpid(), sLocalPort)
			log.Infof("Checking Port %d:%d's status: %s", sLocalPort, sRemotePort, portStatus)
			_ = a.AppendPortForwardStatus(deployment, fmt.Sprintf("%d:%d(%s-%s)", sLocalPort, sRemotePort, strings.ToTitle(way), portStatus))
			return
		default:
			//log.Infof("Check %d:%d port status", sLocalPort, sRemotePort)
			portStatus := port_forward.PidPortStatus(os.Getpid(), sLocalPort)
			log.Infof("Checking Port %d:%d's status: %s", sLocalPort, sRemotePort, portStatus)
			_ = a.AppendPortForwardStatus(deployment, fmt.Sprintf("%d:%d(%s-%s)", sLocalPort, sRemotePort, strings.ToTitle(way), portStatus))
			statusChan <- struct{}{}
			<-time.After(10 * time.Second)
		}
	}
}

// ports format 8080:80
func (a *Application) KillAlreadyExistPortForward(ports, svcName string) error {
	var err error
	//pidList := a.GetSvcProfile(svcName).PortForwardPidList
	pidList := a.GetSvcProfileV2(svcName).PortForwardPidList
	if len(pidList) > 0 {
		for _, v := range pidList {
			portPid := strings.Split(v, "-")
			if len(portPid) < 2 {
				err := errors.New("portForwardPidList format invalid")
				return err
			}
			port := portPid[0]
			// pid := portPid[1]
			if port == ports {
				// should kill
				err = a.StopPortForwardByPort(svcName, ports)
			}
		}
	}
	return err
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

// port-forward use
func (a *Application) DeletePortForwardPidList(svcName string, deletePortList []string) error {
	//existPortList := a.GetSvcProfile(svcName).PortForwardPidList
	existPortList := a.GetSvcProfileV2(svcName).PortForwardPidList
	if len(existPortList) == 0 {
		return errors.New("portForwardPidList empty")
	}
	var newPortList []string
	for _, v := range existPortList {
		needDelete := false
		for _, vv := range deletePortList {
			regexpString, _ := regexp.Compile("\\d+:\\d+")
			localAndRemote := regexpString.FindString(v)
			if localAndRemote == vv {
				needDelete = true
				break
			}
		}
		if !needDelete {
			newPortList = append(newPortList, v)
		}
	}
	a.GetSvcProfileV2(svcName).PortForwardPidList = newPortList
	return a.SaveProfile()
}

func (a *Application) DeletePortForwardStatusList(svcName string, deletePortList []string) error {
	existPortList := a.GetSvcProfileV2(svcName).PortForwardStatusList
	if len(existPortList) == 0 {
		return errors.New("portForwardStatusList empty")
	}
	var newPortList []string
	for _, v := range existPortList {
		needDelete := false
		for _, vv := range deletePortList {
			regexpString, _ := regexp.Compile("\\d+:\\d+")
			localAndRemote := regexpString.FindString(v)
			if localAndRemote == vv {
				needDelete = true
				break
			}
		}
		if !needDelete {
			newPortList = append(newPortList, v)
		}
	}
	a.GetSvcProfileV2(svcName).PortForwardStatusList = newPortList
	return a.SaveProfile()
}

func (a *Application) DeleteDevPortList(svcName string, deletePortList []string) error {
	existPortList := a.GetSvcProfileV2(svcName).DevPortList
	if len(existPortList) == 0 {
		return errors.New("portList empty")
	}
	var newPortList []string
	for _, v := range existPortList {
		needDelete := false
		for _, vv := range deletePortList {
			if v == vv {
				needDelete = true
				break
			}
		}
		if !needDelete {
			newPortList = append(newPortList, v)
		}
	}
	a.GetSvcProfileV2(svcName).DevPortList = newPortList
	return a.SaveProfile()
}

func (a *Application) SetDevPortForward(svcName string, portList []string) error {
	a.GetSvcProfileV2(svcName).DevPortList = portList
	return a.SaveProfile()
}

func (a *Application) AppendDevPortForward(svcName string, portList string) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		return err
	}
	exist := append(a.GetSvcProfileV2(svcName).DevPortList, portList)
	newPodList := tools.RemoveDuplicateElement(exist)
	a.GetSvcProfileV2(svcName).DevPortList = newPodList
	return a.SaveProfile()
}

func (a *Application) AppendDevPortForwardPID(svcName string, portPIDList string) error {
	lock, e := flock.Create(a.GetFileLockPath(svcName))
	if e != nil {
		log.Warnf("lock err %s", e.Error())
	}
	defer lock.Release()

	e = lock.Lock()
	if e != nil {
		log.Warnf("lock err %s", e.Error())
	}
	defer lock.Unlock()

	err := a.ReadBeforeWriteProfile()
	if err != nil {
		return err
	}
	var portStatusList []string
	exist := a.GetSvcProfileV2(svcName).PortForwardPidList
	needAdd := true
	for _, v := range exist {
		if strings.Split(v, "-")[0] == strings.Split(portPIDList, "-")[0] && len(strings.Split(v, "-")) != 0 {
			portStatusList = append(portStatusList, portPIDList)
			needAdd = false
		} else {
			portStatusList = append(portStatusList, v)
		}
	}
	if needAdd {
		portStatusList = append(portStatusList, portPIDList)
	}
	portStatusList = tools.RemoveDuplicateElement(portStatusList)
	a.GetSvcProfileV2(svcName).PortForwardPidList = portStatusList
	return a.SaveProfile()
}

func (a *Application) AppendDevPortManual(svcName, way string, localPorts, remotePorts *[]int) {
	// if from manual port-forward, renturn port
	if way == PortForwardManual {
		return
	}
	// if from devPorts, check previously manual port-forward and add to need port-forward list
	portForwardStatus := a.GetPortForwardStatus(svcName)
	if len(portForwardStatus) == 0 {
		return
	}
	for _, v := range portForwardStatus {
		if strings.Contains(v, strings.ToTitle(PortForwardManual)) {
			exist := false
			// TODO use regex instead of split
			regexp, _ := regexp.Compile("\\d+:\\d+")
			localAndRemote := regexp.FindString(v)
			localAndRemoteArray := strings.Split(localAndRemote, ":")
			if len(localAndRemoteArray) != 2 {
				return
			}
			appendLocalPort, err := strconv.Atoi(localAndRemoteArray[0])
			appendRemotePort, err := strconv.Atoi(localAndRemoteArray[1])
			if err != nil {
				continue
			}
			for _, vv := range *localPorts {
				if vv == appendLocalPort {
					exist = true
				}
			}
			if !exist {
				*localPorts = append(*localPorts, appendLocalPort)
				*remotePorts = append(*remotePorts, appendRemotePort)
			}
		}
	}
}

func (a *Application) GetPortForwardStatus(svcName string) []string {
	return a.GetSvcProfileV2(svcName).PortForwardStatusList
}

func (a *Application) AppendPortForwardStatus(svcName string, portStatus string) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		return err
	}
	var portStatusList []string
	exist := a.GetSvcProfileV2(svcName).PortForwardStatusList
	needAdd := true
	for _, v := range exist {
		if strings.Split(v, "(")[0] == strings.Split(portStatus, "(")[0] && len(strings.Split(v, "(")) != 0 {
			portStatusList = append(portStatusList, portStatus)
			needAdd = false
		} else {
			portStatusList = append(portStatusList, v)
		}
	}
	if needAdd {
		portStatusList = append(portStatusList, portStatus)
	}
	portStatusList = tools.RemoveDuplicateElement(portStatusList)
	a.GetSvcProfileV2(svcName).PortForwardStatusList = portStatusList
	return a.SaveProfile()
}

func (a *Application) GetDevPortForward(svcName string) []string {
	return a.GetSvcProfileV2(svcName).DevPortList
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
