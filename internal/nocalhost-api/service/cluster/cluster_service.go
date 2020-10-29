package cluster

import (
	"context"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/cluster"
)

type ClusterService interface {
	Create(ctx context.Context, name, marks, kubeconfig string, userId uint64) error
	Get(ctx context.Context, id, userId uint64) (model.ClusterModel, error)
	Close()
}

type clusterService struct {
	clusterRepo cluster.ClusterRepo
}

func NewClusterService() ClusterService {
	db := model.GetDB()
	return &clusterService{
		clusterRepo: cluster.NewClusterRepo(db),
	}
}

func (srv *clusterService) Create(ctx context.Context, name, marks, kubeconfig string, userId uint64) error {
	c := model.ClusterModel{
		Name:       name,
		Marks:      marks,
		UserId:     userId,
		KubeConfig: kubeconfig,
	}
	_, err := srv.clusterRepo.Create(ctx, c)
	if err != nil {
		return errors.Wrapf(err, "create cluster")
	}
	return nil
}

func (srv *clusterService) Get(ctx context.Context, id, userId uint64) (model.ClusterModel, error) {
	result, err := srv.clusterRepo.Get(ctx, id, userId)
	if err != nil {
		return result, errors.Wrapf(err, "get cluster")
	}
	return result, nil
}

func (srv *clusterService) Close() {
	srv.clusterRepo.Close()
}
