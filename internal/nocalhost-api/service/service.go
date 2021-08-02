/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package service

import (
	"context"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	Svc                         *Service
	NocalhostDefaultSaNs        = "default"
	NocalhostDefaultRoleBinding = "nocalhost-role-binding"
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

			clusterModel, err := s.ClusterSvc().Get(context.TODO(), result.ClusterId)
			if err != nil {
				return
			}

			goClient, err := clientgo.NewAdminGoClient([]byte(clusterModel.KubeConfig))
			if err != nil {
				return
			}

			err = goClient.UpdateRole(global.NocalhostDevRoleName, result.Namespace)
			if err != nil {
				return
			}
		}()
	}

	wg.Wait()
	return nil
}

func (s *Service) prepareServiceAccountAndClientGo(clusterId, userId uint64) (
	clientGo *clientgo.GoClient, saName string, err error,
) {
	cl, err := s.ClusterSvc().Get(context.TODO(), clusterId)
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

	u, err := s.UserSvc().GetUserByID(context.TODO(), userId)
	if err != nil {
		log.Error(err)
		err = errno.ErrUserNotFound
		return
	}

	if err = createOrUpdateServiceAccountINE(clientGo, u.SaName, NocalhostDefaultSaNs); err != nil {
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
	clientGo, saName, err := s.prepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}

	if err := createNamespaceINE(clientGo, ns); err != nil {
		log.Error(err)
		return errno.ErrNameSpaceCreate
	}

	if err := createOrUpdateRoleBindingINE(clientGo, ns, saName, NocalhostDefaultSaNs); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingCreate
	}

	return nil
}

func (s *Service) UnAuthorizeNsToUser(clusterId, userId uint64, ns string) error {
	clientGo, saName, err := s.prepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}

	if err := removeRoleBindingIfPresent(clientGo, ns, saName, NocalhostDefaultSaNs); err != nil {
		log.Error(err)
		return errno.ErrRoleBindingRemove
	}

	return nil
}

func (s *Service) AuthorizeClusterToUser(clusterId, userId uint64) error {
	clientGo, saName, err := s.prepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}

	if err := createOrUpdateClusterRoleBindingINE(clientGo, saName, NocalhostDefaultSaNs); err != nil {
		log.Error(err)
		return errno.ErrClusterRoleBindingCreate
	}

	return nil
}

func (s *Service) UnAuthorizeClusterToUser(clusterId, userId uint64) error {
	clientGo, saName, err := s.prepareServiceAccountAndClientGo(clusterId, userId)
	if err != nil {
		return err
	}

	if err := removeClusterRoleBindingIfPresent(clientGo, saName, NocalhostDefaultSaNs); err != nil {
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

func createNamespaceINE(client *clientgo.GoClient, ns string) error {
	if _, err := client.CreateNS(ns, ""); err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createClusterAdminRoleINE(client *clientgo.GoClient) error {
	if _, err := client.CreateClusterRole(global.NocalhostDevRoleName); err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// nocalhost use nocalhost-saName for role binding storage container
// and nocalhost create a role binding for each dev space
func createOrUpdateRoleBindingINE(client *clientgo.GoClient, ns, saName, saNs string) error {
	return client.AppendRoleBinding(NocalhostDefaultRoleBinding, ns, global.NocalhostDevRoleName, saName, saNs)
}

func removeRoleBindingIfPresent(client *clientgo.GoClient, ns, saName, saNs string) error {
	return client.RemoveRoleBinding(NocalhostDefaultRoleBinding, ns, saName, saNs)
}

func createOrUpdateClusterRoleBindingINE(client *clientgo.GoClient, saName, saNs string) error {
	// refresh service account to notify dep to update the cache
	defer client.RefreshServiceAccount(saName, saNs)

	return client.AppendClusterRoleBinding(NocalhostDefaultRoleBinding, global.NocalhostDevRoleName, saName, saNs)
}

func removeClusterRoleBindingIfPresent(client *clientgo.GoClient, saName, saNs string) error {
	// refresh service account to notify dep to update the cache
	defer client.RefreshServiceAccount(saName, saNs)

	return client.RemoveClusterRoleBinding(NocalhostDefaultRoleBinding, saName, saNs)
}
