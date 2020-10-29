package cluster

import (
	"context"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

type ClusterRepo interface {
	Create(ctx context.Context, user model.ClusterModel) (id uint64, err error)
	Get(ctx context.Context, clusterId uint64, userId uint64) (model.ClusterModel, error)
	Close()
}

type clusterBaseRepo struct {
	db *gorm.DB
}

func NewClusterRepo(db *gorm.DB) ClusterRepo {
	return &clusterBaseRepo{
		db: db,
	}
}

func (repo *clusterBaseRepo) Create(ctx context.Context, cluster model.ClusterModel) (id uint64, err error) {
	err = repo.db.Create(&cluster).Error
	if err != nil {
		return 0, errors.Wrap(err, "[cluster_repo] create user err")
	}

	return cluster.ID, nil
}

func (repo *clusterBaseRepo) Get(ctx context.Context, clusterId uint64, userId uint64) (model.ClusterModel, error) {
	cluster := model.ClusterModel{}
	if result := repo.db.Where("id=? and user_id=?", clusterId, userId).First(&cluster); result.Error != nil {
		log.Warnf("[cluster_repo] get cluster for user: %v id: %v error", userId, clusterId)
		return cluster, result.Error
	}
	return cluster, nil
}

// Close close db
func (repo *clusterBaseRepo) Close() {
	repo.db.Close()
}
