package application_cluster

import (
	"context"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/application_cluster"
)

type ApplicationClusterService interface {
	Create(ctx context.Context, applicationId uint64, clusterId uint64, userId uint64) error
	Close()
}

type applicationClusterService struct {
	applicationClusterRepo application_cluster.ApplicationClusterRepo
}

func NewApplicationClusterService() ApplicationClusterService {
	db := model.GetDB()
	return &applicationClusterService{applicationClusterRepo: application_cluster.NewApplicationClusterRepo(db)}
}

func (srv *applicationClusterService) Create(ctx context.Context, applicationId uint64, clusterId uint64, userId uint64) error {
	c := model.ApplicationClusterModel{
		ApplicationId: applicationId,
		ClusterId:     clusterId,
	}
	_, err := srv.applicationClusterRepo.Create(ctx, c)
	if err != nil {
		return errors.Wrapf(err, "create application_cluster")
	}
	return nil
}

func (srv *applicationClusterService) Close() {
	srv.applicationClusterRepo.Close()
}
