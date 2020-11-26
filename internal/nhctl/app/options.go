package app

type DevStartOptions struct {
	WorkDir      string
	SideCarImage string
	DevImage     string
	DevLang      string
	Namespace    string
	Kubeconfig   string
}

type FileSyncOptions struct {
	RemoteDir         string
	LocalSharedFolder string
	LocalSshPort      int
}
