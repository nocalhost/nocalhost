package app

import "nocalhost/pkg/nhctl/clientgoutils"

func StandardNocalhostMetas(releaseName, releaseNamespace string) *clientgoutils.ApplyFlags {
	return &clientgoutils.ApplyFlags{
		MergeableLabel: map[string]string{
			clientgoutils.AppManagedByLabel: clientgoutils.AppManagedByNocalhost,
		},

		MergeableAnnotation: map[string]string{
			clientgoutils.NocalhostReleaseNameAnnotation:      releaseName,
			clientgoutils.NocalhostReleaseNamespaceAnnotation: releaseNamespace,
		},
	}
}
