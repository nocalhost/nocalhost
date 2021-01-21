package global

const (
	NocalhostSystemNamespace               = "nocalhost-reserved"
	NocalhostSystemNamespaceServiceAccount = "nocalhost-admin-service-account"
	NocalhostSystemRoleBindingName         = "nocalhost-reserved-role-binding"
	NocalhostSystemNamespaceLabel          = "nocalhost-reserved"
	NocalhostDepName                       = "nocalhost-dep"
	NocalhostDevNamespaceLabel             = "nocalhost"
	NocalhostDevServiceAccountName         = "nocalhost-dev-account"
	NocalhostDevDefaultServiceAccountName  = "default"
	NocalhostDevRoleName                   = "nocalhost-dev-role"
	NocalhostDevDefaultRoleName            = "nocalhost-dev-default-role"
	NocalhostDevRoleBindingName            = "nocalhost-role-binding"
	NocalhostDevRoleDefaultBindingName     = "nocalhost-role-default-biding"
	NocalhostDevServiceAccountSecretCaKey  = "ca.crt"
	NocalhostDevServiceAccountTokenKey     = "token"
	NocalhostDepKubeConfigMapName          = "nocalhost-kubeconfig"
	NocalhostDepKubeConfigMapKey           = "config"
	NocalhostPrePullDSName                 = "nocalhost-prepull"
	NocalhostDefaultReleaseBranch          = "HEAD"
)

var (
	Version  = "default"
	CommitId = "default"
	Branch   = "default"
)
