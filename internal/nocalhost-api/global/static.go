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
	NocalhostName                          = "nocalhost"
	NocalhostDevNamespaceLabel             = "nocalhost"
	NocalhostDevServiceAccountName         = "nocalhost-dev-account"
	NocalhostDevServiceAccountSecretCaKey  = "ca.crt"
	NocalhostDevServiceAccountTokenKey     = "token"
	NocalhostDepJobNamePrefix              = "nocalhost-dep-installer-"
	NocalhostPrePullDSName                 = "nocalhost-prepull"
	//priorityclass
	NocalhostDefaultPriorityclassName         = "nocalhost-container-critical"
	NocalhostDefaultPriorityclassDefaultValue = 1000000
	NocalhostDefaultPriorityclassKey          = "--priority-class"
	NocalhostCreateByLabel                    = "app.kubernetes.io/created-by"
	NocalhostRegistry                         = "nocalhost-docker.pkg.coding.net"
	Nocalhostrepository                       = "nocalhost/public/nocalhost-api"
	NocalhostSaTokenSuffix                    = "-token-gen-by-nocalhost"
)

var (
	Version  = "default"
	CommitId = "default"
	Branch   = "default"

	ServiceInitial = "false"
)
