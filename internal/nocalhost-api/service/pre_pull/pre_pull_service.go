/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
