package app

type SvcProfile struct {
	Name           string              `json:"name" yaml:"name"`
	Type           SvcType             `json:"type" yaml:"type"`
	SshPortForward *PortForwardOptions `json:"ssh_port_forward" yaml:"sshPortForward,omitempty"`
	Developing     bool                `json:"developing" yaml:"developing"`
	PortForwarded  bool                `json:"port_forwarded" yaml:"portForwarded"`
	Syncing        bool                `json:"syncing" yaml:"syncing"`
	WorkDir        string              `json:"work_dir" yaml:"workDir"`
	// same as local available port, use for port-forward
	RemoteSyncthingPort int `json:"remoteSyncthingPort" yaml:"remoteSyncthingPort"`
	// same as local available port, use for port-forward
	RemoteSyncthingGUIPort int `json:"remoteSyncthingGUIPort" yaml:"remoteSyncthingGUIPort"`
	// syncthing local port
	LocalSyncthingPort                     int      `json:"localSyncthingPort" yaml:"localSyncthingPort"`
	LocalSyncthingGUIPort                  int      `json:"localSyncthingGUIPort" yaml:"localSyncthingGUIPort"`
	LocalAbsoluteSyncDirFromDevStartPlugin []string `json:"localAbsoluteSyncDirFromDevStartPlugin" yaml:"localAbsoluteSyncDirFromDevStartPlugin"`
	DevPortList                            []string `json:"devPortList" yaml:"devPortList"`
	//Init           bool                `json:"init" yaml:"init"`
}
