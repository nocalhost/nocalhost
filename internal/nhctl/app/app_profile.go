package app

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type AppProfile struct {
	path                    string
	Name                    string        `json:"name" yaml:"name"`
	ReleaseName             string        `json:"release_name yaml:releaseName"`
	Namespace               string        `json:"namespace" yaml:"namespace"`
	Kubeconfig              string        `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string        `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 AppType       `json:"app_type" yaml:"appType"`
	SvcProfile              []*SvcProfile `json:"svc_profile" yaml:"svcProfile"` // this will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil
	Installed               bool          `json:"installed" yaml:"installed"`
	ResourcePath            []string      `json:"resource_path" yaml:"resourcePath"`
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
