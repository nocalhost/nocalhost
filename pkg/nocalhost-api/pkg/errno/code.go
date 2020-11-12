/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package errno

//nolint: golint
var (
	// Common errors
	OK                  = &Errno{Code: 0, Message: "OK"}
	InternalServerError = &Errno{Code: 10001, Message: "Internal server error"}
	ErrBind             = &Errno{Code: 10002, Message: "Error occurred while binding the request body to the struct."}
	ErrParam            = &Errno{Code: 10003, Message: "parameters Incorrect"}
	ErrSignParam        = &Errno{Code: 10004, Message: "signature parameters Incorrect"}
	RouterNotFound      = &Errno{Code: 10005, Message: "router not found"}

	// user errors
	ErrUserNotFound          = &Errno{Code: 20102, Message: "The user was not found."}
	ErrTokenInvalid          = &Errno{Code: 20103, Message: "Token is invalid or login expired, please log in again"}
	ErrEmailOrPassword       = &Errno{Code: 20111, Message: "Mail or password is incorrect"}
	ErrTwicePasswordNotMatch = &Errno{Code: 20112, Message: "Two password entries are inconsistent"}
	ErrRegisterFailed        = &Errno{Code: 20113, Message: "Registration failed"}
	ErrUserNotAllow          = &Errno{Code: 20114, Message: "User is disabled"}
	ErrCreateUserDenied      = &Errno{Code: 20115, Message: "No user creation permission"}
	ErrUpdateUserDenied      = &Errno{Code: 20116, Message: "No modify user permission"}
	ErrDeleteUser            = &Errno{Code: 20117, Message: "Failed to delete user"}

	// cluster errors
	ErrClusterCreate      = &Errno{Code: 30100, Message: "Failed to add cluster, please try again"}
	ErrClusterExistCreate = &Errno{Code: 30101, Message: "The cluster already exists (Duplicate Server)"}
	ErrClusterKubeCreate  = &Errno{Code: 30102, Message: "It is not allowed to create this type of cluster (there are multiple Kubeconfig Clusters)"}
	ErrClusterKubeErr     = &Errno{Code: 30103, Message: "Kubeconfig parsing error, please check"}
	ErrClusterKubeAdmin   = &Errno{Code: 30104, Message: "Please check Kubeconfig permissions (Admin) and cluster connectivity"}
	ErrClusterDepSetup    = &Errno{Code: 30105, Message: "Initialize cluster: Failed to create dependent component Configmap"}
	ErrClusterDepJobSetup = &Errno{Code: 30106, Message: "Initialize the cluster: Create dependent component Job failed"}

	// application errors
	ErrApplicationCreate        = &Errno{Code: 40100, Message: "Failed to add app, please try again"}
	ErrApplicationGet           = &Errno{Code: 40101, Message: "Failed to get app, please try again"}
	ErrApplicationDelete        = &Errno{Code: 40102, Message: "Failed to delete application, please try again"}
	ErrApplicationUpdate        = &Errno{Code: 40103, Message: "Update application failed, please try again"}
	ErrBindApplicationClsuter   = &Errno{Code: 40104, Message: "Failed to bind cluster, please try again"}
	ErrPermissionApplication    = &Errno{Code: 40105, Message: "No permission for this application"}
	ErrPermissionCluster        = &Errno{Code: 40106, Message: "No permission for this cluster"}
	ErrApplicationInstallUpdate = &Errno{Code: 40107, Message: "Failed to update app installation status, please try again"}

	// cluster-user errors
	ErrBindUserClsuterRepeat                     = &Errno{Code: 50100, Message: "The user has authorized this application"}
	ErrBindNameSpaceCreate                       = &Errno{Code: 50101, Message: "Cluster user authorization failed: failed to create namespace"}
	ErrBindServiceAccountCreateErr               = &Errno{Code: 50102, Message: "Cluster user authorization failed: Failed to create ServiceAccount"}
	ErrBindRoleCreateErr                         = &Errno{Code: 50103, Message: "Cluster user authorization failed: failed to create a role"}
	ErrBindRoleBindingCreateErr                  = &Errno{Code: 50105, Message: "Cluster user authorization failed: failed to create RoleBinding"}
	ErrBindSecretGetErr                          = &Errno{Code: 50106, Message: "Cluster user authorization failed: Failed to obtain ServiceAccount Secret"}
	ErrBindSecretNameGetErr                      = &Errno{Code: 50107, Message: "Cluster user authorization failed: Failed to obtain ServiceAccount SecretName"}
	ErrBindSecretTokenGetErr                     = &Errno{Code: 50108, Message: "Cluster user authorization failed: Failed to obtain ServiceAccount Token"}
	ErrBindSecretCAGetErr                        = &Errno{Code: 50109, Message: "Cluster user authorization failed: Failed to obtain ServiceAccount CA"}
	ErrBindServiceAccountStructEncodeErr         = &Errno{Code: 50110, Message: "Cluster user authorization failed: encoding ServiceAccount Kubeconfig Json to Yaml failed"}
	ErrBindServiceAccountKubeConfigJsonEncodeErr = &Errno{Code: 50110, Message: "Cluster user authorization failed: encoding ServiceAccount Kubeconfig Struct to Json failed"}
)
