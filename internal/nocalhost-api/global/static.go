/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
