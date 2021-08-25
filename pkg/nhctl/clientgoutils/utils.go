/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

import "github.com/pkg/errors"

//func getHomePath() string {
//	u, err := user.Current()
//	if err == nil {
//		return u.HomeDir
//	}
//	return ""
//}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func MustI(err error, s string) {
	if err != nil {
		panic(errors.Wrap(err, s))
	}
}
