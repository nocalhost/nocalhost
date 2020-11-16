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

package pre_pull

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/pre_pull"
)

type PrePullService interface {
	GetAll(ctx context.Context) ([]string, error)
	Close()
}

type prePullService struct {
	prePullRepo pre_pull.PrePullRepo
}

func NewPrePullService() PrePullService {
	db := model.GetDB()
	return &prePullService{prePullRepo: pre_pull.NewPrePullRepoRepo(db)}
}

func (srv *prePullService) GetAll(ctx context.Context) ([]string, error) {
	result, err := srv.prePullRepo.GetAll(ctx)
	if err != nil {
		return []string{}, nil
	}
	images := make([]string, 0)
	for _, value := range result {
		images = append(images, value.Images)
	}
	return images, nil
}

func (srv *prePullService) Close() {
	srv.prePullRepo.Close()
}
