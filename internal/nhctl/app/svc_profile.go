package app

type SvcProfile struct {
	*ServiceDevOptions `yaml:"rawConfig"`
	ActualName         string `json:"actual_name" yaml:"actualName"` // for helm, actualName may be ReleaseName-Name
	//Type           SvcType             `json:"type" yaml:"type"`
	//SshPortForward *PortForwardOptions `json:"ssh_port_forward" yaml:"sshPortForward,omitempty"`
	Developing    bool `json:"developing" yaml:"developing"`
	PortForwarded bool `json:"port_forwarded" yaml:"portForwarded"`
	Syncing       bool `json:"syncing" yaml:"syncing"`
	//WorkDir       string `json:"work_dir" yaml:"workDir"`
	// same as local available port, use for port-forward
	RemoteSyncthingPort int `json:"remoteSyncthingPort" yaml:"remoteSyncthingPort"`
	// same as local available port, use for port-forward
	RemoteSyncthingGUIPort int `json:"remoteSyncthingGUIPort" yaml:"remoteSyncthingGUIPort"`
	// syncthing local port
	LocalSyncthingPort                     int      `json:"localSyncthingPort" yaml:"localSyncthingPort"`
	LocalSyncthingGUIPort                  int      `json:"localSyncthingGUIPort" yaml:"localSyncthingGUIPort"`
	LocalAbsoluteSyncDirFromDevStartPlugin []string `json:"localAbsoluteSyncDirFromDevStartPlugin" yaml:"localAbsoluteSyncDirFromDevStartPlugin"`
	DevPortList                            []string `json:"devPortList" yaml:"devPortList"`
}

type SvcProfileForPlugin struct {
	Name                                   string   `json:"name" yaml:"name"` // for helm, actualName may be ReleaseName-Name
	Type                                   SvcType  `json:"serviceType" yaml:"serviceType"`
	GitUrl                                 string   `json:"gitUrl" yaml:"gitUrl"`
	DevImage                               string   `json:"devContainerImage" yaml:"devContainerImage"`
	WorkDir                                string   `json:"workDir" yaml:"workDir"`
	Sync                                   []string `json:"syncDirs" yaml:"syncDirs"`
	Ignore                                 []string `json:"ignores" yaml:"ignores"` // TODO Ignore file list
	DevPort                                []string `json:"devPorts" yaml:"devPorts"`
	Developing                             bool     `json:"developing" yaml:"developing"`
	PortForwarded                          bool     `json:"port_forwarded" yaml:"portForwarded"`
	Syncing                                bool     `json:"syncing" yaml:"syncing"`
	LocalAbsoluteSyncDirFromDevStartPlugin []string `json:"localAbsoluteSyncDirFromDevStartPlugin" yaml:"localAbsoluteSyncDirFromDevStartPlugin"`
	DevPortList                            []string `json:"devPortList" yaml:"devPortList"`
}
