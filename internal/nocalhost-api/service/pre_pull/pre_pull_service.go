/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pre_pull

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/pre_pull"
)

type PrePull struct {
	prePullRepo *pre_pull.PrePullRepoRepoBase
}

func NewPrePullService() *PrePull {
	db := model.GetDB()
	return &PrePull{prePullRepo: pre_pull.NewPrePullRepoRepo(db)}
}

func (srv *PrePull) GetAll(ctx context.Context) ([]string, error) {
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

func (srv *PrePull) Close() {
	srv.prePullRepo.Close()
}
