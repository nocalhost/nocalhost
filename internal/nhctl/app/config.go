package app

import "errors"

// Deprecated: this struct is deprecated.
type Config struct {
	Name            string     `json:"name" yaml:"name"`
	ManifestType    string     `json:"manifestType" yaml:"manifestType"`
	ResourcePath    []string   `json:"resourcePath" yaml:"resourcePath"`
	MinimalInstall  bool       `json:"minimalInstall" yaml:"minimalInstall"`
	OnPreInstall    []*Hook    `json:"onPreInstall" yaml:"onPreInstall"`
	OnPostInstall   []*Hook    `json:"onPostInstall" yaml:"onPostInstall"`
	OnPreUninstall  []*Hook    `json:"onPreUninstall" yaml:"onPreUninstall"`
	OnPostUninstall []*Hook    `json:"onPostUninstall" yaml:"onPostUninstall"`
	Service         []*Service `json:"service" yaml:"service"`
}

// Deprecated: this struct is deprecated.
type Hook struct {
	Path     string `json:"path" yaml:"path"`
	Name     string `json:"name" yaml:"name"`
	Priority int64  `json:"priority" yaml:"priority"`
}

// Deprecated: this struct is deprecated.
type Service struct {
	Name                    string   `json:"name" yaml:"name"`
	NameRegex               string   `json:"nameRegex" yaml:"nameRegex"`
	ServiceType             string   `json:"serviceType" yaml:"serviceType"`
	GitUrl                  string   `json:"gitUrl" yaml:"gitUrl"`
	DevContainerImage       string   `json:"devContainerImage" yaml:"devContainerImage"`
	DevContainerShell       string   `json:"devContainerShell" yaml:"devContainerShell"`
	SyncType                string   `json:"syncType" yaml:"syncType"`
	SyncDirs                []string `json:"syncDirs" yaml:"syncDirs"`
	IgnoreDirs              []string `json:"ignoreDirs" yaml:"ignoreDirs"`
	DevPort                 []string `json:"devPort" yaml:"devPort"`
	DependPodsLabelSelector []string `json:"dependPodsLabelSelector" yaml:"dependPodsLabelSelector"`
	DependJobsLabelSelector []string `json:"dependJobsLabelSelector" yaml:"dependJobsLabelSelector"`
	WorkDir                 string   `json:"workDir" yaml:"workDir"`
	PersistentVolumeDir     []string `json:"persistentVolumeDir" yaml:"persistentVolumeDir"`
	BuildCommand            string   `json:"buildCommand" yaml:"buildCommand"`
	RunCommand              string   `json:"runCommand" yaml:"runCommand"`
	DebugCommand            string   `json:"debugCommand" yaml:"debugCommand"`
	HotReloadRunCommand     string   `json:"hotReloadRunCommand" yaml:"hotReloadRunCommand"`
	HotReloadDebugCommand   string   `json:"hotReloadDebugCommand" yaml:"hotReloadDebugCommand"`
	RemoteDebugPort         string   `json:"remoteDebugPort" yaml:"remoteDebugPort"`
	UseDevContainer         string   `json:"useDevContainer" yaml:"useDevContainer"`
}

func (c *Config) CheckValid() error {
	if len(c.Name) == 0 {
		return errors.New("Application name is missing")
	}
	if len(c.ManifestType) == 0 {
		return errors.New("Application manifestType is missing")
	}
	if len(c.ResourcePath) == 0 {
		return errors.New("Application resourcePath is missing")
	}
	preInstall := c.OnPreInstall
	if preInstall != nil && len(preInstall) > 0 {
		for _, hook := range preInstall {
			if len(hook.Name) == 0 {
				return errors.New("Application preInstall item's name is missing")
			}
			if len(hook.Path) == 0 {
				return errors.New("Application preInstall item's path is missing")
			}
		}
	}
	postInstall := c.OnPostInstall
	if postInstall != nil && len(postInstall) > 0 {
		for _, hook := range postInstall {
			if len(hook.Name) == 0 {
				return errors.New("Application postInstall item's name is missing")
			}
			if len(hook.Path) == 0 {
				return errors.New("Application postInstall item's path is missing")
			}
		}
	}
	preUninstall := c.OnPreUninstall
	if preUninstall != nil && len(preUninstall) > 0 {
		for _, hook := range preUninstall {
			if len(hook.Name) == 0 {
				return errors.New("Application preUninstall item's name is missing")
			}
			if len(hook.Path) == 0 {
				return errors.New("Application preUninstall item's path is missing")
			}
		}
	}
	postUninstall := c.OnPostUninstall
	if postUninstall != nil && len(postUninstall) > 0 {
		for _, hook := range postUninstall {
			if len(hook.Name) == 0 {
				return errors.New("Application postUninstall item's name is missing")
			}
			if len(hook.Path) == 0 {
				return errors.New("Application postUninstall item's path is missing")
			}
		}
	}
	services := c.Service
	if services != nil && len(services) > 0 {
		for _, s := range services {
			if len(s.ServiceType) == 0 {
				return errors.New("Application services serviceType is missing")
			}
			if len(s.GitUrl) == 0 {
				return errors.New("Application services gitUrl is missing")
			}
			if len(s.DevContainerImage) == 0 {
				return errors.New("Application services devContainerImage is missing")
			}
		}
	}
	return nil
}
