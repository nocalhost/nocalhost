package user

import (
	"context"
	uuid "github.com/satori/go.uuid"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/user"
	"nocalhost/pkg/nocalhost-api/pkg/auth"
	"nocalhost/pkg/nocalhost-api/pkg/token"
)

const (
	// MaxID 最大id
	MaxID = 0xffffffffffff
)

// 如果 userService 没有实现 UserService 报错
var _ UserService = (*userService)(nil)

// UserService 用户服务接口定义
// 使用大写对外暴露方法
type UserService interface {
	Register(ctx context.Context, email, password string) error
	EmailLogin(ctx context.Context, email, password string) (tokenStr string, err error)
	GetUserByID(ctx context.Context, id uint64) (*model.UserBaseModel, error)
	GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error)
	GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error)
	UpdateUser(ctx context.Context, id uint64, userMap map[string]interface{}) error
	Close()
}

// userService 用小写的 service 实现接口中定义的方法
type userService struct {
	userRepo user.BaseRepo
}

// NewUserService 实例化一个userService
// 通过 NewService 函数初始化 Service 接口
// 依赖接口，不要依赖实现，面向接口编程
func NewUserService() UserService {
	db := model.GetDB()
	return &userService{
		userRepo: user.NewUserRepo(db),
	}
}

// Register 注册用户
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
	_, err = srv.userRepo.Create(ctx, u)
	if err != nil {
		return errors.Wrapf(err, "create user")
	}
	return nil
}

// EmailLogin 邮箱登录
func (srv *userService) EmailLogin(ctx context.Context, email, password string) (tokenStr string, err error) {
	u, err := srv.GetUserByEmail(ctx, email)
	if err != nil {
		return "", errors.Wrapf(err, "get user info err by email")
	}

	// Compare the login password with the user password.
	err = auth.Compare(u.Password, password)
	if err != nil {
		return "", errors.Wrapf(err, "password compare err")
	}

	// 签发签名 Sign the json web token.
	tokenStr, err = token.Sign(ctx, token.Context{UserID: u.ID, Username: u.Username, Uuid: u.Uuid, Email: u.Email}, "")
	if err != nil {
		return "", errors.Wrapf(err, "gen token sign err")
	}

	return tokenStr, nil
}

// UpdateUser update user info
func (srv *userService) UpdateUser(ctx context.Context, id uint64, userMap map[string]interface{}) error {
	err := srv.userRepo.Update(ctx, id, userMap)

	if err != nil {
		return err
	}

	return nil
}

// GetUserByID 获取用户信息
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

func (srv *userService) GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error) {
	userModel, err := srv.userRepo.GetUserByEmail(ctx, email)
	if err != nil || gorm.IsRecordNotFoundError(err) {
		return userModel, errors.Wrapf(err, "get user info err from db by email: %s", email)
	}

	return userModel, nil
}

// Close close all user repo
func (srv *userService) Close() {
	srv.userRepo.Close()
}
