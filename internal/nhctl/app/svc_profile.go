package app

type SvcProfile struct {
	Name           string              `json:"name" yaml:"name"`
	Type           SvcType             `json:"type" yaml:"type"`
	SshPortForward *PortForwardOptions `json:"ssh_port_forward" yaml:"sshPortForward,omitempty"`
	Developing     bool                `json:"developing" yaml:"developing"`
	PortForwarded  bool                `json:"port_forwarded" yaml:"portForwarded"`
	Syncing        bool                `json:"syncing" yaml:"syncing"`
	WorkDir        string              `json:"work_dir" yaml:"workDir"`
	//Init           bool                `json:"init" yaml:"init"`
}
