package user

import (
	"context"

	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Get 获取用户信息
// @Summary 通过用户id获取用户信息
// @Description Get an user by user id
// @Tags 用户
// @Accept  json
// @Produce  json
// @Param id path string true "用户id"
// @Success 200 {object} model.UserInfo "用户信息"
// @Router /users [get]
func Get(c *gin.Context) {
	log.Info("Get function called.")

	userID, _ := c.Get("userId")
	if userID == 0 {
		api.SendResponse(c, errno.ErrParam, nil)
		return
	}

	// Get the user by the `user_id` from the database.
	u, err := service.Svc.UserSvc().GetUserByID(context.TODO(), userID.(uint64))
	if err != nil {
		log.Warnf("get user info err: %v", err)
		api.SendResponse(c, errno.ErrUserNotFound, nil)
		return
	}

	api.SendResponse(c, nil, u)
}
