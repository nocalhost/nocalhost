package app

import "nocalhost/pkg/nhctl/clientgoutils"

func StandardNocalhostMetas(releaseName, releaseNamespace string) *clientgoutils.ApplyFlags {
	return &clientgoutils.ApplyFlags{
		MergeableLabel: map[string]string{
			AppManagedByLabel: AppManagedByNocalhost,
		},

		MergeableAnnotation: map[string]string{
			NocalhostReleaseNameAnnotation:      releaseName,
			NocalhostReleaseNamespaceAnnotation: releaseNamespace,
		},
	}
}
