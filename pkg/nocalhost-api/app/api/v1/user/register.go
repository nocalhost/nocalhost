package user

import (
	"github.com/gin-gonic/gin"

	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Register 注册
// @Summary 注册
// @Description 用户注册
// @Tags 用户
// @Produce  json
// @Param register body user.RegisterRequest true "Reg user info"
// @Success 200 {string} json "{"code":0,"message":"OK","data":{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"}}"
// @Router /register [post]
func Register(c *gin.Context) {
	// Binding the data with the u struct.
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("register bind param err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	log.Infof("register req: %#v", req)
	// check param
	if req.Email == "" || req.Password == "" {
		log.Warnf("params is empty: %v", req)
		api.SendResponse(c, errno.ErrParam, nil)
		return
	}

	// 两次密码是否正确
	if req.Password != req.ConfirmPassword {
		log.Warnf("twice password is not same")
		api.SendResponse(c, errno.ErrTwicePasswordNotMatch, nil)
		return
	}

	err := service.Svc.UserSvc().Register(c, req.Email, req.Password)
	if err != nil {
		log.Warnf("register err: %v", err)
		api.SendResponse(c, errno.ErrRegisterFailed, nil)
		return
	}

	api.SendResponse(c, nil, nil)
}
