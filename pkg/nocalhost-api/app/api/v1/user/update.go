/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package user

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service/cooperator/cluster_scope"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nocalhost-api/app/api/v1/cluster_user"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/auth"
	"strings"
	"sync"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

var importStatusMap sync.Map

// Update Update user information
// @Summary Update user information (including disabled users)
// @Description Update a user by ID，Only status is required
// @Tags Users
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "The user's database id index num"
// @Param user body user.UpdateUserRequest true "Update user info"
// @Success 200 {object} model.UserBaseModel
// @Router /v1/users/{id} [put]
func Update(c *gin.Context) {
	// Get the user id from the url parameter.
	userId := cast.ToUint64(c.Param("id"))

	// Binding the user data.
	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		log.Warnf("bind request param err: %+v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	userMap := model.UserBaseModel{}
	if len(req.Email) > 0 {
		userMap.Email = req.Email
	}
	if len(req.Name) > 0 {
		userMap.Name = req.Name
	}
	if len(req.Password) > 0 {
		pwd, err := auth.Encrypt(req.Password)
		if err != nil {
			api.SendResponse(c, errno.InternalServerError, nil)
			return
		}
		userMap.Password = pwd
	}

	// Only administrator can modify status and isAdmin fields
	isAdmin, _ := c.Get("isAdmin")
	if isAdmin.(uint64) == 1 {
		if req.IsAdmin != nil {
			userMap.IsAdmin = req.IsAdmin
		}
		if req.Status != nil {
			userMap.Status = req.Status
		}
	} else {
		uid, _ := c.Get("userId")
		if cast.ToUint64(uid) != userId {
			api.SendResponse(c, errno.ErrPermissionDenied, nil)
			return
		}
	}

	result, err := service.Svc.UserSvc.UpdateUser(context.TODO(), userId, &userMap)
	if err != nil {
		log.Warnf("[user] update user err, %v", err)
		api.SendResponse(c, errno.InternalServerError, nil)
		return
	}

	api.SendResponse(c, nil, result)
}

func Import(c *gin.Context) {

	userId, err := ginbase.LoginUser(c)
	if err != nil {
		api.SendResponse(c, errno.ErrPermissionDenied, err.Error())
		return
	}

	file, err := c.FormFile("upload")
	if err != nil {
		api.SendResponse(c, errno.ErrUserImport, err.Error())
		return
	}

	fh, err := file.Open()
	if err != nil {
		api.SendResponse(c, errno.ErrUserImport, err.Error())
		return
	}

	excFile, err := excelize.OpenReader(fh)
	if err != nil {
		api.SendResponse(c, errno.ErrUserImport, nil)
		return
	}

	// Only get the first sheet
	sheetName := excFile.GetSheetName(1)
	rs := excFile.GetRows(sheetName)
	if len(rs) < 3 {
		api.SendResponse(c, errno.ErrUserImport, fmt.Sprintf("sheet %s no use data found, len is %d", sheetName, len(rs)))
		return
	}

	uu, _ := uuid.NewUUID()
	task := struct {
		TaskId string `json:"taskId"`
	}{TaskId: uu.String()}

	importStatusMap.Store(task.TaskId, &ImportTaskStatus{})

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("%v", err)
			}
		}()
		rs = rs[2:]
		importUsers(c, rs, task.TaskId, userId)
	}()

	api.SendResponse(c, nil, &task)
	return
}

type ItemStatus struct {
	Success            bool
	Email              string
	Username           string
	CooperatorDevSpace string
	ViewerDevSpace     string
	ErrInfo            string
}

type ImportTaskStatus struct {
	Process float32 // 0.0 ~ 1.0
	Items   []ItemStatus
	ErrInfo string
}

func ImportStatus(c *gin.Context) {
	taskId := c.Param("id")
	is, ok := importStatusMap.Load(taskId)
	if !ok {
		api.SendResponse(c, errno.ErrUserImport, fmt.Sprintf("task %s not found", taskId))
		return
	}
	api.SendResponse(c, nil, is)
}

func importUsers(ctx *gin.Context, data [][]string, uuid string, loginUserId uint64) {
	its, _ := importStatusMap.Load(uuid)
	if its == nil {
		log.Error("no import status found in importStatusMap")
		return
	}

	itss, _ := its.(*ImportTaskStatus)

	cum := model.ClusterUserModel{}
	devSpaces, err := cluster_user.DoList(&cum, loginUserId, true, false)
	if err != nil {
		log.Error(err.Error())
	}

	for i, datum := range data {
		if len(datum) < 2 {
			itss.Process = (float32(i) + 1.0) / float32(len(data))
			itss.Items = append(itss.Items, ItemStatus{
				Success: false,
				ErrInfo: "Email or UserName can not be nil",
			})
			continue
		}

		is := ItemStatus{}

		if datum[0] == "" {
			itss.Process = (float32(i) + 1.0) / float32(len(data))
			is.ErrInfo = "Email can not be nil"
			itss.Items = append(itss.Items, is)
			continue
		}
		is.Email = datum[0]

		if datum[1] == "" {
			itss.Process = (float32(i) + 1.0) / float32(len(data))
			is.ErrInfo = "UserName can not be nil"
			itss.Items = append(itss.Items, is)
			continue
		}
		is.Username = datum[1]

		var cooperatorDevSpace []string
		var viewerDevSpace []string
		if datum[2] != "" {
			cooperatorDevSpace = strings.Split(datum[2], "\n")
		}
		is.CooperatorDevSpace = strings.Join(cooperatorDevSpace, ",")

		if datum[3] != "" {
			viewerDevSpace = strings.Split(datum[3], "\n")
		}
		is.ViewerDevSpace = strings.Join(viewerDevSpace, ",")

		usr, err := service.Svc.UserSvc.CreateOrUpdateUserByEmail(context.TODO(), datum[0], datum[1], "", 0, false)
		if err != nil {
			itss.Process = (float32(i) + 1.0) / float32(len(data))
			is.ErrInfo = err.Error()
			if strings.Contains(err.Error(), "record not found") {
				is.ErrInfo = fmt.Sprintf("User %s import failed: %s", is.Username, err.Error())
			}
			itss.Items = append(itss.Items, is)
			continue
		}

		// Associate cooperate dev space
		// Find cooperate dev space
		var errStr string
		for _, s := range cooperatorDevSpace {
			strs := strings.Split(s, "/")
			if len(strs) < 2 {
				errStr = errStr + " " + s + " is invalid format"
				continue
			}
			var spaceFound bool
			for _, space := range devSpaces {
				if space.ClusterName == strs[0] && space.SpaceName == strs[1] {
					spaceFound = true
					cu, errn := cluster_user.LoginUserHasModifyPermissionToSomeDevSpace(ctx, space.ID)
					if errn != nil {
						errStr = errStr + ",user has not permission to " + s
						break
					}
					if cu.IsClusterAdmin() {
						if err := cluster_scope.AsCooperator(cu.ClusterId, cu.UserId, usr.ID); err != nil {
							errStr = errStr + fmt.Sprintf(",Error while adding %d as cluster cooperator", usr.ID)
						}
					} else if err := ns_scope.AsCooperator(cu.ClusterId, usr.ID, cu.Namespace); err != nil {
						errStr = errStr + fmt.Sprintf(",Error while adding %d as cooperator", usr.ID)
					}
					break
				}
			}
			if !spaceFound {
				errStr = errStr + ",DevSpace " + s + " is not found"
			}
		}

		if errStr != "" {
			errStr = strings.TrimLeft(errStr, ",")
			itss.Process = (float32(i) + 1.0) / float32(len(data))
			is.Success = false
			is.ErrInfo = errStr
			itss.Items = append(itss.Items, is)
			continue
		}

		for _, s := range viewerDevSpace {
			strs := strings.Split(s, "/")
			if len(strs) < 2 {
				errStr = errStr + "," + s + " is invalid format"
				continue
			}
			var spaceFound bool
			for _, space := range devSpaces {
				if space.ClusterName == strs[0] && space.SpaceName == strs[1] {
					spaceFound = true
					cu, errn := cluster_user.LoginUserHasModifyPermissionToSomeDevSpace(ctx, space.ID)
					if errn != nil {
						errStr = errStr + ",user has not permission to " + s
						break
					}
					if cu.IsClusterAdmin() {
						if err := cluster_scope.AsViewer(cu.ClusterId, cu.UserId, usr.ID); err != nil {
							errStr = errStr + fmt.Sprintf(",Error while adding %d as cluster viewer", usr.ID)
						}
					} else if err := ns_scope.AsViewer(cu.ClusterId, usr.ID, cu.Namespace); err != nil {
						errStr = errStr + fmt.Sprintf(",Error while adding %d as viewer", usr.ID)
					}
					break
				}
			}
			if !spaceFound {
				errStr = errStr + "," + s + " is not found"
			}
		}

		if errStr != "" {
			errStr = strings.TrimLeft(errStr, ",")
			itss.Process = (float32(i) + 1.0) / float32(len(data))
			is.Success = false
			itss.Items = append(itss.Items, is)
			is.ErrInfo = errStr
			continue
		}

		itss.Process = (float32(i) + 1.0) / float32(len(data))
		is.Success = true
		itss.Items = append(itss.Items, is)
	}
}
