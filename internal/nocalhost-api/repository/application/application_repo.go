package application

import (
	"context"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

type ApplicationRepo interface {
	Create(ctx context.Context, application model.ApplicationModel) (uint64, error)
	Get(ctx context.Context, userId uint64, id uint64) (model.ApplicationModel, error)
	GetList(ctx context.Context, userId uint64) ([]*model.ApplicationModel, error)
	Delete(ctx context.Context, userId uint64, id uint64) error
	Update(ctx context.Context, applicationModel *model.ApplicationModel) error
	Close()
}

type applicationRepo struct {
	db *gorm.DB
}

func NewClusterRepo(db *gorm.DB) ApplicationRepo {
	return &applicationRepo{
		db: db,
	}
}

func (repo *applicationRepo) Create(ctx context.Context, application model.ApplicationModel) (id uint64, err error) {
	err = repo.db.Create(&application).Error
	if err != nil {
		return 0, errors.Wrap(err, "[application_repo] create application err")
	}

	return application.ID, nil
}

func (repo *applicationRepo) Get(ctx context.Context, userId uint64, id uint64) (model.ApplicationModel, error) {
	// 有坑
	// 这里传入的是 Struct 类型，获取不到数据时会抛出 Error
	// 如果传入的是 make([]*model.ApplicationModel,0) Slice 类型，那么获取不到数据永远不会抛出 Error
	application := model.ApplicationModel{}
	result := repo.db.Where("user_id=? and status=1 and id=?", userId, id).First(&application)
	if err := result.Error; err != nil {
		log.Warnf("[application_repo] get application for user: %v id: %v error", userId, id)
		return application, err
	}
	return application, nil
}

func (repo *applicationRepo) GetList(ctx context.Context, userId uint64) ([]*model.ApplicationModel, error) {
	applicationList := make([]*model.ApplicationModel, 0)
	result := repo.db.Where("user_id=? and status=1", userId).Find(&applicationList)

	if err := result.Error; err != nil {
		log.Warnf("[application_repo] get application for user %s err", userId)
		return nil, err
	}

	return applicationList, nil
}

func (repo *applicationRepo) Delete(ctx context.Context, userId uint64, id uint64) error {
	application := model.ApplicationModel{
		ID: id,
	}
	if result := repo.db.Where("user_id=?", userId).Delete(&application); result.RowsAffected > 0 {
		return nil
	}
	return errors.New("application delete denied")
}

func (repo *applicationRepo) Update(ctx context.Context, applicationModel *model.ApplicationModel) error {
	_, err := repo.Get(ctx, applicationModel.UserId, applicationModel.ID)
	if err != nil {
		return errors.Wrap(err, "[application_repo] get application denied")
	}
	affectRow := repo.db.Save(&applicationModel).RowsAffected
	if affectRow > 0 {
		return nil
	}
	return errors.Wrap(err, "[application_repo] update application err")
}

// Close close db
func (repo *applicationRepo) Close() {
	repo.db.Close()
}
