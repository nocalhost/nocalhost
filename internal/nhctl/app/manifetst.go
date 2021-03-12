package app

import "nocalhost/pkg/nhctl/clientgoutils"

func StandardNocalhostMetas(applicationName, applicationNamespace string) *clientgoutils.ApplyFlags {
	return &clientgoutils.ApplyFlags{
		MergeableLabel: map[string]string{
			AppManagedByLabel: AppManagedByNocalhost,
		},

		MergeableAnnotation: map[string]string{
			NocalhostApplicationName:      applicationName,
			NocalhostApplicationNamespace: applicationNamespace,
		},
	}
}
