package cluster_user

import (
	"context"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/cluster_user"
)

type ClusterUserService interface {
	Create(ctx context.Context, clusterId uint64, userId uint64) error
	Close()
}

type clusterUserService struct {
	clusterUserRepo cluster_user.ClusterUserRepo
}

func NewClusterUserService() ClusterUserService {
	db := model.GetDB()
	return &clusterUserService{clusterUserRepo: cluster_user.NewApplicationClusterRepo(db)}
}

func (srv *clusterUserService) Create(ctx context.Context, clusterId, userId uint64) error {
	c := model.ClusterUserModel{
		UserId:    userId,
		ClusterId: clusterId,
	}
	_, err := srv.clusterUserRepo.Create(ctx, c)
	if err != nil {
		return errors.Wrapf(err, "create application_cluster")
	}
	return nil
}

func (srv *clusterUserService) Close() {
	srv.clusterUserRepo.Close()
}
