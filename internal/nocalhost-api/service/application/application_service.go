package application

import (
	"context"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/application"
)

type ApplicationService interface {
	Create(ctx context.Context, context string, status uint8, userId uint64) error
	Get(ctx context.Context, id, userId uint64) (model.ApplicationModel, error)
	GetList(ctx context.Context, userId uint64) ([]*model.ApplicationModel, error)
	Delete(ctx context.Context, userId uint64, id uint64) error
	Update(ctx context.Context, applicationModel *model.ApplicationModel) error
	Close()
}

type applicationService struct {
	applicationRepo application.ApplicationRepo
}

func NewApplicationService() ApplicationService {
	db := model.GetDB()
	return &applicationService{
		applicationRepo: application.NewClusterRepo(db),
	}
}

func (srv *applicationService) Create(ctx context.Context, context string, status uint8, user_id uint64) error {
	c := model.ApplicationModel{
		Context: context,
		UserId:  user_id,
		Status:  status,
	}
	_, err := srv.applicationRepo.Create(ctx, c)
	if err != nil {
		return errors.Wrapf(err, "create application")
	}
	return nil
}

func (srv *applicationService) Get(ctx context.Context, id, userId uint64) (model.ApplicationModel, error) {
	result, err := srv.applicationRepo.Get(ctx, userId, id)
	if err != nil {
		return result, errors.Wrapf(err, "get application")
	}
	return result, nil
}

func (srv *applicationService) GetList(ctx context.Context, userId uint64) ([]*model.ApplicationModel, error) {
	result, err := srv.applicationRepo.GetList(ctx, userId)
	if err != nil {
		return nil, errors.Wrapf(err, "get application")
	}
	return result, nil
}

func (srv *applicationService) Delete(ctx context.Context, userId uint64, id uint64) error {
	err := srv.applicationRepo.Delete(ctx, userId, id)
	if err != nil {
		return errors.Wrapf(err, "delete application error")
	}
	return nil
}

func (srv *applicationService) Update(ctx context.Context, applicationModel *model.ApplicationModel) error {
	err := srv.applicationRepo.Update(ctx, applicationModel)
	if err != nil {
		return errors.Wrapf(err, "update application error")
	}
	return nil
}

func (srv *applicationService) Close() {
	srv.applicationRepo.Close()
}
