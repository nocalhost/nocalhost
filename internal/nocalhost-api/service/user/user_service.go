/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package user

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/cache"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/jinzhu/gorm"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/user"
	"nocalhost/pkg/nocalhost-api/pkg/auth"
	"nocalhost/pkg/nocalhost-api/pkg/token"
)

const (
	// MaxID
	MaxID = 0xffffffffffff
)

var _ UserService = (*userService)(nil)

// UserService
type UserService interface {
	Create(ctx context.Context, email, password, name string, status uint64, isAdmin uint64) (
		model.UserBaseModel, error,
	)
	Delete(ctx context.Context, id uint64) error
	Register(ctx context.Context, email, password string) error
	EmailLogin(ctx context.Context, email, password string) (tokenStr, refreshToken string, err error)
	GetUserByID(ctx context.Context, id uint64) (*model.UserBaseModel, error)
	GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error)
	GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error)

	CreateOrGetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error)
	UpdateUser(ctx context.Context, id uint64, user *model.UserBaseModel) (*model.UserBaseModel, error)
	GetUserList(ctx context.Context) ([]*model.UserList, error)
	UpdateServiceAccountName(ctx context.Context, id uint64, saName string) error

	GetCache(id uint64) (model.UserBaseModel, error)
	GetCacheBySa(sa string) (model.UserBaseModel, error)
	Close()
}

type userService struct {
	userRepo user.BaseRepo
}

func NewUserService() UserService {
	db := model.GetDB()
	return &userService{
		userRepo: user.NewUserRepo(db),
	}
}

func (srv *userService) Evict(id uint64) {
	c := cache.Module(cache.USER)

	value, err := c.Value(id)
	if err != nil {
		return
	}

	clusterModel := value.Data().(*model.UserBaseModel)
	_, _ = c.Delete(id)
	_, _ = c.Delete(clusterModel.SaName)
}

func (srv *userService) GetCacheBySa(sa string) (model.UserBaseModel, error) {
	c := cache.Module(cache.USER)
	value, err := c.Value(sa)
	if err == nil {
		clusterModel := value.Data().(*model.UserBaseModel)
		return *clusterModel, nil
	}

	result, err := srv.userRepo.GetUserBySa(context.TODO(), sa)
	if err != nil {
		return model.UserBaseModel{}, errors.Wrapf(err, "get user")
	}

	c.Add(result.ID, cache.OUT_OF_DATE, result)
	c.Add(result.SaName, cache.OUT_OF_DATE, result)
	return *result, nil
}

func (srv *userService) GetCache(id uint64) (model.UserBaseModel, error) {
	c := cache.Module(cache.USER)
	value, err := c.Value(id)
	if err == nil {
		clusterModel := value.Data().(*model.UserBaseModel)
		return *clusterModel, nil
	}

	result, err := srv.GetUserByID(context.TODO(), id)
	if err != nil {
		return model.UserBaseModel{}, errors.Wrapf(err, "get user")
	}

	c.Add(result.ID, cache.OUT_OF_DATE, result)
	c.Add(result.SaName, cache.OUT_OF_DATE, result)
	return *result, nil
}

func (srv *userService) GetUserList(ctx context.Context) ([]*model.UserList, error) {
	return srv.userRepo.GetUserList(ctx)
}

// Delete
func (srv *userService) Delete(ctx context.Context, id uint64) error {
	err := srv.userRepo.Delete(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "delete user fail")
	}
	srv.Evict(id)
	return nil
}

// Create
func (srv *userService) Create(
	ctx context.Context, email, password, name string, status uint64, isAdmin uint64,
) (model.UserBaseModel, error) {
	pwd, err := auth.Encrypt(password)
	u := model.UserBaseModel{
		SaName:    model.GenerateSaName(),
		Password:  pwd,
		Email:     email,
		Name:      name,
		Status:    &status,
		IsAdmin:   &isAdmin,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
		Uuid:      uuid.NewV4().String(),
	}
	if err != nil {
		return u, errors.Wrapf(err, "encrypt password err")
	}
	result, err := srv.userRepo.Create(ctx, u)
	if err != nil {
		return result, errors.Wrapf(err, "create user")
	}

	srv.Evict(result.ID)
	return result, nil
}

// Register
func (srv *userService) Register(ctx context.Context, email, password string) error {
	pwd, err := auth.Encrypt(password)
	if err != nil {
		return errors.Wrapf(err, "encrypt password err")
	}

	u := model.UserBaseModel{
		Password:  pwd,
		Email:     email,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
		Uuid:      uuid.NewV4().String(),
	}
	result, err := srv.userRepo.Create(ctx, u)
	if err != nil {
		return errors.Wrapf(err, "create user")
	}
	srv.Evict(result.ID)
	return nil
}

// EmailLogin
func (srv *userService) EmailLogin(ctx context.Context, email, password string) (tokenStr, refreshToken string, err error) {
	u, err := srv.GetUserByEmail(ctx, email)
	if err != nil {
		err = errors.Wrapf(err, "get user info err by email")
		return
	}

	// Compare the login password with the user password.
	err = auth.Compare(u.Password, password)
	if err != nil {
		err = errors.Wrapf(err, "password compare err")
		return
	}

	if *u.Status == 0 {
		err = errors.New("user not allow")
		return
	}

	return token.Sign(
		token.Context{UserID: u.ID, Username: u.Username, Uuid: u.Uuid, Email: u.Email, IsAdmin: *u.IsAdmin},
	)
}

// UpdateUser update user info
func (srv *userService) UpdateUser(ctx context.Context, id uint64, user *model.UserBaseModel) (
	*model.UserBaseModel, error,
) {
	_, err := srv.userRepo.Update(ctx, id, user)

	if err != nil {
		return user, err
	}

	srv.Evict(id)
	return user, nil
}

// GetUserByID
func (srv *userService) GetUserByID(ctx context.Context, id uint64) (*model.UserBaseModel, error) {
	userModel, err := srv.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return userModel, errors.Wrapf(err, "get user info err from db by id: %d", id)
	}

	return userModel, nil
}

func (srv *userService) GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error) {
	userModel, err := srv.userRepo.GetUserByPhone(ctx, phone)
	if err != nil || gorm.IsRecordNotFoundError(err) {
		return userModel, errors.Wrapf(err, "get user info err from db by phone: %d", phone)
	}

	return userModel, nil
}

func (srv *userService) CreateOrGetUserByEmail(ctx context.Context, userEmail string) (*model.UserBaseModel, error) {
	userPointer, err := srv.GetUserByEmail(ctx, userEmail)

	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			userName := userEmail[:strings.Index(userEmail, "@")]

			userCreated, err := srv.Create(
				ctx, userEmail, "123456", userName, _const.UintEnable, _const.UintDisable,
			)

			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("Fail to create user by email %s", userEmail))
			}

			return &userCreated, nil
		} else {
			return nil, errors.Wrap(err, fmt.Sprintf("Fail to get user by email %s", userEmail))
		}
	} else {
		return userPointer, nil
	}
}

func (srv *userService) GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error) {
	userModel, err := srv.userRepo.GetUserByEmail(ctx, email)
	if err != nil || gorm.IsRecordNotFoundError(err) {
		return userModel, err
	}

	return userModel, nil
}

func (srv *userService) UpdateServiceAccountName(ctx context.Context, id uint64, saName string) error {
	defer srv.Evict(id)
	return srv.userRepo.UpdateServiceAccountName(ctx, id, saName)
}

// Close close all user repo
func (srv *userService) Close() {
	srv.userRepo.Close()
}
