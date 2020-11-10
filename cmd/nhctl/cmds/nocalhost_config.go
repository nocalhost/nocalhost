package cmds

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"nocalhost/pkg/nhctl/utils"
)

type NocalHostConfig struct {
	PreInstall []*PreInstallItem    `json:"pre_install" yaml:"preInstall"`
	Devs       []*ServiceDevOptions `json:"devs" yaml:"devs"`
}

type PreInstallItem struct {
	Path   string `json:"path" yaml:"path"`
	Weight string `json:"weight" yaml:"weight"`
}

type ServiceDevOptions struct {
	Name     string   `json:"name" yaml:"name"`
	Type     string   `json:"type" yaml:"type"`
	GitUrl   string   `json:"git_url" yaml:"gitUrl"`
	DevEnv   string   `json:"dev_env" yaml:"devEnv"` // java|go|node
	DevImage string   `json:"dev_image" yaml:"devImage"`
	Sync     []string `json:"sync" yaml:"sync"`
	Ignore   []string `json:"ignore" yaml:"ignore"`
	DevPort  []string `json:"dev_port" yaml:"devPort"`
	Command  string   `json:"command" yaml:"command"`
	Jobs     []string `json:"jobs" yaml:"jobs"`
	Pods     []string `json:"pods" yaml:"pods"`
}

func NewNocalHostConfig(configPath string) *NocalHostConfig {
	config := &NocalHostConfig{}
	fileBytes, err := ioutil.ReadFile(configPath)
	utils.Mush(err)
	utils.Mush(yaml.Unmarshal(fileBytes, config))
	return config
}
