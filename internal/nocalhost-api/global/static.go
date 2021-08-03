/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */
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
	NocalhostDepJobNamePrefix              = "nocalhost-dep-installer-"
	NocalhostDepKubeConfigMapName          = "nocalhost-kubeconfig"
	NocalhostDepKubeConfigMapKey           = "config"
	NocalhostPrePullDSName                 = "nocalhost-prepull"
	NocalhostDefaultReleaseBranch          = "HEAD"
	//priorityclass
	NocalhostDefaultPriorityclassName         = "nocalhost-container-critical"
	NocalhostDefaultPriorityclassDefaultValue = 1000000
	NocalhostDefaultPriorityclassKey          = "--priority-class"
)

var (
	Version  = "default"
	CommitId = "default"
	Branch   = "default"

	ServiceInitial = "false"
)
