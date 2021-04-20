/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package middleware

import (
	"errors"
	"time"

	"nocalhost/pkg/nocalhost-api/pkg/sign"

	"github.com/gin-gonic/gin"

	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// SignMd5Middleware md5
func SignMd5Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sn, err := verifySign(c)
		if err != nil {
			api.SendResponse(c, errno.InternalServerError, nil)
			c.Abort()
			return
		}

		if sn != nil {
			api.SendResponse(c, errno.ErrSignParam, sn)
			c.Abort()
			return
		}

		c.Next()
	}
}

// verifySign
func verifySign(c *gin.Context) (map[string]string, error) {
	requestUri := c.Request.RequestURI
	// Initial Verify validator
	verifier := sign.NewVerifier()
	sn := verifier.GetSign()

	// Assuming that the verification parameters are read from RequestUri
	if err := verifier.ParseQuery(requestUri); nil != err {
		return nil, err
	}

	// Check whether the timestamp has timed out。
	if err := verifier.CheckTimeStamp(); nil != err {
		return nil, err
	}

	// Verify signature
	localSign := genSign()
	if sn == "" || sn != localSign {
		return nil, errors.New("sign error")
	}

	return nil, nil
}

// genSign
func genSign() string {
	// TODO: Read configuration
	signer := sign.NewSignerMd5()
	signer.SetAppID("123456")
	signer.SetTimeStamp(time.Now().Unix())
	signer.SetNonceStr("supertempstr")
	signer.SetAppSecretWrapBody("20200711")

	return signer.GetSignature()
}
