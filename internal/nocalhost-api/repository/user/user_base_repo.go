/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */
package user

import (
	"context"
	"github.com/pkg/errors"
	"time"

	"github.com/jinzhu/gorm"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// BaseRepo
//type BaseRepo interface {
//	Create(ctx context.Context, user model.UserBaseModel) (model.UserBaseModel, error)
//	Creates(ctx context.Context, users []*model.UserBaseModel) error
//
//	Update(ctx context.Context, id uint64, userMap *model.UserBaseModel) (*model.UserBaseModel, error)
//	Delete(ctx context.Context, id uint64) error
//	GetUserByID(ctx context.Context, id uint64) (*model.UserBaseModel, error)
//	GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error)
//	GetUserBySa(ctx context.Context, sa string) (*model.UserBaseModel, error)
//	GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error)
//	UpdateServiceAccountName(ctx context.Context, id uint64, saName string) error
//	ListStartById(ctx context.Context, idStart uint64, limit uint64) ([]*model.UserBaseModel, error)
//	GetUserPageable(ctx context.Context, page, limit int) ([]*model.UserBaseModel, error)
//	GetUserHasNotSa(ctx context.Context) ([]*model.UserBaseModel, error)
//	Close()
//
//	DeleteOutOfSyncLdapUser(ldapGen uint64) (int64, error)
//	UpdateUsersLdapGen(list []*model.UserBaseModel, gen uint64) bool
//
//	// deprecated
//	GetUserList(ctx context.Context) ([]*model.UserList, error)
//}

// UserBaseRepo
type UserBaseRepo struct {
	db *gorm.DB
}

// NewUserRepo
func NewUserRepo(db *gorm.DB) *UserBaseRepo {
	return &UserBaseRepo{
		db: db,
	}
}

func (repo *UserBaseRepo) UpdateUsersLdapGen(list []*model.UserBaseModel, gen uint64) bool {
	if len(list) == 0 {
		return true
	}

	sql := "UPDATE users SET ldap_gen=? WHERE id in ("
	args := []interface{}{gen}
	for _, baseModel := range list {
		sql += "?,"
		args = append(args, baseModel.ID)
	}
	sql = sql[:len(sql)-1]
	sql += ")"
	return repo.db.Exec(sql, args...).RowsAffected > 0
}

func (repo *UserBaseRepo) DeleteOutOfSyncLdapUser(ldapGen uint64) (int64, error) {
	exec := repo.db.Exec("UPDATE users SET status=0 WHERE ldap_gen < ? and ldap_dn is not null", ldapGen)
	return exec.RowsAffected, exec.Error
}

// deprecated
func (repo *UserBaseRepo) GetUserList(ctx context.Context) ([]*model.UserList, error) {
	var result []*model.UserList
	repo.db.Raw(
		"select u.id as id,u.name as name,u.sa_name as sa_name,u.email as email," +
			"count(distinct cu.id) as cluster_count,u.status as status," +
			" u.is_admin as is_admin from users as u left join clusters_users as cu on cu.user_id=u.id " +
			"where u.deleted_at is null and cu.deleted_at is null group by u.id",
	).
		Scan(&result)
	return result, nil
}

func (repo *UserBaseRepo) GetUserHasNotSa(ctx context.Context) ([]*model.UserBaseModel, error) {
	var result []*model.UserBaseModel
	repo.db.
		Raw("select * from users where sa_name is null").
		Scan(&result)
	return result, nil
}

func (repo *UserBaseRepo) GetUserPageable(ctx context.Context, page, limit int) ([]*model.UserBaseModel, error) {
	var result []*model.UserBaseModel

	raw := repo.db.
		Raw("select * from users where deleted_at is null")

	if page > 0 && limit > 0 {
		raw = raw.Offset((page - 1) * limit).
			Limit(limit)
	}

	raw.Scan(&result)
	return result, nil
}

func (repo *UserBaseRepo) ListStartById(ctx context.Context, idStart uint64, limit uint64) ([]*model.UserBaseModel, error) {
	var result []*model.UserBaseModel
	repo.db.Raw(
		"SELECT * FROM users "+
			"WHERE id > ? ORDER BY id LIMIT ?", idStart, limit,
	).Scan(&result)
	return result, nil
}

// Delete
func (repo *UserBaseRepo) Delete(ctx context.Context, id uint64) error {
	users := model.UserBaseModel{
		ID: id,
	}
	if result := repo.db.Where("id=?", id).Unscoped().Delete(&users); result.RowsAffected > 0 {
		return nil
	}
	return errors.New("user delete fail")
}

// Create
func (repo *UserBaseRepo) Create(ctx context.Context, user model.UserBaseModel) (model.UserBaseModel, error) {
	err := repo.db.Create(&user).Error
	if err != nil {
		return user, errors.Wrap(err, "[user_repo] create user err")
	}

	return user, nil
}

// Creates
func (repo *UserBaseRepo) Creates(ctx context.Context, users []*model.UserBaseModel) error {
	if len(users) == 0 {
		return nil
	}
	sql := "INSERT INTO users (uuid,username,name,password,avatar,phone,email,is_admin,status,deleted_at," +
		"created_at,updated_at,sa_name,cluster_admin,ldap_dn,ldap_gen) VALUES"

	args := make([]interface{}, 0)
	for _, user := range users {
		sql += "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?),"
		args = append(args, user.Uuid)
		args = append(args, user.Username)
		args = append(args, user.Name)
		args = append(args, user.Password)
		args = append(args, user.Avatar)
		args = append(args, user.Phone)
		args = append(args, user.Email)
		args = append(args, user.IsAdmin)
		args = append(args, user.Status)
		args = append(args, user.DeletedAt)
		args = append(args, user.CreatedAt)
		args = append(args, user.UpdatedAt)
		args = append(args, user.SaName)
		args = append(args, user.ClusterAdmin)
		args = append(args, user.LdapDN)
		args = append(args, user.LdapGen)
	}

	sql = sql[:len(sql)-1]

	return repo.db.Exec(sql, args...).Error
}

// Update
func (repo *UserBaseRepo) Update(ctx context.Context, id uint64, userMap *model.UserBaseModel) (
	*model.UserBaseModel, error,
) {
	user, err := repo.GetUserByID(ctx, id)
	if user.ID != id {
		return user, errors.New("[user_repo] user is not exsit.")
	}
	if err != nil {
		return user, errors.Wrap(err, "[user_repo] update user data err")
	}
	err = repo.db.Model(&user).Updates(&userMap).Where("id=?", id).Error
	if err != nil {
		return user, errors.Wrap(err, "[user_repo] update user data error")
	}
	return user, nil
}

// Update
func (repo *UserBaseRepo) UpdateServiceAccountName(ctx context.Context, id uint64, saName string) error {
	if err := repo.db.Exec("UPDATE users SET sa_name = ? WHERE id = ?", saName, id).Error; err != nil {
		return err
	}

	return nil
}

// GetUserByID
func (repo *UserBaseRepo) GetUserByID(ctx context.Context, uid uint64) (userBase *model.UserBaseModel, err error) {
	start := time.Now()
	defer func() {
		log.Infof("[repo.user_base] get user by uid: %d cost: %d ns", uid, time.Now().Sub(start).Nanoseconds())
	}()

	data := new(model.UserBaseModel)

	err = repo.db.First(data, uid).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.Wrap(err, "[repo.user_base] get user data err")
	}
	return data, nil
}

// GetUserBySa
func (repo *UserBaseRepo) GetUserBySa(ctx context.Context, sa string) (*model.UserBaseModel, error) {
	user := model.UserBaseModel{}
	err := repo.db.Where("sa_name = ?", sa).First(&user).Error
	if err != nil {
		return nil, errors.Wrap(err, "[user_repo] get user err by sa_name")
	}

	return &user, nil
}

// GetUserByPhone
func (repo *UserBaseRepo) GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error) {
	user := model.UserBaseModel{}
	err := repo.db.Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, errors.Wrap(err, "[user_repo] get user err by phone")
	}

	return &user, nil
}

// GetUserByEmail
func (repo *UserBaseRepo) GetUserByEmail(ctx context.Context, phone string) (*model.UserBaseModel, error) {
	user := model.UserBaseModel{}
	err := repo.db.Where("email = ?", phone).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Close close db
func (repo *UserBaseRepo) Close() {
	repo.db.Close()
}
