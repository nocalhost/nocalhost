/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_scope

import (
	"fmt"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

func cooperatorCRB(saName string) string {
	return fmt.Sprintf("%s:%s:%s", "nocalhost", saName, constCooperator)
}

func viewerCRB(saName string) string {
	return fmt.Sprintf("%s:%s:%s", "nocalhost", saName, constViewer)
}

func AsCooperator(clusterId, fromUserId, shareUserId uint64) error {
	clientGo, saName, err := service.Svc.PrepareServiceAccountAndClientGo(clusterId, fromUserId)
	if err != nil {
		return err
	}
	if saName == "" {
		return nil
	}

	u, err := service.Svc.UserSvc.GetCache(shareUserId)
	if err != nil {
		log.Error(err)
		return errno.ErrUserNotFound
	}

	if err := service.CreateOrUpdateClusterRoleBindingINE(
		clientGo, u.SaName,
		_const.NocalhostDefaultSaNs,
		cooperatorCRB(saName),
		_const.NocalhostCooperatorRoleName,
	); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingCreate
	}

	return RemoveFromViewer(clusterId, fromUserId, shareUserId)
}

func RemoveFromCooperator(clusterId, fromUserId, shareUserId uint64) error {
	clientGo, saName, err := service.Svc.PrepareServiceAccountAndClientGo(clusterId, fromUserId)
	if err != nil {
		return err
	}
	if saName == "" {
		return nil
	}

	u, err := service.Svc.UserSvc.GetCache(shareUserId)
	if err != nil {
		log.Error(err)
		return errno.ErrUserNotFound
	}

	if err := service.RemoveClusterRoleBindingIfPresent(
		clientGo, u.SaName,
		_const.NocalhostDefaultSaNs,
		cooperatorCRB(saName),
	); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingCreate
	}

	return nil
}

func AsViewer(clusterId, fromUserId, shareUserId uint64) error {
	clientGo, saName, err := service.Svc.PrepareServiceAccountAndClientGo(clusterId, fromUserId)
	if err != nil {
		return err
	}
	if saName == "" {
		return nil
	}

	u, err := service.Svc.UserSvc.GetCache(shareUserId)
	if err != nil {
		log.Error(err)
		return errno.ErrUserNotFound
	}

	if err := service.CreateOrUpdateClusterRoleBindingINE(
		clientGo, u.SaName,
		_const.NocalhostDefaultSaNs,
		viewerCRB(saName),
		_const.NocalhostViewerRoleName,
	); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingCreate
	}

	return RemoveFromCooperator(clusterId, fromUserId, shareUserId)
}

func RemoveAllFromViewer(clusterId, fromUserId uint64) error {
	clientGo, saName, err := service.Svc.PrepareServiceAccountAndClientGo(clusterId, fromUserId)
	if err != nil {
		return err
	}
	if saName == "" {
		return nil
	}

	for _, sa := range ViewSas(clusterId, fromUserId) {
		if err := service.RemoveClusterRoleBindingIfPresent(
			clientGo, sa,
			_const.NocalhostDefaultSaNs,
			viewerCRB(saName),
		); err != nil {
			log.Error(err)
			return errno.ErrRoleBindingRemove
		}
	}

	return nil
}

func RemoveAllFromCooperator(clusterId, fromUserId uint64) error {
	clientGo, saName, err := service.Svc.PrepareServiceAccountAndClientGo(clusterId, fromUserId)
	if err != nil {
		return err
	}
	if saName == "" {
		return nil
	}

	for _, sa := range CoopSas(clusterId, fromUserId) {
		if err := service.RemoveClusterRoleBindingIfPresent(
			clientGo, sa,
			_const.NocalhostDefaultSaNs,
			cooperatorCRB(saName),
		); err != nil {
			log.Error(err)
			return errno.ErrRoleBindingRemove
		}
	}

	return nil
}

func RemoveFromViewer(clusterId, fromUserId, shareUserId uint64) error {
	clientGo, saName, err := service.Svc.PrepareServiceAccountAndClientGo(clusterId, fromUserId)
	if err != nil {
		return err
	}
	if saName == "" {
		return nil
	}

	u, err := service.Svc.UserSvc.GetCache(shareUserId)
	if err != nil {
		log.Error(err)
		return errno.ErrUserNotFound
	}

	if err := service.RemoveClusterRoleBindingIfPresent(
		clientGo, u.SaName,
		_const.NocalhostDefaultSaNs,
		viewerCRB(saName),
	); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingCreate
	}

	return nil
}
