/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package user

import (
	"nocalhost/internal/nocalhost-api/service/ldap"
	"nocalhost/pkg/nocalhost-api/pkg/token"
	"strings"

	"github.com/gin-gonic/gin"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

func RefreshToken(c *gin.Context) {
	t, rt, err := token.RefreshFromRequest(c)

	if err != nil {
		if strings.Contains(err.Error(), "allow") {
			api.SendResponse(c, errno.ErrUserNotAllow, nil)
			return
		}
		log.Warnf("refresh token err: %v", err)
		api.SendResponse(c, errno.RefreshTokenInvalidOrNotMatch, nil)
		return
	}

	api.SendResponse(
		c, nil, model.Token{
			Token:        t,
			RefreshToken: rt,
		},
	)
}

// Login Web and plug-in login
// @Summary Web and plug-in login
// @Description Web and plug-in login
// @Tags Users
// @Produce  json
// @Param login body user.LoginCredentials true "Login user info"
// @Success 200 {string} json "{"code":0,"message":"OK","data":{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"}}"
// @Router /v1/login [post]
func Login(c *gin.Context) {
	// Binding the data with the u struct.
	var req LoginCredentials
	if err := c.Bind(&req); err != nil {
		log.Warnf("email login bind param err: %v", err)
		api.SendResponse(c, errno.ErrBind, err)
		return
	}

	log.Infof("login req %#v", req)

	// check param
	if req.Email == "" || req.Password == "" {
		log.Warnf("email or password is empty: %v", req)
		api.SendResponse(c, errno.ErrParam, nil)
		return
	}

	// By default, “From” is not passed in web login,
	// and ordinary users are prohibited from logging in to the web interface
	usr, err := service.Svc.UserSvc.GetUserByEmail(c, req.Email)
	if err != nil {
		api.SendResponse(c, errno.ErrEmailOrPassword, nil)
		return
	}

	ldapBindPass := usr.LdapDN != "" &&
		ldap.DoBindForLDAP("ldap://localhost", false, false, usr.LdapDN, req.Password) == nil

	if !ldapBindPass {
		err = service.Svc.UserSvc.EmailLogin(c, req.Email, req.Password)
		if err != nil {
			if strings.Contains(err.Error(), "allow") {
				api.SendResponse(c, errno.ErrUserNotAllow, nil)
				return
			}

			log.Warnf("email login err: %v", err)
			api.SendResponse(c, errno.ErrEmailOrPassword, nil)
			return
		}
	}

	sign, refreshToken, err := token.Sign(
		token.Context{UserID: usr.ID, Username: usr.Username, Uuid: usr.Uuid, Email: usr.Email, IsAdmin: *usr.IsAdmin},
	)

	if err != nil {
		log.Warnf("login err, fail to create token: %v", err)
		api.SendResponse(c, errno.InternalServerError, nil)
		return
	}

	api.SendResponse(
		c, nil, model.Token{
			Token:        sign,
			RefreshToken: refreshToken,
		},
	)
}
