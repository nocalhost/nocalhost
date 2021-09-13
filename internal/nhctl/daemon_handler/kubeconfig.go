/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_handler

import (
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/resouce_cache"
	"nocalhost/pkg/nhctl/log"
)

// HandleKubeconfigOperationRequest this method will operate informer cache, remove or add informer
func HandleKubeconfigOperationRequest(request *command.KubeconfigOperationCommand) error {
	defer log.Infof("receive kubeconfig operation: %s, namespace: %v", request.Operation, request.Namespace)
	switch request.Operation {
	case command.OperationAdd:
		return resouce_cache.AddSearcherByKubeconfig(request.KubeConfigBytes, request.Namespace)
	case command.OperationRemove:
		return resouce_cache.RemoveSearcherByKubeconfig(request.KubeConfigBytes, request.Namespace)
	default:
		return nil
	}
}
