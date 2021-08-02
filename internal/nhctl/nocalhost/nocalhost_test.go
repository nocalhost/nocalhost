/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package nocalhost

import (
	"fmt"
	"github.com/pkg/errors"
	"testing"
)

func TestError(t *testing.T) {
	err := errors.Wrap(CreatePvcFailed, "")
	if errors.Is(err, CreatePvcFailed) {
		fmt.Println(true)
	} else {
		fmt.Println(false)
	}

}
