/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ns_scope

import (
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

func AsCooperator(clusterId, userId uint64, ns string) error {
	clientGo, saName, err := service.Svc.PrepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}
	if saName == "" {
		return nil
	}

	if err := service.CreateNamespaceINE(clientGo, ns); err != nil {
		log.Error(err)
		return errno.ErrNameSpaceCreate
	}

	if err := service.CreateOrUpdateRoleBindingINE(
		clientGo, ns, saName,
		_const.NocalhostDefaultSaNs,
		_const.NocalhostCooperatorRoleBinding,
		_const.NocalhostCooperatorRoleName,
	); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingCreate
	}

	return RemoveFromViewer(clusterId, userId, ns)
}

func RemoveFromCooperator(clusterId, userId uint64, ns string) error {
	clientGo, saName, err := service.Svc.PrepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}
	if saName == "" {
		return nil
	}

	if err := service.RemoveRoleBindingIfPresent(
		clientGo, ns, saName,
		_const.NocalhostDefaultSaNs,
		_const.NocalhostCooperatorRoleBinding,
	); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingRemove
	}

	return nil
}

func AsViewer(clusterId, userId uint64, ns string) error {
	clientGo, saName, err := service.Svc.PrepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}

	if err := service.CreateNamespaceINE(clientGo, ns); err != nil {
		log.Error(err)
		return errno.ErrNameSpaceCreate
	}

	if err := service.CreateOrUpdateRoleBindingINE(
		clientGo, ns, saName,
		_const.NocalhostDefaultSaNs,
		_const.NocalhostViewerRoleBinding,
		_const.NocalhostViewerRoleName,
	); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingCreate
	}

	return RemoveFromCooperator(clusterId, userId, ns)
}

func RemoveFromViewer(clusterId, userId uint64, ns string) error {
	clientGo, saName, err := service.Svc.PrepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}

	if err := service.RemoveRoleBindingIfPresent(
		clientGo, ns, saName,
		_const.NocalhostDefaultSaNs,
		_const.NocalhostViewerRoleBinding,
	); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingRemove
	}

	return nil
}
