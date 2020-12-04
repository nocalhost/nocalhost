package app

func (a *Application) CheckIfSvcIsDeveloping(svcName string) bool {
	profile := a.GetSvcProfile(svcName)
	if profile == nil {
		return false
	}
	return profile.Developing
}

func (a *Application) CheckIfSvcIsSyncthing(svcName string) bool {
	profile := a.GetSvcProfile(svcName)
	if profile == nil {
		return false
	}
	return profile.Syncing
}

func (a *Application) CheckIfSvcIsPortForwaed(svcName string) bool {
	profile := a.GetSvcProfile(svcName)
	if profile == nil {
		return false
	}
	return profile.PortForwarded
}

func (a *Application) GetLocalSshPort(svcName string) int {
	result := DefaultForwardLocalSshPort
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile != nil && svcProfile.SshPortForward != nil && svcProfile.SshPortForward.LocalPort != 0 {
		result = svcProfile.SshPortForward.LocalPort
	}
	return result
}

func (a *Application) GetSvcProfile(svcName string) *SvcProfile {
	if a.AppProfile == nil {
		return nil
	}
	if a.AppProfile.SvcProfile == nil {
		return nil
	}
	for _, svcProfile := range a.AppProfile.SvcProfile {
		if svcProfile.Name == svcName {
			return svcProfile
		}
	}
	return nil
}
