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

package service

import (
	"context"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/application"
	"nocalhost/internal/nocalhost-api/service/application_cluster"
	"nocalhost/internal/nocalhost-api/service/cluster"
	"nocalhost/internal/nocalhost-api/service/cluster_user"
	"nocalhost/internal/nocalhost-api/service/pre_pull"
	"nocalhost/internal/nocalhost-api/service/user"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
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
	}

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

// Ping service
func (s *Service) Ping() error {
	return nil
}

// Close service
func (s *Service) Close() {
	s.userSvc.Close()
}

func (s *Service) updateAllRole() error {
	cu := model.ClusterUserModel{
	}

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

			wg.Done()
		}()
	}

	wg.Wait()
	return nil
}
