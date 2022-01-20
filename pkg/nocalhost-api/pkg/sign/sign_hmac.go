/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package sign

import (
	"crypto/hmac"
	"crypto/sha1"
)

// HmacSign hmac
func HmacSign(secretKey, body string) []byte {
	m := hmac.New(sha1.New, []byte(secretKey))
	m.Write([]byte(body))
	return m.Sum(nil)
}
