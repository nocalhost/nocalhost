/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package util

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// MessageToAny marshals src into a new Any instance.
func MessageToAny(src proto.Message) *anypb.Any {
	dst := new(anypb.Any)
	if err := dst.MarshalFrom(src); err != nil {
		fmt.Println("marshals src into a new any instance error:", err)
		return nil
	}
	return dst
}
