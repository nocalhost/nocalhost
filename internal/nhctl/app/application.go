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
	"path/filepath"
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
	Helm     AppType = "helmGit"
	HelmRepo AppType = "helmRepo"
	Manifest AppType = "rawManifest"
)

type Application struct {
	Name                     string
	config                   *NocalHostAppConfig //  this should not be nil
	NewConfig                *Config
	AppProfile               *AppProfile // runtime info, this will not be nil
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

	err := app.LoadConfig()
	if err != nil {
		return nil, err
	}

	profile, err := NewAppProfile(app.getProfilePath())
	if err != nil {
		return nil, err
	}
	app.AppProfile = profile

	app.client, err = clientgoutils.NewClientGoUtils(app.GetKubeconfig(), app.GetNamespace())
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (a *Application) ReadBeforeWriteProfile() error {
	profile, err := NewAppProfile(a.getProfilePath())
	if err != nil {
		return err
	}
	a.AppProfile = profile
	return nil
}

func (a *Application) InitProfile(profile *AppProfile) {
	if profile != nil {
		a.AppProfile = profile
	}
}

func (a *Application) LoadConfig() error {
	config := &NocalHostAppConfig{}
	if _, err := os.Stat(a.GetConfigPath()); err != nil {
		if os.IsNotExist(err) {
			a.config = config
			return nil
		} else {
			return errors.Wrap(err, "fail to load configs")
		}
	}
	rbytes, err := ioutil.ReadFile(a.GetConfigPath())
	if err != nil {
		return errors.New(fmt.Sprintf("fail to load configFile : %s", a.GetConfigPath()))
	}
	err = yaml.Unmarshal(rbytes, config)
	if err != nil {
		return errors.Wrap(err, err.Error())
	}
	a.config = config
	return nil
}

func (a *Application) SaveConfig() error {
	if a.config != nil {
		bys, err := yaml.Marshal(a.config)
		if err != nil {
			return errors.Wrap(err, err.Error())
		}
		err = ioutil.WriteFile(a.GetConfigPath(), bys, 0644)
		if err != nil {
			return errors.Wrap(err, err.Error())
		}
	}
	return nil
}

func (a *Application) SaveProfile() error {
	return a.AppProfile.Save()
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

func (a *Application) GetSvcConfig(svcName string) *ServiceDevOptions {
	a.LoadConfig() // get the latest config
	if a.config == nil {
		return nil
	}
	for _, config := range a.config.SvcConfigs {
		if config.Name == svcName {
			return config
		}
	}
	return nil
}

func (a *Application) SaveSvcProfile(svcName string, config *ServiceDevOptions) error {

	svcPro := a.GetSvcProfile(svcName)
	if svcPro != nil {
		config.Name = svcName
		svcPro.ServiceDevOptions = config
	} else {
		config.Name = svcName
		svcPro = &SvcProfile{
			ServiceDevOptions: config,
			ActualName:        svcName,
		}
		a.AppProfile.SvcProfile = append(a.AppProfile.SvcProfile, svcPro)
	}

	return a.AppProfile.Save()
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

func (a *Application) CheckConfigFile(file string) error {
	config := &Config{}
	err := yaml.Unmarshal([]byte(file), config)
	if err != nil {
		return errors.New("Application OuterConfig file format error!")
	}
	return config.CheckValid()
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
	if a.AppProfile != nil {
		bytes, err := yaml.Marshal(a.AppProfile)
		if err == nil {
			desc = string(bytes)
		}
	}
	return desc
}

func (a *Application) GetSvcDescription(svcName string) string {
	desc := ""
	profile := a.GetSvcProfile(svcName)
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
	if a.AppProfile != nil {
		// get all service profile
		if service == "" {
			svcProfileForPlugin := make([]*SvcProfileForPlugin, 0)
			for _, value := range a.AppProfile.SvcProfile {
				rows := &SvcProfileForPlugin{
					Name:                                   value.Name,
					Type:                                   value.Type,
					GitUrl:                                 value.GitUrl,
					DevImage:                               value.DevImage,
					WorkDir:                                value.WorkDir,
					Sync:                                   value.Sync,
					DevPort:                                value.DevPort,
					Developing:                             value.Developing,
					PortForwarded:                          value.PortForwarded,
					Syncing:                                value.Syncing,
					LocalAbsoluteSyncDirFromDevStartPlugin: value.LocalAbsoluteSyncDirFromDevStartPlugin,
					DevPortList:                            value.DevPortList,
					SyncedPatterns:                         value.SyncedPatterns,
					IgnoredPatterns:                        value.IgnoredPatterns,
				}
				svcProfileForPlugin = append(svcProfileForPlugin, rows)
			}
			result := &PluginGetApplication{
				Name:                    a.Name,
				ReleaseName:             a.AppProfile.ReleaseName,
				Namespace:               a.AppProfile.Namespace,
				Kubeconfig:              a.AppProfile.Kubeconfig,
				DependencyConfigMapName: a.AppProfile.DependencyConfigMapName,
				AppType:                 a.AppProfile.AppType,
				Installed:               a.AppProfile.Installed,
				ResourcePath:            a.AppProfile.ResourcePath,
				SvcProfile:              svcProfileForPlugin,
			}
			bytes, err := yaml.Marshal(result)
			if err == nil {
				desc = string(bytes)
			}
			return desc
		}
		if service != "" {

			svcProfile := a.GetSvcProfile(service)
			if svcProfile == nil {
				return desc
			}
			svcProfileForPlugin := &SvcProfileForPlugin{
				Type:                                   svcProfile.Type,
				GitUrl:                                 svcProfile.GitUrl,
				DevImage:                               svcProfile.DevImage,
				WorkDir:                                svcProfile.WorkDir,
				Sync:                                   svcProfile.Sync,
				DevPort:                                svcProfile.DevPort,
				Name:                                   svcProfile.Name,
				Developing:                             svcProfile.Developing,
				PortForwarded:                          svcProfile.PortForwarded,
				Syncing:                                svcProfile.Syncing,
				LocalAbsoluteSyncDirFromDevStartPlugin: svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin,
				DevPortList:                            svcProfile.DevPortList,
				SyncedPatterns:                         svcProfile.SyncedPatterns,
				IgnoredPatterns:                        svcProfile.IgnoredPatterns,
			}
			result := &PluginGetApplicationService{
				Name:                    a.Name,
				ReleaseName:             a.AppProfile.ReleaseName,
				Namespace:               a.AppProfile.Namespace,
				Kubeconfig:              a.AppProfile.Kubeconfig,
				DependencyConfigMapName: a.AppProfile.DependencyConfigMapName,
				AppType:                 a.AppProfile.AppType,
				Installed:               a.AppProfile.Installed,
				ResourcePath:            a.AppProfile.ResourcePath,
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
	exist := a.GetSvcProfile(svcName).DevPort
	for k, v := range localPorts {
		checkPorts := fmt.Sprintf("%d:%d", v, remotePorts[k])
		exist = append(exist, checkPorts)
	}
	newPodList := tools.RemoveDuplicateElement(exist)
	a.GetSvcProfile(svcName).DevPort = newPodList
	return a.AppProfile.Save()
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

	// check if already exist manual port-forward, after dev start, pod will lost connection, should reconnect
	a.AppendDevPortManual(deployment, way, &localPorts, &remotePorts)
	for key, sLocalPort := range localPorts {

		// check if already exist port-forward, and kill old
		_ = a.KillAlreadyExistPortForward(fmt.Sprintf("%d:%d", sLocalPort, remotePorts[key]), deployment)

		//key := key
		//sLocalPort := sLocalPort
		//devPod := fmt.Sprintf("%d:%d", sLocalPort, remotePorts[key])
		//addDevPod = append(addDevPod, devPod)
		// log.Infof("Start dev port forward local %d, remote %d", sLocalPort, remotePorts[key])
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
							a.CheckPidPortStatus(endCh, deployment, lPort, rPort, way)
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
		_, err := daemon.Background(a.GetPortForwardLogFile(deployment), a.GetApplicationBackGroundOnlyPortForwardPidFile(deployment), true)
		if err != nil {
			log.Fatal("Failed to run port-forward background, please try again")
		}
	}

	// update profile addDevPod
	// TODO get from channel and set real port-forward status
	//for range localPort {
	//	r := <-portForwardResultCh
	//	portForwardResult = append(portForwardResult, r)
	//}
	//fmt.Printf("portForwardResult %s\n", portForwardResult)

	//_ = a.SetDevPortForward(deployment, portForwardResult)

	// set port forward status
	//if len(portForwardResult) > 0 {
	//	_ = a.SetPortForwardedStatus(deployment, true)
	//}

	//select {
	//case <-sigs:
	//	//case wg.Wait:
	//}
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

func (a *Application) CheckPidPortStatus(stopCh chan struct{}, deployment string, sLocalPort, sRemotePort int, way string) {
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
			<-time.After(10 * time.Second)
		}
	}
}

// ports format 8080:80
func (a *Application) KillAlreadyExistPortForward(ports, svcName string) error {
	var err error
	pidList := a.GetSvcProfile(svcName).PortForwardPidList
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
	existPortList := a.GetSvcProfile(svcName).PortForwardPidList
	if len(existPortList) == 0 {
		return errors.New("portForwardPidList empty")
	}
	var newPortList []string
	for _, v := range existPortList {
		needDelete := false
		for _, vv := range deletePortList {
			if strings.Contains(v, vv) {
				needDelete = true
				break
			}
		}
		if !needDelete {
			newPortList = append(newPortList, v)
		}
	}
	a.GetSvcProfile(svcName).PortForwardPidList = newPortList
	return a.AppProfile.Save()
}

func (a *Application) DeletePortForwardStatusList(svcName string, deletePortList []string) error {
	existPortList := a.GetSvcProfile(svcName).PortForwardStatusList
	if len(existPortList) == 0 {
		return errors.New("portForwardStatusList empty")
	}
	var newPortList []string
	for _, v := range existPortList {
		needDelete := false
		for _, vv := range deletePortList {
			if strings.Contains(v, vv) {
				needDelete = true
				break
			}
		}
		if !needDelete {
			newPortList = append(newPortList, v)
		}
	}
	a.GetSvcProfile(svcName).PortForwardStatusList = newPortList
	return a.AppProfile.Save()
}

func (a *Application) DeleteDevPortList(svcName string, deletePortList []string) error {
	existPortList := a.GetSvcProfile(svcName).DevPortList
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
	a.GetSvcProfile(svcName).DevPortList = newPortList
	return a.AppProfile.Save()
}

func (a *Application) SetDevPortForward(svcName string, portList []string) error {
	a.GetSvcProfile(svcName).DevPortList = portList
	return a.AppProfile.Save()
}

func (a *Application) AppendDevPortForward(svcName string, portList string) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		return err
	}
	exist := append(a.GetSvcProfile(svcName).DevPortList, portList)
	newPodList := tools.RemoveDuplicateElement(exist)
	a.GetSvcProfile(svcName).DevPortList = newPodList
	return a.AppProfile.Save()
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
	exist := a.GetSvcProfile(svcName).PortForwardPidList
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
	a.GetSvcProfile(svcName).PortForwardPidList = portStatusList
	return a.AppProfile.Save()
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
	return a.GetSvcProfile(svcName).PortForwardStatusList
}

func (a *Application) AppendPortForwardStatus(svcName string, portStatus string) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		return err
	}
	var portStatusList []string
	exist := a.GetSvcProfile(svcName).PortForwardStatusList
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
	a.GetSvcProfile(svcName).PortForwardStatusList = portStatusList
	return a.AppProfile.Save()
}

func (a *Application) GetDevPortForward(svcName string) []string {
	return a.GetSvcProfile(svcName).DevPortList
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
	svcProfile := a.GetSvcProfile(svcName)
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
