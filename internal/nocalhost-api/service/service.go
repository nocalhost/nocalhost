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

package service

import (
	"context"
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

	// adapt cluster_user to application_user
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

		_, err := s.applicationUserSvc.GetByApplicationIdAndUserId(
			context.TODO(),
			applicationId,
			userId,
		)
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
				log.Errorf(
					"Error while upgrade %s dep versions, can't not accessing cluster",
					clusterItem.ClusterName,
				)
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
