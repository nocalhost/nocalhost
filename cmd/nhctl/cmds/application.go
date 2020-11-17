package cmds

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"nocalhost/pkg/nhctl/tools"
	"os"
	"strconv"
)

type Application struct {
	Name       string
	Config     *NocalHostAppConfig
	AppProfile *AppProfile // runtime info
}

type AppProfile struct {
	Namespace               string `json:"namespace" yaml:"namespace"`
	Kubeconfig              string `json:"kubeconfig" yaml:"kubeconfig"`
	DependencyConfigMapName string `json:"dependency_config_map_name" yaml:"dependencyConfigMapName"`
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
	return nil
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
