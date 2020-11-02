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
	"nocalhost/internal/nocalhost-api/service/application"
	"nocalhost/internal/nocalhost-api/service/application_cluster"
	"nocalhost/internal/nocalhost-api/service/cluster"
	"nocalhost/internal/nocalhost-api/service/user"
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
}

// New init service
func New() (s *Service) {
	s = &Service{
		userSvc:               user.NewUserService(),
		clusterSvc:            cluster.NewClusterService(),
		applicationSvc:        application.NewApplicationService(),
		applicationClusterSvc: application_cluster.NewApplicationClusterService(),
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

// Ping service
func (s *Service) Ping() error {
	return nil
}

// Close service
func (s *Service) Close() {
	s.userSvc.Close()
}
