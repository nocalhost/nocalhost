package nhctl

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"math/rand"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/third_party/kubectl"
	"nocalhost/pkg/nhctl/tools"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"
)

type Application struct {
	Name       string
	Config     *NocalHostAppConfig
	AppProfile *AppProfile // runtime info
}

type AppProfile struct {
	Namespace               string              `json:"namespace" yaml:"namespace"`
	Kubeconfig              string              `json:"kubeconfig" yaml:"kubeconfig"`
	DependencyConfigMapName string              `json:"dependency_config_map_name" yaml:"dependencyConfigMapName"`
	SshPortForward          *PortForwardOptions `json:"ssh_port_forward" yaml:"sshPortForward"`
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

	err := app.Init()
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (a *Application) GetDependencies() []*SvcDependency {
	result := make([]*SvcDependency, 0)

	if a.Config == nil {
		return nil
	}

	svcConfigs := a.Config.SvcConfigs
	if svcConfigs == nil || len(svcConfigs) == 0 {
		return nil
	}

	for _, svcConfig := range svcConfigs {
		if svcConfig.Pods == nil && svcConfig.Jobs == nil {
			continue
		}
		svcDep := &SvcDependency{
			Name: svcConfig.Name,
			Type: svcConfig.Type,
			Jobs: svcConfig.Jobs,
			Pods: svcConfig.Pods,
		}
		result = append(result, svcDep)
	}
	return result
}

func (a *Application) IsHelm() bool {
	return a.Config.AppConfig.Type == "helm"
}

func (a *Application) IsManifest() bool {
	return a.Config.AppConfig.Type == "manifest"
}

func (a *Application) GetResourceDir() string {
	return fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, a.Config.AppConfig.ResourcePath)
}

func (a *Application) GetNamespace() string {
	return a.AppProfile.Namespace
}

func (a *Application) GetKubeconfig() string {
	return a.AppProfile.Kubeconfig
}

func (a *Application) Init() error {
	var err error
	// init application dir
	if _, err = os.Stat(a.GetHomeDir()); err != nil {
		return err
	}

	// {appName}/port-forward
	forwardDir := a.GetPortForwardDir()
	if _, err = os.Stat(forwardDir); err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(forwardDir, 0755)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	a.Config = &NocalHostAppConfig{}
	fileBytes, err := ioutil.ReadFile(a.GetConfigPath())
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(fileBytes, a.Config)

	a.loadProfile()

	return err
}

func (a *Application) loadProfile() {
	a.AppProfile = &AppProfile{}
	fBytes, err := ioutil.ReadFile(a.getProfilePath())
	if err != nil {
		return
	}
	yaml.Unmarshal(fBytes, a.AppProfile)
	return
}

func (a *Application) SaveProfile() error {
	if a.AppProfile == nil {
		return nil
	}
	bytes, err := yaml.Marshal(a.AppProfile)
	if err != nil {
		return err
	}
	profile := a.getProfilePath()
	err = ioutil.WriteFile(profile, bytes, 0755)
	return err
}

func (a *Application) getProfilePath() string {
	return fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, DefaultApplicationProfilePath)
}

func (a *Application) GetHomeDir() string {
	return fmt.Sprintf("%s%c%s%c%s%c%s", GetHomePath(), os.PathSeparator, ".nhctl", os.PathSeparator, "application", os.PathSeparator, a.Name)
}

func (a *Application) GetConfigPath() string {
	return fmt.Sprintf("%s%c%s%c%s", a.GetHomeDir(), os.PathSeparator, ".nocalhost", os.PathSeparator, "config.yaml")
}

func (a *Application) GetPortForwardDir() string {
	return fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, DefaultPortForwardDir)
}

// .nhctl/application/xxx/port-forward/{pid}
func (a *Application) GetPortForwardPidDir(pid int) string {
	return fmt.Sprintf("%s%c%d", a.GetPortForwardDir(), os.PathSeparator, pid)
}

// .nhctl/application/xxx/port-forward/{pid}/{local_port}_{remote_port}
func (a *Application) SavePortForwardInfo(localPort int, remotePort int) error {
	pid := os.Getpid()
	pidDir := a.GetPortForwardPidDir(pid)
	fileName := fmt.Sprintf("%s%c%d_%d", pidDir, os.PathSeparator, localPort, remotePort)
	f, err := os.Create(fileName)
	defer f.Close()
	if err != nil {
		return err
	}

	a.AppProfile.SshPortForward = &PortForwardOptions{
		LocalPort:  localPort,
		RemotePort: remotePort,
	}
	return a.SaveProfile()
}

func (a *Application) ListPortForwardPid() ([]int, error) {
	result := make([]int, 0)
	pidDir := a.GetPortForwardDir()
	dir, err := ioutil.ReadDir(pidDir)
	if err != nil {
		fmt.Printf("fail to get dirs in port-forward:%v\n", err)
		return nil, err
	}
	for _, fi := range dir {
		pid, err := strconv.Atoi(fi.Name())
		if err != nil {
			fmt.Printf("fail to get file name:%v\n", err)
		} else {
			result = append(result, pid)
		}

	}
	return result, nil
}

func (a *Application) StopAllPortForward() error {
	pids, err := a.ListPortForwardPid()
	if err != nil {
		return err
	}
	fmt.Printf("pids:%v\n", pids)
	for _, pid := range pids {
		_, err = tools.ExecCommand(nil, true, "kill", "-1", fmt.Sprintf("%d", pid))
		if err != nil {
			fmt.Printf("failed to stop port forward pid %d, err: %v\n", pid, err)
		}
		// remove pid dir
		pidDir := a.GetPortForwardPidDir(pid)
		err = os.RemoveAll(pidDir)
		if err != nil {
			fmt.Printf("fail to remove %s\n", pidDir)
		}
	}
	return nil
}

func (a *Application) GetSvcConfig(svcName string) *ServiceDevOptions {
	if a.Config.SvcConfigs != nil && len(a.Config.SvcConfigs) > 0 {
		for _, config := range a.Config.SvcConfigs {
			if config.Name == svcName {
				return config
			}
		}
	}
	return nil
}

func (a *Application) GetDefaultWorkDir(svcName string) string {
	config := a.GetSvcConfig(svcName)
	if config != nil {
		w := config.WorkDir
		if w == "" {
			w = DefaultWorkDir
		}
		return w
	}
	return ""
}

func (a *Application) GetDefaultSideCarImage(svcName string) string {
	config := a.GetSvcConfig(svcName)
	if config != nil {
		w := config.SideCarImage
		if w == "" {
			w = DefaultSideCarImage
		}
		return w
	}
	return ""
}

func (a *Application) GetDefaultDevImage(svcName string) string {
	config := a.GetSvcConfig(svcName)
	if config != nil {
		w := config.DevImage
		if w == "" {
			w = DefaultDevImage
		}
		return w
	}
	return ""
}

func (a *Application) RollBack(svcName string) error {
	clientUtils, err := clientgoutils.NewClientGoUtils(a.GetKubeconfig(), 0)
	if err != nil {
		return err
	}

	dep, err := clientUtils.GetDeployment(context.TODO(), a.GetNamespace(), svcName)
	if err != nil {
		fmt.Printf("failed to get deployment %s , err : %v\n", dep.Name, err)
		return err
	}

	fmt.Printf("rolling deployment back to previous revision\n")
	rss, err := clientUtils.GetReplicaSetsControlledByDeployment(context.TODO(), a.GetNamespace(), svcName)
	if err != nil {
		fmt.Printf("failed to get rs list, err:%v\n", err)
		return err
	}
	// find previous replicaSet
	if len(rss) < 2 {
		fmt.Println("no history to roll back")
		return nil
	}

	keys := make([]int, 0)
	for rs := range rss {
		keys = append(keys, rs)
	}
	sort.Ints(keys)

	dep.Spec.Template = rss[keys[len(keys)-2]].Spec.Template // previous replicaSet is the second largest revision number : keys[len(keys)-2]
	_, err = clientUtils.UpdateDeployment(context.TODO(), a.GetNamespace(), dep, metav1.UpdateOptions{}, true)
	if err != nil {
		fmt.Println("failed rolling back")
	} else {
		fmt.Println("rolling back!")
	}
	return err
}

type PortForwardOptions struct {
	LocalPort  int `json:"local_port" yaml:"localPort"`
	RemotePort int `json:"remote_port" yaml:"remotePort"`
}

func (a *Application) CleanupPid() error {
	pidDir := a.GetPortForwardPidDir(os.Getpid())
	if _, err2 := os.Stat(pidDir); err2 != nil {
		if os.IsNotExist(err2) {
			fmt.Printf("%s not exits, no need to cleanup it\n", pidDir)
			return nil
		} else {
			fmt.Printf("[warning] fails to cleanup %s\n", pidDir)
		}
	}
	err := os.RemoveAll(pidDir)
	if err != nil {
		fmt.Printf("removing .pid failed, please remove it manually, err:%v\n", err)
		return err
	}
	fmt.Printf("%s cleanup\n", pidDir)
	return nil
}

func (a *Application) SshPortForward(svcName string, ops *PortForwardOptions) error {

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT) // kill -1
	ctx, cancel := context.WithCancel(context.TODO())

	go func() {
		<-c
		cancel()
		fmt.Println("stop port forward")
		a.CleanupPid()
	}()

	// todo check if there is a same port-forward exists

	pid := os.Getpid()
	pidDir := a.GetPortForwardPidDir(pid)
	err := os.Mkdir(pidDir, 0755)
	if err != nil {
		return err
	}

	//debug("recording port-forward info...")
	var localPort, remotePort int
	config := a.GetSvcConfig(svcName)
	if config != nil && config.SshPort != nil {
		if config.SshPort.LocalPort != 0 {
			localPort = config.SshPort.LocalPort
		}
		if config.SshPort.SshPort != 0 {
			remotePort = config.SshPort.SshPort
		}
	}

	if ops.LocalPort != 0 {
		localPort = ops.LocalPort
	}
	if ops.RemotePort != 0 {
		remotePort = ops.RemotePort
	}

	if localPort == 0 {
		// random generate a port
		rand.Seed(time.Now().UnixNano())
		localPort = rand.Intn(10000) + 30002
	}
	if remotePort == 0 {
		remotePort = DefaultForwardRemoteSshPort
	}

	err = a.SavePortForwardInfo(localPort, remotePort)
	err = kubectl.PortForward(ctx, a.GetKubeconfig(), a.GetNamespace(), svcName, fmt.Sprintf("%d", localPort), fmt.Sprintf("%d", remotePort)) // eg : ./utils/darwin/kubectl port-forward --address 0.0.0.0 deployment/coding  12345:22
	if err != nil {
		fmt.Printf("failed to forward port : %v\n", err)
		return err
	}

	a.CleanupPid()
	return nil
}
func (a *Application) ReplaceImage(deployment string, ops *DevStartOptions) error {
	clientUtils, err := clientgoutils.NewClientGoUtils(a.GetKubeconfig(), 0)
	if err != nil {
		return err
	}
	deploymentsClient := clientUtils.GetDeploymentClient(a.GetNamespace())

	scale, err := deploymentsClient.GetScale(context.TODO(), deployment, metav1.GetOptions{})
	if err != nil {
		return err
	}

	fmt.Println("developing deployment: " + deployment)
	fmt.Println("scaling replicas to 1")

	if scale.Spec.Replicas > 1 {
		fmt.Printf("deployment %s's replicas is %d now\n", deployment, scale.Spec.Replicas)
		scale.Spec.Replicas = 1
		_, err = deploymentsClient.UpdateScale(context.TODO(), deployment, scale, metav1.UpdateOptions{})
		if err != nil {
			fmt.Println("failed to scale replicas to 1")
		} else {
			time.Sleep(time.Second * 5)
			fmt.Println("replicas has been scaled to 1")
		}
	} else {
		fmt.Printf("deployment %s's replicas is already 1\n", deployment)
	}

	fmt.Println("Updating develop container...")
	dep, err := clientUtils.GetDeployment(context.TODO(), a.GetNamespace(), deployment)
	if err != nil {
		fmt.Printf("failed to get deployment %s , err : %v\n", deployment, err)
		return err
	}

	volName := "nocalhost-shared-volume"
	// shared volume
	vol := corev1.Volume{
		Name: volName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	if dep.Spec.Template.Spec.Volumes == nil {
		//cmds.debug("volume slice define is nil, init slice")
		dep.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, vol)

	// volume mount
	workDir := a.GetDefaultWorkDir(deployment)
	if ops.WorkDir != "" {
		workDir = ops.WorkDir
	}

	volMount := corev1.VolumeMount{
		Name:      volName,
		MountPath: workDir,
	}

	// default : replace the first container
	devImage := a.GetDefaultDevImage(deployment)
	if ops.DevImage != "" {
		devImage = ops.DevImage
	}
	//fmt.Printf("dev image is %s\n", devImage)

	dep.Spec.Template.Spec.Containers[0].Image = devImage
	dep.Spec.Template.Spec.Containers[0].Name = "nocalhost-dev"
	dep.Spec.Template.Spec.Containers[0].Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	dep.Spec.Template.Spec.Containers[0].VolumeMounts = append(dep.Spec.Template.Spec.Containers[0].VolumeMounts, volMount)

	// set the entry
	dep.Spec.Template.Spec.Containers[0].WorkingDir = workDir

	//cmds.debug("disable readiness probes")
	for i := 0; i < len(dep.Spec.Template.Spec.Containers); i++ {
		dep.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		dep.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		dep.Spec.Template.Spec.Containers[i].StartupProbe = nil
	}

	sideCarImage := a.GetDefaultDevImage(deployment)
	if ops.SideCarImage != "" {
		sideCarImage = ops.SideCarImage
	}
	sideCarContainer := corev1.Container{
		Name:    "nocalhost-sidecar",
		Image:   sideCarImage,
		Command: []string{"/bin/sh", "-c", "service ssh start; mutagen daemon start; mutagen-agent install; tail -f /dev/null"},
	}
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, volMount)
	sideCarContainer.LivenessProbe = &corev1.Probe{
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		Handler: corev1.Handler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.IntOrString{
					IntVal: 22,
				},
			},
		},
	}
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer)

	_, err = clientUtils.UpdateDeployment(context.TODO(), a.GetNamespace(), dep, metav1.UpdateOptions{}, true)
	if err != nil {
		fmt.Printf("update develop container failed : %v \n", err)
		return err
	}

	<-time.NewTimer(time.Second * 3).C

	podList, err := clientUtils.ListPodsOfDeployment(a.GetNamespace(), dep.Name)
	if err != nil {
		fmt.Printf("failed to get pods, err: %v\n", err)
		return err
	}

	fmt.Printf("%d pod found\n", len(podList)) // should be 2

	// wait podList to be ready
	fmt.Printf("waiting pod to start.")
	for {
		<-time.NewTimer(time.Second * 2).C
		podList, err = clientUtils.ListPodsOfDeployment(a.GetNamespace(), dep.Name)
		if err != nil {
			fmt.Printf("failed to get pods, err: %v\n", err)
			return err
		}
		if len(podList) == 1 {
			// todo check container status
			break
		}
		fmt.Print(".")
	}

	fmt.Println("develop container has been updated")
	return nil
}
