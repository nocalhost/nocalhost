/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package service

import (
	"context"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/application"
	"nocalhost/internal/nocalhost-api/service/application_cluster"
	"nocalhost/internal/nocalhost-api/service/application_user"
	"nocalhost/internal/nocalhost-api/service/cluster"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
	"nocalhost/internal/nocalhost-api/service/pre_pull"
	"nocalhost/internal/nocalhost-api/service/user"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
	"strings"
	"sync"
)

var (
	// Svc global service var
	Svc *Service
)

// Service struct
type Service struct {
	userSvc               user.UserService
	clusterSvc            cluster.ClusterService
	applicationSvc        application.ApplicationService
	applicationClusterSvc application_cluster.ApplicationClusterService
	clusterUserSvc        cluster_user.ClusterUserService
	prePullSvc            pre_pull.PrePullService
	applicationUserSvc    application_user.ApplicationUserService
}

// New init service
func New() (s *Service) {
	s = &Service{
		userSvc:               user.NewUserService(),
		clusterSvc:            cluster.NewClusterService(),
		applicationSvc:        application.NewApplicationService(),
		applicationClusterSvc: application_cluster.NewApplicationClusterService(),
		clusterUserSvc:        cluster_user.NewClusterUserService(),
		prePullSvc:            pre_pull.NewPrePullService(),
		applicationUserSvc:    application_user.NewApplicationUserService(),
	}

	if global.ServiceInitial == "true" {
		s.init()
	} else {
		log.Infof("Service Initial is disable(enable in build with env: SERVICE_INITIAL=true)")
	}

	s.dataMigrate()
	return s
}

// UserSvc return user service
func (s *Service) UserSvc() user.UserService {
	return s.userSvc
}

func (s *Service) ClusterSvc() cluster.ClusterService {
	return s.clusterSvc
}

func (s *Service) ApplicationSvc() application.ApplicationService {
	return s.applicationSvc
}

func (s *Service) ApplicationClusterSvc() application_cluster.ApplicationClusterService {
	return s.applicationClusterSvc
}

func (s *Service) ClusterUser() cluster_user.ClusterUserService {
	return s.clusterUserSvc
}

func (s *Service) PrePull() pre_pull.PrePullService {
	return s.prePullSvc
}

func (s *Service) ApplicationUser() application_user.ApplicationUserService {
	return s.applicationUserSvc
}

// Ping service
func (s *Service) Ping() error {
	return nil
}

// Close service
func (s *Service) Close() {
	s.userSvc.Close()
}

func (s *Service) dataMigrate() {
	log.Info("Migrate data if needed... ")

	// old version of nocalhost-api did not have saname for user
	s.generateServiceAccountNameForUser()

	// adapt cluster_user to application_user
	s.migrateClusterUseToApplicationUser()

	// adapt devSpace to Sa -> RoleBinding
	s.migrateClusterUseToRoleBinding()
}

func (s *Service) generateServiceAccountNameForUser() {
	list, err := s.userSvc.GetUserList(context.TODO())
	if err != nil {
		log.Infof("Error while generate user sa: %+v", err)
	}

	for _, u := range list {
		u := u
		go func() {
			if u.SaName == "" {
				if err := s.userSvc.UpdateServiceAccountName(context.TODO(), u.ID, model.GenerateSaName()); err != nil {
					log.Infof("Error while generate user %d's sa: %+v", u.ID, err)
				}
			}
		}()
	}
}

func (s *Service) migrateClusterUseToRoleBinding() {
	list, err := s.clusterUserSvc.GetList(context.TODO(), model.ClusterUserModel{})
	if err != nil {
		log.Infof("Error while migrate data: %+v", err)
	}

	for _, clusterUser := range list {
		_ = s.AuthorizeNsToUser(clusterUser.ClusterId, clusterUser.UserId, clusterUser.Namespace)
	}
}

func (s *Service) migrateClusterUseToApplicationUser() {
	list, err := s.clusterUserSvc.GetList(context.TODO(), model.ClusterUserModel{})
	if err != nil {
		log.Infof("Error while migrate data: %+v", err)
	}

	if list == nil {
		return
	}

	for _, userModel := range list {
		applicationId := userModel.ApplicationId
		userId := userModel.UserId

		if applicationId <= 0 {
			continue
		}

		_, err := s.applicationUserSvc.GetByApplicationIdAndUserId(context.TODO(), applicationId, userId)
		if err != nil && strings.Contains(err.Error(), "record not found") {
			err := s.ApplicationUser().BatchInsert(context.TODO(), applicationId, []uint64{userId})
			if err != nil {
				log.Infof("Error while migrate data[BatchInsert]: %+v", err)
			}
		}
	}
}

func (s *Service) init() {
	log.Infof("Upgrading cluster...")

	err := s.upgradeAllClusters()
	if err != nil {
		log.Errorf("Error while upgrading dep: %s", err)
	}

	log.Infof("Upgrading role...")

	err = s.updateAllRole()
	if err != nil {
		log.Errorf("Error while updating role: %s", err)
	}

	// todo update all rolebinding
	if err := s.updateAllRoleBinding(); err != nil {
		log.Errorf("Error while updating role binding: %s", err)
	}
}

// Upgrade all cluster's versions of nocalhost-dep according to nocalhost-api's versions.
func (s *Service) upgradeAllClusters() error {
	result, _ := s.ClusterSvc().GetList(context.TODO())

	wg := sync.WaitGroup{}
	wg.Add(len(result))

	for _, clusterItem := range result {
		clusterItem := clusterItem
		go func() {
			defer wg.Done()
			log.Infof("Checking cluster %s's upgradation if needed ", clusterItem.ClusterName)

			goClient, err := clientgo.NewAdminGoClient([]byte(clusterItem.KubeConfig))

			if err != nil {
				log.Errorf("Error while upgrade %s dep versions, can't not accessing cluster", clusterItem.ClusterName)
				return
			}

			isUpgrade, err := setupcluster.NewSetUpCluster(goClient).UpgradeCluster()

			if err != nil {
				log.Errorf("Error while upgrade dep ", err)
				return
			}

			if isUpgrade {
				log.Infof("Cluster %s's upgrade success ", clusterItem.ClusterName)
			} else {
				log.Infof("Cluster %s's has been up to date ", clusterItem.ClusterName)
			}
		}()
	}

	wg.Wait()
	return nil
}

// elder version of nocalhost may not has nocalhost label
func (s *Service) updateAllRoleBinding() error {
	cu := model.ClusterUserModel{}

	var results []*model.ClusterUserModel
	results, err := s.ClusterUser().GetList(context.TODO(), cu)

	if err != nil {
		return err
	}

	group := map[uint64][]*model.ClusterUserModel{}
	for _, result := range results {
		if _, ok := group[result.ClusterId]; !ok {
			group[result.ClusterId] = []*model.ClusterUserModel{}
		}

		list := group[result.ClusterId]
		list = append(list, result)
		group[result.ClusterId] = list
	}

	wg := sync.WaitGroup{}
	wg.Add(len(group))

	go func() {
		defer wg.Done()
		for clusterId, cus := range group {

			clusterModel, err := s.ClusterSvc().GetCache(clusterId)
			if err != nil {
				log.Error(err)
				return
			}

			goClient, err := clientgo.NewAdminGoClient([]byte(clusterModel.KubeConfig))
			if err != nil {
				log.Error(err)
				return
			}

			err = goClient.UpdateClusterRoleBindingForNocalhostLabel(_const.NocalhostDevRoleName)

			wgInner := sync.WaitGroup{}
			wg.Add(len(cus))

			for _, result := range cus {
				result := result
				go func() {
					defer wgInner.Done()

					err = goClient.UpdateRoleBindingForNocalhostLabel(
						_const.NocalhostDevRoleName, result.Namespace,
					)
					if err != nil {
						log.Error(err)
						return
					}
				}()
			}

			wgInner.Wait()
		}
	}()

	wg.Wait()
	return nil
}

func (s *Service) updateAllRole() error {
	cu := model.ClusterUserModel{}

	var results []*model.ClusterUserModel
	results, err := s.ClusterUser().GetList(context.TODO(), cu)

	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(len(results))

	for _, result := range results {

		result := result
		go func() {
			defer wg.Done()

			clusterModel, err := s.ClusterSvc().GetCache(result.ClusterId)
			if err != nil {
				return
			}

			goClient, err := clientgo.NewAdminGoClient([]byte(clusterModel.KubeConfig))
			if err != nil {
				return
			}

			err = goClient.UpdateRole(
				_const.NocalhostDevRoleName, result.Namespace, []rbacv1.PolicyRule{
					{
						Verbs:     []string{"*"},
						Resources: []string{"*"},
						APIGroups: []string{"*"},
					},
				},
			)
			if err != nil {
				return
			}
		}()
	}

	wg.Wait()
	return nil
}

func (s *Service) PrepareServiceAccountAndClientGo(clusterId, userId uint64) (
	clientGo *clientgo.GoClient, saName string, err error,
) {
	cl, err := s.ClusterSvc().GetCache(clusterId)
	if err != nil {
		log.Error(err)
		err = errno.ErrClusterNotFound
		return
	}

	// new client go
	clientGo, err = clientgo.NewAdminGoClient([]byte(cl.KubeConfig))
	if err != nil {
		log.Error(err)
		err = errno.ErrBindServiceAccountKubeConfigJsonEncodeErr
		return
	}

	u, err := s.UserSvc().GetCache(userId)
	if err != nil {
		log.Error(err)
		err = errno.ErrUserNotFound
		return
	}

	if err = createOrUpdateServiceAccountINE(clientGo, u.SaName, _const.NocalhostDefaultSaNs); err != nil {
		log.Error(err)
		err = errno.ErrServiceAccountCreate
		return
	}

	if err = createClusterAdminRoleINE(clientGo); err != nil {
		log.Error(err)
		err = errno.ErrClusterRoleCreate
		return
	}

	saName = u.SaName
	return
}

func (s *Service) AuthorizeNsToUser(clusterId, userId uint64, ns string) error {
	clientGo, saName, err := s.PrepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}

	if err := CreateNamespaceINE(clientGo, ns); err != nil {
		log.Error(err)
		return errno.ErrNameSpaceCreate
	}

	if err := CreateOrUpdateRoleBindingINE(
		clientGo, ns, saName,
		_const.NocalhostDefaultSaNs,
		_const.NocalhostDefaultRoleBinding,
		_const.NocalhostDevRoleName,
	); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingCreate
	}

	return nil
}

func (s *Service) UnAuthorizeNsToUser(clusterId, userId uint64, ns string) error {
	clientGo, saName, err := s.PrepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}

	if err := RemoveRoleBindingIfPresent(
		clientGo, ns, saName,
		_const.NocalhostDefaultSaNs,
		_const.NocalhostDefaultRoleBinding,
	); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingRemove
	}

	return nil
}

func (s *Service) AuthorizeClusterToUser(clusterId, userId uint64) error {
	clientGo, saName, err := s.PrepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}

	if err := CreateOrUpdateClusterRoleBindingINE(
		clientGo, saName,
		_const.NocalhostDefaultSaNs,
		_const.NocalhostDefaultRoleBinding,
		_const.NocalhostDevRoleName,
	); err != nil {
		log.Error(err)
		return errno.ErrClusterRoleBindingCreate
	}

	return nil
}

func (s *Service) UnAuthorizeClusterToUser(clusterId, userId uint64) error {
	clientGo, saName, err := s.PrepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}

	if err := RemoveClusterRoleBindingIfPresent(
		clientGo, saName,
		_const.NocalhostDefaultSaNs,
		_const.NocalhostDefaultRoleBinding,
	); err != nil && !k8serrors.IsNotFound(err) {
		log.Error(err)
		return errno.ErrClusterRoleBindingRemove
	}

	return nil
}

func createOrUpdateServiceAccountINE(client *clientgo.GoClient, saName string, saNs string) error {
	if _, err := client.CreateServiceAccount(saName, saNs); err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func CreateNamespaceINE(client *clientgo.GoClient, ns string) error {
	if _, err := client.CreateNS(ns, ""); err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createClusterAdminRoleINE(client *clientgo.GoClient) error {
	wg := sync.WaitGroup{}
	wg.Add(3)

	errChan := make(chan error, 3)
	doneChan := make(chan interface{}, 1)
	defer close(errChan)
	defer close(doneChan)

	go func() {
		defer wg.Done()
		if _, err := client.CreateClusterRole(

			// TODO the elder version of _const.NocalhostDevRoleName role may not has nocalhost label
			_const.NocalhostDevRoleName, []rbacv1.PolicyRule{
				{
					Verbs:     []string{"*"},
					Resources: []string{"*"},
					APIGroups: []string{"*"},
				},
			},
		); err != nil && !k8serrors.IsAlreadyExists(err) {
			errChan <- err
		}
	}()

	go func() {
		defer wg.Done()
		if _, err := client.CreateClusterRole(
			_const.NocalhostCooperatorRoleName, []rbacv1.PolicyRule{
				{
					Verbs:     []string{"*"},
					Resources: []string{"*"},
					APIGroups: []string{"*"},
				},
			},
		); err != nil && !k8serrors.IsAlreadyExists(err) {
			errChan <- err
		}
	}()

	go func() {
		defer wg.Done()
		if _, err := client.CreateClusterRole(
			_const.NocalhostViewerRoleName, []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get", "list", "watch"},
					Resources: []string{"*"},
					APIGroups: []string{"*"},
				},
			},
		); err != nil && !k8serrors.IsAlreadyExists(err) {
			errChan <- err
		}
	}()

	go func() {
		wg.Wait()
		doneChan <- "done"
	}()

	select {
	case err := <-errChan:
		return err
	case <-doneChan:
		return nil
	}
}

// nocalhost use nocalhost-saName for role binding storage container
// and nocalhost create a role binding for each dev space
func CreateOrUpdateRoleBindingINE(client *clientgo.GoClient, ns, saName, saNs, rb, role string) error {
	return client.AppendRoleBinding(rb, ns, role, saName, saNs)
}

func RemoveRoleBindingIfPresent(client *clientgo.GoClient, ns, saName, saNs, rb string) error {
	return client.RemoveRoleBinding(rb, ns, saName, saNs)
}

func CreateOrUpdateClusterRoleBindingINE(client *clientgo.GoClient, saName, saNs, crb, role string) error {
	// refresh service account to notify dep to update the cache
	defer client.RefreshServiceAccount(saName, saNs)
	return client.AppendClusterRoleBinding(crb, role, saName, saNs)
}

func RemoveClusterRoleBindingIfPresent(client *clientgo.GoClient, saName, saNs, crb string) error {
	// refresh service account to notify dep to update the cache
	defer client.RefreshServiceAccount(saName, saNs)
	return client.RemoveClusterRoleBinding(crb, saName, saNs)
}
