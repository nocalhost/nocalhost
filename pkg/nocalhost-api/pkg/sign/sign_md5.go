/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package sign

import "crypto/md5"

func Md5Sign(_, body string) []byte {
	m := md5.New()
	m.Write([]byte(body))
	return m.Sum(nil)
}
