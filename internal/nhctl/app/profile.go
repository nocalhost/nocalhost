package app

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type AppProfile struct {
	path                    string
	Name                    string              `json:"name" yaml:"name"`
	Namespace               string              `json:"namespace" yaml:"namespace"`
	Kubeconfig              string              `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string              `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 AppType             `json:"app_type" yaml:"appType"`
	SshPortForward          *PortForwardOptions `json:"ssh_port_forward" yaml:"sshPortForward,omitempty"`
	Installed               bool                `json:"installed" yaml:"installed"`
	Developing              bool                `json:"developing" yaml:"developing"`
	PortForwarded           bool                `json:"port_forwarded" yaml:"portForwarded"`
	Syncing                 bool                `json:"syncing" yaml:"syncing"`
	Init                    bool                `json:"init" yaml:"init"`
	ResourcePath            string              `json:"resource_path" yaml:"resourcePath"`
}

func NewAppProfile(path string) (*AppProfile, error) {
	app := &AppProfile{
		path: path,
	}
	err := app.Load()
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (a *AppProfile) Save() error {
	bytes, err := yaml.Marshal(a)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(a.path, bytes, 0755)
	return err
}

func (a *AppProfile) Load() error {
	fBytes, err := ioutil.ReadFile(a.path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(fBytes, a)
	return err
}
