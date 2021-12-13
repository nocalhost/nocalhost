/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/viper"
)

const (
	JWT_SECRET         = "jwt_secret"
	JWT_REFRESH_SECRET = "jwt_refresh_secret"
)

var (
	// ErrMissingHeader means the `Authorization` header was empty.
	ErrMissingHeader = errors.New("the length of the `Authorization` header is zero")
)

// Context is the context of the JSON web token.
type Context struct {
	UserID   uint64
	Username string
	Uuid     string
	Email    string
	IsAdmin  uint64
}

// secretFunc validates the secret format.
func secretFunc(secret string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		// Make sure the `alg` is what we except.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}

		return []byte(secret), nil
	}
}

func RefreshFromRequest(c *gin.Context) (neoSignToken, neoRefreshToken string, err error) {
	header := c.Request.Header.Get("Authorization")

	if len(header) == 0 {
		return "", "", ErrMissingHeader
	}

	var t string
	// Parse the header to get the token part.
	_, err = fmt.Sscanf(header, "Bearer %s", &t)
	if err != nil {
		fmt.Printf("fmt.Sscanf err: %+v", err)
	}

	rt := c.GetHeader("Reraeb")

	// frontend can not pass the header key with upper-case????
	if rt == "" {
		rt = c.GetHeader("reraeb")
	}

	return refreshToken(t, rt)
}

func refreshToken(signToken, refreshToken string) (neoSignToken, neoRefreshToken string, err error) {
	// Load the jwt secret from the Gin config if the secret does not specified.
	secret := ""
	secret = viper.GetString(JWT_SECRET)

	signTokenCtx, err := Parse(signToken, secret, true)
	if err != nil {
		return "", "", err
	}

	secret = viper.GetString(JWT_REFRESH_SECRET)
	refreshTokenCtx, err := Parse(refreshToken, secret, false)
	if err != nil {
		return "", "", err
	}

	if signTokenCtx.IsAdmin == refreshTokenCtx.IsAdmin &&
		signTokenCtx.UserID == refreshTokenCtx.UserID &&
		signTokenCtx.Email == refreshTokenCtx.Email &&
		signTokenCtx.Uuid == refreshTokenCtx.Uuid &&
		signTokenCtx.Username == refreshTokenCtx.Username {

		return Sign(*refreshTokenCtx)
	}

	return "", "", errors.New("Current refreshToken is not the corresponding token of signToken ")
}

// Parse validates the token with the specified secret,
// and returns the context if the token was valid.
func Parse(tokenString string, secret string, skipValidation bool) (*Context, error) {
	ctx := &Context{}
	token, err := jwt.Parse(tokenString, secretFunc(secret))

	if !skipValidation && err != nil {
		return ctx, err
	} else if token == nil {
		return ctx, errors.New("Token can't not be parsed correctly ")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && (skipValidation || token.Valid) {
		ctx.UserID = uint64(claims["user_id"].(float64))
		ctx.Username = claims["username"].(string)
		ctx.Uuid = claims["uuid"].(string)
		ctx.Email = claims["email"].(string)
		ctx.IsAdmin = uint64(claims["is_admin"].(float64))
		return ctx, nil

		// Other errors.
	} else {
		return ctx, err
	}
}

// ParseRequest gets the token from the header and
// pass it to the Parse function to parses the token.
func ParseRequest(c *gin.Context) (*Context, error) {
	header := c.Request.Header.Get("Authorization")

	// Load the jwt secret from config
	secret := viper.GetString(JWT_SECRET)

	if len(header) == 0 {
		return &Context{}, ErrMissingHeader
	}

	var t string
	// Parse the header to get the token part.
	_, err := fmt.Sscanf(header, "Bearer %s", &t)
	if err != nil {
		fmt.Printf("fmt.Sscanf err: %+v", err)
	}
	return Parse(t, secret, false)
}

func Sign(ctx Context) (tokenString, refreshToken string, err error) {
	if signToken, err := SignToken(ctx); err != nil {
		return "", "", err
	} else {
		if refreshToken, err := SignRefreshToken(ctx); err != nil {
			return "", "", err
		} else {
			return signToken, refreshToken, nil
		}
	}
}

func SignToken(ctx Context) (tokenString string, err error) {
	// Load the jwt secret from the Gin config if the secret does not specified.
	secret := ""
	secret = viper.GetString("jwt_secret")

	return sign(ctx, secret, 1)
}

func SignRefreshToken(ctx Context) (tokenString string, err error) {
	// Load the jwt secret from the Gin config if the secret does not specified.
	secret := ""
	secret = viper.GetString("jwt_refresh_secret")

	return sign(ctx, secret, 14)
}

// Sign signs the context with the specified secret.
func sign(c Context, secret string, expDays int) (tokenString string, err error) {

	// The token content.
	// iss: （Issuer）
	// iat: （Issued At）
	// exp: （Expiration Time）
	// aud: （Audience）
	// sub: （Subject）
	// nbf: （Not Before）
	// jti: （JWT ID）
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  c.UserID,
			"username": c.Username,
			"uuid":     c.Uuid,
			"email":    c.Email,
			"is_admin": c.IsAdmin,
			"nbf":      time.Now().Unix(),
			"iat":      time.Now().Unix(),
			"exp":      time.Now().AddDate(0, 0, expDays).Unix(),
		},
	)
	// Sign the token with the specified secret.
	tokenString, err = token.SignedString([]byte(secret))
	return
}
