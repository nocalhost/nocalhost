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
	k8s_runtime "k8s.io/apimachinery/pkg/util/runtime"
	"net"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/syncthing/ports"
	"os"
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

	port_forward "nocalhost/internal/nhctl/port-forward"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
)

type AppType string

const (
	Helm          AppType = "helmGit"
	HelmRepo      AppType = "helmRepo"
	Manifest      AppType = "rawManifest"
	ManifestLocal AppType = "rawManifestLocal"
	HelmLocal     AppType = "helmLocal"
	KustomizeGit  AppType = "kustomizeGit"

	// default is a special app type, it can be uninstalled neither installed
	// it's a virtual application to managed that those manifest out of Nocalhost management
	DefaultNocalhostApplication           = "default.application"
	DefaultNocalhostApplicationOperateErr = "default.application is a virtual application to managed that those manifest out of Nocalhost management so can't be install, uninstall, reset, etc."

	HelmReleaseName               = "meta.helm.sh/release-name"
	AppManagedByLabel             = "app.kubernetes.io/managed-by"
	AppManagedByNocalhost         = "nocalhost"
	NocalhostApplicationName      = "dev.nocalhost/application-name"
	NocalhostApplicationNamespace = "dev.nocalhost/application-namespace"
	SecretName                    = "dev.nocalhost.application."
	SecretType                    = "dev.nocalhost/application"
)

type Application struct {
	Name      string
	NameSpace string
	//config   *NocalHostAppConfig //  this should not be nil
	configV2 *profile.NocalHostAppConfigV2
	//AppProfile               *AppProfile // runtime info, this will not be nil
	AppProfileV2             *profile.AppProfileV2
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

func (a *Application) moveProfileFromFileToLeveldb() error {

	log.Log("Move profile to leveldb")

	profileV2 := &profile.AppProfileV2{}

	fBytes, err := ioutil.ReadFile(a.getProfileV2Path())
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = yaml.Unmarshal(fBytes, profileV2)
	if err != nil {
		return errors.Wrap(err, "")
	}

	return nocalhost.UpdateProfileV2(a.NameSpace, a.Name, profileV2)
}

func NewApplication(name string, ns string, kubeconfig string, initClient bool) (*Application, error) {

	app := &Application{
		Name:      name,
		NameSpace: ns,
	}

	err := app.LoadConfigV2()
	if err != nil {
		return nil, err
	}

	//
	appProfile, err := nocalhost.GetProfileV2(app.NameSpace, app.Name)
	if err != nil {
		return nil, err
	}
	if appProfile == nil {
		app.moveProfileFromFileToLeveldb()
	}

	err = app.LoadAppProfileV2(true)
	if err != nil {
		return nil, err
	}

	if len(app.AppProfileV2.PreInstall) == 0 && len(app.configV2.ApplicationConfig.PreInstall) > 0 {
		app.AppProfileV2.PreInstall = app.configV2.ApplicationConfig.PreInstall
		_ = app.SaveProfile()
	}

	if kubeconfig != "" && kubeconfig != app.AppProfileV2.Kubeconfig {
		app.AppProfileV2.Kubeconfig = kubeconfig
		_ = app.SaveProfile()
	}

	if initClient {
		app.client, err = clientgoutils.NewClientGoUtils(app.GetKubeconfig(), app.GetNamespace())
		if err != nil {
			return nil, err
		}
	}

	app.convertDevPortForwardList()

	return app, nil
}

func (a *Application) ReadBeforeWriteProfile() error {
	return a.LoadAppProfileV2(true)
}

func (a *Application) LoadConfigV2() error {

	isV2, err := a.checkIfAppConfigIsV2()
	if err != nil {
		return err
	}

	if !isV2 {
		log.Log("Upgrade config V1 to V2 ...")
		err = a.UpgradeAppConfigV1ToV2()
		if err != nil {
			return err
		}
	}

	config := &profile.NocalHostAppConfigV2{}
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
	return nocalhost.UpdateProfileV2(a.NameSpace, a.Name, a.AppProfileV2)
	//v2Bytes, err := yaml.Marshal(a.AppProfileV2)
	//if err != nil {
	//	return errors.Wrap(err, "")
	//}
	//
	//err = ioutil.WriteFile(a.getProfileV2Path(), v2Bytes, 0644)
	//return errors.Wrap(err, "")
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

func (a *Application) GetSvcConfigV2(svcName string) *profile.ServiceConfigV2 {
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

func (a *Application) GetApplicationConfigV2() *profile.ApplicationConfig {
	a.LoadConfigV2() // get the latest config
	if a.configV2 == nil {
		return nil
	}
	return a.configV2.ApplicationConfig
}

func (a *Application) SaveSvcProfileV2(svcName string, config *profile.ServiceConfigV2) error {

	svcPro := a.GetSvcProfileV2(svcName)
	if svcPro != nil {
		config.Name = svcName
		svcPro.ServiceConfigV2 = config
	} else {
		config.Name = svcName
		svcPro = &profile.SvcProfileV2{
			ServiceConfigV2: config,
			ActualName:      svcName,
		}
		a.AppProfileV2.SvcProfile = append(a.AppProfileV2.SvcProfile, svcPro)
	}

	return a.SaveProfile()
}

func (a *Application) GetAppProfileV2() *profile.ApplicationConfig {
	a.LoadAppProfileV2(false)
	a.LoadConfigV2()
	if a.configV2 == nil {
		return nil
	}
	return &profile.ApplicationConfig{
		ResourcePath: a.AppProfileV2.ResourcePath,
		IgnoredPath:  a.AppProfileV2.IgnoredPath,
		PreInstall:   a.AppProfileV2.PreInstall,
		Env:          a.AppProfileV2.Env,
		EnvFrom:      a.AppProfileV2.EnvFrom,
	}
}

func (a *Application) SaveAppProfileV2(config *profile.ApplicationConfig) error {
	a.AppProfileV2.ResourcePath = config.ResourcePath
	a.AppProfileV2.IgnoredPath = config.IgnoredPath
	a.AppProfileV2.PreInstall = config.PreInstall
	a.AppProfileV2.Env = config.Env
	a.AppProfileV2.EnvFrom = config.EnvFrom

	return a.SaveProfile()
}

func (a *Application) RollBack(ctx context.Context, svcName string, reset bool) error {
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
	dep.Annotations[NocalhostApplicationNamespace] = a.GetNamespace()

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
				a.EndDevPortForward(deployment, v.LocalPort, v.RemotePort)
				exist := false
				for _, vv := range localPorts {
					if vv == v.LocalPort {
						exist = true
					}
				}
				if !exist {
					localPorts = append(localPorts, v.LocalPort)
					remotePorts = append(remotePorts, v.RemotePort)
					//a.EndDevPortForward(deployment, v.LocalPort, v.RemotePort)
					os.Args = append(os.Args, "-p", fmt.Sprintf("%d:%d", v.LocalPort, v.RemotePort))
				}
			}
		}

		for _, sLocalPort := range localPorts {
			isAvailable := ports.IsTCP4PortAvailable("0.0.0.0", sLocalPort)
			if isAvailable {
				log.Infof("Port %d is available", sLocalPort)
			} else {
				log.Fatalf("Port %d is unavailable", sLocalPort)
			}
		}

		for key, sLocalPort := range localPorts {
			a.EndDevPortForward(deployment, sLocalPort, remotePorts[key]) // kill existed port-forward
			devPort := &profile.DevPortForward{
				LocalPort:  sLocalPort,
				RemotePort: remotePorts[key],
				Way:        way,
				Status:     "",
				Updated:    time.Now().Format("2006-01-02 15:04:05"),
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
	var lock sync.Mutex

	for key, sLocalPort := range localPorts {
		go func(lPort int, rPort int) {
			lock.Lock()
			_ = a.SetPortForwardPid(deployment, lPort, rPort, os.Getpid())
			lock.Unlock()
			for {
				// stopCh control the port forwarding lifecycle. When it gets closed the
				// port forward will terminate
				stopCh := make(chan struct{}, 1)
				// readyCh communicate when the port forward is ready to get traffic
				readyCh := make(chan struct{})
				endCh := make(chan struct{})

				k8s_runtime.ErrorHandlers = append(k8s_runtime.ErrorHandlers, func(err error) {
					if strings.Contains(err.Error(), "error creating error stream for port") {
						log.Warnf("Port-forward %d:%d failed to create stream, try to reconnecting", lPort, rPort)
						select {
						case _, isOpen := <-stopCh:
							if isOpen {
								log.Infof("Closing Port-forward %d:%d' by stop chan", lPort, rPort)
								close(stopCh)
							} else {
								log.Infof("Port-forward %d:%d has been closed, do nothing", lPort, rPort)
							}
						default:
							log.Infof("Closing Port-forward %d:%d'", lPort, rPort)
							close(stopCh)
						}
					}
				})

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
							a.CheckPidPortStatus(endCh, deployment, lPort, rPort, &lock)
						}()
						go func() {
							a.SendHeartBeat(endCh, listenAddress[0], lPort)
						}()
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
						lock.Lock()
						_ = a.UpdatePortForwardStatus(deployment, lPort, rPort, "DISCONNECTED", fmt.Sprintf("Unable to listen on port %d", lPort))
						lock.Unlock()
						wg.Done()
						return
					}
					log.WarnE(err, "Port-forward failed, reconnecting after 30 seconds...")
					close(endCh)
					lock.Lock()
					_ = a.UpdatePortForwardStatus(deployment, lPort, rPort, "RECONNECTING", "Port-forward failed, reconnecting after 30 seconds...")
					lock.Unlock()
					<-time.After(30 * time.Second)
				} else {
					log.Warn("Reconnecting after 30 seconds...")
					close(endCh)
					lock.Lock()
					_ = a.UpdatePortForwardStatus(deployment, lPort, rPort, "RECONNECTING", "Reconnecting after 30 seconds...")
					lock.Unlock()
					<-time.After(30 * time.Second)
				}
				log.Info("Reconnecting...")
			}
		}(sLocalPort, remotePorts[key])

		// sleep while
		time.Sleep(2 * time.Second)
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

func (a *Application) CheckPidPortStatus(stopCh chan struct{}, deployment string, sLocalPort, sRemotePort int, lock *sync.Mutex) {
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
				lock.Lock()
				_ = a.UpdatePortForwardStatus(deployment, sLocalPort, sRemotePort, portStatus, "Check Pid")
				lock.Unlock()
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
	return a.client.ListPodsByDeployment(deployment)
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
