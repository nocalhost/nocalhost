/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package errno

// managed all Errno for nocalhost-api
// the message in errno will be displayed
// by calling api.SendResponse(c, &errno.Errno{Code: code, Message: err.Error()}, nil)
// such as:
// api.SendResponse(c, ErrBind, nil)
// frontend will receive "Request fail, Please check request parameters"
//nolint: golint
var (
	// Common errors
	OK                         = &Errno{Code: 0, Message: "OK"}
	InternalServerError        = &Errno{Code: 10001, Message: "Internal server error"}
	ErrBind                    = &Errno{Code: 10002, Message: "Request fail, Please check request parameters"}
	ErrParam                   = &Errno{Code: 10003, Message: "parameters Incorrect"}
	ErrSignParam               = &Errno{Code: 10004, Message: "signature parameters Incorrect"}
	RouterNotFound             = &Errno{Code: 10005, Message: "router not found"}
	ErrLoginRequired           = &Errno{Code: 10006, Message: "log in required"}
	InternalServerTimeoutError = &Errno{Code: 10007, Message: "Internal server timeout"}

	// user errors for user module request
	ErrUserNotFound = &Errno{Code: 20102, Message: "The user does not found."}
	ErrTokenInvalid = &Errno{
		Code: 20103, Message: "Token is invalid or login expired, please coloredoutput in again",
	}
	ErrPermissionDenied           = &Errno{Code: 20104, Message: "permission denied"}
	ErrLostPermissionFlag         = &Errno{Code: 20105, Message: "check permission fail, please re-login"}
	ErrEmailOrPassword            = &Errno{Code: 20111, Message: "Mail or password is incorrect"}
	ErrTwicePasswordNotMatch      = &Errno{Code: 20112, Message: "Two password entries are inconsistent"}
	ErrRegisterFailed             = &Errno{Code: 20113, Message: "Registration failed"}
	ErrUserNotAllow               = &Errno{Code: 20114, Message: "User is disabled"}
	ErrCreateUserDenied           = &Errno{Code: 20115, Message: "No user creation permission"}
	ErrUpdateUserDenied           = &Errno{Code: 20116, Message: "No modify user permission"}
	ErrDeleteUser                 = &Errno{Code: 20117, Message: "Failed to delete user"}
	ErrUserLoginWebNotAllow       = &Errno{Code: 20118, Message: "Normal users are not allowed login web interface"}
	RefreshTokenInvalidOrNotMatch = &Errno{Code: 20119, Message: "Refresh token is invalid or token not matched"}

	// cluster errors for cluster module request
	ErrClusterCreate      = &Errno{Code: 30100, Message: "Failed to add cluster, please try again"}
	ErrClusterExistCreate = &Errno{Code: 30101, Message: "The cluster already exists (Duplicate Server)"}
	ErrClusterKubeCreate  = &Errno{
		Code:    30102,
		Message: "It is not allowed to create this type of cluster (there are multiple Kubeconfig Clusters)",
	}
	ErrClusterKubeErr   = &Errno{Code: 30103, Message: "Kubeconfig parsing error, please check"}
	ErrClusterKubeAdmin = &Errno{Code: 30104, Message: "Please check Kubeconfig Admin permissions"}
	ErrClusterDepSetup  = &Errno{
		Code: 30105, Message: "Initialize cluster: Failed to create dependent component Configmap",
	}
	ErrClusterDepJobSetup = &Errno{
		Code: 30106, Message: "Initialize the cluster: Initial dependent component Job failed",
	}
	ErrClusterNotFound        = &Errno{Code: 30107, Message: "Cluster has not found"}
	ErrDeleteClusterNameSpace = &Errno{Code: 30108, Message: "Delete cluster namespace fail, please try again"}
	ErrGetClusterStorageClass = &Errno{Code: 30109, Message: "Get cluster storage class fail, please try again"}
	ErrUpdateCluster          = &Errno{Code: 30110, Message: "Update cluster fail, please try again"}
	ErrClusterContext         = &Errno{
		Code: 30111, Message: "Failed to get current context from kubeconfig, please check context exists",
	}
	ErrClusterName = &Errno{
		Code:    30112,
		Message: "Failed to get current cluster from kubeconfig, please check cluster exists and manage by current context",
	}
	ErrClusterTimeout = &Errno{
		Code:    30113,
		Message: "Failed to get the connection from current cluster after short wait, please make sure the cluster exists and check it's network connectivity",
	}
	ErrClusterKubeConnect  = &Errno{Code: 30114, Message: "Connect cluster fail, Please check cluster connectivity"}
	ErrClusterGenNamespace = &Errno{Code: 30115, Message: "Failed to gen namespace"}
	ErrUserIdRequired      = &Errno{Code: 50116, Message: "User id parameter required"}
	ErrUserIdFormat        = &Errno{Code: 50117, Message: "User id must be an unsigned integer greater than zero"}

	// application errors for application module request
	ErrApplicationCreate        = &Errno{Code: 40100, Message: "Failed to add app, please try again"}
	ErrApplicationGet           = &Errno{Code: 40101, Message: "Failed to get app, please try again"}
	ErrApplicationDelete        = &Errno{Code: 40102, Message: "Failed to delete application, please try again"}
	ErrApplicationUpdate        = &Errno{Code: 40103, Message: "Update application failed, please try again"}
	ErrBindApplicationClsuter   = &Errno{Code: 40104, Message: "Failed to bind cluster, please try again"}
	ErrPermissionApplication    = &Errno{Code: 40105, Message: "application not found or disabled"}
	ErrPermissionCluster        = &Errno{Code: 40106, Message: "No permission for this cluster"}
	ErrApplicationInstallUpdate = &Errno{
		Code: 40107, Message: "Failed to update app installation status, please try again",
	}
	ErrApplicationJsonContext   = &Errno{Code: 40108, Message: "Application context Unmarshal JSON fail"}
	ErrApplicationNameExist     = &Errno{Code: 40109, Message: "Application name already exist"}
	ErrSensitiveApplicationName = &Errno{Code: 40110, Message: "Application name can't not be 'default.application'"}

	// application-cluster for application-cluster module request
	ErrApplicationBoundClusterList = &Errno{
		Code: 40111, Message: "Failed to get application bound cluster list, please try again",
	}

	// cluster-user errors for cluster-user module request
	ErrBindUserApplicationRepeat = &Errno{
		Code: 50099, Message: "The user has authorized this application",
	}
	ErrBindNameSpaceCreate = &Errno{
		Code: 50101, Message: "Cluster user authorization failed: failed to create namespace",
	}
	ErrBindServiceAccountCreateErr = &Errno{
		Code: 50102, Message: "Cluster user authorization failed: Failed to create ServiceAccount",
	}
	ErrBindRoleCreateErr = &Errno{
		Code: 50103, Message: "Cluster user authorization failed: failed to create a role",
	}
	ErrBindRoleBindingCreateErr = &Errno{
		Code: 50105, Message: "Cluster user authorization failed: failed to create RoleBinding",
	}
	ErrBindSecretGetErr = &Errno{
		Code: 50106, Message: "Cluster user authorization failed: Failed to obtain ServiceAccount Secret",
	}
	ErrBindSecretNameGetErr = &Errno{
		Code: 50107, Message: "Cluster user authorization failed: Failed to obtain ServiceAccount SecretName",
	}
	ErrBindSecretTokenGetErr = &Errno{
		Code: 50108, Message: "Cluster user authorization failed: Failed to obtain ServiceAccount Token",
	}
	ErrBindSecretCAGetErr = &Errno{
		Code: 50109, Message: "Cluster user authorization failed: Failed to obtain ServiceAccount CA",
	}
	ErrBindServiceAccountStructEncodeErr = &Errno{
		Code:    50110,
		Message: "Cluster user authorization failed: encoding ServiceAccount Kubeconfig Json to Yaml failed",
	}
	ErrClusterUserNotFound           = &Errno{Code: 50111, Message: "Dev space has not found"}
	ErrDeletedClusterButDatabaseFail = &Errno{
		Code: 50112, Message: "Cluster namespace has deleted, but database record delete fail",
	}
	ErrDeletedClusterDBButClusterDone = &Errno{
		Code: 50113, Message: "Cluster nocalhost resource has deleted, but cluster record delete fail",
	}
	ErrDeletedClusterDevSpaceDBButClusterDone = &Errno{
		Code: 50114, Message: "Cluster nocalhost develop space has deleted, but space record delete fail",
	}
	ErrDeletedClusterRecord = &Errno{
		Code: 50115, Message: "Delete dev space by application fail, please try again",
	}
	ErrResetDevSpaceFail = &Errno{
		Code: 50116, Message: "reset dev space fail, please try again",
	}
	ErrCreateResourceQuota      = &Errno{Code: 50117, Message: "Initial resource limit failed."}
	ErrDeleteResourceQuota      = &Errno{Code: 50118, Message: "Delete resource limit failed."}
	ErrCreateLimitRange         = &Errno{Code: 50119, Message: "Initial limit range failed."}
	ErrDeleteLimitRange         = &Errno{Code: 50120, Message: "Delete limit range failed."}
	ErrFormatResourceLimitParam = &Errno{Code: 50121, Message: "Incorrect Resource limit parameter."}
	ErrValidateResourceQuota    = &Errno{
		Code:    50122,
		Message: "If quota is enabled in a namespace for compute resources like cpu and memory, must specify requests or limits for those values.",
	}
	ErrAlreadyExist = &Errno{
		Code: 50123, Message: "Current user already authorization current cluster's cluster admin",
	}
	ErrBindServiceAccountKubeConfigJsonEncodeErr = &Errno{
		Code:    50124,
		Message: "Cluster user authorization failed: encoding ServiceAccount Kubeconfig Struct to Json failed",
	}
	ErrShareUserSameAsOwner   = &Errno{Code: 50125, Message: "The share user same as owner"}
	ErrSpaceNameAlreadyExists = &Errno{
		Code:    50126,
		Message: "The space name already exists, please change the space name",
	}

	// cluster-user errors for mesh space
	ErrMeshClusterUserNotFound          = &Errno{Code: 50200, Message: "Base dev space has not found"}
	ErrMeshClusterUserNamespaceNotFound = &Errno{Code: 50201, Message: "Base dev namespace has not found"}
	ErrInitMeshSpaceFailed              = &Errno{Code: 50202, Message: "Failed to initialize mesh space"}
	ErrGetDevSpaceAppInfo               = &Errno{Code: 50203, Message: "Failed to get mesh space info"}
	ErrUpdateMeshSpaceFailed            = &Errno{Code: 50204, Message: "Failed to update mesh space"}
	ErrDeleteTracingHeaderFailed        = &Errno{Code: 50205, Message: "Failed to delete tracing header"}
	ErrUpdateBaseSpace                  = &Errno{Code: 50206, Message: "Base space can't be updated"}
	ErrUseAsBaseSpace                   = &Errno{Code: 50207, Message: "Can't be used as base space"}
	ErrValidateMeshInfo                 = &Errno{Code: 50208, Message: "Incorrect mesh space parameter"}
	ErrMeshInfoRequired                 = &Errno{Code: 50209, Message: "Mesh space parameter required"}
	ErrBaseSpaceReSet                   = &Errno{Code: 50210, Message: "Base space can't be reset"}
	ErrAsBothBaseSpaceAndMeshSpace      = &Errno{
		Code:    50211,
		Message: "Cannot be set as both base space and mesh space",
	}
	ErrIstioNotFound     = &Errno{Code: 50212, Message: "Please ensure the Istio is installed and running in your cluster"}
	ErrUpdateSleepConfig = &Errno{Code: 50213, Message: "Failed to update sleep config"}

	// application-user for application-user module request
	ErrListApplicationUser = &Errno{
		Code: 60000, Message: "Failed to list application_user, please check params and try again",
	}
	ErrInsertApplicationUser = &Errno{
		Code: 60001, Message: "Failed to batch insert application_user, please check params and try again",
	}
	ErrDeleteApplicationUser = &Errno{
		Code: 60002, Message: "Failed to batch delete application_user, please check params and try again",
	}

	// service-account for service-account module request
	ErrServiceAccountCreate = &Errno{
		Code: 70000, Message: "Failed to create service account, please check params and try again",
	}
	ErrNameSpaceCreate = &Errno{
		Code: 70001, Message: "Failed to create namespace, please check params and try again",
	}
	ErrClusterRoleCreate = &Errno{
		Code: 70002, Message: "Failed to create nocalhost common cluster role, please check your cluster and try again",
	}
	ErrRoleBindingCreate = &Errno{
		Code: 70003, Message: "Failed to create role binding, please check your cluster and try again",
	}
	ErrRoleBindingRemove = &Errno{
		Code: 70004, Message: "Failed to remove role binding, please check your cluster and try again",
	}
	ErrClusterRoleBindingCreate = &Errno{
		Code: 70005, Message: "Failed to create cluster role binding, please check your cluster and try again",
	}
	ErrClusterRoleBindingRemove = &Errno{
		Code: 70006, Message: "Failed to remove cluster role binding, please check your cluster and try again",
	}
	ErrRoleBindingDelete = &Errno{
		Code: 70007, Message: "Failed to remove role binding, please check your cluster and try again",
	}
)
