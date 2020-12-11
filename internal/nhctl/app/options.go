package app

type DevStartOptions struct {
	WorkDir      string
	SideCarImage string
	DevImage     string
	DevLang      string
	Namespace    string
	Kubeconfig   string
	LocalSyncDir []string
}

type FileSyncOptions struct {
	//RemoteDir         string
	//LocalSharedFolder string
	//LocalSshPort           int
	RemoteSyncthingPort    int
	RemoteSyncthingGUIPort int
	LocalSyncthingPort     int
	LocalSyncthingGUIPort  int
	RunAsDaemon            bool
	SyncDouble             bool
}
