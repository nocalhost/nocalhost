package cluster_user

import (
	"context"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
)

type ClusterUserRepo interface {
	Create(ctx context.Context, model model.ClusterUserModel) (uint64, error)
	Close()
}

type clusterUserRepo struct {
	db *gorm.DB
}

func NewApplicationClusterRepo(db *gorm.DB) ClusterUserRepo {
	return &clusterUserRepo{
		db: db,
	}
}

func (repo *clusterUserRepo) Create(ctx context.Context, model model.ClusterUserModel) (id uint64, err error) {
	err = repo.db.Create(&model).Error
	if err != nil {
		return 0, errors.Wrap(err, "[application_cluster_repo] create application_cluster error")
	}

	return model.ID, nil
}

// Close close db
func (repo *clusterUserRepo) Close() {
	repo.db.Close()
}
