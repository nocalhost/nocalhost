package app

func (a *Application) SetSvcWorkDir(svcName string, workDir string) error {
	a.GetSvcProfile(svcName).WorkDir = workDir
	return a.AppProfile.Save()
}
