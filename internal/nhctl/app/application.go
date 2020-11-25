package app

import (
	"context"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"math/rand"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/third_party/kubectl"
	"nocalhost/pkg/nhctl/third_party/mutagen"
	"nocalhost/pkg/nhctl/tools"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type AppType string

const (
	Helm     AppType = "helm"
	HelmRepo AppType = "helm-repo"
	Manifest AppType = "manifest"
)

type Application struct {
	Name       string
	Config     *NocalHostAppConfig // if config.yaml not exist, this should be nil
	AppProfile *AppProfile         // runtime info
	client     *clientgoutils.ClientGoUtils
}

type SvcDependency struct {
	Name string   `json:"name" yaml:"name"`
	Type string   `json:"type" yaml:"type"`
	Jobs []string `json:"jobs" yaml:"jobs,omitempty"`
	Pods []string `json:"pods" yaml:"pods,omitempty"`
}

// build a new application
func BuildApplication(name string) (*Application, error) {

	app := &Application{
		Name: name,
	}

	err := app.InitDir()
	if err != nil {
		return nil, err
	}

	profile, err := NewAppProfile(app.getProfilePath())
	if err != nil {
		return nil, err
	}
	app.AppProfile = profile
	return app, nil
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

	app.client, err = clientgoutils.NewClientGoUtils(app.GetKubeconfig(), DefaultClientGoTimeOut)
	if err != nil {
		return nil, err
	}

	return app, nil
}

// if namespace is nil, use namespace defined in kubeconfig
func (a *Application) InitClient(kubeconfig string, namespace string) error {
	// check if kubernetes is available
	var err error
	a.client, err = clientgoutils.NewClientGoUtils(kubeconfig, DefaultClientGoTimeOut)
	if err != nil {
		return err
	}
	if namespace == "" {
		namespace, err = a.client.GetDefaultNamespace()
		if err != nil {
			return err
		}
	}

	// save application info
	a.AppProfile.Namespace = namespace
	a.AppProfile.Kubeconfig = kubeconfig
	err = a.AppProfile.Save()
	if err != nil {
		fmt.Println("[error] fail to save nocalhostApp profile")
	}
	return err
}

func (a *Application) InitDir() error {
	var err error
	err = os.Mkdir(a.GetHomeDir(), 0755)
	if err != nil {
		return err
	}

	err = os.Mkdir(a.GetPortForwardDir(), 0755)
	if err != nil {
		return err
	}

	err = os.Mkdir(a.getGitDir(), 0755)
	if err != nil {
		return err
	}

	err = os.Mkdir(a.GetConfigDir(), DefaultNewFilePermission)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(a.getProfilePath(), []byte(""), DefaultNewFilePermission)
	return err
}

func (a *Application) InitConfig(outerConfig string) error {
	configFile := outerConfig
	if outerConfig == "" {
		_, err := os.Stat(a.getConfigPathInGitResourcesDir())
		if err == nil {
			configFile = a.getConfigPathInGitResourcesDir()
		}
	}
	if configFile != "" {
		rbytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			return errors.New(fmt.Sprintf("fail to load configFile : %s", configFile))
		}
		err = ioutil.WriteFile(a.GetConfigPath(), rbytes, DefaultNewFilePermission)
		if err != nil {
			return errors.New(fmt.Sprintf("fail to create configFile : %s", configFile))
		}
		err = a.LoadConfig()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Application) InitProfile(profile *AppProfile) {
	if profile != nil {
		a.AppProfile = profile
	}
}

func (a *Application) LoadConfig() error {
	if _, err := os.Stat(a.GetConfigPath()); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	//fmt.Println(a.GetConfigPath())
	rbytes, err := ioutil.ReadFile(a.GetConfigPath())
	if err != nil {
		return errors.New(fmt.Sprintf("fail to load configFile : %s", a.GetConfigPath()))
	}
	config := &NocalHostAppConfig{}
	err = yaml.Unmarshal(rbytes, config)
	if err != nil {
		return err
	}
	a.Config = config
	return nil
}

func (a *Application) DownloadResourcesFromGit(gitUrl string) error {
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
		gitDirName = strs[len(strs)-1] // todo : for default application anme
		_, err = tools.ExecCommand(nil, true, "git", "clone", "--depth", "1", gitUrl, a.getGitDir())
		if err != nil {
			return err
		}
	}
	return nil
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
	return a.AppProfile.AppType == Helm || a.AppProfile.AppType == HelmRepo
}

func (a *Application) IsManifest() bool {
	return a.AppProfile.AppType == Manifest
}

func (a *Application) GetResourceDir() string {
	if a.AppProfile != nil && a.AppProfile.ResourcePath != "" {
		return fmt.Sprintf("%s%c%s", a.getGitDir(), os.PathSeparator, a.AppProfile.ResourcePath)
	}
	if a.Config != nil {
		return fmt.Sprintf("%s%c%s", a.getGitDir(), os.PathSeparator, a.Config.AppConfig.ResourcePath)
	} else {
		return ""
	}
}

type HelmFlags struct {
	Debug    bool
	Wait     bool
	Set      string
	Values   string
	Chart    string
	RepoName string
	RepoUrl  string
}

func (a *Application) InstallManifest() error {
	var err error
	err = a.InstallDepConfigMap()
	if err != nil {
		return err
	}
	excludeFiles := make([]string, 0)
	if a.Config != nil && a.Config.PreInstall != nil {
		fmt.Println("[nocalhost config] reading pre-install hook")
		excludeFiles, err = a.preInstall(a.GetResourceDir(), a.Config.PreInstall)
		if err != nil {
			return err
		}
	}
	// install manifest recursively, don't install pre-install workload again
	err = a.installManifestRecursively(excludeFiles)
	return err
}

func (a *Application) installManifestRecursively(excludeFiles []string) error {

	files, _, err := a.getYamlFilesAndDirs(a.GetResourceDir())
	if err != nil {
		return err
	}
	start := time.Now()
	wg := sync.WaitGroup{}

outer:
	for _, file := range files {
		for _, ex := range excludeFiles {
			if ex == file {
				fmt.Println("ignore file : " + file)
				continue outer
			}
		}

		// parallel
		wg.Add(1)
		go func(fileName string) {
			fmt.Println("create " + fileName)
			a.client.Create(fileName, a.GetNamespace(), false)
			wg.Done()
		}(file)
	}
	wg.Wait()
	end := time.Now()
	fmt.Printf("installing takes %f seconds\n", end.Sub(start).Seconds())
	return err
}

func (a *Application) getYamlFilesAndDirs(dirPth string) (files []string, dirs []string, err error) {
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, nil, err
	}

	PthSep := string(os.PathSeparator)

	for _, fi := range dir {
		if fi.IsDir() {
			dirs = append(dirs, dirPth+PthSep+fi.Name())
			fs, ds, err := a.getYamlFilesAndDirs(dirPth + PthSep + fi.Name())
			if err != nil {
				return files, dirs, err
			}
			dirs = append(dirs, ds...)
			files = append(files, fs...)
		} else {
			ok := strings.HasSuffix(fi.Name(), ".yaml")
			if ok {
				files = append(files, dirPth+PthSep+fi.Name())
			} else if strings.HasSuffix(fi.Name(), ".yml") {
				files = append(files, dirPth+PthSep+fi.Name())
			}
		}
	}
	return files, dirs, nil
}

func (a *Application) preInstall(basePath string, items []*PreInstallItem) ([]string, error) {
	fmt.Println("run pre-install....")

	// sort
	sort.Sort(ComparableItems(items))

	files := make([]string, 0)
	for _, item := range items {
		fmt.Println(item.Path + " : " + item.Weight)
		itemPath := fmt.Sprintf("%s%c%s", basePath, os.PathSeparator, item.Path)
		files = append(files, itemPath)
		// todo check if item.Path is a valid file
		err := a.client.Create(itemPath, a.GetNamespace(), true)
		if err != nil {
			return files, err
		}
	}
	return files, nil
}

func (a *Application) InstallHelmRepo(releaseName string, flags *HelmFlags) error {
	commonParams := make([]string, 0)
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "--namespace", a.GetNamespace())
	}
	if a.GetKubeconfig() != "" {
		commonParams = append(commonParams, "--kubeconfig", a.GetKubeconfig())
	}
	if flags.Debug {
		commonParams = append(commonParams, "--debug")
	}

	chartName := flags.Chart
	installParams := []string{"install", releaseName}
	if flags.Wait {
		installParams = append(installParams, "--wait")
	}
	//if installFlags.HelmRepoUrl
	if flags.RepoUrl != "" {
		installParams = append(installParams, chartName, "--repo", flags.RepoUrl)
	} else if flags.RepoName != "" {
		installParams = append(installParams, fmt.Sprintf("%s/%s", flags.RepoName, chartName))
	}

	if flags.Set != "" {
		installParams = append(installParams, "--set", flags.Set)
	}
	if flags.Values != "" {
		installParams = append(installParams, "-f", flags.Values)
	}
	installParams = append(installParams, commonParams...)

	fmt.Println("install helm application, this may take several minutes, please waiting...")

	_, err := tools.ExecCommand(nil, true, "helm", installParams...)
	if err != nil {
		//printlnErr("fail to install helm nocalhostApp", err)
		return err
	}
	//debug(output)
	fmt.Printf(`helm nocalhost app installed, use "helm list -n %s" to get the information of the helm release`+"\n", a.GetNamespace())
	return nil
}

func (a *Application) InstallHelm(releaseName string, flags *HelmFlags) error {
	resourcesPath := a.GetResourceDir()

	commonParams := make([]string, 0)
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "-n", a.GetNamespace())
	}
	if a.GetKubeconfig() != "" {
		commonParams = append(commonParams, "--kubeconfig", a.GetKubeconfig())
	}
	if flags.Debug {
		commonParams = append(commonParams, "--debug")
	}

	params := []string{"install", releaseName, resourcesPath}
	if flags.Wait {
		params = append(params, "--wait")
	}
	if flags.Set != "" {
		params = append(params, "--set", flags.Set)
	}
	if flags.Values != "" {
		params = append(params, "-f", flags.Values)
	}
	params = append(params, commonParams...)

	fmt.Println("building dependency...")
	depParams := []string{"dependency", "build", resourcesPath}
	depParams = append(depParams, commonParams...)
	_, err := tools.ExecCommand(nil, true, "helm", depParams...)
	if err != nil {
		fmt.Printf("fail to build dependency for helm app, err: %v\n", err)
		return err
	}
	//debug(depBuildOutput)

	fmt.Println("install helm application, this may take several minutes, please waiting...")
	_, err = tools.ExecCommand(nil, true, "helm", params...)
	if err != nil {
		fmt.Printf("fail to install helm nocalhostApp, err:%v\n", err)
		return err
	}
	//debug(output)
	fmt.Printf(`helm nocalhostApp installed, use "helm list -n %s" to get the information of the helm release`+"\n", a.GetNamespace())
	return nil
}

func (a *Application) InstallDepConfigMap() error {
	appDep := a.GetDependencies()
	if appDep != nil {
		//debug("install dependency config map")
		var depForYaml = &struct {
			Dependency []*SvcDependency `json:"dependency" yaml:"dependency"`
		}{
			Dependency: appDep,
		}

		yamlBytes, err := yaml.Marshal(depForYaml)
		if err != nil {
			return err
		}

		dataMap := make(map[string]string, 0)
		dataMap["nocalhost"] = string(yamlBytes)

		configMap := &corev1.ConfigMap{
			Data: dataMap,
		}
		//fmt.Println("config map : " + string(yamlBytes))

		var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
		rand.Seed(time.Now().UnixNano())
		b := make([]rune, 4)
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}
		generateName := fmt.Sprintf("nocalhost-depends-do-not-overwrite-%s", string(b))
		configMap.Name = generateName
		_, err = a.client.ClientSet.CoreV1().ConfigMaps(a.GetNamespace()).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			fmt.Printf("[error] fail to create dependency config %s, err: %v\n", configMap.Name, err)
			return err
		} else {
			// debug("config map %s has been installed, record it", configMap.Name)
			a.AppProfile.DependencyConfigMapName = configMap.Name
			a.AppProfile.Save()
		}
	}
	return nil
}

func (a *Application) GetNamespace() string {
	return a.AppProfile.Namespace
}

//func (a *Application) GetKubeconfig() string {
//	return a.AppProfile.Kubeconfig
//}

func (a *Application) GetType() (AppType, error) {
	if a.AppProfile != nil && a.AppProfile.AppType != "" {
		return a.AppProfile.AppType, nil
	}
	if a.Config == nil {
		return "", errors.New("config.yaml not found")
	}
	if a.Config.AppConfig != nil && a.Config.AppConfig.Type != "" {
		return a.Config.AppConfig.Type, nil
	}
	return "", errors.New("can not get app type from config.yaml")
}

func (a *Application) GetKubeconfig() string {
	return a.AppProfile.Kubeconfig
}

//func (a *Application) loadProfile() error {
//	a.AppProfile = &AppProfile{}
//	fBytes, err := ioutil.ReadFile(a.getProfilePath())
//	if err != nil {
//		return err
//	}
//	err = yaml.Unmarshal(fBytes, a.AppProfile)
//	return err
//}

//func (a *Application) SaveProfile() error {
//	if a.AppProfile == nil {
//		return nil
//	}
//	bytes, err := yaml.Marshal(a.AppProfile)
//	if err != nil {
//		return err
//	}
//	profile := a.getProfilePath()
//	err = ioutil.WriteFile(profile, bytes, 0755)
//	return err
//}

func (a *Application) getProfilePath() string {
	return fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, DefaultApplicationProfilePath)
}

func (a *Application) GetHomeDir() string {
	return fmt.Sprintf("%s%c%s%c%s%c%s", utils.GetHomePath(), os.PathSeparator, DefaultNhctlHomeDirName, os.PathSeparator, "application", os.PathSeparator, a.Name)
}

func (a *Application) GetConfigDir() string {
	return fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, DefaultApplicationConfigDirName)
}

func (a *Application) GetConfigPath() string {
	return fmt.Sprintf("%s%c%s", a.GetConfigDir(), os.PathSeparator, DefaultApplicationConfigName)
}

func (a *Application) GetPortForwardDir() string {
	return fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, DefaultPortForwardDir)
}

func (a *Application) getGitDir() string {
	return fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, DefaultResourcesDir)
}

func (a *Application) getConfigPathInGitResourcesDir() string {
	return fmt.Sprintf("%s%c%s%c%s", a.getGitDir(), os.PathSeparator, DefaultApplicationConfigDirName, os.PathSeparator, DefaultApplicationConfigName)
}

// .nhctl/application/xxx/port-forward/{pid}
func (a *Application) GetPortForwardPidDir(pid int) string {
	return fmt.Sprintf("%s%c%d", a.GetPortForwardDir(), os.PathSeparator, pid)
}

// .nhctl/application/xxx/port-forward/{pid}/{local_port}_{remote_port}
func (a *Application) SavePortForwardInfo(svcName string, localPort int, remotePort int) error {
	pid := os.Getpid()
	//pidDir := a.GetPortForwardPidDir(pid)
	//fileName := fmt.Sprintf("%s%c%d_%d", pidDir, os.PathSeparator, localPort, remotePort)
	//f, err := os.Create(fileName)
	//defer f.Close()
	//if err != nil {
	//	return err
	//}

	a.GetSvcProfile(svcName).SshPortForward = &PortForwardOptions{
		LocalPort:  localPort,
		RemotePort: remotePort,
		Pid:        pid,
	}
	return a.AppProfile.Save()
}

//func (a *Application)

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
	if a.Config == nil {
		return nil
	}
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
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile != nil && svcProfile.WorkDir != "" {
		return svcProfile.WorkDir
	}
	config := a.GetSvcConfig(svcName)
	result := DefaultWorkDir
	if config != nil && config.WorkDir != "" {
		result = config.WorkDir
	}
	return result
}

func (a *Application) GetDefaultSideCarImage(svcName string) string {
	config := a.GetSvcConfig(svcName)
	result := DefaultSideCarImage
	if config != nil && config.SideCarImage != "" {
		result = config.SideCarImage
	}
	return result
}

func (a *Application) GetDefaultLocalSyncDirs(svcName string) []string {
	config := a.GetSvcConfig(svcName)
	result := []string{DefaultLocalSyncDirName}
	if config != nil && config.Sync != nil && len(config.Sync) > 0 {
		result = config.Sync
	}
	return result
}

func (a *Application) GetDefaultDevImage(svcName string) string {
	config := a.GetSvcConfig(svcName)
	result := DefaultDevImage
	if config != nil && config.DevImage != "" {
		result = config.DevImage
	}
	return result
}

func (a *Application) GetSvcProfile(svcName string) *SvcProfile {
	if a.AppProfile == nil {
		return nil
	}
	if a.AppProfile.SvcProfile == nil {
		return nil
	}
	for _, svcProfile := range a.AppProfile.SvcProfile {
		if svcProfile.Name == svcName {
			return svcProfile
		}
	}
	return nil
}

func (a *Application) GetLocalSshPort(svcName string) int {
	result := DefaultForwardLocalSshPort
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile != nil && svcProfile.SshPortForward != nil && svcProfile.SshPortForward.LocalPort != 0 {
		result = svcProfile.SshPortForward.LocalPort
	}
	return result
}

func (a *Application) RollBack(svcName string) error {
	clientUtils := a.client

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
	Pid        int `json:"pid" yaml:"pid"`
}

//func (a *Application) CleanupPid(svcName string) error {
//	pidDir := a.GetPortForwardPidDir(os.Getpid())
//	if _, err2 := os.Stat(pidDir); err2 != nil {
//		if os.IsNotExist(err2) {
//			fmt.Printf("%s not exits, no need to cleanup it\n", pidDir)
//			return nil
//		} else {
//			fmt.Printf("[warning] fails to cleanup %s\n", pidDir)
//		}
//	}
//	err := os.RemoveAll(pidDir)
//	if err != nil {
//		fmt.Printf("removing .pid failed, please remove it manually, err:%v\n", err)
//		return err
//	}
//	fmt.Printf("%s cleanup\n", pidDir)
//	return nil
//}

func (a *Application) CleanupSshPortForwardInfo(svcName string) error {
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile == nil {
		return errors.New(fmt.Sprintf("\"%s\" not found", svcName))
	}
	svcProfile.SshPortForward = nil
	return a.AppProfile.Save()
}

func (a *Application) SshPortForward(svcName string, ops *PortForwardOptions) error {

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT) // kill -1
	ctx, cancel := context.WithCancel(context.TODO())

	go func() {
		<-c
		cancel()
		fmt.Println("stop port forward")
		//a.CleanupPid()
		a.CleanupSshPortForwardInfo(svcName)
	}()

	// todo check if there is a same port-forward exists

	pid := os.Getpid()
	pidDir := a.GetPortForwardPidDir(pid)
	if _, err2 := os.Stat(pidDir); err2 != nil {
		if os.IsNotExist(err2) {
			err2 := os.Mkdir(pidDir, DefaultNewFilePermission)
			if err2 != nil {
				return err2
			}
		}
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

	err := a.SavePortForwardInfo(svcName, localPort, remotePort)
	if err != nil {
		a.CleanupSshPortForwardInfo(svcName)
		return err
	}
	err = kubectl.PortForward(ctx, a.GetKubeconfig(), a.GetNamespace(), svcName, fmt.Sprintf("%d", localPort), fmt.Sprintf("%d", remotePort)) // eg : ./utils/darwin/kubectl port-forward --address 0.0.0.0 deployment/coding  12345:22
	if err != nil {
		fmt.Printf("failed to forward port : %v\n", err)
		return err
	}

	//a.CleanupPid()
	a.CleanupSshPortForwardInfo(svcName)
	return nil
}

func (a *Application) CreateSvcProfile(name string, svcType SvcType) {
	if a.AppProfile.SvcProfile == nil {
		a.AppProfile.SvcProfile = make([]*SvcProfile, 0)
	}
	svcProfile := &SvcProfile{
		Name: name,
		Type: svcType,
	}
	a.AppProfile.SvcProfile = append(a.AppProfile.SvcProfile, svcProfile)
}

func (a *Application) CheckIfSvcExist(name string, svcType SvcType) (bool, error) {
	switch svcType {
	case Deployment:
		ctx, _ := context.WithTimeout(context.TODO(), DefaultClientGoTimeOut)
		dep, err := a.client.GetDeployment(ctx, a.GetNamespace(), name)
		if err != nil {
			return false, err
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

func (a *Application) ReplaceImage(deployment string, ops *DevStartOptions) error {

	deploymentsClient := a.client.GetDeploymentClient(a.GetNamespace())

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
	dep, err := a.client.GetDeployment(context.TODO(), a.GetNamespace(), deployment)
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
	// delete users SecurityContext
	dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// set the entry
	dep.Spec.Template.Spec.Containers[0].WorkingDir = workDir

	//cmds.debug("disable readiness probes")
	for i := 0; i < len(dep.Spec.Template.Spec.Containers); i++ {
		dep.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		dep.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		dep.Spec.Template.Spec.Containers[i].StartupProbe = nil
	}

	sideCarImage := a.GetDefaultSideCarImage(deployment)
	if ops.SideCarImage != "" {
		sideCarImage = ops.SideCarImage
	}
	sideCarContainer := corev1.Container{
		Name:  "nocalhost-sidecar",
		Image: sideCarImage,
		//Command: []string{"/bin/sh", "-c", "service ssh start; mutagen daemon start; mutagen-agent install; tail -f /dev/null"},
	}
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, volMount)
	sideCarContainer.LivenessProbe = &corev1.Probe{
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		Handler: corev1.Handler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.IntOrString{
					IntVal: DefaultForwardRemoteSshPort,
				},
			},
		},
	}
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer)

	_, err = a.client.UpdateDeployment(context.TODO(), a.GetNamespace(), dep, metav1.UpdateOptions{}, true)
	if err != nil {
		fmt.Printf("update develop container failed : %v \n", err)
		return err
	}

	<-time.NewTimer(time.Second * 3).C

	podList, err := a.client.ListPodsOfDeployment(a.GetNamespace(), dep.Name)
	if err != nil {
		fmt.Printf("failed to get pods, err: %v\n", err)
		return err
	}

	fmt.Printf("%d pod found\n", len(podList)) // should be 2

	// wait podList to be ready
	fmt.Printf("waiting pod to start.")
	for {
		<-time.NewTimer(time.Second * 2).C
		podList, err = a.client.ListPodsOfDeployment(a.GetNamespace(), dep.Name)
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

func (a *Application) FileSync(svcName string, ops *FileSyncOptions) error {
	var err error
	var localSharedDirs = a.GetDefaultLocalSyncDirs(svcName)
	localSshPort := ops.LocalSshPort
	if localSshPort == 0 {
		localSshPort = a.GetLocalSshPort(svcName)
	}
	remoteDir := ops.RemoteDir
	if remoteDir == "" {
		remoteDir = a.GetDefaultWorkDir(svcName)
	}

	if ops.LocalSharedFolder != "" {
		err = mutagen.FileSync(ops.LocalSharedFolder, remoteDir, fmt.Sprintf("%d", localSshPort))
	} else if len(localSharedDirs) > 0 {
		for _, dir := range localSharedDirs {
			err = mutagen.FileSync(dir, remoteDir, fmt.Sprintf("%d", localSshPort))
			if err != nil {
				break
			}
		}
	} else {
		err = errors.New("which dir to sync ?")
	}
	return err
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

func (a *Application) SetDevelopingStatus(svcName string, is bool) error {
	a.GetSvcProfile(svcName).Developing = is
	return a.AppProfile.Save()
}

func (a *Application) SetInstalledStatus(is bool) error {
	a.AppProfile.Installed = is
	return a.AppProfile.Save()
}

func (a *Application) SetAppType(t AppType) error {
	a.AppProfile.AppType = t
	return a.AppProfile.Save()
}

func (a *Application) SetPortForwardedStatus(svcName string, is bool) error {
	a.GetSvcProfile(svcName).PortForwarded = is
	return a.AppProfile.Save()
}

func (a *Application) SetSyncingStatus(svcName string, is bool) error {
	a.GetSvcProfile(svcName).Syncing = is
	return a.AppProfile.Save()
}

func (a *Application) Uninstall() error {

	fmt.Printf("app config is %v\n", a.Config)

	if a.AppProfile.DependencyConfigMapName != "" {
		fmt.Printf("delete config map %s\n", a.AppProfile.DependencyConfigMapName)
		err := a.client.DeleteConfigMapByName(a.AppProfile.DependencyConfigMapName, a.AppProfile.Namespace)
		if err != nil {
			return err
		}
		a.AppProfile.DependencyConfigMapName = ""
		a.AppProfile.Save()
	}

	if a.IsHelm() {
		// todo
		commonParams := make([]string, 0)
		if a.GetNamespace() != "" {
			commonParams = append(commonParams, "--namespace", a.GetNamespace())
		}
		if a.AppProfile.Kubeconfig != "" {
			commonParams = append(commonParams, "--kubeconfig", a.AppProfile.Kubeconfig)
		}
		//if settings.Debug {
		//	commonParams = append(commonParams, "--debug")
		//}
		installParams := []string{"uninstall", a.Name}
		installParams = append(installParams, commonParams...)
		_, err := tools.ExecCommand(nil, true, "helm", installParams...)
		if err != nil {
			//printlnErr("fail to uninstall helm nocalhostApp", err)
			return err
		}
		//debug(output)
		fmt.Printf("\"%s\" has been uninstalled \n", a.Name)
	} else if a.IsManifest() {
		start := time.Now()
		wg := sync.WaitGroup{}
		resourceDir := a.GetResourceDir()
		files, _, err := a.getYamlFilesAndDirs(resourceDir)
		if err != nil {
			return err
		}
		for _, file := range files {
			wg.Add(1)
			fmt.Println("delete " + file)
			go func(fileName string) {
				a.client.Delete(fileName, a.GetNamespace())
				wg.Done()
			}(file)

		}
		wg.Wait()
		end := time.Now()
		fmt.Printf("installing takes %f seconds\n", end.Sub(start).Seconds())
		if err != nil {
			return err
		}
	}

	fmt.Println("remove resource files...")
	homeDir := a.GetHomeDir()
	err := os.RemoveAll(homeDir)
	if err != nil {
		fmt.Printf("[error] fail to remove nocalhostApp dir %s\n", homeDir)
		os.Exit(1)
	}
	return nil
}
