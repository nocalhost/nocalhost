package app

import (
	"context"
	"errors"
	"fmt"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"nocalhost/pkg/nhctl/log"
	"path"
	"path/filepath"
	"runtime"
	"strconv"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	//"github.com/sirupsen/logrus"
	"io/ioutil"
	"math/rand"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/third_party/kubectl"
	"nocalhost/pkg/nhctl/tools"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (a *Application) DownloadResourcesFromGit(gitUrl string, gitRef string) error {
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
		if len(gitRef) > 0 {
			_, err = tools.ExecCommand(nil, true, "git", "clone", "--branch", gitRef, "--depth", "1", gitUrl, a.getGitDir())
		} else {
			_, err = tools.ExecCommand(nil, true, "git", "clone", "--depth", "1", gitUrl, a.getGitDir())
		}
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
		return a.getGitDir()
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
	//err = a.InstallDepConfigMap()
	//if err != nil {
	//	return err
	//}
	excludeFiles := make([]string, 0)
	if a.Config != nil && a.Config.PreInstall != nil {
		fmt.Println("[config] reading pre-install hook")
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
				log.Debugf("ignore file : %s", file)
				continue outer
			}
		}

		// parallel
		wg.Add(1)
		go func(fileName string) {
			log.Debugf("create %s", fileName)
			a.client.Create(fileName, a.GetNamespace(), false, false)
			wg.Done()
		}(file)
	}
	wg.Wait()
	end := time.Now()
	fmt.Printf("installing takes %f seconds\n", end.Sub(start).Seconds())
	return nil
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
		err := a.client.Create(itemPath, a.GetNamespace(), true, false)
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

	fmt.Println("install helm application, this may take several minutes, please waiting...")
	_, err = tools.ExecCommand(nil, true, "helm", params...)
	if err != nil {
		fmt.Printf("fail to install helm nocalhostApp, err:%v\n", err)
		return err
	}
	fmt.Printf(`helm nocalhostApp installed, use "helm list -n %s" to get the information of the helm release`+"\n", a.GetNamespace())
	return nil
}

func (a *Application) InstallDepConfigMap() error {
	appDep := a.GetDependencies()
	if appDep != nil {
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

func (a *Application) getProfilePath() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationProfilePath)
	//fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, DefaultApplicationProfilePath)
}

func (a *Application) GetHomeDir() string {
	return filepath.Join(utils.GetHomePath(), DefaultNhctlHomeDirName, DefaultApplicationDirName, a.Name)
	//return fmt.Sprintf("%s%c%s%c%s%c%s", utils.GetHomePath(), os.PathSeparator, DefaultNhctlHomeDirName, os.PathSeparator, DefaultApplicationDirName, os.PathSeparator, a.Name)
}

func (a *Application) GetApplicationSyncDir(deployment string) string {
	dirPath := path.Join(a.GetHomeDir(), DefaultBinSyncThingDirName, deployment)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0700)
		if err != nil {
			log.Fatalf("fail to create syncthing directory: %s", dirPath)
		}
	}
	return dirPath
}

func (a *Application) GetApplicationBackGroundPortForwardPidFile(deployment string) string {
	return path.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationSyncPortForwardPidFile)
}

func (a *Application) GetApplicationBackGroundOnlyPortForwardPidFile(deployment string) string {
	return path.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationOnlyPortForwardPidFile)
}

func (a *Application) GetApplicationSyncThingPidFile(deployment string) string {
	return path.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationSyncPidFile)
}

func (a *Application) GetApplicationOnlyPortForwardPidFile(deployment string) string {
	return path.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationOnlyPortForwardPidFile)
}

func (a *Application) GetConfigDir() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationConfigDirName)
	//return fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, DefaultApplicationConfigDirName)
}

func (a *Application) GetConfigPath() string {
	return filepath.Join(a.GetConfigDir(), DefaultApplicationConfigName)
	//return fmt.Sprintf("%s%c%s", a.GetConfigDir(), os.PathSeparator, DefaultApplicationConfigName)
}

func (a *Application) GetLogDir() string {
	return path.Join(utils.GetHomePath(), DefaultNhctlHomeDirName, DefaultLogDirName)
}

func (a *Application) GetPortSyncLogFile(deployment string) string {
	return path.Join(a.GetApplicationSyncDir(deployment), DefaultSyncLogFileName)
}

func (a *Application) GetPortForwardLogFile(deployment string) string {
	return path.Join(a.GetApplicationSyncDir(deployment), DefaultBackGroundPortForwardLogFileName)
}

//func (a *Application) GetPortForwardDir() string {
//	return fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, DefaultPortForwardDir)
//}

func (a *Application) getGitDir() string {
	return filepath.Join(a.GetHomeDir(), DefaultResourcesDir)
	//return fmt.Sprintf("%s%c%s", a.GetHomeDir(), os.PathSeparator, DefaultResourcesDir)
}

func (a *Application) getConfigPathInGitResourcesDir() string {
	return filepath.Join(a.getGitDir(), DefaultApplicationConfigDirName, DefaultApplicationConfigName)
	//return fmt.Sprintf("%s%c%s%c%s", a.getGitDir(), os.PathSeparator, DefaultApplicationConfigDirName, os.PathSeparator, DefaultApplicationConfigName)
}

func (a *Application) GetSyncThingBinDir() string {
	return path.Join(utils.GetHomePath(), DefaultNhctlHomeDirName, DefaultBinDirName, DefaultBinSyncThingDirName)
}

//func (a *Application) GetPortForwardPidDir(pid int) string {
//	return fmt.Sprintf("%s%c%d", a.GetPortForwardDir(), os.PathSeparator, pid)
//}

func (a *Application) SavePortForwardInfo(svcName string, localPort int, remotePort int) error {
	pid := os.Getpid()

	a.GetSvcProfile(svcName).SshPortForward = &PortForwardOptions{
		LocalPort:  localPort,
		RemotePort: remotePort,
		Pid:        pid,
	}
	return a.AppProfile.Save()
}

func (a *Application) ListPortForwardPid(svcName string) []int {
	result := make([]int, 0)
	profile := a.GetSvcProfile(svcName)
	if profile == nil || profile.SshPortForward == nil {
		return result
	}
	if profile.SshPortForward.Pid != 0 {
		result = append(result, profile.SshPortForward.Pid)
	}
	return result
}

func (a *Application) StopAllPortForward(svcName string) error {
	pids := a.ListPortForwardPid(svcName)
	for _, pid := range pids {
		_, err := tools.ExecCommand(nil, true, "kill", "-1", fmt.Sprintf("%d", pid))
		if err != nil {
			fmt.Printf("failed to stop port forward pid %d, err: %v\n", pid, err)
			return err
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

func (a *Application) RollBack(ctx context.Context, svcName string) error {
	clientUtils := a.client

	dep, err := clientUtils.GetDeployment(ctx, a.GetNamespace(), svcName)
	if err != nil {
		fmt.Printf("failed to get deployment %s , err : %v\n", dep.Name, err)
		return err
	}

	rss, err := clientUtils.GetSortedReplicaSetsByDeployment(ctx, a.GetNamespace(), svcName)
	if err != nil {
		fmt.Printf("failed to get rs list, err:%v\n", err)
		return err
	}
	// find previous replicaSet
	if len(rss) < 2 {
		fmt.Println("no history to roll back")
		return nil
	}

	var r *v1.ReplicaSet
	for _, rs := range rss {
		if rs.Annotations == nil {
			continue
		}
		if rs.Annotations[DevImageFlagAnnotationKey] == DevImageFlagAnnotationValue {
			r = rs
		}
	}
	if r == nil {
		return errors.New("fail to find the proper revision to rollback")
	}

	dep.Spec.Template = r.Spec.Template

	spinner := utils.NewSpinner(" Rolling container's revision back...")
	spinner.Start()
	_, err = clientUtils.UpdateDeployment(ctx, a.GetNamespace(), dep, metav1.UpdateOptions{}, true)
	spinner.Stop()
	if err != nil {
		coloredoutput.Fail("Failed to roll revision back")
		//fmt.Println("failed rolling back")
	} else {
		coloredoutput.Success("Container has been rollback")
	}

	return err
}

type PortForwardOptions struct {
	LocalPort   int      `json:"local_port" yaml:"localPort"`
	RemotePort  int      `json:"remote_port" yaml:"remotePort"`
	Pid         int      `json:"pid" yaml:"pid"`
	DevPort     []string // 8080:8080 or :8080 means random localPort
	RunAsDaemon bool
}

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
		a.CleanupSshPortForwardInfo(svcName)
	}()

	// todo check if there is a same port-forward exists

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

	a.CleanupSshPortForwardInfo(svcName)
	return nil
}

func (a *Application) CreateSvcProfile(name string, svcType SvcType) {
	if a.AppProfile.SvcProfile == nil {
		a.AppProfile.SvcProfile = make([]*SvcProfile, 0)
	}

	for _, svc := range a.AppProfile.SvcProfile {
		if svc.Name == name {
			return
		}
	}
	svcProfile := &SvcProfile{
		Name: name,
		Type: svcType,
	}
	a.AppProfile.SvcProfile = append(a.AppProfile.SvcProfile, svcProfile)
}

func (a *Application) CheckIfSvcIsDeveloping(svcName string) bool {
	profile := a.GetSvcProfile(svcName)
	if profile == nil {
		return false
	}
	return profile.Developing
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

func (a *Application) CreateSyncThingSecret(syncSecret *corev1.Secret, ops *DevStartOptions) error {
	// check if secret exist
	exist, err := a.client.GetSecret(context.TODO(), ops.Namespace, syncSecret.Name)
	if exist.Name != "" {
		_ = a.client.DeleteSecret(context.TODO(), ops.Namespace, syncSecret.Name)
	}
	_, err = a.client.CreateSecret(context.TODO(), ops.Namespace, syncSecret, metav1.CreateOptions{})
	if err != nil {
		// TODO check configmap first, and end dev should delete that secret
		return err
		//log.Fatalf("create syncthing secret fail, please try to manual delete %s secret first", syncthing.SyncSecretName)
	}
	return nil
}

func (a *Application) ReplaceImage(ctx context.Context, deployment string, ops *DevStartOptions) error {
	deploymentsClient := a.client.GetDeploymentClient(a.GetNamespace())

	// mark current revision for rollback
	rss, err := a.client.GetSortedReplicaSetsByDeployment(ctx, a.GetNamespace(), deployment)
	if err != nil {
		return err
	}
	if rss != nil && len(rss) > 0 {
		rs := rss[len(rss)-1]
		rs.Annotations[DevImageFlagAnnotationKey] = DevImageFlagAnnotationValue
		_, err = a.client.ClientSet.AppsV1().ReplicaSets(a.GetNamespace()).Update(ctx, rs, metav1.UpdateOptions{})
		if err != nil {
			return errors.New("fail to update rs's annotation")
		}
	}

	scale, err := deploymentsClient.GetScale(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return err
	}

	fmt.Println("developing deployment: " + deployment)
	fmt.Println("scaling replicas to 1")

	if scale.Spec.Replicas > 1 {
		fmt.Printf("deployment %s's replicas is %d now\n", deployment, scale.Spec.Replicas)
		scale.Spec.Replicas = 1
		_, err = deploymentsClient.UpdateScale(ctx, deployment, scale, metav1.UpdateOptions{})
		if err != nil {
			fmt.Println("failed to scale replicas to 1")
		} else {
			time.Sleep(time.Second * 5) // todo check replicas
			fmt.Println("replicas has been scaled to 1")
		}
	} else {
		fmt.Printf("deployment %s's replicas is already 1\n", deployment)
	}

	fmt.Println("Updating development container...")
	dep, err := a.client.GetDeployment(ctx, a.GetNamespace(), deployment)
	if err != nil {
		//fmt.Printf("failed to get deployment %s , err : %v\n", deployment, err)
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

	// syncthing secret volume
	syncthingDir := corev1.Volume{
		Name: secret_config.EmptyDir,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	defaultMode := int32(0644)
	syncthingVol := corev1.Volume{
		Name: secret_config.SecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: deployment + "-" + secret_config.SecretName,
				Items: []corev1.KeyToPath{
					{
						Key:  "config.xml",
						Path: "config.xml",
						Mode: &defaultMode,
					},
					{
						Key:  "cert.pem",
						Path: "cert.pem",
						Mode: &defaultMode,
					},
					{
						Key:  "key.pem",
						Path: "key.pem",
						Mode: &defaultMode,
					},
				},
				DefaultMode: &defaultMode,
			},
		},
	}

	if dep.Spec.Template.Spec.Volumes == nil {
		dep.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, vol, syncthingVol, syncthingDir)

	// syncthing volume mount
	syncthingVolHomeDirMount := corev1.VolumeMount{
		Name:      secret_config.EmptyDir,
		MountPath: secret_config.DefaultSyncthingHome,
		SubPath:   "syncthing",
	}

	// syncthing secret volume
	syncthingVolMount := corev1.VolumeMount{
		Name:      secret_config.SecretName,
		MountPath: secret_config.DefaultSyncthingSecretHome,
		ReadOnly:  false,
	}

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

	dep.Spec.Template.Spec.Containers[0].Image = devImage
	dep.Spec.Template.Spec.Containers[0].Name = "nocalhost-dev"
	dep.Spec.Template.Spec.Containers[0].Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	dep.Spec.Template.Spec.Containers[0].VolumeMounts = append(dep.Spec.Template.Spec.Containers[0].VolumeMounts, volMount)
	// delete users SecurityContext
	dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// set the entry
	dep.Spec.Template.Spec.Containers[0].WorkingDir = workDir

	// disable readiness probes
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
		Name:       "nocalhost-sidecar",
		Image:      sideCarImage,
		WorkingDir: workDir,
		//Command: []string{"/bin/sh", "-c", "service ssh start; mutagen daemon start; mutagen-agent install; tail -f /dev/null"},
	}
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, volMount, syncthingVolMount, syncthingVolHomeDirMount)
	//sideCarContainer.LivenessProbe = &corev1.Probe{
	//	InitialDelaySeconds: 10,
	//	PeriodSeconds:       10,
	//	Handler: corev1.Handler{
	//		TCPSocket: &corev1.TCPSocketAction{
	//			Port: intstr.IntOrString{
	//				IntVal: DefaultForwardRemoteSshPort,
	//			},
	//		},
	//	},
	//}
	// over write syncthing command
	sideCarContainer.Command = []string{"/bin/sh", "-c"}
	sideCarContainer.Args = []string{"unset STGUIADDRESS && cp " + secret_config.DefaultSyncthingSecretHome + "/* " + secret_config.DefaultSyncthingHome + "/ && /bin/entrypoint.sh && /bin/syncthing -home /var/syncthing"}
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer)

	_, err = a.client.UpdateDeployment(ctx, a.GetNamespace(), dep, metav1.UpdateOptions{}, true)
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

	log.Debugf("%d pod found", len(podList)) // should be 2

	// wait podList to be ready
	spinner := utils.NewSpinner(" Waiting pod to start")
	spinner.Start()

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
	}
	spinner.Stop()
	coloredoutput.Success("Development container has been updated")
	return nil
}

//func (a *Application) FileSync(svcName string, ops *FileSyncOptions) error {
//	var err error
//	var localSharedDirs = a.GetDefaultLocalSyncDirs(svcName)
//	localSshPort := ops.LocalSshPort
//	if localSshPort == 0 {
//		localSshPort = a.GetLocalSshPort(svcName)
//	}
//	remoteDir := ops.RemoteDir
//	if remoteDir == "" {
//		remoteDir = a.GetDefaultWorkDir(svcName)
//	}
//
//	if ops.LocalSharedFolder != "" {
//		err = mutagen.FileSync(ops.LocalSharedFolder, remoteDir, fmt.Sprintf("%d", localSshPort))
//	} else if len(localSharedDirs) > 0 {
//		for _, dir := range localSharedDirs {
//			err = mutagen.FileSync(dir, remoteDir, fmt.Sprintf("%d", localSshPort))
//			if err != nil {
//				break
//			}
//		}
//	} else {
//		err = errors.New("which dir to sync ?")
//	}
//	return err
//}

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

// for background port-forward
func (a *Application) PortForwardInBackGround(deployment, podName, nameSapce string, localPort, remotePort []int) {
	group := len(localPort)
	if len(localPort) != len(remotePort) {
		log.Fatalf("dev port forward fail, please check you devPort in config\n")
	}
	// wait group
	var wg sync.WaitGroup
	wg.Add(group)
	// stream is used to tell the port forwarder where to place its output or
	// where to expect input if needed. For the port forwarding we just need
	// the output eventually
	stream := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	// managing termination signal from the terminal. As you can see the stopCh
	// gets closed to gracefully handle its termination.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	var addDevPod []string
	for key, sLocalPort := range localPort {
		// stopCh control the port forwarding lifecycle. When it gets closed the
		// port forward will terminate
		stopCh := make(chan struct{}, group)
		// readyCh communicate when the port forward is ready to get traffic
		readyCh := make(chan struct{})
		key := key
		sLocalPort := sLocalPort
		devPod := fmt.Sprintf("%d:%d", sLocalPort, remotePort[key])
		addDevPod = append(addDevPod, devPod)
		fmt.Printf("start dev port forward local %d, remote %d \n", sLocalPort, remotePort[key])
		go func() {
			err := a.PortForwardAPod(clientgoutils.PortForwardAPodRequest{
				Pod: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      podName,
						Namespace: nameSapce,
					},
				},
				LocalPort: sLocalPort,
				PodPort:   remotePort[key],
				Streams:   stream,
				StopCh:    stopCh,
				ReadyCh:   readyCh,
			})
			if err != nil {
				fmt.Printf("port-forward in background fail %s\n", err.Error())
			}
		}()
	}
	fmt.Print("done go routine")
	//select {
	//// wait until port-forward success
	//case <-readyCh:
	//	break
	//}
	// update profile addDevPod
	_ = a.SetDevPortForward(deployment, addDevPod)
	// set port forward status
	_ = a.SetPortForwardedStatus(deployment, true)

	for {
		<-sigs
		fmt.Println("stop port forward")
		//close(stopCh)
		wg.Done()
	}
}

// port-forward use
func (a *Application) SetDevPortForward(svcName string, portList []string) error {
	a.GetSvcProfile(svcName).DevPortList = portList
	return a.AppProfile.Save()
}

func (a *Application) GetDevPortForward(svcName string) []string {
	return a.GetSvcProfile(svcName).DevPortList
}

// for syncthing use
func (a *Application) GetSyncthingPort(svcName string, options *FileSyncOptions) (*FileSyncOptions, error) {
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile == nil {
		return options, errors.New("get " + svcName + " profile fail, please reinstall application")
	}
	options.RemoteSyncthingPort = svcProfile.RemoteSyncthingPort
	options.RemoteSyncthingGUIPort = svcProfile.RemoteSyncthingGUIPort
	options.LocalSyncthingPort = svcProfile.LocalSyncthingPort
	options.LocalSyncthingGUIPort = svcProfile.LocalSyncthingGUIPort
	return options, nil
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

func (a *Application) GetPodsFromDeployment(ctx context.Context, namespace, deployment string) (*corev1.PodList, error) {
	return a.client.GetPodsFromDeployment(ctx, namespace, deployment)
}

func (a *Application) PortForwardAPod(req clientgoutils.PortForwardAPodRequest) error {
	return a.client.PortForwardAPod(req)
}

// set pid file empty
func (a *Application) SetPidFileEmpty(filePath string) error {
	return os.Remove(filePath)
}

func (a *Application) SetDevEndProfileStatus(svcName string) error {
	a.GetSvcProfile(svcName).Developing = false
	a.GetSvcProfile(svcName).PortForwarded = false
	a.GetSvcProfile(svcName).Syncing = false
	a.GetSvcProfile(svcName).DevPortList = []string{}
	a.GetSvcProfile(svcName).LocalAbsoluteSyncDirFromDevStartPlugin = []string{}
	return a.AppProfile.Save()
}

func (a *Application) SetSyncthingPort(svcName string, remotePort, remoteGUIPort, localPort, localGUIPort int) error {
	a.GetSvcProfile(svcName).RemoteSyncthingPort = remotePort
	a.GetSvcProfile(svcName).RemoteSyncthingGUIPort = remoteGUIPort
	a.GetSvcProfile(svcName).LocalSyncthingPort = localPort
	a.GetSvcProfile(svcName).LocalSyncthingGUIPort = localGUIPort
	return a.AppProfile.Save()
}

func (a *Application) SetRemoteSyncthingPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).RemoteSyncthingPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetRemoteSyncthingGUIPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).RemoteSyncthingGUIPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetLocalSyncthingPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).LocalSyncthingPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetLocalSyncthingGUIPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).LocalSyncthingGUIPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetLocalAbsoluteSyncDirFromDevStartPlugin(svcName string, syncDir []string) error {
	a.GetSvcProfile(svcName).LocalAbsoluteSyncDirFromDevStartPlugin = syncDir
	return a.AppProfile.Save()
}

// end syncthing here

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

func (a *Application) Uninstall(force bool) error {

	if a.AppProfile.DependencyConfigMapName != "" {
		log.Debug("delete config map %s\n", a.AppProfile.DependencyConfigMapName)
		err := a.client.DeleteConfigMapByName(a.AppProfile.DependencyConfigMapName, a.AppProfile.Namespace)
		if err != nil && !force {
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
		installParams := []string{"uninstall", a.Name}
		installParams = append(installParams, commonParams...)
		_, err := tools.ExecCommand(nil, true, "helm", installParams...)
		if err != nil && !force {
			return err
		}
		fmt.Printf("\"%s\" has been uninstalled \n", a.Name)
	} else if a.IsManifest() {
		start := time.Now()
		wg := sync.WaitGroup{}
		resourceDir := a.GetResourceDir()
		files, _, err := a.getYamlFilesAndDirs(resourceDir)
		if err != nil && !force {
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

	err := a.CleanupResources()
	if err != nil && !force {
		return err
	}

	return nil
}

func (a *Application) CleanupResources() error {
	fmt.Println("remove resource files...")
	homeDir := a.GetHomeDir()
	err := os.RemoveAll(homeDir)
	if err != nil {
		return errors.New(fmt.Sprintf("fail to remove resources dir %s\n", homeDir))
	}
	return nil
}
