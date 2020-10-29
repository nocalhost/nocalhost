package application_cluster

import (
	"context"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
)

type ApplicationClusterRepo interface {
	Create(ctx context.Context, model model.ApplicationClusterModel) (uint64, error)
	Close()
}

type applicationClusterRepo struct {
	db *gorm.DB
}

func NewApplicationClusterRepo(db *gorm.DB) ApplicationClusterRepo {
	return &applicationClusterRepo{
		db: db,
	}
}

func (repo *applicationClusterRepo) Create(ctx context.Context, clusterModel model.ApplicationClusterModel) (id uint64, err error) {
	err = repo.db.Create(&clusterModel).Error
	if err != nil {
		return 0, errors.Wrap(err, "[application_cluster_repo] create application_cluster error")
	}

	return clusterModel.ID, nil
}

// Close close db
func (repo *applicationClusterRepo) Close() {
	repo.db.Close()
}
